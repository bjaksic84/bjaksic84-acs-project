// Package api serves a small API with chi.
package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Router returns the API's routes.
func Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", health)
	r.Get("/items", listItems)
	r.Post("/items", createItem)
	r.Route("/admin", func(r chi.Router) {
		r.Get("/stats", stats)
		r.Delete("/items/{id}", deleteItem)
	})
	r.Handle("/files/*", http.StripPrefix("/files/", http.FileServer(http.Dir("files"))))
	return r
}

func health(w http.ResponseWriter, r *http.Request)     { fmt.Fprintln(w, "ok") }
func listItems(w http.ResponseWriter, r *http.Request)  { fmt.Fprintln(w, "items") }
func createItem(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "created") }
func stats(w http.ResponseWriter, r *http.Request)      { fmt.Fprintln(w, "stats") }
func deleteItem(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "deleted", chi.URLParam(r, "id"))
}
