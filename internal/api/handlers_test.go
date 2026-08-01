package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"kulee/internal/store"
)

func getTestStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := store.OpenDB(dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	ctx := context.Background()
	if err := store.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return store.New(db)
}

func TestAPIIntegration(t *testing.T) {
	s := getTestStore(t)
	h := NewHandler(s, 5*time.Minute)

	mux := http.NewServeMux()
	h.Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Enqueue a job.
	body := `{"type":"send_email","payload":{"to":"test@example.com"}}`
	resp, err := http.Post(server.URL+"/api/jobs", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var jobResp jobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		t.Fatalf("decode enqueue response: %v", err)
	}
	resp.Body.Close()

	if jobResp.ID == 0 {
		t.Error("expected non-zero job ID")
	}
	if jobResp.Type != "send_email" {
		t.Errorf("expected type send_email, got %s", jobResp.Type)
	}

	// List jobs.
	resp, err = http.Get(server.URL + "/api/jobs")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Stats.
	resp, err = http.Get(server.URL + "/api/stats")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Clean up: mark the job as dead, then delete via API.
	ctx := context.Background()
	if err := s.MarkFailed(ctx, jobResp.ID, "test cleanup", 1, 1, 0); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	req, err := http.NewRequest("DELETE", server.URL+"/api/jobs/"+strconv.FormatInt(jobResp.ID, 10), nil)
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
