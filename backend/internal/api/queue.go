package api

import (
	"context"
	"net/http"
	"time"
)

// HeavyQueue limits concurrent heavy operations (like payroll calculation and CSV export)
// to protect the free-tier database connection pool from starvation.
type HeavyQueue struct {
	sem     chan struct{}
	timeout time.Duration
}

// NewHeavyQueue creates a queue bounded by maxConcurrent slots and a wait timeout.
func NewHeavyQueue(maxConcurrent int, timeout time.Duration) *HeavyQueue {
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HeavyQueue{
		sem:     make(chan struct{}, maxConcurrent),
		timeout: timeout,
	}
}

// Middleware wraps a handler function with the admission queue.
func (q *HeavyQueue) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), q.timeout)
		defer cancel()

		select {
		case q.sem <- struct{}{}:
			defer func() { <-q.sem }()
			next(w, r)
		case <-ctx.Done():
			RespondError(w, http.StatusServiceUnavailable, "system busy processing heavy workloads, please retry in a moment")
		}
	}
}
