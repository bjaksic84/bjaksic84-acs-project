package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"example.com/shop/internal/config"
)

func TestRoutes(t *testing.T) {
	h := New(config.Config{}).Routes()
	for _, tc := range []struct {
		method, path, want string
	}{
		{"GET", "/", "welcome"},
		{"GET", "/products", "products"},
		{"GET", "/products/7", "product 7"},
		{"POST", "/cart", "added"},
		{"GET", "/admin/orders", "orders"},
		{"POST", "/admin/orders/3/refund", "refunded 3"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s %s = %d %q, want 200 containing %q", tc.method, tc.path, rec.Code, rec.Body.String(), tc.want)
		}
	}
}

// A toggle read in a test is not a toggle of the app.
func TestEnvInTests(t *testing.T) {
	_ = os.Getenv("FEATURE_ONLY_IN_TESTS") == "true"
}
