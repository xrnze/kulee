CREATE TABLE IF NOT EXISTS jobs (
    id          BIGSERIAL PRIMARY KEY,
    type        TEXT        NOT NULL,
    payload     JSONB       NOT NULL DEFAULT '{}',
    status      TEXT        NOT NULL DEFAULT 'pending',
    priority    INT         NOT NULL DEFAULT 1,
    attempts    INT         NOT NULL DEFAULT 0,
    max_attempts INT        NOT NULL DEFAULT 5,
    locked_by   TEXT,
    locked_until TIMESTAMPTZ,
    run_after   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error  TEXT
);

CREATE INDEX IF NOT EXISTS idx_jobs_status      ON jobs (status);
CREATE INDEX IF NOT EXISTS idx_jobs_locked_until ON jobs (locked_until);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at   ON jobs (created_at);
