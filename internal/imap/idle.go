// Package imap provides an IMAP server implementation for the Yunt mail server.
package imap

import (
	"context"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"yunt/internal/domain"
	"yunt/internal/service"
)

// IdleConfig holds configuration for IDLE handling.
type IdleConfig struct {
	// Timeout is the maximum duration an IDLE command can stay active.
	// After this duration, the server will end the IDLE state.
	// RFC 2177 recommends servers have an auto-logout timer of at least 30 minutes.
	Timeout time.Duration

	// PollInterval is the interval at which to check for updates
	// when notifications are not available.
	PollInterval time.Duration

	// MaxIdleSessions is the maximum number of concurrent IDLE sessions.
	// 0 means unlimited.
	MaxIdleSessions int
}

// DefaultIdleConfig returns an IdleConfig with sensible defaults.
func DefaultIdleConfig() *IdleConfig {
	return &IdleConfig{
		Timeout:         29 * time.Minute, // Just under 30 minutes per RFC recommendation
		PollInterval:    30 * time.Second, // Poll interval for fallback mode
		MaxIdleSessions: 0,                // Unlimited by default
	}
}

// IdleHandler handles IMAP IDLE command processing.
// IDLE (RFC 2177) allows a client to receive real-time notifications
// about mailbox changes without polling.
type IdleHandler struct {
	config             *IdleConfig
	notificationBridge *NotificationBridge
	logger             zerolog.Logger

	// activeIdleSessions tracks the number of currently active IDLE sessions.
	activeIdleSessions int64
	mu                 sync.RWMutex
}

// NewIdleHandler creates a new IdleHandler.
func NewIdleHandler(config *IdleConfig, bridge *NotificationBridge, logger zerolog.Logger) *IdleHandler {
	if config == nil {
		config = DefaultIdleConfig()
	}

	return &IdleHandler{
		config:             config,
		notificationBridge: bridge,
		logger:             logger.With().Str("component", "idle-handler").Logger(),
	}
}

// HandleIdle processes the IDLE command for a session.
// It waits for mailbox updates or until the client sends DONE.
func (h *IdleHandler) HandleIdle(
	ctx context.Context,
	w *imapserver.UpdateWriter,
	stop <-chan struct{},
	sessionID string,
	mailboxID, userID domain.ID,
) error {
	// Generate a unique subscription ID for this IDLE session
	subscriptionID := uuid.New().String()

	h.logger.Debug().
		Str("sessionID", sessionID).
		Str("subscriptionID", subscriptionID).
		Str("mailboxID", mailboxID.String()).
		Msg("Starting IDLE command")

	// Check if we've exceeded max IDLE sessions
	if !h.incrementIdleSessions() {
		h.logger.Warn().
			Str("sessionID", sessionID).
			Int("maxSessions", h.config.MaxIdleSessions).
			Msg("Maximum IDLE sessions reached")
		// We don't return an error here as we still want to honor IDLE,
		// just without push notifications
	} else {
		defer h.decrementIdleSessions()
	}

	// Register for notifications
	idleInfo := h.notificationBridge.RegisterIdleSession(subscriptionID, mailboxID, userID, w)
	defer h.notificationBridge.UnregisterIdleSession(subscriptionID)

	// Create timeout timer
	timeout := time.NewTimer(h.config.Timeout)
	defer timeout.Stop()

	// Main IDLE loop
	for {
		select {
		case <-stop:
			// Client sent DONE
			h.logger.Debug().
				Str("sessionID", sessionID).
				Msg("IDLE ended by client DONE")
			return nil

		case <-timeout.C:
			// IDLE timeout reached - auto-terminate IDLE
			h.logger.Debug().
				Str("sessionID", sessionID).
				Dur("timeout", h.config.Timeout).
				Msg("IDLE timeout reached")
			return nil

		case notification, ok := <-idleInfo.NotifyChan:
			if !ok {
				// Channel closed, session is ending
				return nil
			}

			// Process notification
			if err := h.processNotification(w, notification, sessionID); err != nil {
				h.logger.Error().
					Err(err).
					Str("sessionID", sessionID).
					Str("notificationType", notification.Type.String()).
					Msg("Failed to process notification")
				// Continue processing other notifications
			}

		case <-ctx.Done():
			// Context cancelled
			h.logger.Debug().
				Str("sessionID", sessionID).
				Msg("IDLE cancelled by context")
			return ctx.Err()
		}
	}
}

