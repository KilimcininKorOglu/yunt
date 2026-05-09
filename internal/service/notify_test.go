package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"yunt/internal/domain"
)

// ---- helpers ----------------------------------------------------------------

const (
	testMailboxID  domain.ID = "mailbox-1"
	testMailboxID2 domain.ID = "mailbox-2"
	testUserID     domain.ID = "user-1"
	testUserID2    domain.ID = "user-2"
	testMessageID  domain.ID = "msg-1"
)

// waitForNotification waits up to maxWait for the counter to reach want.
// Handlers are dispatched in goroutines, so a short wait is required before
// asserting receipt.
func waitForCount(counter *int64, want int64, maxWait time.Duration) bool {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(counter) >= want {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// ---- NotificationType.String ------------------------------------------------

func TestNotificationType_String(t *testing.T) {
	tests := []struct {
		notifType NotificationType
		want      string
	}{
		{NotificationNewMessage, "new_message"},
		{NotificationFlagsChanged, "flags_changed"},
		{NotificationMessageExpunged, "message_expunged"},
		{NotificationMailboxUpdated, "mailbox_updated"},
		{NotificationType(99), "unknown"},
	}

	for _, tc := range tests {
		got := tc.notifType.String()
		if got != tc.want {
			t.Errorf("NotificationType(%d).String() = %q, want %q", tc.notifType, got, tc.want)
		}
	}
}

// ---- NewNotification --------------------------------------------------------

func TestNewNotification(t *testing.T) {
	before := time.Now()
	n := NewNotification(NotificationNewMessage, testMailboxID, testUserID)
	after := time.Now()

	if n == nil {
		t.Fatal("NewNotification() returned nil")
	}
	if n.Type != NotificationNewMessage {
		t.Errorf("Type = %v, want %v", n.Type, NotificationNewMessage)
	}
	if n.MailboxID != testMailboxID {
		t.Errorf("MailboxID = %v, want %v", n.MailboxID, testMailboxID)
	}
	if n.UserID != testUserID {
		t.Errorf("UserID = %v, want %v", n.UserID, testUserID)
	}
	if n.Timestamp.Before(before) || n.Timestamp.After(after) {
		t.Errorf("Timestamp %v is outside [%v, %v]", n.Timestamp, before, after)
	}
}

func TestNewNotification_ZeroFields(t *testing.T) {
	n := NewNotification(NotificationFlagsChanged, testMailboxID, testUserID)

	// Fields not set by NewNotification must be zero-valued.
	if n.MessageID != "" {
		t.Errorf("MessageID should be empty, got %v", n.MessageID)
	}
	if n.SeqNum != 0 {
		t.Errorf("SeqNum should be 0, got %d", n.SeqNum)
	}
	if n.UID != 0 {
		t.Errorf("UID should be 0, got %d", n.UID)
	}
	if n.Flags != nil {
		t.Errorf("Flags should be nil, got %v", n.Flags)
	}
	if n.MessageCount != 0 {
		t.Errorf("MessageCount should be 0, got %d", n.MessageCount)
	}
}

// ---- Subscription.IsClosed --------------------------------------------------

func TestSubscription_IsClosed(t *testing.T) {
	sub := &Subscription{ID: "sub-1"}

	if sub.IsClosed() {
		t.Error("IsClosed() should be false for a new subscription")
	}

	sub.close()

	if !sub.IsClosed() {
		t.Error("IsClosed() should be true after close()")
	}
}

// ---- NewNotifyService -------------------------------------------------------

func TestNewNotifyService(t *testing.T) {
	ns := NewNotifyService()
	if ns == nil {
		t.Fatal("NewNotifyService() returned nil")
	}
	if ns.GetTotalSubscriptions() != 0 {
		t.Errorf("expected 0 total subscriptions, got %d", ns.GetTotalSubscriptions())
	}
}

// ---- Subscribe --------------------------------------------------------------

func TestNotifyService_Subscribe(t *testing.T) {
	ns := NewNotifyService()

	sub := ns.Subscribe("sub-1", testMailboxID, testUserID, nil)

	if sub == nil {
		t.Fatal("Subscribe() returned nil")
	}
	if sub.ID != "sub-1" {
		t.Errorf("sub.ID = %q, want %q", sub.ID, "sub-1")
	}
	if sub.MailboxID != testMailboxID {
		t.Errorf("sub.MailboxID = %v, want %v", sub.MailboxID, testMailboxID)
	}
	if sub.UserID != testUserID {
		t.Errorf("sub.UserID = %v, want %v", sub.UserID, testUserID)
	}
	if sub.IsClosed() {
		t.Error("newly created subscription should not be closed")
	}
	if ns.GetSubscriptionCount(testMailboxID) != 1 {
		t.Errorf("subscription count = %d, want 1", ns.GetSubscriptionCount(testMailboxID))
	}
}

func TestNotifyService_Subscribe_MultipleMailboxes(t *testing.T) {
	ns := NewNotifyService()

	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)
	ns.Subscribe("sub-2", testMailboxID2, testUserID, nil)

	if ns.GetSubscriptionCount(testMailboxID) != 1 {
		t.Errorf("mailbox-1 count = %d, want 1", ns.GetSubscriptionCount(testMailboxID))
	}
	if ns.GetSubscriptionCount(testMailboxID2) != 1 {
		t.Errorf("mailbox-2 count = %d, want 1", ns.GetSubscriptionCount(testMailboxID2))
	}
	if ns.GetTotalSubscriptions() != 2 {
		t.Errorf("total = %d, want 2", ns.GetTotalSubscriptions())
	}
}

func TestNotifyService_Subscribe_UpdatesStats(t *testing.T) {
	ns := NewNotifyService()

	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)
	ns.Subscribe("sub-2", testMailboxID, testUserID2, nil)

	stats := ns.GetStats()
	if stats.TotalSubscriptions != 2 {
		t.Errorf("TotalSubscriptions = %d, want 2", stats.TotalSubscriptions)
	}
	if stats.ActiveSubscriptions != 2 {
		t.Errorf("ActiveSubscriptions = %d, want 2", stats.ActiveSubscriptions)
	}
}

