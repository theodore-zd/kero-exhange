package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/wispberry-tech/kero-exchange/internal/config"
	"github.com/wispberry-tech/kero-exchange/internal/handlers"
)

func TestRouteRegistration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	r := chi.NewRouter()
	cfg := &config.Config{
		AdminPassword:     "test-password",
		AdminPasswordHash: "test-hash",
	}
	handlers.RegisterRoutes(r, testPool, cfg)

	testCases := []struct {
		name         string
		method       string
		route        string
		body         string
		expectedCode int
	}{
		{"Health check", "GET", "/health", "", 200},
		{"Sign in public", "POST", "/api/v1/auth/signin", `{}`, 400},
		{"Wallets need auth", "GET", "/api/v1/wallets", "", 401},
		{"Currencies need auth", "GET", "/api/v1/currencies", "", 401},
		{"Balances need auth", "GET", "/api/v1/balances", "", 401},
		{"Transactions need auth", "GET", "/api/v1/transactions", "", 401},
		{"Signup needs API key", "POST", "/api/v1/auth/signup", `{}`, 401},
		{"Ref code needs API key", "POST", "/api/v1/providers/reference-codes", ``, 401},
		{"Admin providers need auth", "GET", "/api/v1/admin/providers", "", 401},
		{"Admin wallets need auth", "GET", "/api/v1/admin/wallets", "", 401},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.route, strings.NewReader(tc.body))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("Expected %d, got %d", tc.expectedCode, w.Code)
			}
		})
	}
}

func TestMiddlewareOrder(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	r := chi.NewRouter()
	cfg := &config.Config{
		AdminPassword:     "test-password",
		AdminPasswordHash: "test-hash",
	}
	handlers.RegisterRoutes(r, testPool, cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health check failed with status %d", w.Code)
	}

	if w.Header().Get("Content-Type") == "" {
		t.Error("Content-Type header should be set")
	}
}
