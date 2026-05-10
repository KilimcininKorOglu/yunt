// Package service provides business logic and service layer implementations.
package service

import (
	"context"
	"sync"
	"time"

	"yunt/internal/domain"
)

// NotificationType represents the type of mailbox notification.
type NotificationType int

const (
	// NotificationNewMessage indicates a new message has arrived.
	NotificationNewMessage NotificationType = iota
	// NotificationFlagsChanged indicates message flags have changed.
	NotificationFlagsChanged
	// NotificationMessageExpunged indicates a message has been expunged.
	NotificationMessageExpunged
	// NotificationMailboxUpdated indicates mailbox statistics changed.
	NotificationMailboxUpdated
)

// String returns the string representation of the notification type.
func (t NotificationType) String() string {
	switch t {
	case NotificationNewMessage:
		return "new_message"
	case NotificationFlagsChanged:
		return "flags_changed"
	case NotificationMessageExpunged:
		return "message_expunged"
	case NotificationMailboxUpdated:
		return "mailbox_updated"
	default:
		return "unknown"
	}
}

// Notification represents a mailbox change notification.
type Notification struct {
	// Type is the type of notification.
	Type NotificationType

	// MailboxID is the ID of the affected mailbox.
	MailboxID domain.ID

	// UserID is the ID of the user who owns the mailbox.
	UserID domain.ID

	// MessageID is the ID of the affected message (if applicable).
	MessageID domain.ID

	// SeqNum is the sequence number of the message (for EXISTS/EXPUNGE).
	SeqNum uint32

	// UID is the UID of the message.
	UID uint32

	// Flags are the current flags (for flag change notifications).
	Flags []string

	// MessageCount is the new message count (for EXISTS notifications).
	MessageCount uint32

	// Timestamp is when the notification was created.
	Timestamp time.Time
}

// NewNotification creates a new Notification with the given type.
func NewNotification(notifType NotificationType, mailboxID, userID domain.ID) *Notification {
	return &Notification{
		Type:      notifType,
		MailboxID: mailboxID,
		UserID:    userID,
		Timestamp: time.Now(),
	}
}

// NotificationHandler is a function that handles notifications.
type NotificationHandler func(*Notification)

// Subscription represents a subscription to mailbox notifications.
type Subscription struct {
	// ID is the unique subscription identifier.
	ID string

	// MailboxID is the ID of the subscribed mailbox.
	MailboxID domain.ID

	// UserID is the ID of the user who owns the subscription.
	UserID domain.ID

	// Handler is called when a notification is received.
	Handler NotificationHandler

	// Created is when the subscription was created.
	Created time.Time

	// closed indicates if the subscription has been closed.
	closed bool
	mu     sync.RWMutex
}

// IsClosed returns true if the subscription is closed.
func (s *Subscription) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// close marks the subscription as closed.
func (s *Subscription) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// NotifyService manages real-time notifications for mailbox changes.
// It allows IMAP sessions to subscribe to mailbox updates and receive
// notifications when messages arrive, flags change, or messages are expunged.
type NotifyService struct {
	// subscriptions holds all active subscriptions indexed by mailbox ID.
	subscriptions map[domain.ID]map[string]*Subscription
	mu            sync.RWMutex

	// stats tracks notification statistics.
	stats NotifyStats
}

// NotifyStats holds statistics about notifications.
type NotifyStats struct {
	// TotalSubscriptions is the total number of subscriptions ever created.
	TotalSubscriptions int64

	// ActiveSubscriptions is the current number of active subscriptions.
	ActiveSubscriptions int64

	// TotalNotifications is the total number of notifications sent.
	TotalNotifications int64

	// LastNotificationAt is the timestamp of the last notification.
	LastNotificationAt time.Time

	mu sync.RWMutex
}

// GetStats returns a copy of the current statistics.
func (s *NotifyStats) GetStats() NotifyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return NotifyStats{
		TotalSubscriptions:  s.TotalSubscriptions,
		ActiveSubscriptions: s.ActiveSubscriptions,
		TotalNotifications:  s.TotalNotifications,
		LastNotificationAt:  s.LastNotificationAt,
	}
}

// NewNotifyService creates a new NotifyService.
func NewNotifyService() *NotifyService {
	return &NotifyService{
		subscriptions: make(map[domain.ID]map[string]*Subscription),
	}
}

// Subscribe creates a subscription to notifications for a specific mailbox.
// Returns a subscription that can be used to unsubscribe later.
func (ns *NotifyService) Subscribe(subscriptionID string, mailboxID, userID domain.ID, handler NotificationHandler) *Subscription {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	sub := &Subscription{
		ID:        subscriptionID,
		MailboxID: mailboxID,
		UserID:    userID,
		Handler:   handler,
		Created:   time.Now(),
	}

	// Create mailbox subscription map if it doesn't exist
	if ns.subscriptions[mailboxID] == nil {
		ns.subscriptions[mailboxID] = make(map[string]*Subscription)
	}

	ns.subscriptions[mailboxID][subscriptionID] = sub

	// Update statistics
	ns.stats.mu.Lock()
	ns.stats.TotalSubscriptions++
	ns.stats.ActiveSubscriptions++
	ns.stats.mu.Unlock()

	return sub
}

