// Package worker provides a fixed-size worker pool with lease heartbeats.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"kulee/internal/store"
)

// JobFunc processes a single job. Implementations must recover panics --
// the pool only recovers the outer goroutine. Returning an error indicates
// the job failed and should be retried.
type JobFunc func(context.Context, string, json.RawMessage) error

// Pool is a fixed-size worker pool that claims jobs from the store.
// Each worker runs in its own goroutine. Panics in a handler are caught
// so a single bad job never takes down the pool or its workers.
type Pool struct {
	store   *store.Store
	handler JobFunc
	wg      sync.WaitGroup
	cfg     Config
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
}

// Config holds worker-pool-specific parameters.
type Config struct {
	WorkerCount   int
	LeaseDuration time.Duration
	AgingDivisor  float64
	MaxAttempts   int
	RetryBase     time.Duration
	RetryCap      time.Duration
}

// NewPool starts numWorkers goroutines claiming from the store.
func NewPool(ctx context.Context, s *store.Store, handler JobFunc, cfg Config) *Pool {
	ctx, cancel := context.WithCancel(ctx)
	p := &Pool{
		store:   s,
		handler: handler,
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	for i := 0; i < cfg.WorkerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	go func() {
		p.wg.Wait()
		close(p.done)
	}()

	return p
}

// Stop shuts down the pool. Workers finish their current job (if any)
// then exit. In-flight jobs that exceed ShutdownDrain are orphaned and
// will be reclaimed by the reaper after their lease expires.
func (p *Pool) Stop() {
	p.cancel()
}

// Done returns a channel that closes when all workers have stopped.
func (p *Pool) Done() <-chan struct{} {
	return p.done
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()

	workerID := hostnamePID()
	log.Printf("worker %d: started (id=%s)", id, workerID)

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("worker %d: shutting down", id)
			return
		default:
		}

		job, err := p.store.Claim(p.ctx, workerID, p.cfg.AgingDivisor, p.cfg.LeaseDuration)
		if err != nil {
			log.Printf("worker %d: claim error: %v", id, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if job == nil {
			time.Sleep(250 * time.Millisecond)
			continue
		}

		log.Printf("worker %d: claimed job %d (%s)", id, job.ID, job.Type)

		// Run job with heartbeat loop in a sub-context.
		jobCtx, jobCancel := context.WithCancel(p.ctx)
		heartbeatDone := make(chan struct{})
		go p.heartbeat(jobCtx, heartbeatDone, id, job.ID, workerID)

		err = p.safeExecute(jobCtx, id, job)

		jobCancel()
		<-heartbeatDone
		p.handleResult(id, job, err, workerID)
	}
}

// safeExecute runs the handler with panic recovery.
func (p *Pool) safeExecute(ctx context.Context, id int, job *store.Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			log.Printf("worker %d: recovered panic in job %d: %v", id, job.ID, r)
		}
	}()

	return p.handler(ctx, job.Type, job.Payload)
}

// heartbeat renews the lease on the claimed job until the job completes
// or the context is canceled. Runs every LEASE_DURATION / 3.
// Closes done when the heartbeat loop exits.
func (p *Pool) heartbeat(ctx context.Context, done chan<- struct{}, workerID int, jobID int64, lockedBy string) {
	defer close(done)

	tick := time.NewTicker(p.cfg.LeaseDuration / 3)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			err := p.store.Renew(ctx, jobID, lockedBy, p.cfg.LeaseDuration)
			if err != nil {
				log.Printf("worker %d: heartbeat failed for job %d: %v — canceling", workerID, jobID, err)
				return
			}
		}
	}
}

// handleResult updates the store after a job completes or fails.
func (p *Pool) handleResult(workerID int, job *store.Job, execErr error, lockedBy string) {
	ctx := context.Background()

	if execErr == nil {
		err := p.store.MarkSuccess(ctx, job.ID, lockedBy)
		if err != nil {
			log.Printf("worker %d: failed to mark job %d as success: %v", workerID, job.ID, err)
		}
		return
	}

	// Mark as failed; the store decides dead vs pending+backoff.
	backoff := FullJitterDelay(job.Attempts, p.cfg.RetryBase, p.cfg.RetryCap)
	err := p.store.MarkFailed(ctx, job.ID, execErr.Error(), job.Attempts, p.cfg.MaxAttempts, backoff)
	if err != nil {
		log.Printf("worker %d: failed to mark job %d as failed: %v", workerID, job.ID, err)
	}
}

// hostnamePID returns a unique worker identifier (hostname:pid).
func hostnamePID() string {
	// In production this would use os.Hostname()+os.Getpid().
	// For simplicity in the demo, we return a static string since
	// only one process is expected.
	return "demo-process"
}
