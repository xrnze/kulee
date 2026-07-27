// Package queue defines the Job type and in-memory queue.
package queue

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Job status constants.
const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

var nextID atomic.Int64

// Job is a unit of work for the worker pool.
type Job struct {
	ID        string
	Type      string
	Payload   string
	Status    string
	CreatedAt time.Time
}

// NewJob creates a job with a unique ID.
func NewJob(jobType, payload string) Job {
	return Job{
		ID:        fmt.Sprintf("job-%d", nextID.Add(1)),
		Type:      jobType,
		Payload:   payload,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
}

// Queue is an in-memory channel-backed job queue.
type Queue struct {
	ch chan Job
}

// New creates a queue with the given buffer size.
func New(buffer int) *Queue {
	return &Queue{ch: make(chan Job, buffer)}
}

// Enqueue sends a job into the queue.
func (q *Queue) Enqueue(job Job) {
	q.ch <- job
}

// Jobs returns the receive-only channel for consumers.
func (q *Queue) Jobs() <-chan Job {
	return q.ch
}
