package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutes(t *testing.T) {
	mux := http.NewServeMux()
	for _, rt := range append(Routes(), ReportRoutes()...) {
		mux.Handle(rt.Pattern, rt.Handler)
	}
	for _, tc := range []struct {
		method, path string
		admin        bool
		want         int
	}{
		{"GET", "/admin/users", false, 200},
		{"POST", "/admin/users", false, 404},
		{"POST", "/admin/users", true, 200},
		{"GET", "/reports", false, 200},
		{"GET", "/reports/4", false, 200},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if tc.admin {
			req.Header.Set("X-Admin", "1")
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("%s %s admin=%v: %d, want %d", tc.method, tc.path, tc.admin, rec.Code, tc.want)
		}
	}
}
