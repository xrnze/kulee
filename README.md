# Kulee

A self-hosted, lightweight job queue system written in Go, with a React
dashboard for submitting jobs and observing their lifecycle. Demonstrates
production-grade Go concurrency patterns: bounded worker pools, panic
isolation, crash recovery via lease/heartbeat, and safe concurrent job
claiming with `SELECT ... FOR UPDATE SKIP LOCKED`.

## Goals

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

## Architecture

```
Client → API server (writes to Postgres, returns 202 immediately) →
Postgres (jobs table, source of truth) ← Worker pool (claims via
SKIP LOCKED, processes, updates status) + Reaper (sweeps expired leases
back to pending) → Dashboard (SPA served from the same Go process,
manual refresh via TanStack Query fetching /api/jobs).
```

Key architectural decision: Postgres is the single coordination point
between API, workers, and reaper. No separate message broker (Redis,
RabbitMQ, Kafka). This is a deliberate simplicity choice for a portfolio
project.

The Go server serves both the API (`/api/*`) and the React SPA static
files (`web/dist/`) on a single port. Same origin, no CORS configuration
needed. In dev, Vite proxies `/api` to the Go server on a different port
with HMR.

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 20+
- PostgreSQL 16+

### Setup

1. Clone the repo and enter the directory.

2. Create a `.env` file from the example:

   ```
   cp .env.example .env
   ```

   Edit `.env` to set `DATABASE_URL` to your Postgres connection string.

3. Start the Go server:

   ```
   go run ./cmd/server
   ```

   This runs migrations on startup, starts the worker pool and reaper, and
   serves both the API and the frontend on `http://localhost:8080`.

4. In a separate terminal, start the frontend dev server (optional, for HMR):

   ```
   cd web
   npm install
   npm run dev
   ```

   This serves the SPA on `http://localhost:5173` with API proxy to `:8080`.

### Testing

Unit and integration tests. Integration tests hit a real Postgres and skip
when `TEST_DATABASE_URL` is unset; point it at a dedicated test database:

```
TEST_DATABASE_URL=postgres://user:pass@localhost:5432/kulee_test?sslmode=disable \
  go test -p 1 ./...
```

`-p 1` runs packages serially because the DB-backed packages share one test
database and each truncates the `jobs` table before its tests run.

5. Open `http://localhost:5173` (or `http://localhost:8080` for production build).

## Configuration

All configuration is via environment variables, loaded with `godotenv` from
a `.env` file in the project root.

| Variable                  | Default | Description                                      |
|---------------------------|---------|--------------------------------------------------|
| `DATABASE_URL`            | —       | Postgres connection string (required)             |
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

## Project Structure

```
kulee/
├── cmd/
│   └── server/
│       └── main.go              # wires everything, starts API + workers + reaper
├── internal/
│   ├── config/
│   │   └── config.go            # env var loading via godotenv, typed config struct
│   ├── queue/
│   │   └── job.go               # Job struct, status constants
│   ├── worker/
│   │   ├── pool.go              # worker pool, panic recovery, heartbeat loop
│   │   └── retry.go             # full-jitter backoff calculation
│   ├── store/
│   │   ├── postgres.go          # DB connection, migration runner
│   │   ├── claim.go             # claim / renew / reap / enqueue queries
│   │   └── deadletter.go        # dead-letter transition + manual retry
│   ├── api/
│   │   └── handlers.go          # all HTTP handlers
│   └── jobtypes/
│       ├── registry.go          # map[string]JobFunc registry
│       ├── send_email.go        # simulated, ~10% failure rate
│       ├── webhook_delivery.go  # real HTTP POST, I/O-bound
│       └── generate_report.go   # CPU-bound CSV generation
├── migrations/
│   └── 0001_init.sql            # jobs table schema
├── web/                         # TypeScript React SPA
│   ├── src/
│   │   ├── App.tsx              # main page
│   │   ├── main.tsx             # entry point
│   │   ├── components/
│   │   │   ├── JobTable.tsx
│   │   │   ├── JobForm.tsx
│   │   │   ├── StatsChart.tsx
│   │   │   └── DeadLetterView.tsx
│   │   └── lib/
│   │       └── api.ts           # fetch wrappers, TanStack Query hooks
│   ├── vite.config.ts           # dev proxy to Go :8080
│   └── package.json
├── scripts/
│   └── loadtest.sh              # vegeta-based load test
├── Dockerfile                   # multi-stage build
├── fly.toml                     # Fly.io deployment config
├── .env.example                 # documents all configurable env vars
├── PRD.md                       # design document
└── README.md                    # this file
```

## Key Tradeoffs

### Channels + fixed worker pool vs goroutine-per-job

A fixed worker pool (bounded concurrency) prevents the process from
exhausting file descriptors, memory, or downstream connection pools under
load. Goroutine-per-job is simpler to write but unbounded; a burst of
10,000 jobs creates 10,000 goroutines fighting for CPU, and the I/O-bound
jobs queue up behind CPU-bound ones. The pool acts as a admission
controller: only N jobs run concurrently, the rest wait in Postgres.

