// Package httpx wraps http.ServeMux the way Glas's httpx does.
package httpx

import (
	"fmt"
	"net/http"
)

// Server is the app's HTTP server: the app mux and an ops mux.
type Server struct {
	mux *http.ServeMux
	ops *http.ServeMux
}

// New returns a server with its built-in routes.
func New() *Server {
	s := &Server{mux: http.NewServeMux(), ops: http.NewServeMux()}
	s.Handle("GET /version", http.HandlerFunc(version))
	s.HandleOps("GET /version", http.HandlerFunc(version))
	s.HandleOps("GET /livez", http.HandlerFunc(livez))
	return s
}

// Handle registers an app route, with the patterns of http.ServeMux.
func (s *Server) Handle(pattern string, h http.Handler) {
	s.mux.Handle(pattern, h)
}

// HandleOps registers an ops route.
func (s *Server) HandleOps(pattern string, h http.Handler) {
	s.ops.Handle(pattern, h)
}

// ServeHTTP serves the app routes.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Ops serves the ops routes.
func (s *Server) Ops() http.Handler { return s.ops }

func version(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "v1") }

func livez(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") }
