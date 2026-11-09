// Package admin holds the admin pages, as route tables.
package admin

import (
	"fmt"
	"net/http"

	"example.com/lectures/internal/flags"
)

// Route is a mux pattern and its handler.
type Route struct {
	Pattern string
	Handler http.Handler
}

// Routes lists every /admin route.
func Routes() []Route {
	return []Route{
		{"GET /admin/users", flags.Gate(flags.Admin, http.HandlerFunc(users))},
		{"POST /admin/users", flags.Gate(flags.Admin, requireAdmin(http.HandlerFunc(createUser)))},
	}
}

// reportRoutes are the report pages.
var reportRoutes = []Route{
	{Pattern: "GET /reports", Handler: flags.Gate(flags.Reports, http.HandlerFunc(reports))},
	{
		Pattern: "GET /reports/{id}",
		Handler: flags.Gate(flags.Reports, http.HandlerFunc(report)),
	},
}

// ReportRoutes lists the report pages.
func ReportRoutes() []Route { return reportRoutes }

func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Admin") != "1" {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func users(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "users") }

func createUser(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "created") }

func reports(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "reports") }

func report(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "report %s\n", r.PathValue("id")) }
