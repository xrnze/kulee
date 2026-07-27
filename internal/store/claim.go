package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
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
	err = tx.QueryRowContext(ctx, `
		SELECT id, type, payload, status, priority, attempts, max_attempts,
		       locked_by, locked_until, run_after, created_at, updated_at, last_error
		FROM jobs
		WHERE status = 'pending'
		  AND (run_after IS NULL OR run_after <= NOW())
		ORDER BY (priority + EXTRACT(EPOCH FROM NOW() - created_at) / $1) DESC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, agingDivisor).Scan(
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
		workerID, int(leaseDuration.Seconds()), job.ID,
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
		int(leaseDuration.Seconds()), jobID, workerID,
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

// GetJob retrieves a single job by ID.
func (s *Store) GetJob(ctx context.Context, id int64) (*Job, error) {
	job := &Job{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, type, payload, status, priority, attempts, max_attempts,
		        locked_by, locked_until, run_after, created_at, updated_at, last_error
		 FROM jobs WHERE id = $1`, id,
	).Scan(
		&job.ID, &job.Type, &job.Payload, &job.Status,
		&job.Priority, &job.Attempts, &job.MaxAttempts,
		&job.LockedBy, &job.LockedUntil, &job.RunAfter,
		&job.CreatedAt, &job.UpdatedAt, &job.LastError,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get job: job %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return job, nil
}

// ListJobs returns cursor-paginated jobs. Cursor is a job ID (exclusive upper
// bound). Pass 0 for the first page. Returns at most limit+1 rows so the caller
// can detect whether a next page exists.
func (s *Store) ListJobs(ctx context.Context, cursor int64, limit int, status string) ([]*Job, error) {
	q := `SELECT id, type, payload, status, priority, attempts, max_attempts,
	             locked_by, locked_until, run_after, created_at, updated_at, last_error
	      FROM jobs`
	var args []interface{}

	if cursor > 0 && status != "" {
		q += ` WHERE id < $1 AND status = $2`
		args = append(args, cursor, status)
	} else if cursor > 0 {
		q += ` WHERE id < $1`
		args = append(args, cursor)
	} else if status != "" {
		q += ` WHERE status = $1`
		args = append(args, status)
	}
	q += ` ORDER BY id DESC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		if err := rows.Scan(
			&job.ID, &job.Type, &job.Payload, &job.Status,
			&job.Priority, &job.Attempts, &job.MaxAttempts,
			&job.LockedBy, &job.LockedUntil, &job.RunAfter,
			&job.CreatedAt, &job.UpdatedAt, &job.LastError,
		); err != nil {
			return nil, fmt.Errorf("list jobs scan: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list jobs rows: %w", err)
	}
	return jobs, nil
}

// Stats returns counts of jobs grouped by status within the sliding window.
func (s *Store) Stats(ctx context.Context, window time.Duration) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM jobs
		 WHERE created_at >= NOW() - $1::INTERVAL
		 GROUP BY status`, fmt.Sprintf("%.0f seconds", window.Seconds()),
	)
	if err != nil {
		return nil, fmt.Errorf("stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("stats scan: %w", err)
		}
		stats[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stats rows: %w", err)
	}
	return stats, nil
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