// Unsubscribe removes a subscription.
func (ns *NotifyService) Unsubscribe(subscriptionID string, mailboxID domain.ID) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if mailboxSubs, ok := ns.subscriptions[mailboxID]; ok {
		if sub, exists := mailboxSubs[subscriptionID]; exists {
			sub.close()
			delete(mailboxSubs, subscriptionID)

			// Remove mailbox entry if no more subscriptions
			if len(mailboxSubs) == 0 {
				delete(ns.subscriptions, mailboxID)
			}

			// Update statistics
			ns.stats.mu.Lock()
			ns.stats.ActiveSubscriptions--
			ns.stats.mu.Unlock()
		}
	}
}

// UnsubscribeByID removes a subscription by ID across all mailboxes.
// This is useful when a session closes and we need to clean up all its subscriptions.
func (ns *NotifyService) UnsubscribeByID(subscriptionID string) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for mailboxID, mailboxSubs := range ns.subscriptions {
		if sub, exists := mailboxSubs[subscriptionID]; exists {
			sub.close()
			delete(mailboxSubs, subscriptionID)

			// Remove mailbox entry if no more subscriptions
			if len(mailboxSubs) == 0 {
				delete(ns.subscriptions, mailboxID)
			}

			// Update statistics
			ns.stats.mu.Lock()
			ns.stats.ActiveSubscriptions--
			ns.stats.mu.Unlock()
		}
	}
}

// Notify sends a notification to all subscribers of a mailbox.
// It excludes the sender if excludeSubscriptionID is provided.
func (ns *NotifyService) Notify(notification *Notification, excludeSubscriptionID string) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	mailboxSubs, ok := ns.subscriptions[notification.MailboxID]
	if !ok {
		return
	}

	// Update statistics
	ns.stats.mu.Lock()
	ns.stats.TotalNotifications++
	ns.stats.LastNotificationAt = time.Now()
	ns.stats.mu.Unlock()

	// Notify all subscribers except the excluded one
	for subID, sub := range mailboxSubs {
		if subID == excludeSubscriptionID {
			continue
		}

		if !sub.IsClosed() && sub.Handler != nil {
			// Call handler in a goroutine to avoid blocking
			go sub.Handler(notification)
		}
	}
}

// NotifyNewMessage sends a new message notification.
func (ns *NotifyService) NotifyNewMessage(mailboxID, userID, messageID domain.ID, seqNum, uid, messageCount uint32) {
	notification := NewNotification(NotificationNewMessage, mailboxID, userID)
	notification.MessageID = messageID
	notification.SeqNum = seqNum
	notification.UID = uid
	notification.MessageCount = messageCount

	ns.Notify(notification, "")
}

// NotifyFlagsChanged sends a flags changed notification.
func (ns *NotifyService) NotifyFlagsChanged(mailboxID, userID, messageID domain.ID, seqNum, uid uint32, flags []string, excludeSubscriptionID string) {
	notification := NewNotification(NotificationFlagsChanged, mailboxID, userID)
	notification.MessageID = messageID
	notification.SeqNum = seqNum
	notification.UID = uid
	notification.Flags = flags

	ns.Notify(notification, excludeSubscriptionID)
}

// NotifyMessageExpunged sends a message expunged notification.
func (ns *NotifyService) NotifyMessageExpunged(mailboxID, userID, messageID domain.ID, seqNum uint32) {
	notification := NewNotification(NotificationMessageExpunged, mailboxID, userID)
	notification.MessageID = messageID
	notification.SeqNum = seqNum

	ns.Notify(notification, "")
}

// NotifyMailboxUpdated sends a mailbox updated notification.
func (ns *NotifyService) NotifyMailboxUpdated(mailboxID, userID domain.ID, messageCount uint32) {
	notification := NewNotification(NotificationMailboxUpdated, mailboxID, userID)
	notification.MessageCount = messageCount

	ns.Notify(notification, "")
}

// GetSubscriptionCount returns the number of active subscriptions for a mailbox.
func (ns *NotifyService) GetSubscriptionCount(mailboxID domain.ID) int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if mailboxSubs, ok := ns.subscriptions[mailboxID]; ok {
		return len(mailboxSubs)
	}
	return 0
}

// GetTotalSubscriptions returns the total number of active subscriptions.
func (ns *NotifyService) GetTotalSubscriptions() int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	count := 0
	for _, mailboxSubs := range ns.subscriptions {
		count += len(mailboxSubs)
	}
	return count
}

// GetStats returns the notification service statistics.
func (ns *NotifyService) GetStats() NotifyStats {
	return ns.stats.GetStats()
}

// HasSubscribers returns true if the mailbox has any active subscribers.
func (ns *NotifyService) HasSubscribers(mailboxID domain.ID) bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if mailboxSubs, ok := ns.subscriptions[mailboxID]; ok {
		return len(mailboxSubs) > 0
	}
	return false
}

// GetSubscribersForUser returns all subscription IDs for a specific user.
func (ns *NotifyService) GetSubscribersForUser(userID domain.ID) []string {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	var subscriptionIDs []string
	for _, mailboxSubs := range ns.subscriptions {
		for subID, sub := range mailboxSubs {
			if sub.UserID == userID {
				subscriptionIDs = append(subscriptionIDs, subID)
			}
		}
	}
	return subscriptionIDs
}

// Close closes all subscriptions and cleans up resources.
func (ns *NotifyService) Close(_ context.Context) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	// Close all subscriptions
	for _, mailboxSubs := range ns.subscriptions {
		for _, sub := range mailboxSubs {
			sub.close()
		}
	}

	// Clear all subscriptions
	ns.subscriptions = make(map[domain.ID]map[string]*Subscription)

	// Reset active count
	ns.stats.mu.Lock()
	ns.stats.ActiveSubscriptions = 0
	ns.stats.mu.Unlock()

	return nil
}
