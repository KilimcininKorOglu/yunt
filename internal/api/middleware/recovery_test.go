package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"yunt/internal/config"
)

func TestRecovery(t *testing.T) {
	e := echo.New()
	logger := config.NewDefaultLogger()

	e.Use(RecoveryWithLogger(logger))

	e.GET("/panic", func(c echo.Context) error {
		panic("test panic")
	})

	e.GET("/normal", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Test panic recovery
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["success"] != false {
		t.Error("expected success to be false")
	}

	errData, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error to be present")
	}

	if errData["code"] != "INTERNAL_SERVER_ERROR" {
		t.Errorf("expected code INTERNAL_SERVER_ERROR, got %v", errData["code"])
	}
}

func TestRecoveryNormalRequest(t *testing.T) {
	e := echo.New()
	logger := config.NewDefaultLogger()

	e.Use(RecoveryWithLogger(logger))

	e.GET("/normal", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/normal", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %s", rec.Body.String())
	}
}

func TestRecoveryWithConfig(t *testing.T) {
	e := echo.New()
	logger := config.NewDefaultLogger()

	cfg := RecoveryConfig{
		Logger:            logger,
		StackSize:         2 << 10, // 2KB
		DisableStackAll:   true,
		DisablePrintStack: true,
	}

	e.Use(Recovery(cfg))

	e.GET("/panic", func(c echo.Context) error {
		panic("custom config panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestRecoveryErrorPanic(t *testing.T) {
	e := echo.New()
	logger := config.NewDefaultLogger()

	e.Use(RecoveryWithLogger(logger))

	e.GET("/error-panic", func(c echo.Context) error {
		panic(http.ErrAbortHandler)
	})

	req := httptest.NewRequest(http.MethodGet, "/error-panic", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestDefaultRecoveryConfig(t *testing.T) {
	cfg := DefaultRecoveryConfig()

	if cfg.StackSize != 4<<10 {
		t.Errorf("expected StackSize %d, got %d", 4<<10, cfg.StackSize)
	}

	if cfg.DisableStackAll != false {
		t.Error("expected DisableStackAll to be false")
	}

	if cfg.DisablePrintStack != false {
		t.Error("expected DisablePrintStack to be false")
	}
}
