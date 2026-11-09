package web

import "net/http"

// Routes returns the shop's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /products", s.listProducts)
	mux.HandleFunc("GET /products/{id}", s.showProduct)
	mux.HandleFunc("POST /cart", s.addToCart)
	mux.HandleFunc("GET /admin/orders", s.adminOrders)
	mux.HandleFunc("POST /admin/orders/{id}/refund", s.refundOrder)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	return mux
}
