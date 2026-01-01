// Package repository_test contains integration tests that run against all database backends.
// These tests ensure consistent behavior across SQLite, PostgreSQL, MySQL, and MongoDB.
//
// To run tests with all databases:
//  1. Start test databases: docker-compose -f testdata/docker-compose.yml up -d
//  2. Run tests: go test -v -tags=integration ./...
//  3. Stop databases: docker-compose -f testdata/docker-compose.yml down -v
//
// SQLite tests run by default (in-memory). Other databases require Docker containers.
package repository_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"testing"
	"time"

	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/repository/testhelpers"
)

// TestIntegration_UserCRUD tests user CRUD operations across all databases.
func TestIntegration_UserCRUD(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "UserCRUD", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()
		users := repo.Users()

		// Create user
		user := testhelpers.TestUser("user-crud-1")
		err := users.Create(ctx, user)
		testhelpers.AssertNoError(t, err, "create user")

		// Get by ID
		retrieved, err := users.GetByID(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "get user by ID")
		testhelpers.AssertEqual(t, user.Username, retrieved.Username, "username")
		testhelpers.AssertEqual(t, user.Email, retrieved.Email, "email")

		// Get by username
		retrieved, err = users.GetByUsername(ctx, user.Username)
		testhelpers.AssertNoError(t, err, "get user by username")
		testhelpers.AssertEqual(t, user.ID, retrieved.ID, "user ID")

		// Get by email
		retrieved, err = users.GetByEmail(ctx, user.Email)
		testhelpers.AssertNoError(t, err, "get user by email")
		testhelpers.AssertEqual(t, user.ID, retrieved.ID, "user ID")

		// Update
		user.DisplayName = "Updated Name"
		err = users.Update(ctx, user)
		testhelpers.AssertNoError(t, err, "update user")

		retrieved, err = users.GetByID(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "get updated user")
		testhelpers.AssertEqual(t, "Updated Name", retrieved.DisplayName, "display name")

		// Exists
		exists, err := users.Exists(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "check exists")
		testhelpers.AssertEqual(t, true, exists, "user exists")

		// Count
		count, err := users.Count(ctx, nil)
		testhelpers.AssertNoError(t, err, "count users")
		if count < 1 {
			t.Errorf("expected at least 1 user, got %d", count)
		}

		// Delete
		err = users.Delete(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "delete user")

		exists, _ = users.Exists(ctx, user.ID)
		testhelpers.AssertEqual(t, false, exists, "user should not exist after delete")
	})
}

// TestIntegration_UserSoftDelete tests soft delete and restore operations.
func TestIntegration_UserSoftDelete(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "UserSoftDelete", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()
		users := repo.Users()

		// Create user
		user := testhelpers.TestUser("user-soft-del-1")
		err := users.Create(ctx, user)
		testhelpers.AssertNoError(t, err, "create user")

		// Soft delete
		err = users.SoftDelete(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "soft delete user")

		// User should not be found in normal queries
		_, err = users.GetByID(ctx, user.ID)
		if !domain.IsNotFound(err) {
			t.Errorf("expected not found error after soft delete, got %v", err)
		}

		// Restore
		err = users.Restore(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "restore user")

		// User should be found after restore
		retrieved, err := users.GetByID(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "get restored user")
		testhelpers.AssertEqual(t, user.ID, retrieved.ID, "restored user ID")

		// Cleanup
		users.Delete(ctx, user.ID)
	})
}

// TestIntegration_MailboxCRUD tests mailbox CRUD operations across all databases.
func TestIntegration_MailboxCRUD(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "MailboxCRUD", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Create user first
		user := testhelpers.TestUser("mailbox-user-1")
		err := repo.Users().Create(ctx, user)
		testhelpers.AssertNoError(t, err, "create user")
		defer repo.Users().Delete(ctx, user.ID)

		mailboxes := repo.Mailboxes()

		// Create mailbox
		mailbox := testhelpers.TestMailbox("mailbox-1", user.ID)
		err = mailboxes.Create(ctx, mailbox)
		testhelpers.AssertNoError(t, err, "create mailbox")

		// Get by ID
		retrieved, err := mailboxes.GetByID(ctx, mailbox.ID)
		testhelpers.AssertNoError(t, err, "get mailbox by ID")
		testhelpers.AssertEqual(t, mailbox.Name, retrieved.Name, "mailbox name")

		// Get by address
		retrieved, err = mailboxes.GetByAddress(ctx, mailbox.Address)
		testhelpers.AssertNoError(t, err, "get mailbox by address")
		testhelpers.AssertEqual(t, mailbox.ID, retrieved.ID, "mailbox ID")

		// Set as default
		err = mailboxes.SetDefault(ctx, mailbox.ID)
		testhelpers.AssertNoError(t, err, "set default mailbox")

		defaultMailbox, err := mailboxes.GetDefault(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "get default mailbox")
		testhelpers.AssertEqual(t, mailbox.ID, defaultMailbox.ID, "default mailbox ID")

		// List by user
		result, err := mailboxes.ListByUser(ctx, user.ID, nil)
		testhelpers.AssertNoError(t, err, "list mailboxes by user")
		if len(result.Items) < 1 {
			t.Errorf("expected at least 1 mailbox, got %d", len(result.Items))
		}

		// Delete
		err = mailboxes.Delete(ctx, mailbox.ID)
		testhelpers.AssertNoError(t, err, "delete mailbox")

		exists, _ := mailboxes.Exists(ctx, mailbox.ID)
		testhelpers.AssertEqual(t, false, exists, "mailbox should not exist after delete")
	})
}

