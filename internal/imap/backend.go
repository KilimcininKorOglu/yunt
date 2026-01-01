package imap

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// Backend provides the IMAP backend functionality, managing user sessions
// and providing access to user-specific data through the repository.
type Backend struct {
	repo          repository.Repository
	authenticator *Authenticator
	logger        zerolog.Logger

	// Session management
	sessions   map[string]*UserSession
	sessionsMu sync.RWMutex

	// Configuration
	sessionTimeout time.Duration
}

// BackendConfig holds configuration options for the Backend.
type BackendConfig struct {
	// SessionTimeout is the duration after which inactive sessions expire.
	// Default is 30 minutes.
	SessionTimeout time.Duration
}

// DefaultBackendConfig returns a BackendConfig with default values.
func DefaultBackendConfig() *BackendConfig {
	return &BackendConfig{
		SessionTimeout: 30 * time.Minute,
	}
}

// UserSession represents an authenticated user's IMAP session.
type UserSession struct {
	// ID is the unique session identifier.
	ID string

	// User is the authenticated user.
	User *domain.User

	// CreatedAt is when the session was created.
	CreatedAt time.Time

	// LastActivityAt is when the session was last active.
	LastActivityAt time.Time

	// SelectedMailbox is the currently selected mailbox, if any.
	SelectedMailbox *domain.Mailbox

	// IsReadOnly indicates if the selected mailbox is read-only.
	IsReadOnly bool

	// mu protects session state modifications.
	mu sync.RWMutex
}

// NewBackend creates a new IMAP Backend instance.
func NewBackend(repo repository.Repository, logger zerolog.Logger, cfg *BackendConfig) *Backend {
	if cfg == nil {
		cfg = DefaultBackendConfig()
	}

	b := &Backend{
		repo:           repo,
		authenticator:  NewAuthenticator(repo),
		logger:         logger.With().Str("component", "imap-backend").Logger(),
		sessions:       make(map[string]*UserSession),
		sessionTimeout: cfg.SessionTimeout,
	}

	// Start session cleanup goroutine
	go b.sessionCleanupLoop()

	return b
}

// Login authenticates a user with username and password.
// Returns a UserSession on success, or an error on failure.
func (b *Backend) Login(ctx context.Context, username, password string) (*UserSession, error) {
	b.logger.Debug().
		Str("username", username).
		Msg("Login attempt")

	// Authenticate the user
	result, err := b.authenticator.Authenticate(ctx, username, password)
	if err != nil {
		b.logger.Warn().
			Str("username", username).
			Err(err).
			Msg("Login failed")
		return nil, err
	}

	// Create a new session
	session := b.createSession(result.User)

	b.logger.Info().
		Str("username", username).
		Str("userID", result.User.ID.String()).
		Str("sessionID", session.ID).
		Msg("Login successful")

	return session, nil
}

// Logout terminates a user session.
func (b *Backend) Logout(sessionID string) {
	b.sessionsMu.Lock()
	defer b.sessionsMu.Unlock()

	if session, ok := b.sessions[sessionID]; ok {
		b.logger.Info().
			Str("sessionID", sessionID).
			Str("username", session.User.Username).
			Dur("sessionDuration", time.Since(session.CreatedAt)).
			Msg("Session logged out")
		delete(b.sessions, sessionID)
	}
}

// GetSession retrieves an active session by ID.
// Returns nil if the session doesn't exist or has expired.
func (b *Backend) GetSession(sessionID string) *UserSession {
	b.sessionsMu.RLock()
	session, ok := b.sessions[sessionID]
	b.sessionsMu.RUnlock()

	if !ok {
		return nil
	}

	// Check if session has expired
	if time.Since(session.LastActivityAt) > b.sessionTimeout {
		b.Logout(sessionID)
		return nil
	}

	// Update last activity
	session.Touch()
	return session
}

// Repository returns the underlying repository for data access.
func (b *Backend) Repository() repository.Repository {
	return b.repo
}

// Authenticator returns the authenticator for direct authentication needs.
func (b *Backend) Authenticator() *Authenticator {
	return b.authenticator
}

// SessionCount returns the current number of active sessions.
func (b *Backend) SessionCount() int {
	b.sessionsMu.RLock()
	defer b.sessionsMu.RUnlock()
	return len(b.sessions)
}

// GetSessionsByUser returns all active sessions for a specific user.
func (b *Backend) GetSessionsByUser(userID domain.ID) []*UserSession {
	b.sessionsMu.RLock()
	defer b.sessionsMu.RUnlock()

	var userSessions []*UserSession
	for _, session := range b.sessions {
		if session.User.ID == userID {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions
}

// TerminateUserSessions terminates all sessions for a specific user.
func (b *Backend) TerminateUserSessions(userID domain.ID) int {
	b.sessionsMu.Lock()
	defer b.sessionsMu.Unlock()

	count := 0
	for id, session := range b.sessions {
		if session.User.ID == userID {
			delete(b.sessions, id)
			count++
		}
	}

	if count > 0 {
		b.logger.Info().
			Str("userID", userID.String()).
			Int("terminatedSessions", count).
			Msg("Terminated all user sessions")
	}

	return count
}

// createSession creates a new user session and registers it.
func (b *Backend) createSession(user *domain.User) *UserSession {
	now := time.Now()
	session := &UserSession{
		ID:             uuid.New().String(),
		User:           user,
		CreatedAt:      now,
		LastActivityAt: now,
	}

	b.sessionsMu.Lock()
	b.sessions[session.ID] = session
	b.sessionsMu.Unlock()

	return session
}

// sessionCleanupLoop periodically removes expired sessions.
func (b *Backend) sessionCleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		b.cleanupExpiredSessions()
	}
}

// cleanupExpiredSessions removes all expired sessions.
func (b *Backend) cleanupExpiredSessions() {
	b.sessionsMu.Lock()
	defer b.sessionsMu.Unlock()

	expiredCount := 0
	for id, session := range b.sessions {
		if time.Since(session.LastActivityAt) > b.sessionTimeout {
			delete(b.sessions, id)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		b.logger.Debug().
			Int("expiredSessions", expiredCount).
			Int("activeSessions", len(b.sessions)).
			Msg("Cleaned up expired sessions")
	}
}

// Touch updates the last activity timestamp for the session.
func (s *UserSession) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivityAt = time.Now()
}

// SelectMailbox sets the currently selected mailbox for the session.
func (s *UserSession) SelectMailbox(mailbox *domain.Mailbox, readOnly bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SelectedMailbox = mailbox
	s.IsReadOnly = readOnly
	s.LastActivityAt = time.Now()
}

// GetSelectedMailbox returns the currently selected mailbox.
func (s *UserSession) GetSelectedMailbox() *domain.Mailbox {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SelectedMailbox
}

// ClearSelectedMailbox clears the currently selected mailbox.
func (s *UserSession) ClearSelectedMailbox() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SelectedMailbox = nil
	s.IsReadOnly = false
	s.LastActivityAt = time.Now()
}

// IsMailboxSelected returns true if a mailbox is currently selected.
func (s *UserSession) IsMailboxSelected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SelectedMailbox != nil
}

// GetUser returns a copy of the session's user.
func (s *UserSession) GetUser() *domain.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.User
}

// Duration returns how long the session has been active.
func (s *UserSession) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.CreatedAt)
}

// IdleTime returns how long since the last activity.
func (s *UserSession) IdleTime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.LastActivityAt)
}
