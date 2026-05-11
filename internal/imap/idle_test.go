package imap

import (
	"testing"
	"time"
)

func TestDefaultIdleConfig(t *testing.T) {
	cfg := DefaultIdleConfig()

	if cfg.Timeout != 29*time.Minute {
		t.Errorf("expected timeout 29m, got %v", cfg.Timeout)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("expected poll interval 30s, got %v", cfg.PollInterval)
	}
	if cfg.MaxIdleSessions != 0 {
		t.Errorf("expected unlimited max sessions, got %d", cfg.MaxIdleSessions)
	}
}

func TestIdleHandlerIncrementDecrementSessions(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)
	handler := NewIdleHandler(nil, bridge, testLogger())

	if handler.GetActiveIdleSessions() != 0 {
		t.Fatalf("expected 0 active sessions, got %d", handler.GetActiveIdleSessions())
	}

	if !handler.incrementIdleSessions() {
		t.Fatal("expected increment to succeed")
	}
	if handler.GetActiveIdleSessions() != 1 {
		t.Fatalf("expected 1 active session, got %d", handler.GetActiveIdleSessions())
	}

	handler.decrementIdleSessions()
	if handler.GetActiveIdleSessions() != 0 {
		t.Fatalf("expected 0 active sessions after decrement, got %d", handler.GetActiveIdleSessions())
	}
}

func TestIdleHandlerMaxSessionsLimit(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)

	cfg := &IdleConfig{
		Timeout:         5 * time.Minute,
		PollInterval:    10 * time.Second,
		MaxIdleSessions: 2,
	}
	handler := NewIdleHandler(cfg, bridge, testLogger())

	if !handler.incrementIdleSessions() {
		t.Fatal("first increment should succeed")
	}
	if !handler.incrementIdleSessions() {
		t.Fatal("second increment should succeed")
	}
	if handler.incrementIdleSessions() {
		t.Fatal("third increment should fail (max=2)")
	}

	handler.decrementIdleSessions()
	if !handler.incrementIdleSessions() {
		t.Fatal("increment after decrement should succeed")
	}
}

func TestIdleHandlerDecrementBelowZero(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)
	handler := NewIdleHandler(nil, bridge, testLogger())

	handler.decrementIdleSessions()
	if handler.GetActiveIdleSessions() != 0 {
		t.Fatalf("expected 0 after decrementing from 0, got %d", handler.GetActiveIdleSessions())
	}
}

func TestIdleManagerCreateRemoveHandler(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)
	mgr := NewIdleManager(nil, bridge, testLogger())

	h := mgr.CreateHandler("sess-1")
	if h == nil {
		t.Fatal("expected non-nil handler")
	}

	stats := mgr.GetStats()
	if stats.ActiveSessions != 1 {
		t.Fatalf("expected 1 active session, got %d", stats.ActiveSessions)
	}
	if stats.TotalSessions != 1 {
		t.Fatalf("expected total 1, got %d", stats.TotalSessions)
	}
	if stats.MaxConcurrentSessions != 1 {
		t.Fatalf("expected max concurrent 1, got %d", stats.MaxConcurrentSessions)
	}

	mgr.CreateHandler("sess-2")
	stats = mgr.GetStats()
	if stats.ActiveSessions != 2 {
		t.Fatalf("expected 2 active sessions, got %d", stats.ActiveSessions)
	}
	if stats.MaxConcurrentSessions != 2 {
		t.Fatalf("expected max concurrent 2, got %d", stats.MaxConcurrentSessions)
	}

	mgr.RemoveHandler("sess-1")
	stats = mgr.GetStats()
	if stats.ActiveSessions != 1 {
		t.Fatalf("expected 1 active session after remove, got %d", stats.ActiveSessions)
	}
	if stats.MaxConcurrentSessions != 2 {
		t.Fatalf("max concurrent should remain 2, got %d", stats.MaxConcurrentSessions)
	}

	if mgr.GetHandler("sess-1") != nil {
		t.Fatal("removed handler should be nil")
	}
	if mgr.GetHandler("sess-2") == nil {
		t.Fatal("sess-2 handler should still exist")
	}
}

func TestIdleManagerGetNotificationBridge(t *testing.T) {
	ns := newTestNotifyService()
	bridge := newTestBridge(ns)
	mgr := NewIdleManager(nil, bridge, testLogger())

	if mgr.GetNotificationBridge() != bridge {
		t.Fatal("expected same bridge instance")
	}
}
