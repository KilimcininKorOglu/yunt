package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"yunt/internal/domain"
)

func setupWebhooksTest(t *testing.T) (*fullTestEnv, string) {
	t.Helper()
	env := setupFullTest()
	token := env.loginAdmin(t)
	return env, token
}

func TestWebhookHandler_CreateWebhook(t *testing.T) {
	env, token := setupWebhooksTest(t)

	body := map[string]interface{}{
		"name":   "Test Webhook",
		"url":    "https://example.com/hook",
		"events": []string{"message.received"},
	}
	req := makeAuthReq(http.MethodPost, "/api/v1/webhooks", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_GetWebhook(t *testing.T) {
	env, token := setupWebhooksTest(t)

	wh := createTestWebhook("wh-get", domain.ID("admin-ft"))
	env.repo.webhooks.add(wh)

	req := makeAuthReq(http.MethodGet, "/api/v1/webhooks/wh-get", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_GetWebhook_NotFound(t *testing.T) {
	env, token := setupWebhooksTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/webhooks/nonexistent", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_ListWebhooks(t *testing.T) {
	env, token := setupWebhooksTest(t)

	wh := createTestWebhook("wh-list", domain.ID("admin-ft"))
	env.repo.webhooks.add(wh)

	req := makeAuthReq(http.MethodGet, "/api/v1/webhooks", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_UpdateWebhook(t *testing.T) {
	env, token := setupWebhooksTest(t)

	wh := createTestWebhook("wh-upd", domain.ID("admin-ft"))
	env.repo.webhooks.add(wh)

	body := map[string]interface{}{
		"name": "Updated Webhook",
	}
	req := makeAuthReq(http.MethodPut, "/api/v1/webhooks/wh-upd", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_DeleteWebhook(t *testing.T) {
	env, token := setupWebhooksTest(t)

	wh := createTestWebhook("wh-del", domain.ID("admin-ft"))
	env.repo.webhooks.add(wh)

	req := makeAuthReq(http.MethodDelete, "/api/v1/webhooks/wh-del", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_ActivateWebhook(t *testing.T) {
	env, token := setupWebhooksTest(t)

	wh := createTestWebhook("wh-act", domain.ID("admin-ft"))
	wh.Status = domain.WebhookStatusInactive
	env.repo.webhooks.add(wh)

	req := makeAuthReq(http.MethodPost, "/api/v1/webhooks/wh-act/activate", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_DeactivateWebhook(t *testing.T) {
	env, token := setupWebhooksTest(t)

	wh := createTestWebhook("wh-deact", domain.ID("admin-ft"))
	env.repo.webhooks.add(wh)

	req := makeAuthReq(http.MethodPost, "/api/v1/webhooks/wh-deact/deactivate", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_ListDeliveries(t *testing.T) {
	env, token := setupWebhooksTest(t)

	wh := createTestWebhook("wh-dlvr", domain.ID("admin-ft"))
	env.repo.webhooks.add(wh)

	req := makeAuthReq(http.MethodGet, "/api/v1/webhooks/wh-dlvr/deliveries", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebhookHandler_GetDeliveryStats(t *testing.T) {
	env, token := setupWebhooksTest(t)

	wh := createTestWebhook("wh-stats", domain.ID("admin-ft"))
	env.repo.webhooks.add(wh)

	req := makeAuthReq(http.MethodGet, "/api/v1/webhooks/wh-stats/stats", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
