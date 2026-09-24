// Package jobtypes provides a registry of job handler functions.
package jobtypes

import (
	"context"
	"encoding/json"
	"fmt"
)

// JobFunc processes a job given its typed payload.
type JobFunc func(context.Context, json.RawMessage) error

// Registry maps job type names to handlers and payload validators.
type Registry struct {
	handlers   map[string]JobFunc
	validators map[string]func(json.RawMessage) error
}

// NewRegistry returns an empty registry.
func NewRegistry() Registry {
	return Registry{
		handlers:   make(map[string]JobFunc),
		validators: make(map[string]func(json.RawMessage) error),
	}
}

// Register adds a handler and payload validator for the given job type.
func (r Registry) Register(jobType string, fn JobFunc, validate func(json.RawMessage) error) {
	r.handlers[jobType] = fn
	r.validators[jobType] = validate
}

// Lookup returns the handler for a job type, or an error if unknown.
func (r Registry) Lookup(jobType string) (JobFunc, error) {
	fn, ok := r.handlers[jobType]
	if !ok {
		return nil, fmt.Errorf("unknown job type: %s", jobType)
	}
	return fn, nil
}

// Validate checks that a job type exists and accepts the payload.
func (r Registry) Validate(jobType string, payload json.RawMessage) error {
	if _, err := r.Lookup(jobType); err != nil {
		return err
	}
	if validate := r.validators[jobType]; validate != nil {
		return validate(payload)
	}
	return nil
}
