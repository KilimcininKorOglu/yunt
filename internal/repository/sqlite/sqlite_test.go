package sqlite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// testRepo creates a new in-memory SQLite repository for testing.
func testRepo(t *testing.T) *Repository {
	t.Helper()

	cfg := &ConnectionConfig{
		DSN:               ":memory:",
		MaxOpenConns:      1,
		MaxIdleConns:      1,
		EnableForeignKeys: true,
		JournalMode:       "MEMORY",
		SynchronousMode:   "OFF",
	}

	pool, err := NewConnectionPool(cfg)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	repo, err := NewWithOptions(pool, true, false)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create repository: %v", err)
	}

	t.Cleanup(func() {
		repo.Close()
	})

	return repo
}

// TestConnectionPool tests the connection pool.
func TestConnectionPool(t *testing.T) {
	cfg := DefaultConnectionConfig()
	cfg.DSN = ":memory:"

	pool, err := NewConnectionPool(cfg)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	// Test health check
	ctx := context.Background()
	if err := pool.Health(ctx); err != nil {
		t.Errorf("health check failed: %v", err)
	}

	// Test version
	version, err := pool.Version(ctx)
	if err != nil {
		t.Errorf("failed to get version: %v", err)
	}
	if version == "" {
		t.Error("version should not be empty")
	}

	// Test stats
	stats := pool.Stats()
	if stats.OpenConnections < 1 {
		t.Error("should have at least one open connection")
	}
}

// TestRepository tests the main repository.
func TestRepository(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Test health
	if err := repo.Health(ctx); err != nil {
		t.Errorf("health check failed: %v", err)
	}

	// Test database info
	info, err := repo.DatabaseInfo(ctx)
	if err != nil {
		t.Errorf("failed to get database info: %v", err)
	}
	if info.Driver != domain.DatabaseDriverSQLite {
		t.Errorf("expected SQLite driver, got %v", info.Driver)
	}

	// Test sub-repositories are available
	if repo.Users() == nil {
		t.Error("users repository should not be nil")
	}
	if repo.Mailboxes() == nil {
		t.Error("mailboxes repository should not be nil")
	}
	if repo.Messages() == nil {
		t.Error("messages repository should not be nil")
	}
	if repo.Attachments() == nil {
		t.Error("attachments repository should not be nil")
	}
	if repo.Webhooks() == nil {
		t.Error("webhooks repository should not be nil")
	}
	if repo.Settings() == nil {
		t.Error("settings repository should not be nil")
	}
}

// TestUserRepository tests user CRUD operations.
func TestUserRepository(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	users := repo.Users()

	// Create user
	user := &domain.User{
		ID:           domain.ID("user-1"),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}

	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Get by ID
	retrieved, err := users.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to get user by ID: %v", err)
	}
	if retrieved.Username != user.Username {
		t.Errorf("expected username %s, got %s", user.Username, retrieved.Username)
	}

	// Get by username
	retrieved, err = users.GetByUsername(ctx, user.Username)
	if err != nil {
		t.Fatalf("failed to get user by username: %v", err)
	}
	if retrieved.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, retrieved.ID)
	}

	// Get by email
	retrieved, err = users.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}
	if retrieved.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, retrieved.ID)
	}

	// Exists
	exists, err := users.Exists(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to check existence: %v", err)
	}
	if !exists {
		t.Error("user should exist")
	}

	// Update
	user.DisplayName = "Test User"
	if err := users.Update(ctx, user); err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	retrieved, _ = users.GetByID(ctx, user.ID)
	if retrieved.DisplayName != "Test User" {
		t.Errorf("expected display name 'Test User', got '%s'", retrieved.DisplayName)
	}

	// Update password
	if err := users.UpdatePassword(ctx, user.ID, "newhash"); err != nil {
		t.Fatalf("failed to update password: %v", err)
	}

	// Update status
	if err := users.UpdateStatus(ctx, user.ID, domain.StatusInactive); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	// Update role
	if err := users.UpdateRole(ctx, user.ID, domain.RoleAdmin); err != nil {
		t.Fatalf("failed to update role: %v", err)
	}

	// Update last login
	if err := users.UpdateLastLogin(ctx, user.ID); err != nil {
		t.Fatalf("failed to update last login: %v", err)
	}

	// Count
	count, err := users.Count(ctx, nil)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}

	// List
	result, err := users.List(ctx, nil, nil)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 user, got %d", len(result.Items))
	}

	// Search
	result, err = users.Search(ctx, "test", nil)
	if err != nil {
		t.Fatalf("failed to search users: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 user in search results, got %d", len(result.Items))
	}

	// Soft delete
	if err := users.SoftDelete(ctx, user.ID); err != nil {
		t.Fatalf("failed to soft delete user: %v", err)
	}

	// User should not be found after soft delete
	_, err = users.GetByID(ctx, user.ID)
	if !domain.IsNotFound(err) {
		t.Error("user should not be found after soft delete")
	}

	// Restore
	if err := users.Restore(ctx, user.ID); err != nil {
		t.Fatalf("failed to restore user: %v", err)
	}

	// User should be found after restore
	retrieved, err = users.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("user should be found after restore: %v", err)
	}

	// Delete
	if err := users.Delete(ctx, user.ID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// User should not exist after delete
	exists, _ = users.Exists(ctx, user.ID)
	if exists {
		t.Error("user should not exist after delete")
	}
}