// ---- Unsubscribe ------------------------------------------------------------

func TestNotifyService_Unsubscribe(t *testing.T) {
	ns := NewNotifyService()
	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)

	ns.Unsubscribe("sub-1", testMailboxID)

	if ns.GetSubscriptionCount(testMailboxID) != 0 {
		t.Errorf("subscription count = %d, want 0 after unsubscribe", ns.GetSubscriptionCount(testMailboxID))
	}
}

func TestNotifyService_Unsubscribe_MarksClosed(t *testing.T) {
	ns := NewNotifyService()
	sub := ns.Subscribe("sub-1", testMailboxID, testUserID, nil)

	ns.Unsubscribe("sub-1", testMailboxID)

	if !sub.IsClosed() {
		t.Error("unsubscribed subscription should be closed")
	}
}

func TestNotifyService_Unsubscribe_RemovesMailboxEntry(t *testing.T) {
	ns := NewNotifyService()
	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)

	ns.Unsubscribe("sub-1", testMailboxID)

	// After the last subscription for a mailbox is removed the mailbox entry
	// itself must be deleted so HasSubscribers returns false.
	if ns.HasSubscribers(testMailboxID) {
		t.Error("HasSubscribers() should return false after last subscription removed")
	}
}

func TestNotifyService_Unsubscribe_DecreasesActiveCount(t *testing.T) {
	ns := NewNotifyService()
	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)
	ns.Subscribe("sub-2", testMailboxID, testUserID2, nil)

	ns.Unsubscribe("sub-1", testMailboxID)

	stats := ns.GetStats()
	if stats.ActiveSubscriptions != 1 {
		t.Errorf("ActiveSubscriptions = %d, want 1", stats.ActiveSubscriptions)
	}
}

func TestNotifyService_Unsubscribe_NonExistent(t *testing.T) {
	// Must not panic when unsubscribing an ID that was never registered.
	ns := NewNotifyService()
	ns.Unsubscribe("ghost", testMailboxID)
}

// ---- UnsubscribeByID --------------------------------------------------------

func TestNotifyService_UnsubscribeByID(t *testing.T) {
	ns := NewNotifyService()
	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)
	ns.Subscribe("sub-1", testMailboxID2, testUserID, nil) // same sub ID, different mailbox

	ns.UnsubscribeByID("sub-1")

	if ns.GetSubscriptionCount(testMailboxID) != 0 {
		t.Errorf("mailbox-1 count = %d, want 0", ns.GetSubscriptionCount(testMailboxID))
	}
	if ns.GetSubscriptionCount(testMailboxID2) != 0 {
		t.Errorf("mailbox-2 count = %d, want 0", ns.GetSubscriptionCount(testMailboxID2))
	}
}

