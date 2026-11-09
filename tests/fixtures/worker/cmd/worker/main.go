// Command worker runs background jobs until it is stopped.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"example.com/worker/internal/jobs"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	w := jobs.NewWorker()
	go w.Cleanup(ctx)
	go jobs.SendDigests(ctx, 5)
	go func() {
		<-ctx.Done()
		log.Print("stopping")
	}()
	jobs.StartReports(w)
	<-ctx.Done()
}