// TestMailboxRepository tests mailbox CRUD operations.
func TestMailboxRepository(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Create a user first
	user := &domain.User{
		ID:           domain.ID("user-1"),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	if err := repo.Users().Create(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	mailboxes := repo.Mailboxes()

	// Create mailbox
	mailbox := &domain.Mailbox{
		ID:        domain.ID("mailbox-1"),
		UserID:    user.ID,
		Name:      "Inbox",
		Address:   "inbox@example.com",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}

	if err := mailboxes.Create(ctx, mailbox); err != nil {
		t.Fatalf("failed to create mailbox: %v", err)
	}

	// Get by ID
	retrieved, err := mailboxes.GetByID(ctx, mailbox.ID)
	if err != nil {
		t.Fatalf("failed to get mailbox: %v", err)
	}
	if retrieved.Name != "Inbox" {
		t.Errorf("expected name 'Inbox', got '%s'", retrieved.Name)
	}

	// Get by address
	retrieved, err = mailboxes.GetByAddress(ctx, mailbox.Address)
	if err != nil {
		t.Fatalf("failed to get mailbox by address: %v", err)
	}
	if retrieved.ID != mailbox.ID {
		t.Errorf("expected ID %s, got %s", mailbox.ID, retrieved.ID)
	}

	// Set default
	if err := mailboxes.SetDefault(ctx, mailbox.ID); err != nil {
		t.Fatalf("failed to set default: %v", err)
	}

	// Get default
	defaultMailbox, err := mailboxes.GetDefault(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to get default mailbox: %v", err)
	}
	if defaultMailbox.ID != mailbox.ID {
		t.Errorf("expected default mailbox ID %s, got %s", mailbox.ID, defaultMailbox.ID)
	}

	// Update stats
	if err := mailboxes.IncrementMessageCount(ctx, mailbox.ID, 1024); err != nil {
		t.Fatalf("failed to increment message count: %v", err)
	}

	retrieved, _ = mailboxes.GetByID(ctx, mailbox.ID)
	if retrieved.MessageCount != 1 {
		t.Errorf("expected message count 1, got %d", retrieved.MessageCount)
	}

	// Count
	count, err := mailboxes.Count(ctx, nil)
	if err != nil {
		t.Fatalf("failed to count mailboxes: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 mailbox, got %d", count)
	}

	// List by user
	result, err := mailboxes.ListByUser(ctx, user.ID, nil)
	if err != nil {
		t.Fatalf("failed to list mailboxes by user: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 mailbox, got %d", len(result.Items))
	}

	// Get stats
	stats, err := mailboxes.GetStats(ctx, mailbox.ID)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.TotalMessages != 1 {
		t.Errorf("expected 1 total message in stats, got %d", stats.TotalMessages)
	}

	// Delete
	if err := mailboxes.Delete(ctx, mailbox.ID); err != nil {
		t.Fatalf("failed to delete mailbox: %v", err)
	}

	exists, _ := mailboxes.Exists(ctx, mailbox.ID)
	if exists {
		t.Error("mailbox should not exist after delete")
	}
}

// TestMessageRepository tests message CRUD operations.
func TestMessageRepository(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Create user and mailbox
	user := &domain.User{
		ID:           domain.ID("user-1"),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	repo.Users().Create(ctx, user)

	mailbox := &domain.Mailbox{
		ID:        domain.ID("mailbox-1"),
		UserID:    user.ID,
		Name:      "Inbox",
		Address:   "inbox@example.com",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	repo.Mailboxes().Create(ctx, mailbox)

	messages := repo.Messages()

	// Create message
	msg := &domain.Message{
		ID:          domain.ID("msg-1"),
		MailboxID:   mailbox.ID,
		MessageID:   "<test@example.com>",
		From:        domain.EmailAddress{Name: "Sender", Address: "sender@example.com"},
		To:          []domain.EmailAddress{{Address: "inbox@example.com"}},
		Subject:     "Test Subject",
		TextBody:    "This is a test message.",
		ContentType: domain.ContentTypePlain,
		Size:        100,
		Status:      domain.MessageUnread,
		ReceivedAt:  domain.Now(),
		CreatedAt:   domain.Now(),
		UpdatedAt:   domain.Now(),
	}

	if err := messages.Create(ctx, msg); err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// Get by ID
	retrieved, err := messages.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("failed to get message: %v", err)
	}
	if retrieved.Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got '%s'", retrieved.Subject)
	}

	// Get by Message-ID
	retrieved, err = messages.GetByMessageID(ctx, msg.MessageID)
	if err != nil {
		t.Fatalf("failed to get message by Message-ID: %v", err)
	}
	if retrieved.ID != msg.ID {
		t.Errorf("expected ID %s, got %s", msg.ID, retrieved.ID)
	}

	// Mark as read
	changed, err := messages.MarkAsRead(ctx, msg.ID)
	if err != nil {
		t.Fatalf("failed to mark as read: %v", err)
	}
	if !changed {
		t.Error("status should have changed")
	}

	retrieved, _ = messages.GetByID(ctx, msg.ID)
	if retrieved.Status != domain.MessageRead {
		t.Errorf("expected status Read, got %s", retrieved.Status)
	}

	// Mark as unread
	changed, err = messages.MarkAsUnread(ctx, msg.ID)
	if err != nil {
		t.Fatalf("failed to mark as unread: %v", err)
	}
	if !changed {
		t.Error("status should have changed")
	}

	// Star
	if err := messages.Star(ctx, msg.ID); err != nil {
		t.Fatalf("failed to star message: %v", err)
	}

	retrieved, _ = messages.GetByID(ctx, msg.ID)
	if !retrieved.IsStarred {
		t.Error("message should be starred")
	}

	// Unstar
	if err := messages.Unstar(ctx, msg.ID); err != nil {
		t.Fatalf("failed to unstar message: %v", err)
	}

	// Mark as spam
	if err := messages.MarkAsSpam(ctx, msg.ID); err != nil {
		t.Fatalf("failed to mark as spam: %v", err)
	}

	retrieved, _ = messages.GetByID(ctx, msg.ID)
	if !retrieved.IsSpam {
		t.Error("message should be marked as spam")
	}

	// Count
	count, err := messages.Count(ctx, nil)
	if err != nil {
		t.Fatalf("failed to count messages: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 message, got %d", count)
	}

	// List by mailbox
	result, err := messages.ListByMailbox(ctx, mailbox.ID, nil)
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 message, got %d", len(result.Items))
	}

	// Search
	result, err = messages.Search(ctx, &repository.SearchOptions{Query: "test"}, nil, nil)
	if err != nil {
		t.Fatalf("failed to search messages: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 message in search, got %d", len(result.Items))
	}

	// Delete
	if err := messages.Delete(ctx, msg.ID); err != nil {
		t.Fatalf("failed to delete message: %v", err)
	}

	exists, _ := messages.Exists(ctx, msg.ID)
	if exists {
		t.Error("message should not exist after delete")
	}
}

// TestAttachmentRepository tests attachment CRUD operations.
func TestAttachmentRepository(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Setup user, mailbox, message
	user := &domain.User{
		ID: domain.ID("user-1"), Username: "test", Email: "test@example.com",
		PasswordHash: "hash", Role: domain.RoleUser, Status: domain.StatusActive,
		CreatedAt: domain.Now(), UpdatedAt: domain.Now(),
	}
	repo.Users().Create(ctx, user)

	mailbox := &domain.Mailbox{
		ID: domain.ID("mailbox-1"), UserID: user.ID, Name: "Inbox", Address: "inbox@example.com",
		CreatedAt: domain.Now(), UpdatedAt: domain.Now(),
	}
	repo.Mailboxes().Create(ctx, mailbox)

	msg := &domain.Message{
		ID: domain.ID("msg-1"), MailboxID: mailbox.ID, MessageID: "<test@example.com>",
		From: domain.EmailAddress{Address: "sender@example.com"},
		To: []domain.EmailAddress{{Address: "inbox@example.com"}}, Subject: "Test",
		ContentType: domain.ContentTypePlain, Size: 100, Status: domain.MessageUnread,
		ReceivedAt: domain.Now(), CreatedAt: domain.Now(), UpdatedAt: domain.Now(),
	}
	repo.Messages().Create(ctx, msg)

	attachments := repo.Attachments()

	// Create attachment
	att := &domain.Attachment{
		ID:          domain.ID("att-1"),
		MessageID:   msg.ID,
		Filename:    "test.pdf",
		ContentType: "application/pdf",
		Size:        1024,
		Disposition: domain.DispositionAttachment,
		CreatedAt:   domain.Now(),
	}

	if err := attachments.Create(ctx, att); err != nil {
		t.Fatalf("failed to create attachment: %v", err)
	}

	// Get by ID
	retrieved, err := attachments.GetByID(ctx, att.ID)
	if err != nil {
		t.Fatalf("failed to get attachment: %v", err)
	}
	if retrieved.Filename != "test.pdf" {
		t.Errorf("expected filename 'test.pdf', got '%s'", retrieved.Filename)
	}

	// List by message
	list, err := attachments.ListByMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("failed to list attachments: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(list))
	}

	// Store content
	content := []byte("PDF content here")
	if err := attachments.StoreContent(ctx, att.ID, bytes.NewReader(content)); err != nil {
		t.Fatalf("failed to store content: %v", err)
	}

	// Get content
	reader, err := attachments.GetContent(ctx, att.ID)
	if err != nil {
		t.Fatalf("failed to get content: %v", err)
	}
	defer reader.Close()

	data, _ := io.ReadAll(reader)
	if string(data) != string(content) {
		t.Errorf("content mismatch: expected '%s', got '%s'", content, data)
	}

	// Get total size
	totalSize, err := attachments.GetTotalSize(ctx)
	if err != nil {
		t.Fatalf("failed to get total size: %v", err)
	}
	if totalSize != att.Size {
		t.Errorf("expected total size %d, got %d", att.Size, totalSize)
	}

	// Delete
	if err := attachments.Delete(ctx, att.ID); err != nil {
		t.Fatalf("failed to delete attachment: %v", err)
	}

	exists, _ := attachments.Exists(ctx, att.ID)
	if exists {
		t.Error("attachment should not exist after delete")
	}
}

// TestWebhookRepository tests webhook CRUD operations.
func TestWebhookRepository(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Create user
	user := &domain.User{
		ID: domain.ID("user-1"), Username: "test", Email: "test@example.com",
		PasswordHash: "hash", Role: domain.RoleUser, Status: domain.StatusActive,
		CreatedAt: domain.Now(), UpdatedAt: domain.Now(),
	}
	repo.Users().Create(ctx, user)

	webhooks := repo.Webhooks()

	// Create webhook
	webhook := &domain.Webhook{
		ID:             domain.ID("webhook-1"),
		UserID:         user.ID,
		Name:           "Test Webhook",
		URL:            "https://example.com/webhook",
		Events:         []domain.WebhookEvent{domain.WebhookEventMessageReceived},
		Status:         domain.WebhookStatusActive,
		MaxRetries:     3,
		TimeoutSeconds: 30,
		CreatedAt:      domain.Now(),
		UpdatedAt:      domain.Now(),
	}

	if err := webhooks.Create(ctx, webhook); err != nil {
		t.Fatalf("failed to create webhook: %v", err)
	}

	// Get by ID
	retrieved, err := webhooks.GetByID(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("failed to get webhook: %v", err)
	}
	if retrieved.Name != "Test Webhook" {
		t.Errorf("expected name 'Test Webhook', got '%s'", retrieved.Name)
	}

	// List by user
	result, err := webhooks.ListByUser(ctx, user.ID, nil)
	if err != nil {
		t.Fatalf("failed to list webhooks: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(result.Items))
	}

	// List by event
	list, err := webhooks.ListActiveByEvent(ctx, domain.WebhookEventMessageReceived)
	if err != nil {
		t.Fatalf("failed to list by event: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(list))
	}

	// Record success
	if err := webhooks.RecordSuccess(ctx, webhook.ID); err != nil {
		t.Fatalf("failed to record success: %v", err)
	}

	retrieved, _ = webhooks.GetByID(ctx, webhook.ID)
	if retrieved.SuccessCount != 1 {
		t.Errorf("expected success count 1, got %d", retrieved.SuccessCount)
	}

	// Record failure
	if err := webhooks.RecordFailure(ctx, webhook.ID, "connection refused"); err != nil {
		t.Fatalf("failed to record failure: %v", err)
	}

	retrieved, _ = webhooks.GetByID(ctx, webhook.ID)
	if retrieved.FailureCount != 1 {
		t.Errorf("expected failure count 1, got %d", retrieved.FailureCount)
	}

	// Deactivate
	if err := webhooks.Deactivate(ctx, webhook.ID); err != nil {
		t.Fatalf("failed to deactivate: %v", err)
	}

	retrieved, _ = webhooks.GetByID(ctx, webhook.ID)
	if retrieved.Status != domain.WebhookStatusInactive {
		t.Errorf("expected status Inactive, got %s", retrieved.Status)
	}

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

	if err := webhooks.CreateDelivery(ctx, delivery); err != nil {
		t.Fatalf("failed to create delivery: %v", err)
	}

	// Get delivery
	retrievedDelivery, err := webhooks.GetDelivery(ctx, delivery.ID)
	if err != nil {
		t.Fatalf("failed to get delivery: %v", err)
	}
	if !retrievedDelivery.Success {
		t.Error("delivery should be successful")
	}

	// Get delivery stats
	stats, err := webhooks.GetDeliveryStats(ctx, webhook.ID)
	if err != nil {
		t.Fatalf("failed to get delivery stats: %v", err)
	}
	if stats.TotalDeliveries != 1 {
		t.Errorf("expected 1 delivery, got %d", stats.TotalDeliveries)
	}

	// Delete
	if err := webhooks.Delete(ctx, webhook.ID); err != nil {
		t.Fatalf("failed to delete webhook: %v", err)
	}

	exists, _ := webhooks.Exists(ctx, webhook.ID)
	if exists {
		t.Error("webhook should not exist after delete")
	}
}

// TestSettingsRepository tests settings CRUD operations.
func TestSettingsRepository(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	settings := repo.Settings()

	// Get default settings
	s, err := settings.Get(ctx)
	if err != nil {
		t.Fatalf("failed to get settings: %v", err)
	}
	if s.SMTP.Port != 1025 {
		t.Errorf("expected SMTP port 1025, got %d", s.SMTP.Port)
	}

	// Save settings
	s.SMTP.Port = 2025
	s.UpdatedAt = domain.Now()
	if err := settings.Save(ctx, s); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	// Get again
	s, _ = settings.Get(ctx)
	if s.SMTP.Port != 2025 {
		t.Errorf("expected SMTP port 2025, got %d", s.SMTP.Port)
	}

	// Update SMTP
	port := 3025
	if err := settings.UpdateSMTP(ctx, &domain.SMTPSettingsUpdate{Port: &port}); err != nil {
		t.Fatalf("failed to update SMTP: %v", err)
	}

	smtp, err := settings.GetSMTP(ctx)
	if err != nil {
		t.Fatalf("failed to get SMTP: %v", err)
	}
	if smtp.Port != 3025 {
		t.Errorf("expected SMTP port 3025, got %d", smtp.Port)
	}

	// Export
	export, err := settings.Export(ctx)
	if err != nil {
		t.Fatalf("failed to export: %v", err)
	}
	if export.Settings == nil {
		t.Error("exported settings should not be nil")
	}

	// Reset
	if err := settings.Reset(ctx); err != nil {
		t.Fatalf("failed to reset: %v", err)
	}

	s, _ = settings.Get(ctx)
	if s.SMTP.Port != 1025 {
		t.Errorf("expected default SMTP port 1025 after reset, got %d", s.SMTP.Port)
	}

	// Validate
	errors, err := settings.Validate(ctx)
	if err != nil {
		t.Fatalf("failed to validate: %v", err)
	}
	if len(errors) > 0 {
		t.Errorf("expected no validation errors, got %d", len(errors))
	}

	// Test database connection
	if err := settings.TestDatabaseConnection(ctx); err != nil {
		t.Errorf("database connection test failed: %v", err)
	}
}

// TestTransaction tests transaction support.
func TestTransaction(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Test successful transaction
	err := repo.Transaction(ctx, func(tx repository.Repository) error {
		user := &domain.User{
			ID: domain.ID("tx-user-1"), Username: "txuser", Email: "tx@example.com",
			PasswordHash: "hash", Role: domain.RoleUser, Status: domain.StatusActive,
			CreatedAt: domain.Now(), UpdatedAt: domain.Now(),
		}
		return tx.Users().Create(ctx, user)
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	// Verify user was created
	_, err = repo.Users().GetByID(ctx, domain.ID("tx-user-1"))
	if err != nil {
		t.Error("user should exist after successful transaction")
	}

	// Test failed transaction (rollback)
	err = repo.Transaction(ctx, func(tx repository.Repository) error {
		user := &domain.User{
			ID: domain.ID("tx-user-2"), Username: "txuser2", Email: "tx2@example.com",
			PasswordHash: "hash", Role: domain.RoleUser, Status: domain.StatusActive,
			CreatedAt: domain.Now(), UpdatedAt: domain.Now(),
		}
		if err := tx.Users().Create(ctx, user); err != nil {
			return err
		}
		return fmt.Errorf("simulated error")
	})
	if err == nil {
		t.Error("transaction should have failed")
	}

	// Verify user was NOT created due to rollback
	_, err = repo.Users().GetByID(ctx, domain.ID("tx-user-2"))
	if !domain.IsNotFound(err) {
		t.Error("user should not exist after rolled back transaction")
	}
}

// TestPagination tests pagination functionality.
func TestPagination(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Create multiple users
	for i := 1; i <= 25; i++ {
		user := &domain.User{
			ID:           domain.ID(fmt.Sprintf("user-%d", i)),
			Username:     fmt.Sprintf("user%d", i),
			Email:        fmt.Sprintf("user%d@example.com", i),
			PasswordHash: "hash",
			Role:         domain.RoleUser,
			Status:       domain.StatusActive,
			CreatedAt:    domain.Now(),
			UpdatedAt:    domain.Now(),
		}
		repo.Users().Create(ctx, user)
	}

	// Test pagination
	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: 10,
		},
	}

	result, err := repo.Users().List(ctx, nil, opts)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}

	if len(result.Items) != 10 {
		t.Errorf("expected 10 items, got %d", len(result.Items))
	}
	if result.Total != 25 {
		t.Errorf("expected total 25, got %d", result.Total)
	}
	if !result.HasMore {
		t.Error("should have more pages")
	}

	// Test second page
	opts.Pagination.Page = 2
	result, _ = repo.Users().List(ctx, nil, opts)
	if len(result.Items) != 10 {
		t.Errorf("expected 10 items on page 2, got %d", len(result.Items))
	}

	// Test last page
	opts.Pagination.Page = 3
	result, _ = repo.Users().List(ctx, nil, opts)
	if len(result.Items) != 5 {
		t.Errorf("expected 5 items on last page, got %d", len(result.Items))
	}
	if result.HasMore {
		t.Error("should not have more pages")
	}
}

// TestMigrator tests the migration system.
func TestMigrator(t *testing.T) {
	cfg := &ConnectionConfig{
		DSN:               ":memory:",
		MaxOpenConns:      1,
		MaxIdleConns:      1,
		EnableForeignKeys: true,
		JournalMode:       "MEMORY",
		SynchronousMode:   "OFF",
	}

	pool, err := NewConnectionPool(cfg)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}
	defer pool.Close()

	migrator, err := NewMigrator(pool)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	ctx := context.Background()

	// Test initial migration version
	version, err := migrator.MigrationVersion(ctx)
	if err != nil {
		t.Fatalf("failed to get migration version: %v", err)
	}
	if version != 0 {
		t.Errorf("expected version 0, got %d", version)
	}

	// Test pending migrations
	pending, err := migrator.IsPending(ctx)
	if err != nil {
		t.Fatalf("failed to check pending: %v", err)
	}
	if !pending {
		t.Error("should have pending migrations")
	}

	// Get pending migrations list
	pendingMigrations, err := migrator.GetPendingMigrations(ctx)
	if err != nil {
		t.Fatalf("failed to get pending migrations: %v", err)
	}
	if len(pendingMigrations) < 2 {
		t.Errorf("expected at least 2 pending migrations, got %d", len(pendingMigrations))
	}

	// Run all migrations
	if err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Check version after migration
	version, err = migrator.MigrationVersion(ctx)
	if err != nil {
		t.Fatalf("failed to get migration version: %v", err)
	}
	if version < 2 {
		t.Errorf("expected version >= 2, got %d", version)
	}

	// Check no pending migrations
	pending, err = migrator.IsPending(ctx)
	if err != nil {
		t.Fatalf("failed to check pending: %v", err)
	}
	if pending {
		t.Error("should not have pending migrations")
	}

	// Test migration status
	status, err := migrator.MigrationStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get migration status: %v", err)
	}
	if len(status) < 2 {
		t.Errorf("expected at least 2 migration statuses, got %d", len(status))
	}
	for _, s := range status {
		if !s.Applied {
			t.Errorf("migration %d should be applied", s.Version)
		}
		if s.AppliedAt == nil {
			t.Errorf("migration %d should have applied_at timestamp", s.Version)
		}
	}

	// Test migrations are idempotent (running again should not fail)
	if err := migrator.Migrate(ctx); err != nil {
		t.Errorf("running migrations again should not fail: %v", err)
	}
}

