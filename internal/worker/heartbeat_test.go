package worker

import (
	"context"
	"os"
	"testing"
	"time"

	"kulee/internal/store"
)

func getTestStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := store.OpenDB(dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	ctx := context.Background()
	if err := store.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	// The tests claim jobs by ordering, so the table must start empty.
	if _, err := db.ExecContext(ctx, "TRUNCATE jobs RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate jobs: %v", err)
	}
	return store.New(db)
}

// TestHeartbeatCancelsJobOnRenewFailure covers the contract that a worker
// which can no longer renew its lease cancels the job context instead of
// letting the job keep running (which could duplicate side effects when the
// job is reaped and re-claimed by another worker).
func TestHeartbeatCancelsJobOnRenewFailure(t *testing.T) {
	s := getTestStore(t)
	ctx := context.Background()

	id, err := s.Enqueue(ctx, "send_email", []byte(`{}`), 1, 5)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	job, err := s.Claim(ctx, "worker-a", 600, 30*time.Second)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if job == nil || job.ID != id {
		t.Fatalf("expected to claim job %d", id)
	}

	// The worker loses its lease: release the lock as the reaper would.
	if err := s.MarkFailed(ctx, id, "worker-a", "lease lost", 1, 5, time.Minute); err != nil {
		t.Fatalf("release lock: %v", err)
	}

	jobCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	p := &Pool{store: s, cfg: Config{LeaseDuration: 300 * time.Millisecond}}
	go p.heartbeat(jobCtx, done, 1, id, "worker-a", cancel)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("heartbeat did not exit after renew failure")
	}

	if jobCtx.Err() == nil {
		t.Error("expected job context to be canceled when renewal fails")
	}
}
