// Package jobtypes provides a registry of job handler functions.
package jobtypes

import (
	"context"
	"encoding/json"
	"fmt"
)

// JobFunc processes a job given its typed payload.
type JobFunc func(context.Context, json.RawMessage) error

// Registry maps job type names to handler functions.
type Registry map[string]JobFunc

// NewRegistry returns an empty registry.
func NewRegistry() Registry {
	return make(Registry)
}

// Register adds a handler for the given job type.
func (r Registry) Register(jobType string, fn JobFunc) {
	r[jobType] = fn
}

// Lookup returns the handler for a job type, or an error if unknown.
func (r Registry) Lookup(jobType string) (JobFunc, error) {
	fn, ok := r[jobType]
	if !ok {
		return nil, fmt.Errorf("unknown job type: %s", jobType)
	}
	return fn, nil
}
