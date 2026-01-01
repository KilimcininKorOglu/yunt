package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCORS(t *testing.T) {
	e := echo.New()

	cfg := DefaultCORSConfig()
	e.Use(CORS(cfg))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Test normal request with origin
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin to be *, got %s", origin)
	}
}

func TestCORSPreflight(t *testing.T) {
	e := echo.New()

	cfg := DefaultCORSConfig()
	e.Use(CORS(cfg))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Test preflight request
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}

	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Error("expected Access-Control-Allow-Headers to be set")
	}

	maxAge := rec.Header().Get("Access-Control-Max-Age")
	if maxAge == "" {
		t.Error("expected Access-Control-Max-Age to be set")
	}
}

func TestCORSSpecificOrigins(t *testing.T) {
	e := echo.New()

	cfg := CORSConfig{
		AllowOrigins: []string{"http://allowed.com"},
	}
	e.Use(CORS(cfg))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Test allowed origin
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://allowed.com")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://allowed.com" {
		t.Errorf("expected Access-Control-Allow-Origin to be http://allowed.com, got %s", origin)
	}

	// Test disallowed origin
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://disallowed.com")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	origin = rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("expected no Access-Control-Allow-Origin header, got %s", origin)
	}
}

func TestCORSWithCredentials(t *testing.T) {
	e := echo.New()

	cfg := CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}
	e.Use(CORS(cfg))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	credentials := rec.Header().Get("Access-Control-Allow-Credentials")
	if credentials != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials to be true, got %s", credentials)
	}
}

func TestCORSExposeHeaders(t *testing.T) {
	e := echo.New()

	cfg := CORSConfig{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{"X-Custom-Header", "X-Another-Header"},
	}
	e.Use(CORS(cfg))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	expose := rec.Header().Get("Access-Control-Expose-Headers")
	if expose == "" {
		t.Error("expected Access-Control-Expose-Headers to be set")
	}
}

func TestCORSWithOrigins(t *testing.T) {
	e := echo.New()

	e.Use(CORSWithOrigins([]string{"http://example.com"}))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin to be http://example.com, got %s", origin)
	}
}

func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	if len(cfg.AllowOrigins) != 1 || cfg.AllowOrigins[0] != "*" {
		t.Error("expected AllowOrigins to contain *")
	}

	if len(cfg.AllowMethods) == 0 {
		t.Error("expected AllowMethods to not be empty")
	}

	if len(cfg.AllowHeaders) == 0 {
		t.Error("expected AllowHeaders to not be empty")
	}

	if cfg.MaxAge != 86400 {
		t.Errorf("expected MaxAge to be 86400, got %d", cfg.MaxAge)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{86400, "86400"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}
