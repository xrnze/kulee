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

```ts
Browser                              // React dashboard entry point
  → Caddy :8080                      // Production edge router
    → /api/* and /health → Go API    // API and health requests
    → /                 → nginx      // Static frontend server
                          → React    // Dashboard assets

Go API → Postgres jobs table         // Durable source of truth
                       ← Worker pool // Claims and processes jobs
                       ← Reaper      // Recovers expired leases
```

Evidence: `docker-compose.yaml:17-47`, `Caddyfile:3-12`,
`web/src/main.tsx:7-15`, `cmd/server/main.go:70-107`.

The Go process contains the HTTP API, fixed-size worker pool, and reaper.
Postgres is the single coordination point and source of truth for job state.
Workers claim jobs with `SELECT ... FOR UPDATE SKIP LOCKED`; the reaper
returns expired leases to `pending`. There is no separate message broker
such as Redis, RabbitMQ, or Kafka. This is a deliberate simplicity choice
for a portfolio project.

Deployment is a four-service stack on a single Docker bridge network: the
Go API, an nginx static server for the SPA, Postgres (internal-only, no
host ports), and a Caddy reverse proxy in front of both the API and the
frontend. The proxy is the only service exposed to the host, so the app
stays same-origin and needs no CORS configuration. In dev, Vite proxies
`/api` to the Go API with HMR.

### End-to-end flows

#### Happy path: enqueue, process, and complete

```ts
Browser                              // User submits a job
  → Caddy /api/jobs                  // Routes API traffic in production
    → api.Handler.enqueueJob         // Decodes request and applies defaults
      → store.Store.Enqueue          // Inserts a pending job
        → Postgres jobs table        // Durable source of truth
      → HTTP 201                     // Returns the persisted job
  → dashboard refresh                // Reloads jobs and stats

worker.Pool.worker                   // Repeated worker claim loop
  → store.Store.Claim                // Claims one eligible pending job
    → FOR UPDATE SKIP LOCKED         // Prevents competing workers colliding
      → status=running               // Sets lease and increments attempts
  → worker.Pool.heartbeat            // Renews the lease while running
    → store.Store.Renew              // Extends locked_until
  → registered job handler           // Executes the selected job type
    → store.Store.MarkSuccess        // Records success if ownership matches
```

Evidence: `Caddyfile:4-6`, `internal/api/handlers.go:73-108`,
`internal/store/claim.go:22-35`, `internal/store/claim.go:37-109`,
`internal/worker/pool.go:95-146`, `internal/store/deadletter.go:84-100`.

The API acknowledges persistence first. Workers then independently claim
pending jobs, execute the registered handler, renew the lease for long jobs,
and conditionally mark successful work as `success`.

#### Sad path: bad input, failure, retry, and recovery

```ts
Browser or client                    // Untrusted request boundary
  → invalid JSON or missing type     // Request cannot be accepted
    → HTTP 400                       // Client receives a validation error

worker.Pool.worker                   // Job was accepted and claimed
  → handler error or panic           // Execution did not complete
    → worker.FullJitterDelay         // Calculates retry delay
      → store.Store.MarkFailed       // Fenced failure update
        → pending + run_after        // Retry later while attempts remain
        → dead + last_error          // Stop after the attempt limit

running job                          // Worker loses lease or process crashes
  → lease expires                    // Heartbeat no longer renews it
    → store.Store.Reap               // Reclaims the orphaned job
      → status=pending               // Makes it claimable again
        → another worker claims      // Job may execute again

stale worker completion              // Old worker finishes after lease loss
  → conditional update               // Ownership check does not match
    → update rejected                // Current owner is protected
```

Evidence: `internal/api/handlers.go:73-98`, `internal/worker/pool.go:136-195`,
`internal/worker/retry.go:13-19`, `internal/store/deadletter.go:9-47`,
`internal/worker/pool.go:148-173`, `internal/store/claim.go:207-217`.

Failures are handled as at-least-once processing, not exactly-once
processing. A job may run again after a crash or lease loss, so external job
side effects should be idempotent. Dead jobs can be manually retried or
deleted from the dashboard.

## Quick Start

### Prerequisites

- Docker with Docker Compose
- Go 1.24+ and Node.js 20+ (only for local dev and tests)

### Setup

Run everything with one command:

```
cp .env.example .env
docker compose up --build
```

This builds and starts the whole stack on a private Docker network: Caddy
(edge proxy) -> `http://localhost:8080`, Go API (internal), nginx static
server (internal), and Postgres (internal, no host port). Migrations run
automatically on API startup.

For close-in development, run the Go API and Vite dev server directly:

```
# terminal 1: postgres + proxy only, or the full stack
docker compose up --build db proxy

# terminal 2: Go API on :8080 (needs a reachable Postgres)
go run ./cmd/server

# terminal 3: Vite dev server with HMR, API proxied via localhost:8080
cd web
npm install
npm run dev
```

### Testing

Unit and integration tests. Integration tests hit a real Postgres and skip
when `TEST_DATABASE_URL` is unset; point it at a dedicated test database:

```
TEST_DATABASE_URL=postgres://user:pass@localhost:5432/kulee_test?sslmode=disable \
  go test -p 1 ./...
```

`-p 1` runs packages serially because the DB-backed packages share one test
database and each truncates the `jobs` table before its tests run.

Open `http://localhost:5173` (Vite dev) or `http://localhost:8080` (production build via the proxy).

## Configuration

All configuration is via environment variables, loaded with `godotenv` from
a `.env` file in the project root.

| Variable                  | Default | Description                                      |
|---------------------------|---------|--------------------------------------------------|
| `DATABASE_URL`            | —       | Postgres connection string (required)             |
| `LISTEN_ADDR`             | `:8080` | API listen address                                 |
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
│   ├── vite.config.ts           # dev proxy to localhost:8080
│   ├── Dockerfile               # multi-stage SPA build (node -> nginx)
│   └── package.json
├── scripts/
│   └── loadtest.sh              # vegeta-based load test
├── Dockerfile                   # Go API-only build
├── docker-compose.yaml          # db + api + web + proxy stack
├── Caddyfile                    # edge reverse proxy config
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

### Reverse proxy vs single-port serving

The API is deliberately API-only; it does not serve the SPA. A Caddy reverse
proxy fronts both the API (`/api/*`, `/health`) and the nginx static server
for the SPA (`/`). Routing both through the proxy keeps the app same-origin
(no CORS needed) while letting the frontend and API deploy and scale
independently. Postgres is internal-only, reachable only from the Docker
bridge network.

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

### Docker Compose (recommended)

Build and run the full stack:

```bash
docker compose up --build -d
```

Caddy exposes `http://<host>:8080`; the API, frontend, and Postgres stay on
the internal network.

For production HTTPS on a VPS, point a DNS A record at the host and add the
hostname to the top of `Caddyfile` (e.g. `jobs.example.com {`); Caddy then
provisions and renews a Let's Encrypt certificate automatically.

### Legacy single-container

The `Dockerfile` builds only the Go API. To run it standalone you must
provide Postgres separately:

```bash
docker build -t kulee-api .
docker run -p 8080:8080 -e DATABASE_URL=<your-postgres-dsn> kulee-api
```

## License

MIT
