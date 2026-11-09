package jobs

// StartReports starts the report job. It has no context of its own.
func StartReports(w *Worker) {
	go w.Reports()
}

// Reports builds one report.
func (w *Worker) Reports() {
	w.reports.Add(1)
}
