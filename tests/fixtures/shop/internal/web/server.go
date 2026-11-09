// Package web serves the shop's pages.
package web

import (
	"fmt"
	"net/http"

	"example.com/shop/internal/config"
)

// Server holds what the handlers need.
type Server struct {
	cfg config.Config
}

// New returns a server for cfg.
func New(cfg config.Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "welcome to the shop")
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "products: tea, coffee")
	if s.cfg.Recommendations {
		fmt.Fprintln(w, "recommended: tea")
	}
}

func (s *Server) showProduct(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "product %s\n", r.PathValue("id"))
}

func (s *Server) addToCart(w http.ResponseWriter, r *http.Request) {
	if s.cfg.BetaCheckout {
		fmt.Fprintln(w, "added (beta checkout)")
		return
	}
	fmt.Fprintln(w, "added")
}

func (s *Server) adminOrders(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "orders: 1")
}

func (s *Server) refundOrder(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "refunded %s\n", r.PathValue("id"))
}