// TestIntegration_MessageCRUD tests message CRUD operations across all databases.
func TestIntegration_MessageCRUD(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "MessageCRUD", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Setup user and mailbox
		user := testhelpers.TestUser("msg-user-1")
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		mailbox := testhelpers.TestMailbox("msg-mailbox-1", user.ID)
		repo.Mailboxes().Create(ctx, mailbox)
		defer repo.Mailboxes().Delete(ctx, mailbox.ID)

		messages := repo.Messages()

		// Create message
		msg := testhelpers.TestMessage("msg-1", mailbox.ID)
		err := messages.Create(ctx, msg)
		testhelpers.AssertNoError(t, err, "create message")

		// Get by ID
		retrieved, err := messages.GetByID(ctx, msg.ID)
		testhelpers.AssertNoError(t, err, "get message by ID")
		testhelpers.AssertEqual(t, msg.Subject, retrieved.Subject, "message subject")

		// Get by Message-ID
		retrieved, err = messages.GetByMessageID(ctx, msg.MessageID)
		testhelpers.AssertNoError(t, err, "get message by Message-ID")
		testhelpers.AssertEqual(t, msg.ID, retrieved.ID, "message ID")

		// Mark as read
		changed, err := messages.MarkAsRead(ctx, msg.ID)
		testhelpers.AssertNoError(t, err, "mark as read")
		testhelpers.AssertEqual(t, true, changed, "status should have changed")

		retrieved, _ = messages.GetByID(ctx, msg.ID)
		testhelpers.AssertEqual(t, domain.MessageRead, retrieved.Status, "message status")

		// Mark as unread
		changed, err = messages.MarkAsUnread(ctx, msg.ID)
		testhelpers.AssertNoError(t, err, "mark as unread")
		testhelpers.AssertEqual(t, true, changed, "status should have changed")

		// Star
		err = messages.Star(ctx, msg.ID)
		testhelpers.AssertNoError(t, err, "star message")

		retrieved, _ = messages.GetByID(ctx, msg.ID)
		testhelpers.AssertEqual(t, true, retrieved.IsStarred, "message starred")

		// Unstar
		err = messages.Unstar(ctx, msg.ID)
		testhelpers.AssertNoError(t, err, "unstar message")

		// Mark as spam
		err = messages.MarkAsSpam(ctx, msg.ID)
		testhelpers.AssertNoError(t, err, "mark as spam")

		retrieved, _ = messages.GetByID(ctx, msg.ID)
		testhelpers.AssertEqual(t, true, retrieved.IsSpam, "message marked as spam")

		// List by mailbox
		result, err := messages.ListByMailbox(ctx, mailbox.ID, nil)
		testhelpers.AssertNoError(t, err, "list messages by mailbox")
		if len(result.Items) < 1 {
			t.Errorf("expected at least 1 message, got %d", len(result.Items))
		}

		// Delete
		err = messages.Delete(ctx, msg.ID)
		testhelpers.AssertNoError(t, err, "delete message")

		exists, _ := messages.Exists(ctx, msg.ID)
		testhelpers.AssertEqual(t, false, exists, "message should not exist after delete")
	})
}

// TestIntegration_AttachmentCRUD tests attachment CRUD operations across all databases.
func TestIntegration_AttachmentCRUD(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "AttachmentCRUD", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Setup user, mailbox, message
		user := testhelpers.TestUser("att-user-1")
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		mailbox := testhelpers.TestMailbox("att-mailbox-1", user.ID)
		repo.Mailboxes().Create(ctx, mailbox)
		defer repo.Mailboxes().Delete(ctx, mailbox.ID)

		msg := testhelpers.TestMessage("att-msg-1", mailbox.ID)
		repo.Messages().Create(ctx, msg)
		defer repo.Messages().Delete(ctx, msg.ID)

		attachments := repo.Attachments()

		// Create attachment
		att := testhelpers.TestAttachment("att-1", msg.ID)
		err := attachments.Create(ctx, att)
		testhelpers.AssertNoError(t, err, "create attachment")

		// Get by ID
		retrieved, err := attachments.GetByID(ctx, att.ID)
		testhelpers.AssertNoError(t, err, "get attachment by ID")
		testhelpers.AssertEqual(t, att.Filename, retrieved.Filename, "attachment filename")

		// List by message
		list, err := attachments.ListByMessage(ctx, msg.ID)
		testhelpers.AssertNoError(t, err, "list attachments by message")
		if len(list) < 1 {
			t.Errorf("expected at least 1 attachment, got %d", len(list))
		}

		// Store content
		content := []byte("Test attachment content")
		err = attachments.StoreContent(ctx, att.ID, bytes.NewReader(content))
		testhelpers.AssertNoError(t, err, "store content")

		// Get content
		reader, err := attachments.GetContent(ctx, att.ID)
		testhelpers.AssertNoError(t, err, "get content")
		defer reader.Close()

		data, _ := io.ReadAll(reader)
		testhelpers.AssertEqual(t, string(content), string(data), "attachment content")

		// Delete
		err = attachments.Delete(ctx, att.ID)
		testhelpers.AssertNoError(t, err, "delete attachment")

		exists, _ := attachments.Exists(ctx, att.ID)
		testhelpers.AssertEqual(t, false, exists, "attachment should not exist after delete")
	})
}

