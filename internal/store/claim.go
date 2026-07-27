package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Store wraps a *sql.DB with job lifecycle operations.
type Store struct {
	db *sql.DB
}

// New wraps a DB connection in a Store.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Enqueue inserts a new pending job and returns its ID.
func (s *Store) Enqueue(ctx context.Context, jobType string, payload json.RawMessage, priority, maxAttempts int) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO jobs (type, payload, priority, max_attempts)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		jobType, payload, priority, maxAttempts,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("enqueue: %w", err)
	}
	return id, nil
}

// Claim tries to claim one pending job using SELECT FOR UPDATE SKIP LOCKED.
// Returns nil, nil when no job is available.
func (s *Store) Claim(ctx context.Context, workerID string, agingDivisor float64, leaseDuration time.Duration) (*Job, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("claim begin tx: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	job := &Job{}
	err = tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT id, type, payload, status, priority, attempts, max_attempts,
		       locked_by, locked_until, run_after, created_at, updated_at, last_error
		FROM jobs
		WHERE status = 'pending'
		  AND (run_after IS NULL OR run_after <= NOW())
		ORDER BY (priority + EXTRACT(EPOCH FROM NOW() - created_at) / %f) DESC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, agingDivisor)).Scan(
		&job.ID, &job.Type, &job.Payload, &job.Status,
		&job.Priority, &job.Attempts, &job.MaxAttempts,
		&job.LockedBy, &job.LockedUntil, &job.RunAfter,
		&job.CreatedAt, &job.UpdatedAt, &job.LastError,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim select: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE jobs
		 SET status = 'running', locked_by = $1,
		     locked_until = NOW() + ($2 || ' seconds')::INTERVAL,
		     attempts = attempts + 1
		 WHERE id = $3`,
		workerID, fmt.Sprintf("%d", int(leaseDuration.Seconds())), job.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("claim update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("claim commit: %w", err)
	}

	job.Status = "running"
	now := time.Now()
	job.LockedBy = &workerID
	lu := now.Add(leaseDuration)
	job.LockedUntil = &lu
	job.Attempts++
	return job, nil
}

// Renew extends the lease on a running job.
func (s *Store) Renew(ctx context.Context, jobID int64, workerID string, leaseDuration time.Duration) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs
		 SET locked_until = NOW() + ($1 || ' seconds')::INTERVAL
		 WHERE id = $2 AND locked_by = $3 AND status = 'running'`,
		fmt.Sprintf("%d", int(leaseDuration.Seconds())), jobID, workerID,
	)
	if err != nil {
		return fmt.Errorf("renew: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("renew: no running row found for job %d", jobID)
	}
	return nil
}

// Reap sweeps expired leases back to pending.
func (s *Store) Reap(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs
		 SET status = 'pending', locked_by = NULL, locked_until = NULL
		 WHERE status = 'running' AND locked_until < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("reap: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
