// Package store provides the Postgres-backed job persistence layer.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Job represents a persisted job row.
type Job struct {
	ID          int64
	Type        string
	Payload     []byte
	Status      string
	Priority    int
	Attempts    int
	MaxAttempts int
	LockedBy    *string
	LockedUntil *time.Time
	RunAfter    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastError   *string
}

// OpenDB opens the Postgres connection pool and verifies connectivity.
func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

// RunMigrations executes the 0001_init.sql migration.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	// Serialize concurrent migration runs (parallel tests, multiple server
	// instances starting at once): CREATE TABLE IF NOT EXISTS races with
	// concurrent identical DDL and can fail on a unique-constraint check.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(717101)"); err != nil {
		return fmt.Errorf("lock migration: %w", err)
	}

	q := `
	CREATE TABLE IF NOT EXISTS jobs (
		id           BIGSERIAL PRIMARY KEY,
		type         TEXT        NOT NULL,
		payload      JSONB       NOT NULL DEFAULT '{}',
		status       TEXT        NOT NULL DEFAULT 'pending',
		priority     INT         NOT NULL DEFAULT 1,
		attempts     INT         NOT NULL DEFAULT 0,
		max_attempts INT         NOT NULL DEFAULT 5,
		locked_by    TEXT,
		locked_until TIMESTAMPTZ,
		run_after    TIMESTAMPTZ,
		created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		last_error   TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status      ON jobs (status);
	CREATE INDEX IF NOT EXISTS idx_jobs_locked_until ON jobs (locked_until);
	CREATE INDEX IF NOT EXISTS idx_jobs_created_at   ON jobs (created_at);`
	if _, err := tx.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("run migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}