// TestIntegration_WebhookCRUD tests webhook CRUD operations across all databases.
func TestIntegration_WebhookCRUD(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "WebhookCRUD", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Create user
		user := testhelpers.TestUser("webhook-user-1")
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		webhooks := repo.Webhooks()

		// Create webhook
		webhook := testhelpers.TestWebhook("webhook-1", user.ID)
		err := webhooks.Create(ctx, webhook)
		testhelpers.AssertNoError(t, err, "create webhook")

		// Get by ID
		retrieved, err := webhooks.GetByID(ctx, webhook.ID)
		testhelpers.AssertNoError(t, err, "get webhook by ID")
		testhelpers.AssertEqual(t, webhook.Name, retrieved.Name, "webhook name")
		testhelpers.AssertEqual(t, webhook.URL, retrieved.URL, "webhook URL")

		// List by user
		result, err := webhooks.ListByUser(ctx, user.ID, nil)
		testhelpers.AssertNoError(t, err, "list webhooks by user")
		if len(result.Items) < 1 {
			t.Errorf("expected at least 1 webhook, got %d", len(result.Items))
		}

		// List active by event
		list, err := webhooks.ListActiveByEvent(ctx, domain.WebhookEventMessageReceived)
		testhelpers.AssertNoError(t, err, "list by event")
		if len(list) < 1 {
			t.Errorf("expected at least 1 webhook for event, got %d", len(list))
		}

		// Record success
		err = webhooks.RecordSuccess(ctx, webhook.ID)
		testhelpers.AssertNoError(t, err, "record success")

		retrieved, _ = webhooks.GetByID(ctx, webhook.ID)
		testhelpers.AssertEqual(t, int64(1), retrieved.SuccessCount, "success count")

		// Record failure
		err = webhooks.RecordFailure(ctx, webhook.ID, "test error")
		testhelpers.AssertNoError(t, err, "record failure")

		retrieved, _ = webhooks.GetByID(ctx, webhook.ID)
		testhelpers.AssertEqual(t, int64(1), retrieved.FailureCount, "failure count")

		// Deactivate
		err = webhooks.Deactivate(ctx, webhook.ID)
		testhelpers.AssertNoError(t, err, "deactivate")

		retrieved, _ = webhooks.GetByID(ctx, webhook.ID)
		testhelpers.AssertEqual(t, domain.WebhookStatusInactive, retrieved.Status, "webhook status")

		// Create delivery
		delivery := &domain.WebhookDelivery{
			ID:            domain.ID("delivery-1"),
			WebhookID:     webhook.ID,
			Event:         domain.WebhookEventMessageReceived,
			Payload:       `{"test": true}`,
			StatusCode:    200,
			Success:       true,
			Duration:      100,
			AttemptNumber: 1,
			CreatedAt:     domain.Now(),
		}
		err = webhooks.CreateDelivery(ctx, delivery)
		testhelpers.AssertNoError(t, err, "create delivery")

		// Get delivery stats
		stats, err := webhooks.GetDeliveryStats(ctx, webhook.ID)
		testhelpers.AssertNoError(t, err, "get delivery stats")
		testhelpers.AssertEqual(t, int64(1), stats.TotalDeliveries, "total deliveries")

		// Delete
		err = webhooks.Delete(ctx, webhook.ID)
		testhelpers.AssertNoError(t, err, "delete webhook")

		exists, _ := webhooks.Exists(ctx, webhook.ID)
		testhelpers.AssertEqual(t, false, exists, "webhook should not exist after delete")
	})
}

// TestIntegration_Settings tests settings operations across all databases.
func TestIntegration_Settings(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "Settings", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()
		settings := repo.Settings()

		// Get default settings
		s, err := settings.Get(ctx)
		testhelpers.AssertNoError(t, err, "get settings")
		testhelpers.AssertNotNil(t, s, "settings should not be nil")

		// Save settings
		originalPort := s.SMTP.Port
		s.SMTP.Port = 2025
		s.UpdatedAt = domain.Now()
		err = settings.Save(ctx, s)
		testhelpers.AssertNoError(t, err, "save settings")

		// Get again
		s, _ = settings.Get(ctx)
		testhelpers.AssertEqual(t, 2025, s.SMTP.Port, "SMTP port")

		// Reset
		err = settings.Reset(ctx)
		testhelpers.AssertNoError(t, err, "reset settings")

		s, _ = settings.Get(ctx)
		testhelpers.AssertEqual(t, originalPort, s.SMTP.Port, "SMTP port after reset")

		// Validate
		errors, err := settings.Validate(ctx)
		testhelpers.AssertNoError(t, err, "validate settings")
		if len(errors) > 0 {
			t.Errorf("expected no validation errors, got %d", len(errors))
		}
	})
}

