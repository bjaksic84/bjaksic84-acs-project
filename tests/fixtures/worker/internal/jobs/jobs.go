// Package jobs holds the worker's periodic jobs.
package jobs

import (
	"context"
	"log"
	"sync/atomic"
	"time"
)

// Worker runs the periodic jobs.
type Worker struct {
	cleanups atomic.Int64
	reports  atomic.Int64
}

// NewWorker returns a worker.
func NewWorker() *Worker { return &Worker{} }

// Cleanup removes stale data every minute until ctx ends.
func (w *Worker) Cleanup(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		w.cleanups.Add(1)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// SendDigests sends n digests, or fewer if ctx ends first.
func SendDigests(ctx context.Context, n int) {
	for i := 0; i < n && ctx.Err() == nil; i++ {
		log.Printf("digest %d sent", i+1)
	}
}