// TestMigratorRollback tests migration rollback functionality.
func TestMigratorRollback(t *testing.T) {
	cfg := &ConnectionConfig{
		DSN:               ":memory:",
		MaxOpenConns:      1,
		MaxIdleConns:      1,
		EnableForeignKeys: true,
		JournalMode:       "MEMORY",
		SynchronousMode:   "OFF",
	}

	pool, err := NewConnectionPool(cfg)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}
	defer pool.Close()

	migrator, err := NewMigrator(pool)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	ctx := context.Background()

	// Run all migrations
	if err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Get version after full migration
	versionBefore, err := migrator.MigrationVersion(ctx)
	if err != nil {
		t.Fatalf("failed to get migration version: %v", err)
	}

	// Rollback one migration
	if err := migrator.MigrateDown(ctx, 1); err != nil {
		t.Fatalf("failed to rollback migration: %v", err)
	}

	// Check version decreased
	versionAfter, err := migrator.MigrationVersion(ctx)
	if err != nil {
		t.Fatalf("failed to get migration version: %v", err)
	}
	if versionAfter >= versionBefore {
		t.Errorf("version should have decreased: before=%d, after=%d", versionBefore, versionAfter)
	}

	// Migrate up again
	if err := migrator.MigrateUp(ctx, 1); err != nil {
		t.Fatalf("failed to migrate up: %v", err)
	}

	// Check version increased
	versionFinal, err := migrator.MigrationVersion(ctx)
	if err != nil {
		t.Fatalf("failed to get migration version: %v", err)
	}
	if versionFinal != versionBefore {
		t.Errorf("expected version %d, got %d", versionBefore, versionFinal)
	}
}

