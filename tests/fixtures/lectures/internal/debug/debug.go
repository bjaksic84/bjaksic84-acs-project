// Package debug registers debugging pages on the default mux.
package debug

import (
	"fmt"
	"net/http"
	"os"
)

// Register adds the debug pages, unless LECTURES_DEBUG is "off".
func Register() {
	if os.Getenv("LECTURES_DEBUG") == "off" {
		return
	}
	http.HandleFunc("/debug/vars", vars)
	pattern := "/debug/" + os.Getenv("LECTURES_DEBUG_PAGE")
	http.HandleFunc(pattern, vars) // not a constant pattern: not a candidate
}

func vars(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "vars") }