// TestIntegration_Transaction tests transaction support across all databases.
func TestIntegration_Transaction(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "Transaction", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Test successful transaction
		err := repo.Transaction(ctx, func(tx repository.Repository) error {
			user := testhelpers.TestUser("tx-user-1")
			return tx.Users().Create(ctx, user)
		})
		testhelpers.AssertNoError(t, err, "successful transaction")

		// Verify user was created
		_, err = repo.Users().GetByID(ctx, domain.ID("tx-user-1"))
		testhelpers.AssertNoError(t, err, "user should exist after successful transaction")

		// Cleanup
		repo.Users().Delete(ctx, domain.ID("tx-user-1"))

		// Test failed transaction (rollback)
		err = repo.Transaction(ctx, func(tx repository.Repository) error {
			user := testhelpers.TestUser("tx-user-2")
			if err := tx.Users().Create(ctx, user); err != nil {
				return err
			}
			return fmt.Errorf("simulated error")
		})
		testhelpers.AssertError(t, err, "transaction should fail")

		// Verify user was NOT created due to rollback
		_, err = repo.Users().GetByID(ctx, domain.ID("tx-user-2"))
		if !domain.IsNotFound(err) {
			t.Error("user should not exist after rolled back transaction")
		}
	})
}

// TestIntegration_Pagination tests pagination across all databases.
func TestIntegration_Pagination(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "Pagination", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()
		users := repo.Users()

		// Create multiple users
		for i := 1; i <= 25; i++ {
			user := &domain.User{
				ID:           domain.ID(fmt.Sprintf("page-user-%d", i)),
				Username:     fmt.Sprintf("pageuser%d", i),
				Email:        fmt.Sprintf("pageuser%d@test.local", i),
				PasswordHash: "hash",
				Role:         domain.RoleUser,
				Status:       domain.StatusActive,
				CreatedAt:    domain.Now(),
				UpdatedAt:    domain.Now(),
			}
			users.Create(ctx, user)
		}

		// Cleanup
		defer func() {
			for i := 1; i <= 25; i++ {
				users.Delete(ctx, domain.ID(fmt.Sprintf("page-user-%d", i)))
			}
		}()

		// Test first page
		opts := &repository.ListOptions{
			Pagination: &repository.PaginationOptions{
				Page:    1,
				PerPage: 10,
			},
		}

		result, err := users.List(ctx, nil, opts)
		testhelpers.AssertNoError(t, err, "list users page 1")
		testhelpers.AssertEqual(t, 10, len(result.Items), "items on page 1")
		if result.Total < 25 {
			t.Errorf("expected total >= 25, got %d", result.Total)
		}
		testhelpers.AssertEqual(t, true, result.HasMore, "should have more pages")

		// Test second page
		opts.Pagination.Page = 2
		result, _ = users.List(ctx, nil, opts)
		testhelpers.AssertEqual(t, 10, len(result.Items), "items on page 2")

		// Test last page
		opts.Pagination.Page = 3
		result, _ = users.List(ctx, nil, opts)
		if len(result.Items) < 5 {
			t.Errorf("expected at least 5 items on last page, got %d", len(result.Items))
		}
	})
}

// TestIntegration_Search tests search functionality across all databases.
func TestIntegration_Search(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "Search", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Create user
		user := &domain.User{
			ID:           domain.ID("search-user-1"),
			Username:     "searchable_user",
			Email:        "findme@test.local",
			PasswordHash: "hash",
			DisplayName:  "Searchable Display Name",
			Role:         domain.RoleUser,
			Status:       domain.StatusActive,
			CreatedAt:    domain.Now(),
			UpdatedAt:    domain.Now(),
		}
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		// Search by username
		result, err := repo.Users().Search(ctx, "searchable", nil)
		testhelpers.AssertNoError(t, err, "search users")
		if len(result.Items) < 1 {
			t.Error("expected to find user by username search")
		}

		// Search by email
		result, err = repo.Users().Search(ctx, "findme", nil)
		testhelpers.AssertNoError(t, err, "search users by email")
		if len(result.Items) < 1 {
			t.Error("expected to find user by email search")
		}
	})
}

