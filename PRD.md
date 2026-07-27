# PRD: Go Job Processing Pipeline & Worker Queue

## 1. Overview

A self-hosted, lightweight job queue system written in Go, with a dashboard
for submitting jobs and observing their lifecycle. The project exists to
demonstrate production-grade Go concurrency patterns (bounded worker pools,
panic isolation, crash recovery, safe concurrent claiming) rather than to
compete with mature systems like Sidekiq or River.

**Target audience for this PRD:** yourself, as a planning and scope-locking
document, and anyone reviewing your portfolio who wants to understand your
design decisions before reading code.

## 2. Goals

- Demonstrate bounded concurrency (fixed worker pool, not goroutine-per-job).
- Demonstrate crash safety (lease/heartbeat pattern, orphaned job recovery).
- Demonstrate safe concurrent job claiming (`SELECT ... FOR UPDATE SKIP LOCKED`).
- Demonstrate retry semantics (exponential backoff with full jitter, dead-letter queue).
- Demonstrate priority handling with starvation prevention (aging via configurable divisor).
- Demonstrate graceful shutdown (drain workers with deadline, conditional lease cleanup).
- Demonstrate context propagation (heartbeat failure cancels job context).
- Provide a dashboard (TypeScript React SPA, manual refresh) showing job state.
- Ship a README that explains *why*, not just *what* — the tradeoffs are the
  actual portfolio artifact, the code is the proof.

## 3. Non-goals (explicitly out of scope)

- Distributed/multi-node job queue (single Postgres instance, single API
  process is fine for this scope).
- Horizontal scaling of the API server itself.
- A generic plugin system for arbitrary job types beyond the 3 demo types.
- Authentication/authorization (assume local/demo use, note this as a
  known limitation rather than building it).
- Exactly-once delivery guarantees (this system is at-least-once; state
  that explicitly rather than pretending otherwise).
- SSE / WebSocket real-time updates (dashboard uses manual refresh; noted
  as a deliberate simplicity choice, with polling as a documented upgrade path).
- Auto-cleanup of dead-lettered jobs (manual delete button only, so the
  developer can inspect failures before removing them).

## 4. Architecture

Client → API server (writes to Postgres, returns 202 immediately) →
Postgres (jobs table, source of truth) ← Worker pool (claims via
`SKIP LOCKED`, processes, updates status) + Reaper (sweeps expired leases
back to `pending`) → Dashboard (SPA served from the same Go process,
manual refresh via TanStack Query fetching `/api/jobs`).

Key architectural decision: Postgres is the single coordination point
between API, workers, and reaper. No separate message broker (Redis,
RabbitMQ, Kafka). This is a deliberate simplicity choice for a portfolio
project, and worth stating as such rather than treating it as an
oversight.

The Go server serves both the API (`/api/*`) and the React SPA static
files (`web/dist/`) on a single port. Same origin, no CORS configuration
needed. In dev, Vite proxies `/api` to the Go server on a different port
with HMR.

## 5. Project structure

```
job-queue/
├── cmd/
│   └── server/
│       └── main.go              # wires everything, starts API + workers + reaper
├── internal/
│   ├── config/
│   │   └── config.go            # env var loading via godotenv, typed config struct
│   ├── queue/
│   │   ├── job.go               # Job struct, status constants
│   │   └── channel_queue.go     # in-memory channel-based queue (Step 1 only)
│   ├── worker/
│   │   ├── pool.go              # worker pool, panic recovery, heartbeat loop
│   │   └── retry.go             # full-jitter backoff calculation
│   ├── store/
│   │   ├── postgres.go          # DB connection, migration runner
│   │   ├── claim.go             # claim / renew / reap / enqueue queries
│   │   └── deadletter.go        # dead-letter transition + manual retry
│   ├── api/
│   │   ├── handlers.go          # POST /api/jobs, GET /api/jobs, GET /api/jobs/:id
│   │   └── stats.go             # GET /api/stats
│   └── jobtypes/
│       ├── registry.go          # map[string]JobFunc registry
│       ├── send_email.go        # simulated, ~10% failure rate
│       ├── webhook_delivery.go  # real HTTP POST, I/O-bound
│       └── generate_report.go   # CPU-bound CSV generation
├── migrations/
│   └── 0001_init.sql            # jobs table per section 6
├── web/                         # TypeScript React SPA
│   ├── src/
│   │   ├── routes/
│   │   │   └── index.tsx        # job list + submission form
│   │   ├── components/
│   │   │   ├── JobTable.tsx
│   │   │   ├── StatsChart.tsx
│   │   │   └── DeadLetterView.tsx
│   │   └── lib/
│   │       └── api.ts           # fetch wrappers, TanStack Query hooks
│   ├── vite.config.ts           # dev proxy to Go :8080
│   └── package.json
├── .env.example                 # documents all configurable env vars
├── PRD.md                       # this document
└── README.md                    # written last, in Step 4
```

