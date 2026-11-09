// Command lectures serves lecture pages, in the style of the Glas app: a
// server wrapper, route tables, and configuration read through a lookup
// function.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"example.com/lectures/internal/admin"
	"example.com/lectures/internal/config"
	"example.com/lectures/internal/debug"
	"example.com/lectures/internal/httpx"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		log.Print(err)
		return 1
	}
	srv := httpx.New()
	srv.Handle("GET /login", http.HandlerFunc(loginPage))
	srv.Handle("POST /login", http.HandlerFunc(login))
	for _, rt := range admin.Routes() {
		srv.Handle(rt.Pattern, rt.Handler)
	}
	for _, rt := range admin.ReportRoutes() {
		srv.Handle(rt.Pattern, rt.Handler)
	}
	if cfg.QA {
		srv.Handle("GET /qa", http.HandlerFunc(questions))
	}
	debug.Register()
	if err := http.ListenAndServe(cfg.Addr, srv); err != nil {
		log.Print(err)
		return 1
	}
	return 0
}

func loginPage(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "login form") }

func login(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "logged in") }

func questions(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "questions") }
