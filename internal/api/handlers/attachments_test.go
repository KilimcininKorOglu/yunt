package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"yunt/internal/domain"
)

func setupAttachmentsTest(t *testing.T) (*fullTestEnv, string) {
	t.Helper()
	env := setupFullTest()
	token := env.loginAdmin(t)

	mbx := createTestMailbox("mbx-att", domain.ID("admin-ft"), "attach@localhost")
	env.repo.mboxes.add(mbx)

	msg := createTestMessage("msg-att", "mbx-att")
	env.repo.messages.add(msg)

	att := createTestAttachment("att-1", "msg-att")
	env.repo.attachs.add(att, []byte("file content"))

	return env, token
}

func TestAttachmentHandler_ListAttachments(t *testing.T) {
	env, token := setupAttachmentsTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/attachments", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAttachmentHandler_GetAttachment(t *testing.T) {
	env, token := setupAttachmentsTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/attachments/att-1", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAttachmentHandler_GetAttachment_NotFound(t *testing.T) {
	env, token := setupAttachmentsTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/attachments/nonexistent", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAttachmentHandler_DownloadAttachment(t *testing.T) {
	env, token := setupAttachmentsTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/attachments/att-1/download", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/plain" {
		t.Errorf("expected text/plain content-type, got %q", ct)
	}

	cd := rec.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header")
	}
}