**Rule for step-by-step work:** each step in section 9 lists exactly which
files it creates or touches. An agent picking up a step should only read
those files plus this section, not the whole repo, unless a step explicitly
says otherwise.

## 6. Data model

**`jobs` table**

| Column         | Type        | Notes                                          |
|----------------|-------------|-------------------------------------------------|
| id             | bigserial   | PK                                              |
| type           | text        | e.g. `send_email`, `webhook_delivery`, `generate_report` |
| payload        | jsonb       | job-specific input                              |
| status         | text        | pending / running / success / failed / dead      |
| priority       | int         | 1-10, higher = more urgent                      |
| attempts       | int         | current attempt count (incremented at claim time)|
| max_attempts   | int         | global default 5 from config                    |
| locked_by      | text        | worker id owning the current claim (nullable)    |
| locked_until   | timestamptz | lease expiry (nullable)                          |
| run_after      | timestamptz | earliest time this job can be claimed (for retry backoff; nullable) |
| created_at     | timestamptz | used for FIFO + aging calc                      |
| updated_at     | timestamptz | last state change                                |
| last_error     | text        | most recent failure message (nullable)           |

**Effective priority (for claim ordering)**

```
effective = priority + (EXTRACT(EPOCH FROM NOW() - created_at) / AGING_DIVISOR_SECONDS)
```

Default divisor: 600 (10 min). A priority-1 job catches a priority-10 job
after ~90 minutes of waiting. Configurable via `AGING_DIVISOR_SECONDS` env var.

**Claim query**
```sql
BEGIN;
SELECT * FROM jobs
WHERE status = 'pending'
  AND (run_after IS NULL OR run_after <= NOW())
ORDER BY (priority + EXTRACT(EPOCH FROM NOW() - created_at) / $aging_divisor) DESC
LIMIT 1
FOR UPDATE SKIP LOCKED;

UPDATE jobs
SET status = 'running', locked_by = $worker_id,
    locked_until = NOW() + ($lease_duration_seconds || ' seconds')::INTERVAL,
    attempts = attempts + 1
WHERE id = $1;
COMMIT;
```

**Renewal query (heartbeat)**
```sql
UPDATE jobs
SET locked_until = NOW() + ($lease_duration_seconds || ' seconds')::INTERVAL
WHERE id = $1 AND locked_by = $worker_id AND status = 'running';
```

Heartbeat interval: `LEASE_DURATION_SECONDS / 3` (every 10s for a 30s lease).
If heartbeat fails, the job context is canceled and the worker stops work.
On the next heartbeat failure: worker attempts a conditional update to
record the failure, then moves on.

**Reaper query**
```sql
UPDATE jobs
SET status = 'pending', locked_by = NULL, locked_until = NULL
WHERE status = 'running' AND locked_until < NOW();
```

Sweep interval: configurable via `REAPER_INTERVAL_SECONDS` (default 5s).

**Retry after failure**
```sql
UPDATE jobs
SET status = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'pending' END,
    run_after = CASE WHEN attempts < max_attempts THEN NOW() + $backoff_delay ELSE run_after END,
    locked_by = NULL, locked_until = NULL, last_error = $err
WHERE id = $1;
```

Backoff: full jitter — `delay = random(0, min(RETRY_CAP_MS, RETRY_BASE_MS * 2^attempt))`.
Defaults: base=1000ms, cap=60000ms.