// TestSeeder tests the database seeder.
func TestSeeder(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	seeder := repo.Seeder()
	if seeder == nil {
		t.Fatal("seeder should not be nil")
	}

	// Seed explicitly (testRepo does not auto-seed)
	if err := seeder.Seed(ctx); err != nil {
		t.Fatalf("failed to seed: %v", err)
	}

	isSeeded, err := seeder.IsSeeded(ctx)
	if err != nil {
		t.Fatalf("failed to check if seeded: %v", err)
	}
	if !isSeeded {
		t.Error("database should be seeded")
	}

	// Get seed status
	status, err := seeder.GetSeedStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get seed status: %v", err)
	}
	if !status.HasAdmin {
		t.Error("should have admin user")
	}
	if status.UserCount < 1 {
		t.Error("should have at least 1 user")
	}
	if status.MailboxCount < 1 {
		t.Error("should have at least 1 mailbox")
	}

	// Verify admin user exists
	admins, err := repo.Users().GetAdmins(ctx)
	if err != nil {
		t.Fatalf("failed to get admins: %v", err)
	}
	if len(admins) < 1 {
		t.Error("should have at least 1 admin")
	}

	// Seeding again should be idempotent (not create duplicate admin)
	if err := seeder.Seed(ctx); err != nil {
		t.Errorf("seeding again should not fail: %v", err)
	}

	// Check admin count is still 1
	admins, err = repo.Users().GetAdmins(ctx)
	if err != nil {
		t.Fatalf("failed to get admins: %v", err)
	}
	if len(admins) != 1 {
		t.Errorf("should still have 1 admin, got %d", len(admins))
	}
}

