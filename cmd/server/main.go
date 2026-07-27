// Command server is the binary entry point for the job queue system.
//
// Step 2: runs a CLI-driven demo backed by Postgres. The worker pool
// claims jobs from the store, heartbeats, and handles retries with
// full-jitter backoff. A reaper goroutine sweeps expired leases.
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"kulee/internal/config"
	"kulee/internal/jobtypes"
	"kulee/internal/store"
	"kulee/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := store.OpenDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	if err := store.RunMigrations(ctx, db); err != nil {
		log.Fatalf("migration: %v", err)
	}
	log.Println("migrations applied")

	st := store.New(db)

	// Register job types.
	reg := jobtypes.NewRegistry()
	reg.Register("send_email", jobtypes.SendEmail)
	reg.Register("webhook_delivery", jobtypes.WebhookDelivery)
	reg.Register("generate_report", jobtypes.GenerateReport)

	// Build the job handler: lookup from the registry.
	handler := func(ctx context.Context, jobType string, payload json.RawMessage) error {
		fn, err := reg.Lookup(jobType)
		if err != nil {
			return err
		}
		return fn(ctx, payload)
	}

	poolCfg := worker.Config{
		WorkerCount:   cfg.WorkerCount,
		LeaseDuration: cfg.LeaseDuration,
		AgingDivisor:  cfg.AgingDivisor,
		MaxAttempts:   cfg.MaxAttempts,
		RetryBase:     cfg.RetryBase,
		RetryCap:      cfg.RetryCap,
	}

	pool := worker.NewPool(ctx, st, handler, poolCfg)

	// Start reaper goroutine.
	reaperCtx, reaperCancel := context.WithCancel(ctx)
	defer reaperCancel()
	go func() {
		tick := time.NewTicker(cfg.ReaperInterval)
		defer tick.Stop()
		for {
			select {
			case <-reaperCtx.Done():
				return
			case <-tick.C:
				n, err := st.Reap(reaperCtx)
				if err != nil {
					log.Printf("reaper: %v", err)
					continue
				}
				if n > 0 {
					log.Printf("reaper: reclaimed %d expired leases", n)
				}
			}
		}
	}()

	// Enqueue demo jobs.
	types := []string{"send_email", "webhook_delivery", "generate_report"}
	payloads := map[string]string{
		"send_email":       `{"to":"user@example.com","subject":"Hello","body":"Test"}`,
		"webhook_delivery": `{"url":"https://httpbin.org/post","body":{"key":"value"},"timeout_seconds":10}`,
		"generate_report":  `{"rows":1000,"output_format":"csv"}`,
	}

	log.Printf("enqueueing %d demo jobs...", len(types)*3)
	for i := 0; i < 3; i++ {
		for _, t := range types {
			id, err := st.Enqueue(ctx, t, []byte(payloads[t]), 5, cfg.MaxAttempts)
			if err != nil {
				log.Printf("enqueue error: %v", err)
				continue
			}
			log.Printf("enqueued job %d (%s)", id, t)
		}
	}

	// Let the pool process jobs for a while, then stop.
	log.Println("workers running for 15 seconds...")
	time.Sleep(15 * time.Second)

	pool.Stop()
	<-pool.Done()
	log.Println("all workers stopped")
}