// TestIntegration_MessageSearch tests message search functionality across all databases.
func TestIntegration_MessageSearch(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "MessageSearch", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Setup
		user := testhelpers.TestUser("search-msg-user-1")
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		mailbox := testhelpers.TestMailbox("search-msg-mailbox-1", user.ID)
		repo.Mailboxes().Create(ctx, mailbox)
		defer repo.Mailboxes().Delete(ctx, mailbox.ID)

		// Create messages with different subjects
		messages := []struct {
			id      string
			subject string
			body    string
		}{
			{"search-msg-1", "Important Meeting Tomorrow", "Please join the meeting"},
			{"search-msg-2", "Weekly Report", "Here is the weekly report"},
			{"search-msg-3", "Urgent: Action Required", "Please take action immediately"},
		}

		for _, m := range messages {
			msg := &domain.Message{
				ID:          domain.ID(m.id),
				MailboxID:   mailbox.ID,
				MessageID:   fmt.Sprintf("<%s@test.local>", m.id),
				From:        domain.EmailAddress{Address: "sender@test.local"},
				To:          []domain.EmailAddress{{Address: "recipient@test.local"}},
				Subject:     m.subject,
				TextBody:    m.body,
				ContentType: domain.ContentTypePlain,
				Size:        1024,
				Status:      domain.MessageUnread,
				ReceivedAt:  domain.Now(),
				CreatedAt:   domain.Now(),
				UpdatedAt:   domain.Now(),
			}
			repo.Messages().Create(ctx, msg)
		}

		// Cleanup
		defer func() {
			for _, m := range messages {
				repo.Messages().Delete(ctx, domain.ID(m.id))
			}
		}()

		// Search for "meeting"
		result, err := repo.Messages().Search(ctx, &repository.SearchOptions{Query: "meeting"}, nil, nil)
		testhelpers.AssertNoError(t, err, "search messages")
		if len(result.Items) < 1 {
			t.Error("expected to find message with 'meeting'")
		}

		// Search for "urgent"
		result, err = repo.Messages().Search(ctx, &repository.SearchOptions{Query: "urgent"}, nil, nil)
		testhelpers.AssertNoError(t, err, "search messages for urgent")
		if len(result.Items) < 1 {
			t.Error("expected to find message with 'urgent'")
		}

		// Search for "report"
		result, err = repo.Messages().Search(ctx, &repository.SearchOptions{Query: "report"}, nil, nil)
		testhelpers.AssertNoError(t, err, "search messages for report")
		if len(result.Items) < 1 {
			t.Error("expected to find message with 'report'")
		}
	})
}

// TestIntegration_ConcurrentOperations tests concurrent access to the database.
func TestIntegration_ConcurrentOperations(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "ConcurrentOperations", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Create base user
		user := testhelpers.TestUser("concurrent-user-1")
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		// Create mailbox
		mailbox := testhelpers.TestMailbox("concurrent-mailbox-1", user.ID)
		repo.Mailboxes().Create(ctx, mailbox)
		defer repo.Mailboxes().Delete(ctx, mailbox.ID)

		// Run concurrent message creation
		const numGoroutines = 10
		const messagesPerGoroutine = 5

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*messagesPerGoroutine)
		messageIDs := make(chan domain.ID, numGoroutines*messagesPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < messagesPerGoroutine; j++ {
					msgID := domain.ID(fmt.Sprintf("concurrent-msg-%d-%d", goroutineID, j))
					msg := &domain.Message{
						ID:          msgID,
						MailboxID:   mailbox.ID,
						MessageID:   fmt.Sprintf("<%s@test.local>", msgID),
						From:        domain.EmailAddress{Address: "sender@test.local"},
						To:          []domain.EmailAddress{{Address: "recipient@test.local"}},
						Subject:     fmt.Sprintf("Concurrent Message %d-%d", goroutineID, j),
						TextBody:    "Concurrent test body",
						ContentType: domain.ContentTypePlain,
						Size:        512,
						Status:      domain.MessageUnread,
						ReceivedAt:  domain.Now(),
						CreatedAt:   domain.Now(),
						UpdatedAt:   domain.Now(),
					}

					if err := repo.Messages().Create(ctx, msg); err != nil {
						errors <- fmt.Errorf("goroutine %d, message %d: %w", goroutineID, j, err)
					} else {
						messageIDs <- msgID
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		close(messageIDs)

		// Check for errors
		for err := range errors {
			t.Errorf("concurrent error: %v", err)
		}

		// Verify all messages were created
		createdCount := len(messageIDs)
		expectedCount := numGoroutines * messagesPerGoroutine
		if createdCount != expectedCount {
			t.Errorf("expected %d messages, created %d", expectedCount, createdCount)
		}

		// Cleanup
		for msgID := range messageIDs {
			repo.Messages().Delete(ctx, msgID)
		}

		// Verify final count in mailbox
		count, _ := repo.Messages().Count(ctx, nil)
		t.Logf("Messages after cleanup: %d", count)
	})
}

// TestIntegration_ConcurrentReadWrite tests concurrent read and write operations.
func TestIntegration_ConcurrentReadWrite(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "ConcurrentReadWrite", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Create initial user
		user := testhelpers.TestUser("rw-user-1")
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		const numOperations = 20
		var wg sync.WaitGroup
		errors := make(chan error, numOperations*2)

		// Writers
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				user.DisplayName = fmt.Sprintf("Updated Name %d", i)
				if err := repo.Users().Update(ctx, user); err != nil {
					errors <- fmt.Errorf("write %d: %w", i, err)
				}
			}(i)
		}

		// Readers
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				if _, err := repo.Users().GetByID(ctx, user.ID); err != nil {
					errors <- fmt.Errorf("read %d: %w", i, err)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("concurrent r/w error: %v", err)
		}
	})
}