**Dead-letter manual retry**
```sql
UPDATE jobs SET status = 'pending', attempts = 0, last_error = NULL, run_after = NULL
WHERE id = $1 AND status = 'dead';
```

## 7. API surface

| Method | Path                   | Purpose                                          |
|--------|------------------------|---------------------------------------------------|
| POST   | `/api/jobs`            | enqueue a job, returns 202 + job id               |
| GET    | `/api/jobs`            | list jobs (cursor-paginated, filterable by status)|
| GET    | `/api/jobs/:id`        | job detail                                        |
| POST   | `/api/jobs/:id/retry`  | manually retry a dead-lettered job                |
| DELETE | `/api/jobs/:id`        | delete a dead-lettered job                        |
| DELETE | `/api/jobs/dead`       | delete all dead-lettered jobs                     |
| GET    | `/api/stats`           | queue depth, throughput, failure rate (sliding window) |

**Pagination**: composite cursor on `(created_at, id)`.
`GET /api/jobs?status=pending&limit=50&cursor=2024-01-01T00:00:00Z,42`
Response includes a `nextCursor` field.

**Payload validation**: API server only validates `type` is non-empty and
`payload` is valid JSON. Worker validates semantics and records errors in
`last_error`.

## 8. Demo job types

All registered via `map[string]func(context.Context, json.RawMessage) error`.

1. **`send_email`** — simulated I/O. Reads `{"to","subject","body"}`.
   Sleeps 2-5s. Fails ~10% of the time. Demonstrates retry/backoff.

2. **`webhook_delivery`** — real I/O. Reads `{"url","body","timeout_seconds"}`.
   Does `http.Post` with context timeout. Fails on non-2xx, timeout, or DNS
   error. Demonstrates real HTTP client patterns and context cancellation.

3. **`generate_report`** — CPU-bound. Reads `{"rows","output_format"}`.
   Generates N rows of fake CSV data in a tight loop. Demonstrates the
   contrast between I/O-bound and CPU-bound job handling.

## 9. Build plan (step-based)

**Step 1 — core queue + worker pool**
Files: `internal/queue/job.go`, `internal/queue/channel_queue.go`,
`internal/worker/pool.go`, `cmd/server/main.go` (minimal, CLI-driven).
- [x] Define `Job` struct and in-memory channel-based queue
- [x] Implement fixed-size worker pool consuming from the channel
- [x] Add `defer/recover()` around each job execution (not the worker loop)
- [x] Prove it with a CLI test: submit jobs, confirm bounded concurrency,
      confirm a panicking job doesn't kill the worker

**Step 2 — persistence and durability**
Files: `internal/config/config.go`, `.env.example`,
`migrations/0001_init.sql`, `internal/store/*`,
`internal/worker/retry.go`, `internal/jobtypes/*`.
- [x] Config loading via `godotenv` + typed struct with defaults
- [x] Set up Postgres schema (`jobs` table per section 6)
- [x] Implement claim query (`SELECT ... FOR UPDATE SKIP LOCKED`)
- [x] Implement heartbeat renewal loop per worker (ticker + cancellable context;
      on failure, cancel job context and conditional-update status)
- [x] Implement reaper goroutine (sweep expired leases back to `pending`)
- [x] Implement full-jitter retry backoff + `run_after` column
- [x] Implement dead-letter transition after `max_attempts`
- [x] Implement 3 demo job types with registry map
- [ ] Add `/api/stats` endpoint (DB sliding window)

**Step 3 — API + dashboard skeleton**
Files: `internal/api/handlers.go`, `internal/api/stats.go`,
`web/src/lib/api.ts`, `web/src/routes/index.tsx`,
`web/src/components/JobTable.tsx`, `web/vite.config.ts`,
`cmd/server/main.go` (wired up).
- [ ] API handlers: POST /api/jobs, GET /api/jobs (cursor-paginated),
      GET /api/jobs/:id, POST /api/jobs/:id/retry, DELETE /api/jobs/:id,
      DELETE /api/jobs/dead
- [ ] Go server serves `web/dist/` as static files on `/`
- [ ] Frontend: initial job list via TanStack Query
- [ ] Frontend: job submission form (pick type, paste JSON payload)
- [ ] Frontend: manual refresh button
- [ ] Vite dev proxy to Go backend

