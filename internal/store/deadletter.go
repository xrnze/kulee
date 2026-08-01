package store

import (
	"context"
	"fmt"
	"time"
)

// MarkFailed transitions a job after a failure. If attempts >= max_attempts
// the job becomes 'dead'; otherwise it reverts to 'pending' with a backoff
// delay in run_after. The caller should compute the backoff delay.
// The update is fenced on locked_by so a worker that lost its lease (job
// reaped and possibly re-claimed by another worker) cannot modify the row
// the new owner is now processing.
func (s *Store) MarkFailed(ctx context.Context, jobID int64, workerID, lastError string, attempts, maxAttempts int, backoff time.Duration) error {
	newStatus := "pending"
	if attempts >= maxAttempts {
		newStatus = "dead"
	}

	var runAfter interface{}
	if newStatus == "pending" {
		runAfter = time.Now().Add(backoff)
	} else {
		runAfter = nil
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs
		 SET status = $1,
		     run_after = $2,
		     locked_by = NULL,
		     locked_until = NULL,
		     last_error = $3,
		     updated_at = NOW()
		 WHERE id = $4 AND locked_by = $5 AND (status = 'running' OR status = 'pending')`,
		newStatus, runAfter, lastError, jobID, workerID,
	)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mark failed: no row found for job %d", jobID)
	}
	return nil
}

// RetryDead resets a dead-lettered job back to pending with zero attempts.
func (s *Store) RetryDead(ctx context.Context, jobID int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs
		 SET status = 'pending', attempts = 0, last_error = NULL,
		     run_after = NULL, locked_by = NULL, locked_until = NULL,
		     updated_at = NOW()
		 WHERE id = $1 AND status = 'dead'`,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("retry dead: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("retry dead: job %d not found or not dead", jobID)
	}
	return nil
}

// DeleteDead removes a single dead-lettered job.
func (s *Store) DeleteDead(ctx context.Context, jobID int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM jobs WHERE id = $1 AND status = 'dead'`, jobID,
	)
	if err != nil {
		return fmt.Errorf("delete dead: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("delete dead: job %d not found or not dead", jobID)
	}
	return nil
}

// MarkSuccess marks a running job as completed successfully.
// Uses a conditional update so it only affects the row if the worker
// still holds the lock.
func (s *Store) MarkSuccess(ctx context.Context, jobID int64, workerID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = 'success', updated_at = NOW()
		 WHERE id = $1 AND locked_by = $2 AND status = 'running'`,
		jobID, workerID,
	)
	if err != nil {
		return fmt.Errorf("mark success: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mark success: no running row found for job %d", jobID)
	}
	return nil
}

// DeleteAllDead removes all dead-lettered jobs.
func (s *Store) DeleteAllDead(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM jobs WHERE status = 'dead'`,
	)
	if err != nil {
		return 0, fmt.Errorf("delete all dead: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