// TestSeederWithConfig tests the seeder with custom configuration.
func TestSeederWithConfig(t *testing.T) {
	cfg := &ConnectionConfig{
		DSN:               ":memory:",
		MaxOpenConns:      1,
		MaxIdleConns:      1,
		EnableForeignKeys: true,
		JournalMode:       "MEMORY",
		SynchronousMode:   "OFF",
	}

	pool, err := NewConnectionPool(cfg)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	// Create repo without auto-seed
	repo, err := NewWithOptions(pool, true, false)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Verify not seeded
	isSeeded, err := repo.IsSeeded(ctx)
	if err != nil {
		t.Fatalf("failed to check if seeded: %v", err)
	}
	if isSeeded {
		t.Error("database should not be seeded yet")
	}

	// Seed with custom config
	config := &SeedConfig{
		AdminUsername:          "customadmin",
		AdminEmail:             "customadmin@test.local",
		AdminPassword:          "custompassword123",
		CreateDefaultMailboxes: true,
		DefaultMailboxAddress:  "custom@test.local",
		CreateCatchAll:         false,
	}

	if err := repo.SeedWithConfig(ctx, config); err != nil {
		t.Fatalf("failed to seed with config: %v", err)
	}

	// Verify admin was created with custom username
	admin, err := repo.Users().GetByUsername(ctx, "customadmin")
	if err != nil {
		t.Fatalf("failed to get custom admin: %v", err)
	}
	if admin.Email != "customadmin@test.local" {
		t.Errorf("expected email customadmin@test.local, got %s", admin.Email)
	}
	if admin.Role != domain.RoleAdmin {
		t.Errorf("expected role admin, got %s", admin.Role)
	}
	if admin.Status != domain.StatusActive {
		t.Errorf("expected status active, got %s", admin.Status)
	}

	// Verify mailbox was created
	mailbox, err := repo.Mailboxes().GetByAddress(ctx, "custom@test.local")
	if err != nil {
		t.Fatalf("failed to get custom mailbox: %v", err)
	}
	if mailbox.UserID != admin.ID {
		t.Errorf("mailbox should belong to admin user")
	}
	if !mailbox.IsDefault {
		t.Error("mailbox should be default")
	}
}

