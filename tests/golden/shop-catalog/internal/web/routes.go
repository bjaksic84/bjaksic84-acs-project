package web

import (
	"net/http"

	"example.com/shop/internal/flags"
)

// Routes returns the shop's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /products", flags.GateFunc(flags.Catalog, s.listProducts))
	mux.HandleFunc("GET /products/{id}", flags.GateFunc(flags.Catalog, s.showProduct))
	mux.HandleFunc("POST /cart", s.addToCart)
	mux.HandleFunc("GET /admin/orders", s.adminOrders)
	mux.HandleFunc("POST /admin/orders/{id}/refund", s.refundOrder)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	return mux
}
