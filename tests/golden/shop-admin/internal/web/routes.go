package web

import (
	"net/http"

	"example.com/shop/internal/flags"
)

// Routes returns the shop's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /products", s.listProducts)
	mux.HandleFunc("GET /products/{id}", s.showProduct)
	mux.HandleFunc("POST /cart", s.addToCart)
	mux.HandleFunc("GET /admin/orders", flags.GateFunc(flags.Admin, s.adminOrders))
	mux.HandleFunc("POST /admin/orders/{id}/refund", flags.GateFunc(flags.Admin, s.refundOrder))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	return mux
}