// TestCreateUserWithMailbox tests creating a user with default mailbox.
func TestCreateUserWithMailbox(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	seeder := repo.Seeder()

	// Create new user with mailbox
	user, mailbox, err := seeder.CreateUserWithMailbox(ctx, "newuser", "newuser@test.local", "password123")
	if err != nil {
		t.Fatalf("failed to create user with mailbox: %v", err)
	}

	// Verify user
	if user.Username != "newuser" {
		t.Errorf("expected username newuser, got %s", user.Username)
	}
	if user.Role != domain.RoleUser {
		t.Errorf("expected role user, got %s", user.Role)
	}

	// Verify mailbox
	if mailbox.UserID != user.ID {
		t.Errorf("mailbox should belong to created user")
	}
	if mailbox.Address != "newuser@localhost" {
		t.Errorf("expected address newuser@localhost, got %s", mailbox.Address)
	}
	if !mailbox.IsDefault {
		t.Error("mailbox should be default")
	}
}

// TestParseMigrationContent tests the migration content parser.
func TestParseMigrationContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectedUp  string
		expectedDn  string
	}{
		{
			name:        "no markers",
			content:     "CREATE TABLE test (id INT);",
			expectedUp:  "CREATE TABLE test (id INT);",
			expectedDn:  "",
		},
		{
			name: "with markers",
			content: `-- +migrate Up
CREATE TABLE test (id INT);
-- +migrate Down
DROP TABLE test;`,
			expectedUp: "CREATE TABLE test (id INT);",
			expectedDn: "DROP TABLE test;",
		},
		{
			name: "up only",
			content: `-- +migrate Up
CREATE TABLE test (id INT);`,
			expectedUp: "CREATE TABLE test (id INT);",
			expectedDn: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upSQL, downSQL := parseMigrationContent(tt.content)
			if upSQL != tt.expectedUp {
				t.Errorf("expected up '%s', got '%s'", tt.expectedUp, upSQL)
			}
			if downSQL != tt.expectedDn {
				t.Errorf("expected down '%s', got '%s'", tt.expectedDn, downSQL)
			}
		})
	}
}

