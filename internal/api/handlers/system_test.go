package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"yunt/internal/domain"
)

func setupSystemTest(t *testing.T) (*fullTestEnv, string) {
	t.Helper()
	env := setupFullTest()
	token := env.loginAdmin(t)
	return env, token
}

func TestSystemHandler_GetVersion(t *testing.T) {
	env := setupFullTest()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/version", nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &raw)
	if data, ok := raw["data"].(map[string]interface{}); ok {
		if v, _ := data["version"].(string); v != "test-v1" {
			t.Errorf("expected version 'test-v1', got %q", v)
		}
	}
}

func TestSystemHandler_GetStats(t *testing.T) {
	env, token := setupSystemTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/system/stats", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSystemHandler_GetStats_Unauthorized(t *testing.T) {
	env := setupFullTest()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/stats", nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestSystemHandler_GetSystemInfo(t *testing.T) {
	env, token := setupSystemTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/system/info", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &raw)
	if data, ok := raw["data"].(map[string]interface{}); ok {
		if _, ok := data["runtime"]; !ok {
			t.Error("expected 'runtime' field in system info")
		}
	}
}

func TestSystemHandler_GetSystemInfo_Forbidden(t *testing.T) {
	env := setupFullTest()

	regularUser := createTestUser("user-regular", "regular", "password123")
	regularUser.Role = domain.RoleUser
	env.repo.users.addUser(regularUser)

	mbx := createTestMailbox("mbx-reg", regularUser.ID, "regular@localhost")
	mbx.IsDefault = true
	env.repo.mboxes.add(mbx)

	token := loginForToken(t, env.echo, "regular", "password123")

	req := makeAuthReq(http.MethodGet, "/api/v1/system/info", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSystemHandler_DeleteAllMessages(t *testing.T) {
	env, token := setupSystemTest(t)

	mbx := createTestMailbox("mbx-delall", domain.ID("admin-ft"), "delall@localhost")
	env.repo.mboxes.add(mbx)

	msg := createTestMessage("msg-delall", "mbx-delall")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodDelete, "/api/v1/system/messages", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