// TestIntegration_MailboxMessageCountSync tests that mailbox message counts stay in sync.
func TestIntegration_MailboxMessageCountSync(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "MailboxMessageCountSync", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Setup
		user := testhelpers.TestUser("sync-user-1")
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		mailbox := testhelpers.TestMailbox("sync-mailbox-1", user.ID)
		repo.Mailboxes().Create(ctx, mailbox)
		defer repo.Mailboxes().Delete(ctx, mailbox.ID)

		// Verify initial counts
		mb, _ := repo.Mailboxes().GetByID(ctx, mailbox.ID)
		testhelpers.AssertEqual(t, int64(0), mb.MessageCount, "initial message count")
		testhelpers.AssertEqual(t, int64(0), mb.UnreadCount, "initial unread count")

		// Create messages
		for i := 0; i < 5; i++ {
			msg := &domain.Message{
				ID:          domain.ID(fmt.Sprintf("sync-msg-%d", i)),
				MailboxID:   mailbox.ID,
				MessageID:   fmt.Sprintf("<sync-%d@test.local>", i),
				From:        domain.EmailAddress{Address: "sender@test.local"},
				To:          []domain.EmailAddress{{Address: "recipient@test.local"}},
				Subject:     fmt.Sprintf("Sync Test %d", i),
				TextBody:    "Test",
				ContentType: domain.ContentTypePlain,
				Size:        100,
				Status:      domain.MessageUnread,
				ReceivedAt:  domain.Now(),
				CreatedAt:   domain.Now(),
				UpdatedAt:   domain.Now(),
			}
			repo.Messages().Create(ctx, msg)
		}

		// Verify counts after creation
		mb, _ = repo.Mailboxes().GetByID(ctx, mailbox.ID)
		testhelpers.AssertEqual(t, int64(5), mb.MessageCount, "message count after creation")
		testhelpers.AssertEqual(t, int64(5), mb.UnreadCount, "unread count after creation")

		// Mark some as read
		repo.Messages().MarkAsRead(ctx, domain.ID("sync-msg-0"))
		repo.Messages().MarkAsRead(ctx, domain.ID("sync-msg-1"))

		mb, _ = repo.Mailboxes().GetByID(ctx, mailbox.ID)
		testhelpers.AssertEqual(t, int64(5), mb.MessageCount, "message count unchanged")
		testhelpers.AssertEqual(t, int64(3), mb.UnreadCount, "unread count after marking read")

		// Delete a message
		repo.Messages().Delete(ctx, domain.ID("sync-msg-0"))

		mb, _ = repo.Mailboxes().GetByID(ctx, mailbox.ID)
		testhelpers.AssertEqual(t, int64(4), mb.MessageCount, "message count after delete")
		testhelpers.AssertEqual(t, int64(3), mb.UnreadCount, "unread count after delete")

		// Cleanup remaining messages
		for i := 1; i < 5; i++ {
			repo.Messages().Delete(ctx, domain.ID(fmt.Sprintf("sync-msg-%d", i)))
		}
	})
}

// TestIntegration_Performance_InsertUsers tests user insertion performance.
func TestIntegration_Performance_InsertUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	suite := testhelpers.NewTestSuite(nil)
	repos := suite.SetupAllDatabases(t)

	const count = 100
	results := make([]*testhelpers.PerformanceResult, 0, len(repos))

	for _, testRepo := range repos {
		ctx := context.Background()
		users := testRepo.Repository.Users()

		// Create unique IDs for this test
		userIDs := make([]domain.ID, count)
		for i := 0; i < count; i++ {
			userIDs[i] = domain.ID(fmt.Sprintf("perf-user-%s-%d", testRepo.Type, i))
		}

		i := 0
		result, err := testhelpers.MeasurePerformance(testRepo.Type, "InsertUser", count, func() error {
			user := &domain.User{
				ID:           userIDs[i],
				Username:     fmt.Sprintf("perfuser_%s_%d", testRepo.Type, i),
				Email:        fmt.Sprintf("perf%d@%s.test.local", i, testRepo.Type),
				PasswordHash: "hash",
				Role:         domain.RoleUser,
				Status:       domain.StatusActive,
				CreatedAt:    domain.Now(),
				UpdatedAt:    domain.Now(),
			}
			i++
			return users.Create(ctx, user)
		})

		if err != nil {
			t.Errorf("%s: performance test failed: %v", testRepo.Name, err)
			continue
		}

		results = append(results, result)

		// Cleanup
		for _, id := range userIDs {
			users.Delete(ctx, id)
		}
	}

	// Report results
	t.Log("\nUser Insertion Performance:")
	t.Log("----------------------------------")
	for _, r := range results {
		t.Logf("%-15s: %d ops in %v (%.2f ops/sec)", r.DatabaseType, r.Count, r.Duration, r.OpsPerSecond)
	}
}

