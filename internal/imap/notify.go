package imap

import (
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/rs/zerolog"

	"yunt/internal/domain"
	"yunt/internal/service"
)

// NotificationBridge bridges the service notification system to IMAP update writers.
// It translates service-level notifications into IMAP protocol responses.
type NotificationBridge struct {
	notifyService *service.NotifyService
	logger        zerolog.Logger

	// idleSessions tracks active IDLE sessions for efficient notification routing.
	idleSessions map[string]*IdleSessionInfo
	mu           sync.RWMutex
}

// IdleSessionInfo holds information about an active IDLE session.
type IdleSessionInfo struct {
	// SessionID is the unique session identifier.
	SessionID string

	// SubscriptionID is the notification subscription ID.
	SubscriptionID string

	// MailboxID is the ID of the selected mailbox.
	MailboxID domain.ID

	// UserID is the ID of the authenticated user.
	UserID domain.ID

	// UpdateWriter is used to send unsolicited updates.
	UpdateWriter *imapserver.UpdateWriter

	// NotifyChan is a channel to send notifications to the IDLE handler.
	NotifyChan chan *service.Notification

	// Closed indicates if the session has been closed.
	Closed bool
	mu     sync.RWMutex
}

// IsClosed returns true if the session is closed.
func (s *IdleSessionInfo) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Closed
}

// Close marks the session as closed.
func (s *IdleSessionInfo) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Closed = true
}

// NewNotificationBridge creates a new NotificationBridge.
func NewNotificationBridge(notifyService *service.NotifyService, logger zerolog.Logger) *NotificationBridge {
	return &NotificationBridge{
		notifyService: notifyService,
		logger:        logger.With().Str("component", "notification-bridge").Logger(),
		idleSessions:  make(map[string]*IdleSessionInfo),
	}
}

// RegisterIdleSession registers an IDLE session to receive notifications.
// Returns an IdleSessionInfo that can be used to track the session.
func (nb *NotificationBridge) RegisterIdleSession(
	sessionID string,
	mailboxID, userID domain.ID,
	updateWriter *imapserver.UpdateWriter,
) *IdleSessionInfo {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	// Create notification channel with buffer to avoid blocking
	notifyChan := make(chan *service.Notification, 100)

	info := &IdleSessionInfo{
		SessionID:      sessionID,
		SubscriptionID: sessionID, // Use session ID as subscription ID
		MailboxID:      mailboxID,
		UserID:         userID,
		UpdateWriter:   updateWriter,
		NotifyChan:     notifyChan,
	}

	// Store in our tracking map
	nb.idleSessions[sessionID] = info

	// Subscribe to notifications
	nb.notifyService.Subscribe(sessionID, mailboxID, userID, func(notification *service.Notification) {
		// Send notification to the channel (non-blocking)
		select {
		case notifyChan <- notification:
		default:
			nb.logger.Warn().
				Str("sessionID", sessionID).
				Str("notificationType", notification.Type.String()).
				Msg("Notification channel full, dropping notification")
		}
	})

	nb.logger.Debug().
		Str("sessionID", sessionID).
		Str("mailboxID", mailboxID.String()).
		Msg("Registered IDLE session for notifications")

	return info
}

// UnregisterIdleSession unregisters an IDLE session.
func (nb *NotificationBridge) UnregisterIdleSession(sessionID string) {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	if info, ok := nb.idleSessions[sessionID]; ok {
		info.Close()

		// Close the notification channel
		close(info.NotifyChan)

		// Unsubscribe from notifications
		nb.notifyService.Unsubscribe(sessionID, info.MailboxID)

		delete(nb.idleSessions, sessionID)

		nb.logger.Debug().
			Str("sessionID", sessionID).
			Msg("Unregistered IDLE session")
	}
}

// GetIdleSessionCount returns the number of active IDLE sessions.
func (nb *NotificationBridge) GetIdleSessionCount() int {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return len(nb.idleSessions)
}

// GetIdleSessionsForMailbox returns the number of IDLE sessions for a specific mailbox.
func (nb *NotificationBridge) GetIdleSessionsForMailbox(mailboxID domain.ID) int {
	nb.mu.RLock()
	defer nb.mu.RUnlock()

	count := 0
	for _, info := range nb.idleSessions {
		if info.MailboxID == mailboxID {
			count++
		}
	}
	return count
}

