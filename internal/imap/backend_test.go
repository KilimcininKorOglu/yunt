package imap

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"yunt/internal/domain"
)

func TestNewBackend(t *testing.T) {
	repo := newMockRepository()
	logger := zerolog.Nop()

	t.Run("with default config", func(t *testing.T) {
		backend := NewBackend(repo, logger, nil)
		if backend == nil {
			t.Error("NewBackend() returned nil")
		}
		if backend.SessionCount() != 0 {
			t.Errorf("SessionCount() = %v, want 0", backend.SessionCount())
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &BackendConfig{
			SessionTimeout: 1 * time.Hour,
		}
		backend := NewBackend(repo, logger, cfg)
		if backend == nil {
			t.Error("NewBackend() returned nil")
		}
	})
}

func TestBackend_Login(t *testing.T) {
	repo := newMockRepository()
	logger := zerolog.Nop()

	// Add a test user
	activeUser := createTestUser("testuser", "test@example.com", "password123", domain.StatusActive)
	repo.userRepo.AddUser(activeUser)

	backend := NewBackend(repo, logger, nil)
	ctx := context.Background()

	t.Run("successful login", func(t *testing.T) {
		session, err := backend.Login(ctx, "testuser", "password123")
		if err != nil {
			t.Errorf("Login() error = %v", err)
			return
		}
		if session == nil {
			t.Error("Login() returned nil session")
			return
		}
		if session.ID == "" {
			t.Error("Session ID should not be empty")
		}
		if session.User == nil {
			t.Error("Session User should not be nil")
		}
		if session.User.Username != "testuser" {
			t.Errorf("Session User.Username = %v, want testuser", session.User.Username)
		}
	})

	t.Run("failed login - wrong password", func(t *testing.T) {
		session, err := backend.Login(ctx, "testuser", "wrongpassword")
		if err == nil {
			t.Error("Login() expected error for wrong password")
		}
		if session != nil {
			t.Error("Login() should return nil session on failure")
		}
	})

	t.Run("failed login - non-existent user", func(t *testing.T) {
		session, err := backend.Login(ctx, "nobody", "password123")
		if err == nil {
			t.Error("Login() expected error for non-existent user")
		}
		if session != nil {
			t.Error("Login() should return nil session on failure")
		}
	})
}

func TestBackend_SessionManagement(t *testing.T) {
	repo := newMockRepository()
	logger := zerolog.Nop()

	activeUser := createTestUser("testuser", "test@example.com", "password123", domain.StatusActive)
	repo.userRepo.AddUser(activeUser)

	backend := NewBackend(repo, logger, nil)
	ctx := context.Background()

	// Create a session
	session, err := backend.Login(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	t.Run("get session", func(t *testing.T) {
		retrieved := backend.GetSession(session.ID)
		if retrieved == nil {
			t.Error("GetSession() returned nil for existing session")
			return
		}
		if retrieved.ID != session.ID {
			t.Errorf("GetSession() ID = %v, want %v", retrieved.ID, session.ID)
		}
	})

	t.Run("get non-existent session", func(t *testing.T) {
		retrieved := backend.GetSession("non-existent-id")
		if retrieved != nil {
			t.Error("GetSession() should return nil for non-existent session")
		}
	})

	t.Run("session count", func(t *testing.T) {
		count := backend.SessionCount()
		if count < 1 {
			t.Errorf("SessionCount() = %v, want at least 1", count)
		}
	})

	t.Run("logout", func(t *testing.T) {
		backend.Logout(session.ID)
		retrieved := backend.GetSession(session.ID)
		if retrieved != nil {
			t.Error("GetSession() should return nil after Logout")
		}
	})
}

func TestBackend_GetSessionsByUser(t *testing.T) {
	repo := newMockRepository()
	logger := zerolog.Nop()

	user1 := createTestUser("user1", "user1@example.com", "password123", domain.StatusActive)
	user2 := createTestUser("user2", "user2@example.com", "password123", domain.StatusActive)
	repo.userRepo.AddUser(user1)
	repo.userRepo.AddUser(user2)

	backend := NewBackend(repo, logger, nil)
	ctx := context.Background()

	// Create sessions for user1
	_, _ = backend.Login(ctx, "user1", "password123")
	_, _ = backend.Login(ctx, "user1", "password123")

	// Create a session for user2
	_, _ = backend.Login(ctx, "user2", "password123")

	t.Run("get sessions for user1", func(t *testing.T) {
		sessions := backend.GetSessionsByUser(user1.ID)
		if len(sessions) != 2 {
			t.Errorf("GetSessionsByUser() returned %v sessions, want 2", len(sessions))
		}
	})

	t.Run("get sessions for user2", func(t *testing.T) {
		sessions := backend.GetSessionsByUser(user2.ID)
		if len(sessions) != 1 {
			t.Errorf("GetSessionsByUser() returned %v sessions, want 1", len(sessions))
		}
	})

	t.Run("get sessions for non-existent user", func(t *testing.T) {
		sessions := backend.GetSessionsByUser(domain.ID("non-existent"))
		if len(sessions) != 0 {
			t.Errorf("GetSessionsByUser() returned %v sessions, want 0", len(sessions))
		}
	})
}

func TestBackend_TerminateUserSessions(t *testing.T) {
	repo := newMockRepository()
	logger := zerolog.Nop()

	user := createTestUser("testuser", "test@example.com", "password123", domain.StatusActive)
	repo.userRepo.AddUser(user)

	backend := NewBackend(repo, logger, nil)
	ctx := context.Background()

	// Create multiple sessions
	_, _ = backend.Login(ctx, "testuser", "password123")
	_, _ = backend.Login(ctx, "testuser", "password123")
	_, _ = backend.Login(ctx, "testuser", "password123")

	initialCount := backend.SessionCount()
	if initialCount != 3 {
		t.Errorf("Initial SessionCount() = %v, want 3", initialCount)
	}

	// Terminate all sessions for the user
	terminated := backend.TerminateUserSessions(user.ID)
	if terminated != 3 {
		t.Errorf("TerminateUserSessions() = %v, want 3", terminated)
	}

	// Verify all sessions are gone
	if backend.SessionCount() != 0 {
		t.Errorf("SessionCount() after termination = %v, want 0", backend.SessionCount())
	}
}

func TestUserSession_Methods(t *testing.T) {
	session := &UserSession{
		ID:             "test-session-id",
		User:           &domain.User{Username: "testuser"},
		CreatedAt:      time.Now().Add(-10 * time.Minute),
		LastActivityAt: time.Now().Add(-5 * time.Minute),
	}

	t.Run("touch", func(t *testing.T) {
		oldActivity := session.LastActivityAt
		time.Sleep(1 * time.Millisecond)
		session.Touch()
		if !session.LastActivityAt.After(oldActivity) {
			t.Error("Touch() should update LastActivityAt")
		}
	})

	t.Run("select mailbox", func(t *testing.T) {
		mailbox := &domain.Mailbox{
			ID:   domain.ID("mailbox-1"),
			Name: "INBOX",
		}
		session.SelectMailbox(mailbox, false)

		if session.GetSelectedMailbox() == nil {
			t.Error("GetSelectedMailbox() should return the selected mailbox")
		}
		if !session.IsMailboxSelected() {
			t.Error("IsMailboxSelected() should return true")
		}
		if session.IsReadOnly {
			t.Error("IsReadOnly should be false")
		}
	})

	t.Run("select mailbox read-only", func(t *testing.T) {
		mailbox := &domain.Mailbox{
			ID:   domain.ID("mailbox-2"),
			Name: "Archive",
		}
		session.SelectMailbox(mailbox, true)

		if !session.IsReadOnly {
			t.Error("IsReadOnly should be true")
		}
	})

	t.Run("clear selected mailbox", func(t *testing.T) {
		session.ClearSelectedMailbox()
		if session.IsMailboxSelected() {
			t.Error("IsMailboxSelected() should return false after clearing")
		}
		if session.GetSelectedMailbox() != nil {
			t.Error("GetSelectedMailbox() should return nil after clearing")
		}
	})

	t.Run("duration", func(t *testing.T) {
		duration := session.Duration()
		if duration <= 0 {
			t.Error("Duration() should return positive value")
		}
	})

	t.Run("idle time", func(t *testing.T) {
		// Touch to reset, then wait briefly
		session.Touch()
		time.Sleep(5 * time.Millisecond)
		idleTime := session.IdleTime()
		if idleTime <= 0 {
			t.Error("IdleTime() should return positive value")
		}
	})

	t.Run("get user", func(t *testing.T) {
		user := session.GetUser()
		if user == nil {
			t.Error("GetUser() should not return nil")
		}
		if user.Username != "testuser" {
			t.Errorf("GetUser().Username = %v, want testuser", user.Username)
		}
	})
}

func TestDefaultBackendConfig(t *testing.T) {
	cfg := DefaultBackendConfig()
	if cfg == nil {
		t.Error("DefaultBackendConfig() returned nil")
	}
	if cfg.SessionTimeout <= 0 {
		t.Error("SessionTimeout should be positive")
	}
}
