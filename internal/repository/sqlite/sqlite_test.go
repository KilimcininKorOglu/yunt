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

	repo, err := New(pool)
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
