package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"yunt/internal/domain"
)

func setupMailboxesTest(t *testing.T) (*fullTestEnv, string) {
	t.Helper()
	env := setupFullTest()
	token := env.loginAdmin(t)
	return env, token
}

func TestMailboxHandler_ListMailboxes(t *testing.T) {
	env, token := setupMailboxesTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/mailboxes", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_ListMailboxes_Unauthorized(t *testing.T) {
	env := setupFullTest()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mailboxes", nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMailboxHandler_CreateMailbox(t *testing.T) {
	env, token := setupMailboxesTest(t)

	body := map[string]interface{}{
		"name":    "Test Mailbox",
		"address": "testcreate@localhost",
	}
	req := makeAuthReq(http.MethodPost, "/api/v1/mailboxes", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_GetMailbox(t *testing.T) {
	env, token := setupMailboxesTest(t)

	mbx := createTestMailbox("mbx-get", domain.ID("admin-ft"), "get@localhost")
	env.repo.mboxes.add(mbx)

	req := makeAuthReq(http.MethodGet, "/api/v1/mailboxes/mbx-get", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_GetMailbox_NotFound(t *testing.T) {
	env, token := setupMailboxesTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/mailboxes/nonexistent", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_UpdateMailbox(t *testing.T) {
	env, token := setupMailboxesTest(t)

	mbx := createTestMailbox("mbx-upd", domain.ID("admin-ft"), "update@localhost")
	env.repo.mboxes.add(mbx)

	body := map[string]interface{}{
		"name":        "Updated Name",
		"description": "Updated description",
	}
	req := makeAuthReq(http.MethodPut, "/api/v1/mailboxes/mbx-upd", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_DeleteMailbox(t *testing.T) {
	env, token := setupMailboxesTest(t)

	mbx := createTestMailbox("mbx-del", domain.ID("admin-ft"), "delete@localhost")
	env.repo.mboxes.add(mbx)

	req := makeAuthReq(http.MethodDelete, "/api/v1/mailboxes/mbx-del", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_GetMailboxStats(t *testing.T) {
	env, token := setupMailboxesTest(t)

	mbx := createTestMailbox("mbx-stats", domain.ID("admin-ft"), "stats@localhost")
	env.repo.mboxes.add(mbx)

	req := makeAuthReq(http.MethodGet, "/api/v1/mailboxes/mbx-stats/stats", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_GetUserStats(t *testing.T) {
	env, token := setupMailboxesTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/mailboxes/stats", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_SetDefaultMailbox(t *testing.T) {
	env, token := setupMailboxesTest(t)

	mbx := createTestMailbox("mbx-setdef", domain.ID("admin-ft"), "setdefault@localhost")
	env.repo.mboxes.add(mbx)

	req := makeAuthReq(http.MethodPost, "/api/v1/mailboxes/mbx-setdef/default", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMailboxHandler_CreateMailbox_ResponseFormat(t *testing.T) {
	env, token := setupMailboxesTest(t)

	body := map[string]interface{}{
		"name":    "Format Test",
		"address": "formattest@localhost",
	}
	req := makeAuthReq(http.MethodPost, "/api/v1/mailboxes", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &raw)
	if _, ok := raw["data"]; !ok {
		t.Error("expected 'data' field in response")
	}
}