**Step 4 — polish and ship**
Files: `web/src/components/StatsChart.tsx`,
`web/src/components/DeadLetterView.tsx`, `README.md`.
- [ ] Stats dashboard view (throughput + queue depth chart)
- [ ] Dead-letter view with manual retry and delete buttons
- [ ] Graceful shutdown (drain workers with `SHUTDOWN_DRAIN_SECONDS` deadline)
- [ ] Load test with `vegeta` at a few worker pool sizes, record results
- [ ] Deploy (Fly.io or Railway)
- [ ] Write README covering section 11 tradeoffs

## 10. Success criteria

- Worker pool survives a panicking job without crashing or losing a worker.
- Killing the process mid-job and restarting it results in the job being
  correctly reclaimed and retried, not stuck or double-processed.
- Two workers can safely claim from a saturated queue without ever
  processing the same job twice under normal (non-crash) conditions.
- Graceful shutdown: workers finish in-progress jobs within the drain
  deadline; jobs exceeding the deadline are left for the reaper.
- Heartbeat failure cancels the job context and the worker stops work
  without corrupting the job state.
- Dashboard reflects job state accurately on manual refresh.
- README explains every tradeoff in section 11 clearly enough that a
  reader doesn't need to ask "why did you do it this way."

## 11. Key tradeoffs to document in the README

- Channels + fixed worker pool vs goroutine-per-job — bounding resource use
  and downstream load.
- Single Postgres instance vs a message broker — simplicity over scale,
  appropriate for this project's scope.
- Manual refresh vs SSE/WebSocket/polling — dashboard simplicity; `refetchInterval`
  on TanStack Query is the documented upgrade path if near-real-time is needed.
- At-least-once delivery, not exactly-once — fencing tokens (leases) and
  conditional updates mitigate double-execution risk, they don't eliminate it.
- Lease timeout tuning — based on realistic p99 job duration, with an
  explicit tradeoff between false reclaims (too short) and slow crash
  recovery (too long). Heartbeat renewal extends the lease for long-running
  healthy jobs.
- Priority via computed "effective priority" in SQL vs an in-memory heap —
  chosen deliberately as the less complex, equally correct option at this
  scale. Aging divisor is the single tuning knob.
- Full-jitter backoff vs scheduled/calendar retries — simpler to reason
  about, eliminates thundering herd, and the `run_after` column keeps the
  worker pool free while waiting.
- Heartbeat failure → cancel context vs keep running — cancel is safer;
  prevents a worker from completing work it can't prove it still owns.
- Cursor pagination (composite on `created_at, id`) vs offset — correct
  under concurrent insertions, no duplicate/missing rows at page boundaries.
- Single-port, same-origin serving (Go static files + API) vs separate
  services — dev simplicity; add a reverse proxy in front when needed.

## 12. Environment variables

All configurable via `.env`, loaded with `godotenv`. Documented in `.env.example`.

| Variable                  | Default | Description                                      |
|---------------------------|---------|--------------------------------------------------|
| `DATABASE_URL`            | —       | Postgres connection string                        |
| `LISTEN_ADDR`             | `:8080` | API + static file listen address                  |
| `WORKER_COUNT`            | `4`     | Number of worker goroutines in the pool           |
| `LEASE_DURATION_SECONDS`  | `30`    | How long a claim lease lasts before expiry        |
| `SHUTDOWN_DRAIN_SECONDS`  | `60`    | Max time to wait for in-flight jobs on SIGTERM    |
| `REAPER_INTERVAL_SECONDS` | `5`     | How often the reaper sweeps expired leases        |
| `AGING_DIVISOR_SECONDS`   | `600`   | Seconds of waiting = 1 effective priority point   |
| `MAX_ATTEMPTS`            | `5`     | Global max retry attempts per job                 |
| `RETRY_BASE_MS`           | `1000`  | Base delay for exponential backoff (ms)           |
| `RETRY_CAP_MS`            | `60000` | Maximum backoff delay (ms)                        |
| `STATS_WINDOW_MINUTES`    | `5`     | Sliding window for throughput/failure rate stats  |
