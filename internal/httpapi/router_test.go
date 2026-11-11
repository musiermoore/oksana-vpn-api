package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/musiermoore/oksana-vpn-api/internal/account"
	"github.com/musiermoore/oksana-vpn-api/internal/auth"
)

func TestRoutesKeepExistingPathsAndProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(nil, auth.NewHandler(nil, ""), account.NewHandler(nil), nil)

	want := map[string]bool{
		"GET /health":                              true,
		"GET /readyz":                              true,
		"POST /api/v1/auth/login":                  true,
		"POST /api/v1/auth/telegram":               true,
		"GET /api/v1/auth/telegram/oidc/start":     true,
		"GET /api/v1/auth/telegram/oidc/callback":  true,
		"POST /api/v1/auth/telegram/oidc/exchange": true,
		"GET /api/v1/auth/me":                      true,
		"POST /api/v1/auth/logout":                 true,
		"PUT /api/v1/auth/credentials":             true,
		"GET /api/v1/vless/link":                   true,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if !want[key] {
			t.Errorf("unexpected route %s", key)
		}
		delete(want, key)
	}
	for route := range want {
		t.Errorf("missing route %s", route)
	}

	for _, path := range []string{"/api/v1/auth/me", "/api/v1/auth/logout", "/api/v1/vless/link"} {
		method := http.MethodGet
		if path == "/api/v1/auth/logout" {
			method = http.MethodPost
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s without a token returned %d; want 401", path, response.Code)
		}
	}

	for _, path := range []string{"/api/v1/auth/login", "/api/v1/auth/telegram"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, nil)
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Errorf("%s without a token returned %d; want public request validation (400)", path, response.Code)
		}
	}

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/auth/telegram/oidc/start"},
		{http.MethodGet, "/api/v1/auth/telegram/oidc/callback"},
		{http.MethodPost, "/api/v1/auth/telegram/oidc/exchange"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(route.method, route.path, nil)
		router.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s without configuration returned %d; want 503", route.method, route.path, response.Code)
		}
	}
}
