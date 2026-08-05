// Command server is the binary entry point for the job queue system.
//
// Starts an HTTP API server, a worker pool, and a reaper goroutine.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kulee/internal/api"
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

	poolCtx, poolCancel := context.WithCancel(context.Background())
	defer poolCancel()

	pool := worker.NewPool(poolCtx, st, handler, poolCfg)

	// Start reaper goroutine.
	reaperCtx, reaperCancel := context.WithCancel(context.Background())
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

	// Set up HTTP server.
	mux := http.NewServeMux()
	api.NewHandler(st, cfg.StatsWindow).Register(mux)

	// Health check for orchestration and the reverse proxy.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownDrain)
		defer shutdownCancel()

		// Stop claiming new jobs; in-flight jobs get to finish.
		pool.Stop()

		select {
		case <-pool.Done():
			log.Println("drain: all in-flight jobs finished")
		case <-shutdownCtx.Done():
			log.Println("drain deadline exceeded, aborting remaining jobs")
			pool.Abort()
			<-pool.Done()
		}

		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
	log.Println("server stopped")
}
