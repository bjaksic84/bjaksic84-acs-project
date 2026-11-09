package jobs

import (
	"context"
	"testing"
	"time"
)

func TestCleanupStops(t *testing.T) {
	w := NewWorker()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { // a go statement in a test is not a job
		w.Cleanup(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Cleanup did not stop")
	}
}

func TestReports(t *testing.T) {
	w := NewWorker()
	w.Reports()
	if w.reports.Load() != 1 {
		t.Fatal("no report")
	}
}