// processNotification processes a single notification and writes the update.
func (h *IdleHandler) processNotification(
	w *imapserver.UpdateWriter,
	notification *service.Notification,
	sessionID string,
) error {
	h.logger.Debug().
		Str("sessionID", sessionID).
		Str("notificationType", notification.Type.String()).
		Str("mailboxID", notification.MailboxID.String()).
		Msg("Processing IDLE notification")

	return WriteUpdate(w, notification, h.logger)
}

// incrementIdleSessions increments the active session count.
// Returns false if max sessions has been reached.
func (h *IdleHandler) incrementIdleSessions() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.config.MaxIdleSessions > 0 && int(h.activeIdleSessions) >= h.config.MaxIdleSessions {
		return false
	}

	h.activeIdleSessions++
	return true
}

// decrementIdleSessions decrements the active session count.
func (h *IdleHandler) decrementIdleSessions() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.activeIdleSessions > 0 {
		h.activeIdleSessions--
	}
}

// GetActiveIdleSessions returns the current number of active IDLE sessions.
func (h *IdleHandler) GetActiveIdleSessions() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.activeIdleSessions
}

// GetConfig returns the current IDLE configuration.
func (h *IdleHandler) GetConfig() *IdleConfig {
	return h.config
}

// IdleStats holds statistics about IDLE sessions.
type IdleStats struct {
	// ActiveSessions is the current number of active IDLE sessions.
	ActiveSessions int64

	// TotalSessions is the total number of IDLE sessions ever created.
	TotalSessions int64

	// MaxConcurrentSessions is the maximum number of concurrent sessions observed.
	MaxConcurrentSessions int64
}

// IdleManager manages IDLE sessions across multiple IMAP connections.
// It coordinates with the notification service to deliver real-time updates.
type IdleManager struct {
	config             *IdleConfig
	notificationBridge *NotificationBridge
	logger             zerolog.Logger

	// handlers tracks IdleHandlers by session ID.
	handlers map[string]*IdleHandler
	mu       sync.RWMutex

	// stats tracks IDLE statistics.
	stats IdleStats
}

// NewIdleManager creates a new IdleManager.
func NewIdleManager(config *IdleConfig, bridge *NotificationBridge, logger zerolog.Logger) *IdleManager {
	if config == nil {
		config = DefaultIdleConfig()
	}

	return &IdleManager{
		config:             config,
		notificationBridge: bridge,
		logger:             logger.With().Str("component", "idle-manager").Logger(),
		handlers:           make(map[string]*IdleHandler),
	}
}

// CreateHandler creates a new IdleHandler for a session.
func (m *IdleManager) CreateHandler(sessionID string) *IdleHandler {
	m.mu.Lock()
	defer m.mu.Unlock()

	handler := NewIdleHandler(m.config, m.notificationBridge, m.logger)
	m.handlers[sessionID] = handler

	m.stats.TotalSessions++
	if int64(len(m.handlers)) > m.stats.MaxConcurrentSessions {
		m.stats.MaxConcurrentSessions = int64(len(m.handlers))
	}

	return handler
}

// RemoveHandler removes an IdleHandler for a session.
func (m *IdleManager) RemoveHandler(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.handlers, sessionID)
}

// GetHandler returns the IdleHandler for a session.
func (m *IdleManager) GetHandler(sessionID string) *IdleHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.handlers[sessionID]
}

// GetStats returns the current IDLE statistics.
func (m *IdleManager) GetStats() IdleStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := m.stats
	stats.ActiveSessions = int64(len(m.handlers))
	return stats
}

// GetNotificationBridge returns the notification bridge.
func (m *IdleManager) GetNotificationBridge() *NotificationBridge {
	return m.notificationBridge
}

// Close closes all IDLE sessions and cleans up resources.
func (m *IdleManager) Close(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers = make(map[string]*IdleHandler)
	m.logger.Info().Msg("IdleManager closed")

	return nil
}