// TestSplitStatements tests SQL statement splitting.
func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "single statement",
			sql:      "SELECT 1",
			expected: 1,
		},
		{
			name:     "two statements",
			sql:      "SELECT 1; SELECT 2",
			expected: 2,
		},
		{
			name:     "semicolon in string",
			sql:      "INSERT INTO t VALUES ('a;b');",
			expected: 1,
		},
		{
			name:     "empty",
			sql:      "",
			expected: 0,
		},
		{
			name:     "multiple with newlines",
			sql:      "CREATE TABLE t1 (id INT);\nCREATE TABLE t2 (id INT);\nCREATE TABLE t3 (id INT);",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmts := splitStatements(tt.sql)
			if len(stmts) != tt.expected {
				t.Errorf("expected %d statements, got %d: %v", tt.expected, len(stmts), stmts)
			}
		})
	}
}

// TestStatsRepository tests statistics repository operations.
func TestStatsRepository(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	stats := repo.Stats()

	// Create a user
	user := &domain.User{
		ID:           domain.ID("stats-user-1"),
		Username:     "statsuser",
		Email:        "stats@example.com",
		PasswordHash: "hash",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	if err := repo.Users().Create(ctx, user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a mailbox
	mailbox := &domain.Mailbox{
		ID:        domain.ID("stats-mailbox-1"),
		UserID:    user.ID,
		Name:      "Stats Test Inbox",
		Address:   "stats-test@example.com",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	if err := repo.Mailboxes().Create(ctx, mailbox); err != nil {
		t.Fatalf("failed to create mailbox: %v", err)
	}

	// Get initial stats
	systemStats, err := stats.GetStats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	// Verify initial user stats
	if systemStats.Users.TotalUsers < 1 {
		t.Errorf("expected at least 1 user, got %d", systemStats.Users.TotalUsers)
	}

	// Verify initial mailbox stats
	if systemStats.Mailboxes.TotalMailboxes < 1 {
		t.Errorf("expected at least 1 mailbox, got %d", systemStats.Mailboxes.TotalMailboxes)
	}

	// Create messages
	for i := 0; i < 3; i++ {
		msg := &domain.Message{
			ID:        domain.ID(fmt.Sprintf("stats-msg-%d", i)),
			MailboxID: mailbox.ID,
			From:      domain.EmailAddress{Address: "sender@example.com"},
			To:        []domain.EmailAddress{{Address: "stats-test@example.com"}},
			Subject:   fmt.Sprintf("Test Message %d", i),
			TextBody:  "Test body",
			Size:      int64(1000 + i*100),
			Status:    domain.MessageUnread,
			CreatedAt: domain.Now(),
			UpdatedAt: domain.Now(),
		}
		if i == 0 {
			msg.IsStarred = true
		}
		if i == 2 {
			msg.IsSpam = true
		}
		if err := repo.Messages().Create(ctx, msg); err != nil {
			t.Fatalf("failed to create message %d: %v", i, err)
		}
	}

	// Get message stats
	messageStats, err := stats.GetMessageAggregateStats(ctx)
	if err != nil {
		t.Fatalf("failed to get message aggregate stats: %v", err)
	}

	if messageStats.TotalMessages < 3 {
		t.Errorf("expected at least 3 messages, got %d", messageStats.TotalMessages)
	}

	if messageStats.StarredMessages < 1 {
		t.Errorf("expected at least 1 starred message, got %d", messageStats.StarredMessages)
	}

	if messageStats.SpamMessages < 1 {
		t.Errorf("expected at least 1 spam message, got %d", messageStats.SpamMessages)
	}

	if messageStats.UnreadMessages < 3 {
		t.Errorf("expected at least 3 unread messages, got %d", messageStats.UnreadMessages)
	}

	// Get mailbox-specific stats
	mailboxStats, err := stats.GetMessageStatsByMailbox(ctx, mailbox.ID)
	if err != nil {
		t.Fatalf("failed to get mailbox stats: %v", err)
	}

	if mailboxStats.Count != 3 {
		t.Errorf("expected 3 messages in mailbox, got %d", mailboxStats.Count)
	}

	if mailboxStats.UnreadCount != 3 {
		t.Errorf("expected 3 unread in mailbox, got %d", mailboxStats.UnreadCount)
	}

	if mailboxStats.StarredCount != 1 {
		t.Errorf("expected 1 starred in mailbox, got %d", mailboxStats.StarredCount)
	}

	// Get user stats
	userStats, err := stats.GetMessageStatsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to get user stats: %v", err)
	}

	if userStats.Count != 3 {
		t.Errorf("expected 3 messages for user, got %d", userStats.Count)
	}

	// Get storage stats
	storageStats, err := stats.GetStorageStats(ctx)
	if err != nil {
		t.Fatalf("failed to get storage stats: %v", err)
	}

	if storageStats.TotalSize <= 0 {
		t.Errorf("expected positive total size, got %d", storageStats.TotalSize)
	}

	// Get top senders
	topSenders, err := stats.GetTopSenders(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get top senders: %v", err)
	}

	if len(topSenders) == 0 {
		t.Error("expected at least one sender")
	} else if topSenders[0].MessageCount < 3 {
		t.Errorf("expected sender to have at least 3 messages, got %d", topSenders[0].MessageCount)
	}

	// Get counts
	starredCount, err := stats.GetStarredCount(ctx, nil)
	if err != nil {
		t.Fatalf("failed to get starred count: %v", err)
	}
	if starredCount < 1 {
		t.Errorf("expected at least 1 starred, got %d", starredCount)
	}

	spamCount, err := stats.GetSpamCount(ctx, nil)
	if err != nil {
		t.Fatalf("failed to get spam count: %v", err)
	}
	if spamCount < 1 {
		t.Errorf("expected at least 1 spam, got %d", spamCount)
	}

	unreadCount, err := stats.GetUnreadCount(ctx, nil)
	if err != nil {
		t.Fatalf("failed to get unread count: %v", err)
	}
	if unreadCount < 3 {
		t.Errorf("expected at least 3 unread, got %d", unreadCount)
	}

	// Test filtered counts
	mailboxFilter := &domain.StatsFilter{MailboxID: &mailbox.ID}
	filteredStarred, err := stats.GetStarredCount(ctx, mailboxFilter)
	if err != nil {
		t.Fatalf("failed to get filtered starred count: %v", err)
	}
	if filteredStarred != 1 {
		t.Errorf("expected 1 starred in mailbox, got %d", filteredStarred)
	}
}

// TestStatsRecalculation tests that statistics can be recalculated accurately.
func TestStatsRecalculation(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	stats := repo.Stats()

	// Create a user and mailbox
	user := &domain.User{
		ID:           domain.ID("recalc-user"),
		Username:     "recalcuser",
		Email:        "recalc@example.com",
		PasswordHash: "hash",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	repo.Users().Create(ctx, user)

	mailbox := &domain.Mailbox{
		ID:        domain.ID("recalc-mailbox"),
		UserID:    user.ID,
		Name:      "Recalc Inbox",
		Address:   "recalc@example.com",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	repo.Mailboxes().Create(ctx, mailbox)

	// Create some messages
	for i := 0; i < 5; i++ {
		msg := &domain.Message{
			ID:        domain.ID(fmt.Sprintf("recalc-msg-%d", i)),
			MailboxID: mailbox.ID,
			From:      domain.EmailAddress{Address: "sender@example.com"},
			To:        []domain.EmailAddress{{Address: "recalc@example.com"}},
			Subject:   fmt.Sprintf("Recalc Message %d", i),
			Size:      int64(500 + i*50),
			Status:    domain.MessageUnread,
			CreatedAt: domain.Now(),
			UpdatedAt: domain.Now(),
		}
		repo.Messages().Create(ctx, msg)
	}

	// Mark some as read
	repo.Messages().MarkAsRead(ctx, domain.ID("recalc-msg-0"))
	repo.Messages().MarkAsRead(ctx, domain.ID("recalc-msg-1"))

	// Verify mailbox counters are accurate
	mailboxData, _ := repo.Mailboxes().GetByID(ctx, mailbox.ID)
	if mailboxData.MessageCount != 5 {
		t.Errorf("expected message count 5, got %d", mailboxData.MessageCount)
	}
	if mailboxData.UnreadCount != 3 {
		t.Errorf("expected unread count 3, got %d", mailboxData.UnreadCount)
	}

	// Verify stats match actual data
	verifyResult, err := stats.VerifyMailboxStats(ctx)
	if err != nil {
		t.Fatalf("failed to verify mailbox stats: %v", err)
	}

	// The mailbox should pass verification (no mismatches)
	found := false
	for _, id := range verifyResult {
		if id == mailbox.ID {
			found = true
			break
		}
	}
	if found {
		t.Error("mailbox should pass verification but was flagged as mismatched")
	}

	// Recalculate all stats
	affected, err := stats.RecalculateAllMailboxStats(ctx)
	if err != nil {
		t.Fatalf("failed to recalculate stats: %v", err)
	}
	if affected == 0 {
		t.Error("expected some mailboxes to be affected by recalculation")
	}
}

// TestStatsDenormalizedCounters tests that denormalized counters stay in sync.
func TestStatsDenormalizedCounters(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Create user and mailbox
	user := &domain.User{
		ID:           domain.ID("counter-user"),
		Username:     "counteruser",
		Email:        "counter@example.com",
		PasswordHash: "hash",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	repo.Users().Create(ctx, user)

	mailbox := &domain.Mailbox{
		ID:        domain.ID("counter-mailbox"),
		UserID:    user.ID,
		Name:      "Counter Inbox",
		Address:   "counter@example.com",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	repo.Mailboxes().Create(ctx, mailbox)

	// Verify initial counters
	mb, _ := repo.Mailboxes().GetByID(ctx, mailbox.ID)
	if mb.MessageCount != 0 || mb.UnreadCount != 0 || mb.TotalSize != 0 {
		t.Error("initial counters should be zero")
	}

	// Create a message - counters should increment
	msg := &domain.Message{
		ID:        domain.ID("counter-msg-1"),
		MailboxID: mailbox.ID,
		From:      domain.EmailAddress{Address: "sender@example.com"},
		To:        []domain.EmailAddress{{Address: "counter@example.com"}},
		Subject:   "Counter Test",
		Size:      1024,
		Status:    domain.MessageUnread,
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	repo.Messages().Create(ctx, msg)

	mb, _ = repo.Mailboxes().GetByID(ctx, mailbox.ID)
	if mb.MessageCount != 1 {
		t.Errorf("expected message count 1, got %d", mb.MessageCount)
	}
	if mb.UnreadCount != 1 {
		t.Errorf("expected unread count 1, got %d", mb.UnreadCount)
	}
	if mb.TotalSize != 1024 {
		t.Errorf("expected total size 1024, got %d", mb.TotalSize)
	}

	// Mark as read - unread should decrement
	repo.Messages().MarkAsRead(ctx, msg.ID)
	mb, _ = repo.Mailboxes().GetByID(ctx, mailbox.ID)
	if mb.UnreadCount != 0 {
		t.Errorf("expected unread count 0 after mark read, got %d", mb.UnreadCount)
	}

	// Mark as unread - unread should increment
	repo.Messages().MarkAsUnread(ctx, msg.ID)
	mb, _ = repo.Mailboxes().GetByID(ctx, mailbox.ID)
	if mb.UnreadCount != 1 {
		t.Errorf("expected unread count 1 after mark unread, got %d", mb.UnreadCount)
	}

	// Delete message - all counters should decrement
	repo.Messages().Delete(ctx, msg.ID)
	mb, _ = repo.Mailboxes().GetByID(ctx, mailbox.ID)
	if mb.MessageCount != 0 {
		t.Errorf("expected message count 0 after delete, got %d", mb.MessageCount)
	}
	if mb.UnreadCount != 0 {
		t.Errorf("expected unread count 0 after delete, got %d", mb.UnreadCount)
	}
	if mb.TotalSize != 0 {
		t.Errorf("expected total size 0 after delete, got %d", mb.TotalSize)
	}
}
