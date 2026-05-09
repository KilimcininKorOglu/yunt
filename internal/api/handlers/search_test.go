package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"yunt/internal/domain"
)

func setupSearchTest(t *testing.T) (*fullTestEnv, string) {
	t.Helper()
	env := setupFullTest()
	token := env.loginAdmin(t)

	mbx := createTestMailbox("mbx-search", domain.ID("admin-ft"), "search@localhost")
	env.repo.mboxes.add(mbx)

	msg := createTestMessage("msg-search", "mbx-search")
	env.repo.messages.add(msg)

	return env, token
}

func TestSearchHandler_SimpleSearch(t *testing.T) {
	env, token := setupSearchTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/search/simple?q=test+subject", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchHandler_SimpleSearch_EmptyQuery(t *testing.T) {
	env, token := setupSearchTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/search/simple?q=", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchHandler_SimpleSearch_ShortQuery(t *testing.T) {
	env, token := setupSearchTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/search/simple?q=x", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchHandler_AdvancedSearch(t *testing.T) {
	env, token := setupSearchTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/search/advanced?from=sender", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchHandler_AdvancedSearch_Unauthorized(t *testing.T) {
	env := setupFullTest()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/advanced?from=sender", nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
