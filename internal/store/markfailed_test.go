package store

import (
	"context"
	"os"
	"testing"
	"time"
)

func getTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := OpenDB(dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	// The tests claim jobs by ordering, so the table must start empty.
	if _, err := db.ExecContext(ctx, "TRUNCATE jobs RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate jobs: %v", err)
	}
	return New(db)
}

// TestMarkFailedFencedByWorker covers the fencing contract: a worker that
// lost its lease (job reaped and re-claimed by another worker) must not be
// able to modify the row the new owner is processing.
func TestMarkFailedFencedByWorker(t *testing.T) {
	s := getTestStore(t)
	ctx := context.Background()

	id, err := s.Enqueue(ctx, "send_email", []byte(`{}`), 1, 5)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// Worker A claims with a short lease, then loses it (reaped).
	if _, err := s.Claim(ctx, "worker-a", 600, 20*time.Millisecond); err != nil {
		t.Fatalf("claim A: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := s.Reap(ctx); err != nil {
		t.Fatalf("reap: %v", err)
	}

	// Worker B claims the reaped job.
	b, err := s.Claim(ctx, "worker-b", 600, 30*time.Second)
	if err != nil {
		t.Fatalf("claim B: %v", err)
	}
	if b == nil || b.ID != id {
		t.Fatalf("expected worker B to claim job %d", id)
	}

	// Worker A finishes late with an error; its MarkFailed must be rejected.
	if err := s.MarkFailed(ctx, id, "worker-a", "stale failure", 1, 5, time.Second); err == nil {
		t.Fatal("expected stale MarkFailed to be rejected")
	}

	job, err := s.GetJob(ctx, id)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.Status != "running" {
		t.Errorf("stale MarkFailed changed status: got %q, want running", job.Status)
	}
	if job.LockedBy == nil || *job.LockedBy != "worker-b" {
		t.Errorf("stale MarkFailed changed lock: got %v, want worker-b", job.LockedBy)
	}
	if job.LastError != nil {
		t.Errorf("stale MarkFailed set last_error: %v", *job.LastError)
	}
}
