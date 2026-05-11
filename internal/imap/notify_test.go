package imap

import (
	"testing"
	"time"

	"github.com/rs/zerolog"

	"yunt/internal/domain"
	"yunt/internal/service"
)

func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

func newTestNotifyService() *service.NotifyService {
	return service.NewNotifyService()
}

func newTestBridge(ns *service.NotifyService) *NotificationBridge {
	return NewNotificationBridge(ns, testLogger())
}

func TestNotificationBridgeRegisterUnregister(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)

	if bridge.GetIdleSessionCount() != 0 {
		t.Fatalf("expected 0 sessions, got %d", bridge.GetIdleSessionCount())
	}

	info := bridge.RegisterIdleSession("sess-1", domain.ID("mbox-1"), domain.ID("user-1"), nil)
	if info == nil {
		t.Fatal("expected non-nil idle session info")
	}
	if info.SessionID != "sess-1" {
		t.Fatalf("expected session ID sess-1, got %s", info.SessionID)
	}
	if info.MailboxID != domain.ID("mbox-1") {
		t.Fatalf("expected mailbox ID mbox-1, got %s", info.MailboxID)
	}
	if bridge.GetIdleSessionCount() != 1 {
		t.Fatalf("expected 1 session, got %d", bridge.GetIdleSessionCount())
	}

	bridge.UnregisterIdleSession("sess-1")
	if bridge.GetIdleSessionCount() != 0 {
		t.Fatalf("expected 0 sessions after unregister, got %d", bridge.GetIdleSessionCount())
	}
}

func TestNotificationBridgeSessionsForMailbox(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)

	bridge.RegisterIdleSession("s1", domain.ID("mbox-A"), domain.ID("u1"), nil)
	bridge.RegisterIdleSession("s2", domain.ID("mbox-A"), domain.ID("u1"), nil)
	bridge.RegisterIdleSession("s3", domain.ID("mbox-B"), domain.ID("u1"), nil)

	if bridge.GetIdleSessionsForMailbox(domain.ID("mbox-A")) != 2 {
		t.Fatalf("expected 2 sessions for mbox-A, got %d", bridge.GetIdleSessionsForMailbox(domain.ID("mbox-A")))
	}
	if bridge.GetIdleSessionsForMailbox(domain.ID("mbox-B")) != 1 {
		t.Fatalf("expected 1 session for mbox-B, got %d", bridge.GetIdleSessionsForMailbox(domain.ID("mbox-B")))
	}
	if bridge.GetIdleSessionsForMailbox(domain.ID("mbox-C")) != 0 {
		t.Fatalf("expected 0 sessions for mbox-C, got %d", bridge.GetIdleSessionsForMailbox(domain.ID("mbox-C")))
	}

	bridge.UnregisterIdleSession("s1")
	if bridge.GetIdleSessionsForMailbox(domain.ID("mbox-A")) != 1 {
		t.Fatalf("expected 1 session for mbox-A after unregister, got %d", bridge.GetIdleSessionsForMailbox(domain.ID("mbox-A")))
	}
}

func TestNotificationBridgeUnregisterNonexistent(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)

	bridge.UnregisterIdleSession("nonexistent")
	if bridge.GetIdleSessionCount() != 0 {
		t.Fatalf("expected 0, got %d", bridge.GetIdleSessionCount())
	}
}

func TestNotificationBridgeChannelReceivesNotifications(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)

	info := bridge.RegisterIdleSession("sess-1", domain.ID("mbox-1"), domain.ID("user-1"), nil)

	ns.NotifyNewMessage(domain.ID("mbox-1"), domain.ID("user-1"), domain.ID("msg-1"), 1, 1, 1)

	select {
	case n := <-info.NotifyChan:
		if n.Type != service.NotificationNewMessage {
			t.Fatalf("expected NotificationNewMessage, got %s", n.Type.String())
		}
		if n.MessageID != domain.ID("msg-1") {
			t.Fatalf("expected msg-1, got %s", n.MessageID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for notification")
	}

	bridge.UnregisterIdleSession("sess-1")
}

func TestIdleSessionInfoClosedState(t *testing.T) {
	info := &IdleSessionInfo{SessionID: "test"}

	if info.IsClosed() {
		t.Fatal("new session should not be closed")
	}

	info.Close()
	if !info.IsClosed() {
		t.Fatal("session should be closed after Close()")
	}
}

func TestNotificationBridgeGetNotifyService(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)

	if bridge.GetNotifyService() != ns {
		t.Fatal("expected same notify service instance")
	}
}
