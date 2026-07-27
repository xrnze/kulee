// Package api provides HTTP handlers for the job queue API.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"kulee/internal/store"
)

// Handler holds dependencies for API handlers.
type Handler struct {
	store       *store.Store
	statsWindow time.Duration
}

// NewHandler creates an API handler with the given store.
func NewHandler(s *store.Store, statsWindow time.Duration) *Handler {
	return &Handler{store: s, statsWindow: statsWindow}
}

// Register adds all API routes to the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/jobs", h.enqueueJob)
	mux.HandleFunc("GET /api/jobs", h.listJobs)
	mux.HandleFunc("GET /api/jobs/{id}", h.getJob)
	mux.HandleFunc("POST /api/jobs/{id}/retry", h.retryJob)
	mux.HandleFunc("DELETE /api/jobs/{id}", h.deleteJob)
	mux.HandleFunc("DELETE /api/jobs/dead", h.deleteAllDead)
	mux.HandleFunc("GET /api/stats", h.stats)
}

// jobResponse is the JSON shape for a job.
type jobResponse struct {
	ID          int64   `json:"id"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Priority    int     `json:"priority"`
	Attempts    int     `json:"attempts"`
	MaxAttempts int     `json:"max_attempts"`
	LockedBy    *string `json:"locked_by,omitempty"`
	LastError   *string `json:"last_error,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func toJobResponse(j *store.Job) jobResponse {
	r := jobResponse{
		ID:          j.ID,
		Type:        j.Type,
		Status:      j.Status,
		Priority:    j.Priority,
		Attempts:    j.Attempts,
		MaxAttempts: j.MaxAttempts,
		LockedBy:    j.LockedBy,
		LastError:   j.LastError,
		CreatedAt:   j.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   j.UpdatedAt.Format(time.RFC3339),
	}
	return r
}

type enqueueRequest struct {
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Priority    int             `json:"priority"`
	MaxAttempts int             `json:"max_attempts"`
}

func (h *Handler) enqueueJob(w http.ResponseWriter, r *http.Request) {
	var req enqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if req.Payload == nil {
		req.Payload = []byte("{}")
	}
	if req.Priority == 0 {
		req.Priority = 1
	}
	if req.MaxAttempts == 0 {
		req.MaxAttempts = 5
	}

	id, err := h.store.Enqueue(r.Context(), req.Type, req.Payload, req.Priority, req.MaxAttempts)
	if err != nil {
		log.Printf("enqueue error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}

	job, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		log.Printf("get job after enqueue: %v", err)
		writeError(w, http.StatusInternalServerError, "job created but failed to retrieve")
		return
	}

	writeJSON(w, http.StatusCreated, toJobResponse(job))
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	cursor := 0
	if c := r.URL.Query().Get("cursor"); c != "" {
		cursor, _ = strconv.Atoi(c)
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	status := r.URL.Query().Get("status")

	jobs, err := h.store.ListJobs(r.Context(), int64(cursor), limit, status)
	if err != nil {
		log.Printf("list jobs error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}

	hasMore := len(jobs) > limit
	if hasMore {
		jobs = jobs[:limit]
	}

	resp := make([]jobResponse, 0, len(jobs))
	for _, j := range jobs {
		resp = append(resp, toJobResponse(j))
	}

	nextCursor := int64(0)
	if hasMore && len(jobs) > 0 {
		nextCursor = jobs[len(jobs)-1].ID
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":        resp,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toJobResponse(job))
}

func (h *Handler) retryJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	if err := h.store.RetryDead(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	job, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "job retried but failed to retrieve")
		return
	}

	writeJSON(w, http.StatusOK, toJobResponse(job))
}

func (h *Handler) deleteJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	if err := h.store.DeleteDead(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteAllDead(w http.ResponseWriter, r *http.Request) {
	n, err := h.store.DeleteAllDead(r.Context())
	if err != nil {
		log.Printf("delete all dead error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete dead jobs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"deleted": n})
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.Stats(r.Context(), h.statsWindow)
	if err != nil {
		log.Printf("stats error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
