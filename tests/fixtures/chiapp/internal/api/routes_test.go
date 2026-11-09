package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutes(t *testing.T) {
	h := Router()
	for _, tc := range []struct{ method, path string }{
		{"GET", "/health"}, {"GET", "/items"}, {"POST", "/items"}, {"GET", "/admin/stats"}, {"DELETE", "/admin/items/3"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s %s = %d", tc.method, tc.path, rec.Code)
		}
	}
}
