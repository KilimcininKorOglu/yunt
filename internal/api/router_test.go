package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"yunt/internal/config"
)

func setupTestRouter() *RouterConfig {
	logger := config.NewDefaultLogger()
	return &RouterConfig{
		Logger:        logger,
		CORSOrigins:   []string{"*"},
		EnableSwagger: false,
	}
}

func TestNewRouter(t *testing.T) {
	cfg := setupTestRouter()
	e := NewRouter(*cfg)

	if e == nil {
		t.Fatal("NewRouter returned nil")
	}

	if !e.HideBanner {
		t.Error("expected HideBanner to be true")
	}

	if !e.HidePort {
		t.Error("expected HidePort to be true")
	}
}

func TestVersionEndpoint(t *testing.T) {
	cfg := setupTestRouter()
	e := NewRouter(*cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if _, ok := data["version"]; !ok {
		t.Error("expected version field to be present")
	}

	if _, ok := data["goVersion"]; !ok {
		t.Error("expected goVersion field to be present")
	}
}

func TestSetVersion(t *testing.T) {
	oldVersion := version
	defer func() { version = oldVersion }()

	SetVersion("1.0.0")
	if version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", version)
	}
}

func TestCORSHeaders(t *testing.T) {
	cfg := setupTestRouter()
	e := NewRouter(*cfg)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/version", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d for OPTIONS, got %d", http.StatusNoContent, rec.Code)
	}

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin to be *, got %s", origin)
	}
}

func TestNotFoundRoute(t *testing.T) {
	cfg := setupTestRouter()
	e := NewRouter(*cfg)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