func TestNotifyService_UnsubscribeByID_NonExistent(t *testing.T) {
	// Must not panic when subscription ID is unknown.
	ns := NewNotifyService()
	ns.UnsubscribeByID("ghost")
}

// ---- HasSubscribers / GetSubscriptionCount ----------------------------------

func TestNotifyService_HasSubscribers(t *testing.T) {
	ns := NewNotifyService()

	if ns.HasSubscribers(testMailboxID) {
		t.Error("HasSubscribers() should return false for empty service")
	}

	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)

	if !ns.HasSubscribers(testMailboxID) {
		t.Error("HasSubscribers() should return true after subscribe")
	}
}

func TestNotifyService_GetSubscriptionCount_NoMailbox(t *testing.T) {
	ns := NewNotifyService()

	if ns.GetSubscriptionCount("nonexistent") != 0 {
		t.Error("GetSubscriptionCount() should return 0 for unknown mailbox")
	}
}

// ---- GetSubscribersForUser --------------------------------------------------

func TestNotifyService_GetSubscribersForUser(t *testing.T) {
	ns := NewNotifyService()
	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)
	ns.Subscribe("sub-2", testMailboxID2, testUserID, nil)
	ns.Subscribe("sub-3", testMailboxID, testUserID2, nil) // different user

	ids := ns.GetSubscribersForUser(testUserID)

	if len(ids) != 2 {
		t.Fatalf("GetSubscribersForUser() returned %d IDs, want 2", len(ids))
	}

	found := make(map[string]bool)
	for _, id := range ids {
		found[id] = true
	}
	if !found["sub-1"] || !found["sub-2"] {
		t.Errorf("GetSubscribersForUser() = %v, want [sub-1 sub-2]", ids)
	}
}

func TestNotifyService_GetSubscribersForUser_NoneFound(t *testing.T) {
	ns := NewNotifyService()
	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)

	ids := ns.GetSubscribersForUser(testUserID2)
	if len(ids) != 0 {
		t.Errorf("GetSubscribersForUser() = %v, want empty slice", ids)
	}
}

// ---- Notify -----------------------------------------------------------------

func TestNotifyService_Notify_DeliverToSubscriber(t *testing.T) {
	ns := NewNotifyService()

	var counter int64
	ns.Subscribe("sub-1", testMailboxID, testUserID, func(n *Notification) {
		atomic.AddInt64(&counter, 1)
	})

	notification := NewNotification(NotificationNewMessage, testMailboxID, testUserID)
	ns.Notify(notification, "")

	if !waitForCount(&counter, 1, 500*time.Millisecond) {
		t.Error("handler was not called within timeout")
	}
}

func TestNotifyService_Notify_ExcludesSubscriptionID(t *testing.T) {
	ns := NewNotifyService()

	var counter1, counter2 int64
	ns.Subscribe("sub-1", testMailboxID, testUserID, func(n *Notification) {
		atomic.AddInt64(&counter1, 1)
	})
	ns.Subscribe("sub-2", testMailboxID, testUserID2, func(n *Notification) {
		atomic.AddInt64(&counter2, 1)
	})

	notification := NewNotification(NotificationNewMessage, testMailboxID, testUserID)
	ns.Notify(notification, "sub-1") // exclude sub-1

	if !waitForCount(&counter2, 1, 500*time.Millisecond) {
		t.Error("sub-2 handler was not called within timeout")
	}
	// Give sub-1 handler extra time to confirm it was not called.
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt64(&counter1) != 0 {
		t.Error("excluded subscription handler should not have been called")
	}
}

func TestNotifyService_Notify_NoSubscribers(t *testing.T) {
	// Must not panic when there are no subscribers.
	ns := NewNotifyService()
	notification := NewNotification(NotificationNewMessage, testMailboxID, testUserID)
	ns.Notify(notification, "")
}

func TestNotifyService_Notify_ClosedSubscription(t *testing.T) {
	ns := NewNotifyService()

	var counter int64
	sub := ns.Subscribe("sub-1", testMailboxID, testUserID, func(n *Notification) {
		atomic.AddInt64(&counter, 1)
	})

	// Close the subscription without unsubscribing to simulate an internal close.
	sub.close()

	notification := NewNotification(NotificationNewMessage, testMailboxID, testUserID)
	ns.Notify(notification, "")

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt64(&counter) != 0 {
		t.Error("handler of a closed subscription should not be called")
	}
}

