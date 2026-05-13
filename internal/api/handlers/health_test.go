package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"yunt/internal/repository"
)

type mockHealthRepo struct {
	healthErr error
}

func (r *mockHealthRepo) Health(ctx context.Context) error                                      { return r.healthErr }
func (r *mockHealthRepo) Close() error                                                          { return nil }
func (r *mockHealthRepo) Users() repository.UserRepository                                      { return nil }
func (r *mockHealthRepo) Mailboxes() repository.MailboxRepository                                { return nil }
func (r *mockHealthRepo) Messages() repository.MessageRepository                                 { return nil }
func (r *mockHealthRepo) Attachments() repository.AttachmentRepository                           { return nil }
func (r *mockHealthRepo) Webhooks() repository.WebhookRepository                                 { return nil }
func (r *mockHealthRepo) Settings() repository.SettingsRepository                                { return nil }
func (r *mockHealthRepo) JMAP() repository.JMAPRepository                                       { return nil }
func (r *mockHealthRepo) Transaction(_ context.Context, _ func(tx repository.Repository) error) error { return nil }
func (r *mockHealthRepo) TransactionWithOptions(_ context.Context, _ repository.TransactionOptions, _ func(tx repository.Repository) error) error { return nil }

var _ repository.Repository = (*mockHealthRepo)(nil)

type mockServiceChecker struct {
	running bool
}

func (m *mockServiceChecker) IsRunning() bool { return m.running }

func setupHealthHandler(repo repository.Repository, opts ...func(*HealthHandler)) (*echo.Echo, *HealthHandler) {
	e := echo.New()
	h := NewHealthHandler(repo, "test-v1")
	for _, opt := range opts {
		opt(h)
	}
	h.RegisterRoutes(e)
	return e, h
}

func TestHealthHandler_Healthz(t *testing.T) {
	e, _ := setupHealthHandler(&mockHealthRepo{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("expected OK, got %s", rec.Body.String())
	}
}

func TestHealthHandler_Ready(t *testing.T) {
	tests := []struct {
		name       string
		repo       *mockHealthRepo
		setup      func(*HealthHandler)
		wantStatus int
	}{
		{
			name:       "ready with healthy db",
			repo:       &mockHealthRepo{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not ready with db error",
			repo:       &mockHealthRepo{healthErr: fmt.Errorf("connection refused")},
			wantStatus: 503,
		},
		{
			name: "not ready with smtp down",
			repo: &mockHealthRepo{},
			setup: func(h *HealthHandler) {
				h.smtpEnabled = true
				h.smtpServer = &mockServiceChecker{running: false}
			},
			wantStatus: 503,
		},
		{
			name: "ready with smtp up",
			repo: &mockHealthRepo{},
			setup: func(h *HealthHandler) {
				h.smtpEnabled = true
				h.smtpServer = &mockServiceChecker{running: true}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not ready with imap down",
			repo: &mockHealthRepo{},
			setup: func(h *HealthHandler) {
				h.imapEnabled = true
				h.imapServer = &mockServiceChecker{running: false}
			},
			wantStatus: 503,
		},
		{
			name:       "ready with nil repo",
			repo:       nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var opts []func(*HealthHandler)
			if tc.setup != nil {
				opts = append(opts, tc.setup)
			}
			var e *echo.Echo
			if tc.repo == nil {
				e = echo.New()
				h := &HealthHandler{version: "test"}
				if tc.setup != nil {
					tc.setup(h)
				}
				h.RegisterRoutes(e)
			} else {
				e, _ = setupHealthHandler(tc.repo, opts...)
			}

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d, body: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHealthHandler_Health(t *testing.T) {
	tests := []struct {
		name       string
		repo       *mockHealthRepo
		setup      func(*HealthHandler)
		wantStatus int
		wantHealth string
	}{
		{
			name:       "all healthy",
			repo:       &mockHealthRepo{},
			wantStatus: http.StatusOK,
			wantHealth: HealthStatusHealthy,
		},
		{
			name:       "unhealthy db",
			repo:       &mockHealthRepo{healthErr: fmt.Errorf("db down")},
			wantStatus: 503,
			wantHealth: HealthStatusUnhealthy,
		},
		{
			name: "smtp enabled but not running",
			repo: &mockHealthRepo{},
			setup: func(h *HealthHandler) {
				h.smtpEnabled = true
				h.smtpServer = &mockServiceChecker{running: false}
			},
			wantStatus: 503,
			wantHealth: HealthStatusUnhealthy,
		},
		{
			name: "smtp enabled and running",
			repo: &mockHealthRepo{},
			setup: func(h *HealthHandler) {
				h.smtpEnabled = true
				h.smtpServer = &mockServiceChecker{running: true}
			},
			wantStatus: http.StatusOK,
			wantHealth: HealthStatusHealthy,
		},
		{
			name: "smtp enabled but nil server",
			repo: &mockHealthRepo{},
			setup: func(h *HealthHandler) {
				h.smtpEnabled = true
			},
			wantStatus: 503,
			wantHealth: HealthStatusUnhealthy,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var opts []func(*HealthHandler)
			if tc.setup != nil {
				opts = append(opts, tc.setup)
			}
			e, _ := setupHealthHandler(tc.repo, opts...)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d, body: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}

			var raw map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			var data map[string]interface{}
			if d, ok := raw["data"].(map[string]interface{}); ok {
				data = d
			} else if errObj, ok := raw["error"].(map[string]interface{}); ok {
				if details, ok := errObj["details"].(map[string]interface{}); ok {
					data = details
				}
			}
			if data == nil {
				t.Fatalf("could not extract health data from response: %s", rec.Body.String())
			}
			if status, _ := data["status"].(string); status != tc.wantHealth {
				t.Errorf("expected health status %q, got %q", tc.wantHealth, status)
			}
		})
	}
}

func TestHealthHandler_HealthDetails(t *testing.T) {
	e, _ := setupHealthHandler(&mockHealthRepo{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data field in response: %s", rec.Body.String())
	}

	details, ok := data["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected details in data: %s", rec.Body.String())
	}

	for _, component := range []string{"api", "database", "smtp", "imap"} {
		if _, ok := details[component]; !ok {
			t.Errorf("missing detail component: %s", component)
		}
	}

	if uptime, ok := data["uptime"].(float64); ok && uptime < 0 {
		t.Error("expected non-negative uptime")
	}
}

func TestGetRuntimeInfo(t *testing.T) {
	info := GetRuntimeInfo()
	if info.GoVersion == "" {
		t.Error("expected GoVersion to be set")
	}
	if info.NumCPU <= 0 {
		t.Error("expected NumCPU > 0")
	}
	if info.OS == "" {
		t.Error("expected OS to be set")
	}
	if info.Arch == "" {
		t.Error("expected Arch to be set")
	}
}
