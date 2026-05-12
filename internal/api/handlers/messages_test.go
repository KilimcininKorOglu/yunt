package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"yunt/internal/domain"
)

func setupMessagesTest(t *testing.T) (*fullTestEnv, string) {
	t.Helper()
	env := setupFullTest()
	token := env.loginAdmin(t)

	mbx := createTestMailbox("mbx-msg", domain.ID("admin-ft"), "inbox@localhost")
	mbx.IsDefault = true
	env.repo.mboxes.add(mbx)

	return env, token
}

func TestMessageHandler_ListMessages(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-1", "mbx-msg")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodGet, "/api/v1/messages", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &raw)
	if data, ok := raw["data"].(map[string]interface{}); ok {
		if items, ok := data["items"].([]interface{}); ok {
			if len(items) == 0 {
				t.Error("expected at least 1 message")
			}
		}
	}
}

func TestMessageHandler_ListMessages_Unauthorized(t *testing.T) {
	env := setupFullTest()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMessageHandler_GetMessage(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-get", "mbx-msg")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodGet, "/api/v1/messages/msg-get", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_GetMessage_NotFound(t *testing.T) {
	env, token := setupMessagesTest(t)

	req := makeAuthReq(http.MethodGet, "/api/v1/messages/nonexistent", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_GetMessageHTML(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-html", "mbx-msg")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodGet, "/api/v1/messages/msg-html/html", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=UTF-8" && ct != "text/html" {
		t.Errorf("expected text/html content-type, got %q", ct)
	}
}

func TestMessageHandler_GetMessageText(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-text", "mbx-msg")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodGet, "/api/v1/messages/msg-text/text", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_GetMessageRaw(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-raw", "mbx-msg")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodGet, "/api/v1/messages/msg-raw/raw", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "message/rfc822" {
		t.Errorf("expected message/rfc822 content-type, got %q", ct)
	}
}

func TestMessageHandler_DeleteMessage(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-del", "mbx-msg")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodDelete, "/api/v1/messages/msg-del", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_MarkAsRead(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-read", "mbx-msg")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodPut, "/api/v1/messages/msg-read/read", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_MarkAsUnread(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-unread", "mbx-msg")
	msg.Status = domain.MessageRead
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodPut, "/api/v1/messages/msg-unread/unread", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_Star(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-star", "mbx-msg")
	env.repo.messages.add(msg)

	req := makeAuthReq(http.MethodPut, "/api/v1/messages/msg-star/star", token, nil)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_MoveMessage(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-move", "mbx-msg")
	env.repo.messages.add(msg)

	target := createTestMailbox("mbx-target", domain.ID("admin-ft"), "archive@localhost")
	env.repo.mboxes.add(target)

	body := map[string]string{"targetMailboxId": "mbx-target"}
	req := makeAuthReq(http.MethodPut, "/api/v1/messages/msg-move/move", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_MoveMessage_BadRequest(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg := createTestMessage("msg-move2", "mbx-msg")
	env.repo.messages.add(msg)

	body := map[string]string{"targetMailboxId": ""}
	req := makeAuthReq(http.MethodPut, "/api/v1/messages/msg-move2/move", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_BulkMarkAsRead(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg1 := createTestMessage("msg-bulk1", "mbx-msg")
	msg2 := createTestMessage("msg-bulk2", "mbx-msg")
	env.repo.messages.add(msg1)
	env.repo.messages.add(msg2)

	body := map[string]interface{}{"ids": []string{"msg-bulk1", "msg-bulk2"}}
	req := makeAuthReq(http.MethodPost, "/api/v1/messages/bulk/read", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMessageHandler_BulkDelete(t *testing.T) {
	env, token := setupMessagesTest(t)

	msg1 := createTestMessage("msg-bdel1", "mbx-msg")
	msg2 := createTestMessage("msg-bdel2", "mbx-msg")
	env.repo.messages.add(msg1)
	env.repo.messages.add(msg2)

	body := map[string]interface{}{"ids": []string{"msg-bdel1", "msg-bdel2"}}
	req := makeAuthReq(http.MethodPost, "/api/v1/messages/bulk/delete", token, body)
	rec := httptest.NewRecorder()
	env.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
