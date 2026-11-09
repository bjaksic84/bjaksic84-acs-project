// Command shop serves a small web shop.
package main

import (
	"log"
	"net/http"

	"example.com/shop/internal/config"
	"example.com/shop/internal/flags"
	"example.com/shop/internal/web"
)

func main() {
	flags.Setup()
	cfg := config.Load()
	srv := web.New(cfg)
	log.Printf("shop listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, srv.Routes()))
}
