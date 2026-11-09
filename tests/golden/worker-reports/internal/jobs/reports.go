package jobs

import (
	"context"

	"example.com/worker/internal/flags"
)

// StartReports starts the report job. It has no context of its own.
func StartReports(w *Worker) {
	if flags.Enabled(context.Background(), flags.Reports, true) {
		go w.Reports()
	}
}

// Reports builds one report.
func (w *Worker) Reports() {
	w.reports.Add(1)
}