### Single Postgres instance vs a message broker

Postgres handles persistence, claiming, and coordination. A message broker
(Redis, RabbitMQ, Kafka) would give lower latency and better throughput at
scale, but adds operational complexity. For a single-process portfolio
project, Postgres is the right tool: it's already running, already the
source of truth, and `SKIP LOCKED` makes safe concurrent claiming
surprisingly simple.

### Manual refresh vs SSE/WebSocket/polling

The dashboard refreshes on manual click and via TanStack Query's
`refetchInterval`. SSE or WebSocket would give sub-second updates but
add server-side complexity (connection tracking, reconnection handling).
For a developer tool, a 5-second poll is pragmatic. Upgrade path: pass
`refetchInterval: 5000` to the query, which is already done.

### At-least-once delivery, not exactly-once

The system guarantees that a job is processed at least once. Fencing tokens
(leases with `locked_by` and `locked_until`) and conditional updates
(`WHERE id = ? AND locked_by = ?`) mitigate double-execution risk but do
not eliminate it. True exactly-once would require idempotency tokens or
two-phase commit across the job execution and its side effects, which is
out of scope.

### Lease timeout tuning

The lease timeout (`LEASE_DURATION_SECONDS`, default 30s) is a tradeoff:
too short, and healthy jobs are falsely reclaimed; too long, and crashed
workers keep jobs stuck until the reaper sweeps. Heartbeat renewal (every
`LEASE_DURATION_SECONDS / 3`) extends the lease for long-running healthy
jobs, so the timeout should be set to the p99 job duration under normal
load. The reaper interval (default 5s) determines how quickly truly
orphaned jobs are recovered.

### Priority via effective priority in SQL vs an in-memory heap

The `ORDER BY (priority + aging)` clause computes priority directly in the
query. An in-memory priority queue would be more efficient at high
throughput, but requires synchronizing the heap with the database state
across crashes and restarts. The SQL approach is simpler, equally correct
at this scale, and the aging divisor is the single tuning knob.

### Full-jitter backoff vs scheduled/calendar retries

Full-jitter exponential backoff (`random(0, min(cap, base * 2^attempt))`)
eliminates thundering herd problems when many jobs fail simultaneously.
The `run_after` column keeps the worker pool free while a job is waiting
for its next retry window. Calendar-based retries (e.g., "retry at 2pm
tomorrow") are more expressive but add complexity; the full-jitter approach
is simple and sufficient for demo job types.

### Heartbeat failure → cancel context vs keep running

When a heartbeat fails (the lease cannot be renewed), the worker cancels
the job's context and stops execution. This is safer than allowing the job
to continue: a worker that cannot prove it still owns the lease might be
competing with a reaper-reclaimed copy of the same job. Canceling the
context prevents the job from completing work it can't commit, at the cost
of potentially aborting a job that would have finished successfully.

### Cursor pagination vs offset

Cursor pagination (composite on `created_at, id`) is correct under
concurrent insertions: no duplicate or missing rows at page boundaries.
Offset pagination is simpler but breaks when new rows are inserted before
the current page. The cursor is the last job's ID, which is monotonic and
stable.

### Single-port, same-origin serving vs separate services

The Go server serves both the API and the React SPA on the same port.
This avoids CORS configuration, simplifies deployment, and is the standard
pattern for small Go + frontend projects. Add a reverse proxy (nginx,
Caddy) in front when scaling.

## Demo Job Types

All registered via a `map[string]func(context.Context, json.RawMessage) error`.

1. **`send_email`** — simulated I/O. Reads `{"to","subject","body"}`.
   Sleeps 2-5s. Fails ~10% of the time. Demonstrates retry/backoff.

2. **`webhook_delivery`** — real I/O. Reads `{"url","body","timeout_seconds"}`.
   Does `http.Post` with context timeout. Fails on non-2xx, timeout, or DNS
   error. Demonstrates real HTTP client patterns and context cancellation.

3. **`generate_report`** — CPU-bound. Reads `{"rows","output_format"}`.
   Generates N rows of fake CSV data in a tight loop. Demonstrates the
   contrast between I/O-bound and CPU-bound job handling.

## Load Testing

A vegeta-based load test script is provided at `scripts/loadtest.sh`:

```bash
# Install vegeta
go install github.com/tsenart/vegeta/v12@latest

# Run the load test (requires a running server)
./scripts/loadtest.sh
```

This runs attacks against `POST /api/jobs` at worker pool sizes 1, 4, and
8, recording results in `scripts/loadtest-results/`.

## Deploy

### Fly.io

```bash
fly launch --from fly.toml
fly secrets set DATABASE_URL=<your-postgres-dsn>
fly deploy
```

### Docker

```bash
docker build -t kulee .
docker run -p 8080:8080 -e DATABASE_URL=<your-postgres-dsn> kulee
```

## License

MIT