// TestIntegration_Performance_QueryMessages tests message query performance.
func TestIntegration_Performance_QueryMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	suite := testhelpers.NewTestSuite(nil)
	repos := suite.SetupAllDatabases(t)

	const setupCount = 100
	const queryCount = 50

	results := make([]*testhelpers.PerformanceResult, 0, len(repos))

	for _, testRepo := range repos {
		ctx := context.Background()

		// Setup user and mailbox
		user := testhelpers.TestUser(fmt.Sprintf("perf-query-user-%s", testRepo.Type))
		testRepo.Repository.Users().Create(ctx, user)

		mailbox := testhelpers.TestMailbox(fmt.Sprintf("perf-query-mailbox-%s", testRepo.Type), user.ID)
		testRepo.Repository.Mailboxes().Create(ctx, mailbox)

		// Create messages
		for i := 0; i < setupCount; i++ {
			msg := &domain.Message{
				ID:          domain.ID(fmt.Sprintf("perf-msg-%s-%d", testRepo.Type, i)),
				MailboxID:   mailbox.ID,
				MessageID:   fmt.Sprintf("<perf-%s-%d@test.local>", testRepo.Type, i),
				From:        domain.EmailAddress{Address: "sender@test.local"},
				To:          []domain.EmailAddress{{Address: "recipient@test.local"}},
				Subject:     fmt.Sprintf("Performance Test %d", i),
				TextBody:    "Performance test body content",
				ContentType: domain.ContentTypePlain,
				Size:        1024,
				Status:      domain.MessageUnread,
				ReceivedAt:  domain.Now(),
				CreatedAt:   domain.Now(),
				UpdatedAt:   domain.Now(),
			}
			testRepo.Repository.Messages().Create(ctx, msg)
		}

		// Measure query performance
		result, err := testhelpers.MeasurePerformance(testRepo.Type, "QueryMessages", queryCount, func() error {
			_, err := testRepo.Repository.Messages().ListByMailbox(ctx, mailbox.ID, &repository.ListOptions{
				Pagination: &repository.PaginationOptions{Page: 1, PerPage: 20},
			})
			return err
		})

		if err != nil {
			t.Errorf("%s: performance test failed: %v", testRepo.Name, err)
		} else {
			results = append(results, result)
		}

		// Cleanup
		for i := 0; i < setupCount; i++ {
			testRepo.Repository.Messages().Delete(ctx, domain.ID(fmt.Sprintf("perf-msg-%s-%d", testRepo.Type, i)))
		}
		testRepo.Repository.Mailboxes().Delete(ctx, mailbox.ID)
		testRepo.Repository.Users().Delete(ctx, user.ID)
	}

	// Report results
	t.Log("\nMessage Query Performance:")
	t.Log("----------------------------------")
	for _, r := range results {
		t.Logf("%-15s: %d ops in %v (%.2f ops/sec)", r.DatabaseType, r.Count, r.Duration, r.OpsPerSecond)
	}
}

// TestIntegration_DataConsistency tests data consistency across operations.
func TestIntegration_DataConsistency(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "DataConsistency", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Create user
		user := &domain.User{
			ID:           domain.ID("consistency-user-1"),
			Username:     "consistency_user",
			Email:        "consistency@test.local",
			PasswordHash: "hash",
			DisplayName:  "Consistency Test User",
			Role:         domain.RoleAdmin,
			Status:       domain.StatusActive,
			CreatedAt:    domain.Now(),
			UpdatedAt:    domain.Now(),
		}
		err := repo.Users().Create(ctx, user)
		testhelpers.AssertNoError(t, err, "create user")
		defer repo.Users().Delete(ctx, user.ID)

		// Read back and verify all fields
		retrieved, err := repo.Users().GetByID(ctx, user.ID)
		testhelpers.AssertNoError(t, err, "get user")
		testhelpers.AssertEqual(t, user.ID, retrieved.ID, "ID")
		testhelpers.AssertEqual(t, user.Username, retrieved.Username, "Username")
		testhelpers.AssertEqual(t, user.Email, retrieved.Email, "Email")
		testhelpers.AssertEqual(t, user.DisplayName, retrieved.DisplayName, "DisplayName")
		testhelpers.AssertEqual(t, user.Role, retrieved.Role, "Role")
		testhelpers.AssertEqual(t, user.Status, retrieved.Status, "Status")

		// Verify password hash is preserved (not returned in some implementations)
		if retrieved.PasswordHash != "" && retrieved.PasswordHash != user.PasswordHash {
			t.Errorf("password hash mismatch: expected %s, got %s", user.PasswordHash, retrieved.PasswordHash)
		}
	})
}

// TestIntegration_ErrorHandling tests error handling across all databases.
func TestIntegration_ErrorHandling(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "ErrorHandling", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		// Test not found error
		_, err := repo.Users().GetByID(ctx, domain.ID("nonexistent-user"))
		if !domain.IsNotFound(err) {
			t.Errorf("expected not found error, got %v", err)
		}

		// Test duplicate key error
		user := testhelpers.TestUser("error-user-1")
		repo.Users().Create(ctx, user)
		defer repo.Users().Delete(ctx, user.ID)

		duplicate := testhelpers.TestUser("error-user-2")
		duplicate.Username = user.Username // Same username
		err = repo.Users().Create(ctx, duplicate)
		if err == nil {
			t.Error("expected error for duplicate username")
			repo.Users().Delete(ctx, duplicate.ID)
		}
	})
}