// WriteUpdate writes an update to the IMAP update writer based on the notification type.
func WriteUpdate(w *imapserver.UpdateWriter, notification *service.Notification, logger zerolog.Logger) error {
	switch notification.Type {
	case service.NotificationNewMessage:
		return writeExistsUpdate(w, notification, logger)
	case service.NotificationFlagsChanged:
		return writeFetchUpdate(w, notification, logger)
	case service.NotificationMessageExpunged:
		return writeExpungeUpdate(w, notification, logger)
	case service.NotificationMailboxUpdated:
		return writeStatusUpdate(w, notification, logger)
	default:
		logger.Warn().
			Str("notificationType", notification.Type.String()).
			Msg("Unknown notification type")
		return nil
	}
}

// writeExistsUpdate writes an EXISTS response for new messages.
func writeExistsUpdate(w *imapserver.UpdateWriter, notification *service.Notification, logger zerolog.Logger) error {
	// Write EXISTS to indicate new message count
	err := w.WriteNumMessages(notification.MessageCount)
	if err != nil {
		logger.Error().
			Err(err).
			Uint32("messageCount", notification.MessageCount).
			Msg("Failed to write EXISTS update")
		return err
	}

	logger.Debug().
		Uint32("messageCount", notification.MessageCount).
		Msg("Sent EXISTS notification")

	return nil
}

// writeFetchUpdate writes a FETCH response for flag changes.
func writeFetchUpdate(w *imapserver.UpdateWriter, notification *service.Notification, logger zerolog.Logger) error {
	// Convert string flags to imap.Flag
	flags := make([]imap.Flag, len(notification.Flags))
	for i, f := range notification.Flags {
		flags[i] = imap.Flag(f)
	}

	// Use WriteMessageFlags to send a FLAGS update for the message
	err := w.WriteMessageFlags(notification.SeqNum, imap.UID(notification.UID), flags)
	if err != nil {
		logger.Error().
			Err(err).
			Uint32("seqNum", notification.SeqNum).
			Msg("Failed to write FETCH update")
		return err
	}

	logger.Debug().
		Uint32("seqNum", notification.SeqNum).
		Strs("flags", notification.Flags).
		Msg("Sent FETCH notification for flag change")

	return nil
}

// writeExpungeUpdate writes an EXPUNGE response for deleted messages.
func writeExpungeUpdate(w *imapserver.UpdateWriter, notification *service.Notification, logger zerolog.Logger) error {
	err := w.WriteExpunge(notification.SeqNum)
	if err != nil {
		logger.Error().
			Err(err).
			Uint32("seqNum", notification.SeqNum).
			Msg("Failed to write EXPUNGE update")
		return err
	}

	logger.Debug().
		Uint32("seqNum", notification.SeqNum).
		Msg("Sent EXPUNGE notification")

	return nil
}

// writeStatusUpdate writes a status update for mailbox changes.
func writeStatusUpdate(w *imapserver.UpdateWriter, notification *service.Notification, logger zerolog.Logger) error {
	// Write EXISTS to update the message count
	err := w.WriteNumMessages(notification.MessageCount)
	if err != nil {
		logger.Error().
			Err(err).
			Uint32("messageCount", notification.MessageCount).
			Msg("Failed to write status update")
		return err
	}

	logger.Debug().
		Uint32("messageCount", notification.MessageCount).
		Msg("Sent mailbox status update")

	return nil
}

// NotifyNewMessage notifies all IDLE sessions about a new message.
// This is a convenience method for use from SMTP delivery or other services.
func (nb *NotificationBridge) NotifyNewMessage(mailboxID, userID, messageID domain.ID, seqNum, uid, messageCount uint32) {
	nb.notifyService.NotifyNewMessage(mailboxID, userID, messageID, seqNum, uid, messageCount)
}

// NotifyFlagsChanged notifies all IDLE sessions about flag changes.
func (nb *NotificationBridge) NotifyFlagsChanged(mailboxID, userID, messageID domain.ID, seqNum, uid uint32, flags []string, excludeSessionID string) {
	nb.notifyService.NotifyFlagsChanged(mailboxID, userID, messageID, seqNum, uid, flags, excludeSessionID)
}

// NotifyMessageExpunged notifies all IDLE sessions about an expunged message.
func (nb *NotificationBridge) NotifyMessageExpunged(mailboxID, userID, messageID domain.ID, seqNum uint32) {
	nb.notifyService.NotifyMessageExpunged(mailboxID, userID, messageID, seqNum)
}

// GetNotifyService returns the underlying notify service.
func (nb *NotificationBridge) GetNotifyService() *service.NotifyService {
	return nb.notifyService
}
