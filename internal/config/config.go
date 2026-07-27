// Package config loads typed configuration from environment variables.
// Uses godotenv to optionally load from .env, then reads os.Getenv.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configurable parameters for the job queue system.
type Config struct {
	DatabaseURL    string
	ListenAddr     string
	WorkerCount    int
	LeaseDuration  time.Duration
	ShutdownDrain  time.Duration
	ReaperInterval time.Duration
	AgingDivisor   float64
	MaxAttempts    int
	RetryBase      time.Duration
	RetryCap       time.Duration
	StatsWindow    time.Duration
}

// Load reads .env (if present) then environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load() // optional; ignore missing .env

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	var err error
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.ListenAddr, err = envStr("LISTEN_ADDR", ":8080"); err != nil {
		return nil, err
	}
	if cfg.WorkerCount, err = envInt("WORKER_COUNT", 4); err != nil {
		return nil, err
	}
	var ls, sd, ri, rb, rc, sw int
	if ls, err = envInt("LEASE_DURATION_SECONDS", 30); err != nil {
		return nil, err
	}
	cfg.LeaseDuration = time.Duration(ls) * time.Second
	if sd, err = envInt("SHUTDOWN_DRAIN_SECONDS", 60); err != nil {
		return nil, err
	}
	cfg.ShutdownDrain = time.Duration(sd) * time.Second
	if ri, err = envInt("REAPER_INTERVAL_SECONDS", 5); err != nil {
		return nil, err
	}
	cfg.ReaperInterval = time.Duration(ri) * time.Second
	var ad float64
	ad, err = envFloat("AGING_DIVISOR_SECONDS", 600)
	if err != nil {
		return nil, err
	}
	cfg.AgingDivisor = ad
	if cfg.MaxAttempts, err = envInt("MAX_ATTEMPTS", 5); err != nil {
		return nil, err
	}
	if rb, err = envInt("RETRY_BASE_MS", 1000); err != nil {
		return nil, err
	}
	cfg.RetryBase = time.Duration(rb) * time.Millisecond
	if rc, err = envInt("RETRY_CAP_MS", 60000); err != nil {
		return nil, err
	}
	cfg.RetryCap = time.Duration(rc) * time.Millisecond
	if sw, err = envInt("STATS_WINDOW_MINUTES", 5); err != nil {
		return nil, err
	}
	cfg.StatsWindow = time.Duration(sw) * time.Minute

	return cfg, nil
}

func envStr(key, def string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	return v, nil
}

func envInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q", key, v)
	}
	return n, nil
}

func envFloat(key string, def float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q", key, v)
	}
	return f, nil
}