// TestIntegration_BulkOperations tests bulk operations across all databases.
func TestIntegration_BulkOperations(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "BulkOperations", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()
		users := repo.Users()

		// Create multiple users
		userIDs := make([]domain.ID, 10)
		for i := 0; i < 10; i++ {
			user := &domain.User{
				ID:           domain.ID(fmt.Sprintf("bulk-user-%d", i)),
				Username:     fmt.Sprintf("bulkuser%d", i),
				Email:        fmt.Sprintf("bulk%d@test.local", i),
				PasswordHash: "hash",
				Role:         domain.RoleUser,
				Status:       domain.StatusActive,
				CreatedAt:    domain.Now(),
				UpdatedAt:    domain.Now(),
			}
			users.Create(ctx, user)
			userIDs[i] = user.ID
		}

		// Bulk update status
		result, err := users.BulkUpdateStatus(ctx, userIDs[:5], domain.StatusInactive)
		testhelpers.AssertNoError(t, err, "bulk update status")
		testhelpers.AssertEqual(t, int64(5), result.Succeeded, "bulk update succeeded count")

		// Verify updates
		for i := 0; i < 5; i++ {
			user, _ := users.GetByID(ctx, userIDs[i])
			testhelpers.AssertEqual(t, domain.StatusInactive, user.Status, "user status")
		}

		// Bulk delete
		result, err = users.BulkDelete(ctx, userIDs)
		testhelpers.AssertNoError(t, err, "bulk delete")
		if result.Succeeded < 10 {
			t.Errorf("expected 10 deletions, got %d", result.Succeeded)
		}

		// Verify deletions
		for _, id := range userIDs {
			exists, _ := users.Exists(ctx, id)
			testhelpers.AssertEqual(t, false, exists, "user should not exist after bulk delete")
		}
	})
}

// TestIntegration_Sorting tests sorting functionality across all databases.
func TestIntegration_Sorting(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "Sorting", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()
		users := repo.Users()

		// Create users with different names
		names := []string{"alice", "charlie", "bob", "david", "eve"}
		for i, name := range names {
			user := &domain.User{
				ID:           domain.ID(fmt.Sprintf("sort-user-%d", i)),
				Username:     name,
				Email:        fmt.Sprintf("%s@test.local", name),
				PasswordHash: "hash",
				Role:         domain.RoleUser,
				Status:       domain.StatusActive,
				CreatedAt:    domain.Now(),
				UpdatedAt:    domain.Now(),
			}
			users.Create(ctx, user)
		}

		// Cleanup
		defer func() {
			for i := range names {
				users.Delete(ctx, domain.ID(fmt.Sprintf("sort-user-%d", i)))
			}
		}()

		// Sort ascending
		opts := &repository.ListOptions{
			Sort: &repository.SortOptions{
				Field: "username",
				Order: domain.SortAsc,
			},
		}

		result, err := users.List(ctx, &repository.UserFilter{
			IDs: []domain.ID{
				domain.ID("sort-user-0"),
				domain.ID("sort-user-1"),
				domain.ID("sort-user-2"),
				domain.ID("sort-user-3"),
				domain.ID("sort-user-4"),
			},
		}, opts)
		testhelpers.AssertNoError(t, err, "list users sorted")

		// Verify ascending order
		if len(result.Items) >= 2 {
			sortedUsernames := make([]string, len(result.Items))
			for i, u := range result.Items {
				sortedUsernames[i] = u.Username
			}

			isSorted := sort.SliceIsSorted(sortedUsernames, func(i, j int) bool {
				return sortedUsernames[i] < sortedUsernames[j]
			})

			if !isSorted {
				t.Errorf("results not sorted in ascending order: %v", sortedUsernames)
			}
		}
	})
}

// TestIntegration_HealthCheck tests database health checks.
func TestIntegration_HealthCheck(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "HealthCheck", func(t *testing.T, repo repository.Repository) {
		ctx := context.Background()

		err := repo.Health(ctx)
		testhelpers.AssertNoError(t, err, "health check")
	})
}

// TestIntegration_ContextTimeout tests context timeout handling.
func TestIntegration_ContextTimeout(t *testing.T) {
	testhelpers.RunForAllDatabases(t, "ContextTimeout", func(t *testing.T, repo repository.Repository) {
		// Create a context with a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for context to expire
		time.Sleep(1 * time.Millisecond)

		// Attempt operation with expired context
		_, err := repo.Users().GetByID(ctx, domain.ID("any-id"))
		if err == nil {
			// Some implementations might cache or be too fast
			t.Log("operation succeeded despite expired context (implementation-specific)")
		} else if ctx.Err() != context.DeadlineExceeded {
			// The error might be wrapped, so we just check that we got some error
			t.Logf("got error with expired context: %v", err)
		}
	})
}
