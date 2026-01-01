package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"yunt/internal/config"
)

func TestLogger(t *testing.T) {
	e := echo.New()
	logger := config.NewDefaultLogger()

	cfg := LoggerConfig{
		Logger:    logger,
		SkipPaths: []string{"/health"},
	}

	e.Use(Logger(cfg))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Test normal request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Test skipped path
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestLoggerWithConfig(t *testing.T) {
	e := echo.New()
	logger := config.NewDefaultLogger()

	e.Use(LoggerWithConfig(logger, "/health", "/ready"))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestLoggerStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"success", http.StatusOK},
		{"client error", http.StatusBadRequest},
		{"not found", http.StatusNotFound},
		{"server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			logger := config.NewDefaultLogger()

			e.Use(LoggerWithConfig(logger))

			e.GET("/test", func(c echo.Context) error {
				return c.String(tt.statusCode, "test")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, rec.Code)
			}
		})
	}
}