func TestNotifyService_Notify_UpdatesStats(t *testing.T) {
	ns := NewNotifyService()
	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)

	notification := NewNotification(NotificationNewMessage, testMailboxID, testUserID)
	ns.Notify(notification, "")

	stats := ns.GetStats()
	if stats.TotalNotifications != 1 {
		t.Errorf("TotalNotifications = %d, want 1", stats.TotalNotifications)
	}
	if stats.LastNotificationAt.IsZero() {
		t.Error("LastNotificationAt should not be zero after notification")
	}
}

func TestNotifyService_Notify_NilHandler(t *testing.T) {
	// Must not panic when a subscriber has a nil handler.
	ns := NewNotifyService()
	ns.Subscribe("sub-1", testMailboxID, testUserID, nil)

	notification := NewNotification(NotificationNewMessage, testMailboxID, testUserID)
	ns.Notify(notification, "")
}

// ---- Convenience notify helpers ---------------------------------------------

func TestNotifyService_NotifyNewMessage(t *testing.T) {
	ns := NewNotifyService()

	var received *Notification
	var mu sync.Mutex
	ns.Subscribe("sub-1", testMailboxID, testUserID, func(n *Notification) {
		mu.Lock()
		received = n
		mu.Unlock()
	})

	ns.NotifyNewMessage(testMailboxID, testUserID, testMessageID, 1, 42, 10)

	var ok bool
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok = received != nil
		mu.Unlock()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !ok {
		t.Fatal("NotifyNewMessage() did not deliver notification within timeout")
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Type != NotificationNewMessage {
		t.Errorf("Type = %v, want %v", received.Type, NotificationNewMessage)
	}
	if received.MessageID != testMessageID {
		t.Errorf("MessageID = %v, want %v", received.MessageID, testMessageID)
	}
	if received.SeqNum != 1 {
		t.Errorf("SeqNum = %d, want 1", received.SeqNum)
	}
	if received.UID != 42 {
		t.Errorf("UID = %d, want 42", received.UID)
	}
	if received.MessageCount != 10 {
		t.Errorf("MessageCount = %d, want 10", received.MessageCount)
	}
}

func TestNotifyService_NotifyFlagsChanged(t *testing.T) {
	ns := NewNotifyService()

	var received *Notification
	var mu sync.Mutex
	ns.Subscribe("sub-1", testMailboxID, testUserID, func(n *Notification) {
		mu.Lock()
		received = n
		mu.Unlock()
	})

	flags := []string{`\Seen`, `\Flagged`}
	ns.NotifyFlagsChanged(testMailboxID, testUserID, testMessageID, 2, 55, flags, "")

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := received != nil
		mu.Unlock()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if received == nil {
		t.Fatal("NotifyFlagsChanged() did not deliver notification within timeout")
	}
	if received.Type != NotificationFlagsChanged {
		t.Errorf("Type = %v, want %v", received.Type, NotificationFlagsChanged)
	}
	if len(received.Flags) != 2 || received.Flags[0] != `\Seen` {
		t.Errorf("Flags = %v, want %v", received.Flags, flags)
	}
}

func TestNotifyService_NotifyFlagsChanged_ExcludesSender(t *testing.T) {
	ns := NewNotifyService()

	var counter1, counter2 int64
	ns.Subscribe("sender", testMailboxID, testUserID, func(n *Notification) {
		atomic.AddInt64(&counter1, 1)
	})
	ns.Subscribe("other", testMailboxID, testUserID2, func(n *Notification) {
		atomic.AddInt64(&counter2, 1)
	})

	ns.NotifyFlagsChanged(testMailboxID, testUserID, testMessageID, 1, 1, nil, "sender")

	if !waitForCount(&counter2, 1, 500*time.Millisecond) {
		t.Error("other subscriber was not notified")
	}
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt64(&counter1) != 0 {
		t.Error("sender should have been excluded from flags-changed notification")
	}
}

func TestNotifyService_NotifyMessageExpunged(t *testing.T) {
	ns := NewNotifyService()

	var received *Notification
	var mu sync.Mutex
	ns.Subscribe("sub-1", testMailboxID, testUserID, func(n *Notification) {
		mu.Lock()
		received = n
		mu.Unlock()
	})

	ns.NotifyMessageExpunged(testMailboxID, testUserID, testMessageID, 3)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := received != nil
		mu.Unlock()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if received == nil {
		t.Fatal("NotifyMessageExpunged() did not deliver notification within timeout")
	}
	if received.Type != NotificationMessageExpunged {
		t.Errorf("Type = %v, want %v", received.Type, NotificationMessageExpunged)
	}
	if received.SeqNum != 3 {
		t.Errorf("SeqNum = %d, want 3", received.SeqNum)
	}
}

func TestNotifyService_NotifyMailboxUpdated(t *testing.T) {
	ns := NewNotifyService()

	var received *Notification
	var mu sync.Mutex
	ns.Subscribe("sub-1", testMailboxID, testUserID, func(n *Notification) {
		mu.Lock()
		received = n
		mu.Unlock()
	})

	ns.NotifyMailboxUpdated(testMailboxID, testUserID, 99)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := received != nil
		mu.Unlock()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if received == nil {
		t.Fatal("NotifyMailboxUpdated() did not deliver notification within timeout")
	}
	if received.Type != NotificationMailboxUpdated {
		t.Errorf("Type = %v, want %v", received.Type, NotificationMailboxUpdated)
	}
	if received.MessageCount != 99 {
		t.Errorf("MessageCount = %d, want 99", received.MessageCount)
	}
}

// ---- Close ------------------------------------------------------------------

func TestNotifyService_Close(t *testing.T) {
	ns := NewNotifyService()
	sub1 := ns.Subscribe("sub-1", testMailboxID, testUserID, nil)
	sub2 := ns.Subscribe("sub-2", testMailboxID2, testUserID2, nil)

	err := ns.Close(context.Background())
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !sub1.IsClosed() {
		t.Error("sub-1 should be closed after Close()")
	}
	if !sub2.IsClosed() {
		t.Error("sub-2 should be closed after Close()")
	}
	if ns.GetTotalSubscriptions() != 0 {
		t.Errorf("total subscriptions = %d, want 0 after Close()", ns.GetTotalSubscriptions())
	}

	stats := ns.GetStats()
	if stats.ActiveSubscriptions != 0 {
		t.Errorf("ActiveSubscriptions = %d, want 0 after Close()", stats.ActiveSubscriptions)
	}
}

func TestNotifyService_Close_Empty(t *testing.T) {
	// Must not panic on empty service.
	ns := NewNotifyService()
	if err := ns.Close(context.Background()); err != nil {
		t.Fatalf("Close() on empty service error = %v", err)
	}
}

// ---- Concurrent operations --------------------------------------------------

func TestNotifyService_ConcurrentSubscribeUnsubscribe(t *testing.T) {
	ns := NewNotifyService()
	const goroutines = 20

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			subID := domain.ID(fmt.Sprintf("sub-%d", idx)).String()
			ns.Subscribe(subID, testMailboxID, testUserID, nil)
			ns.Unsubscribe(subID, testMailboxID)
		}(i)
	}
	wg.Wait()

	// All subscriptions should be cleaned up.
	if ns.GetSubscriptionCount(testMailboxID) != 0 {
		t.Errorf("subscription count = %d, want 0 after concurrent subscribe/unsubscribe", ns.GetSubscriptionCount(testMailboxID))
	}
}

func TestNotifyService_ConcurrentNotify(t *testing.T) {
	ns := NewNotifyService()
	const subscribers = 10
	const notifications = 50

	var counter int64
	for i := range subscribers {
		subID := domain.ID(fmt.Sprintf("sub-%d", i)).String()
		ns.Subscribe(subID, testMailboxID, testUserID, func(n *Notification) {
			atomic.AddInt64(&counter, 1)
		})
	}

	var wg sync.WaitGroup
	for range notifications {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n := NewNotification(NotificationNewMessage, testMailboxID, testUserID)
			ns.Notify(n, "")
		}()
	}
	wg.Wait()

	want := int64(subscribers * notifications)
	if !waitForCount(&counter, want, 2*time.Second) {
		t.Errorf("concurrent notifications: got %d calls, want %d", atomic.LoadInt64(&counter), want)
	}
}
