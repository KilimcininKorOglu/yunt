package smtp

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// mockMailboxRepository is a mock implementation of MailboxRepository for testing.
type mockMailboxRepository struct {
	mailboxes       map[string]*domain.Mailbox
	findMatchingErr error
}

func newMockMailboxRepository() *mockMailboxRepository {
	return &mockMailboxRepository{
		mailboxes: make(map[string]*domain.Mailbox),
	}
}

func (m *mockMailboxRepository) addMailbox(mailbox *domain.Mailbox) {
	m.mailboxes[mailbox.Address] = mailbox
}

func (m *mockMailboxRepository) setFindMatchingError(err error) {
	m.findMatchingErr = err
}

func (m *mockMailboxRepository) FindMatchingMailbox(ctx context.Context, address string) (*domain.Mailbox, error) {
	if m.findMatchingErr != nil {
		return nil, m.findMatchingErr
	}

	if mailbox, ok := m.mailboxes[address]; ok {
		return mailbox, nil
	}

	// Check for catch-all
	for _, mailbox := range m.mailboxes {
		if mailbox.IsCatchAll {
			return mailbox, nil
		}
	}

	return nil, domain.NewNotFoundError("mailbox", address)
}

// Stub implementations for the rest of MailboxRepository interface
func (m *mockMailboxRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Mailbox, error) {
	return nil, nil
}
func (m *mockMailboxRepository) GetByAddress(ctx context.Context, address string) (*domain.Mailbox, error) {
	return nil, nil
}
func (m *mockMailboxRepository) GetCatchAll(ctx context.Context, domainName string) (*domain.Mailbox, error) {
	return nil, nil
}
func (m *mockMailboxRepository) GetDefault(ctx context.Context, userID domain.ID) (*domain.Mailbox, error) {
	return nil, nil
}
func (m *mockMailboxRepository) List(ctx context.Context, filter *repository.MailboxFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}
func (m *mockMailboxRepository) ListByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}
func (m *mockMailboxRepository) Create(ctx context.Context, mailbox *domain.Mailbox) error {
	return nil
}
func (m *mockMailboxRepository) Update(ctx context.Context, mailbox *domain.Mailbox) error {
	return nil
}
func (m *mockMailboxRepository) Delete(ctx context.Context, id domain.ID) error { return nil }
func (m *mockMailboxRepository) DeleteWithMessages(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *mockMailboxRepository) DeleteByUser(ctx context.Context, userID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mockMailboxRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *mockMailboxRepository) ExistsByAddress(ctx context.Context, address string) (bool, error) {
	return false, nil
}
func (m *mockMailboxRepository) Count(ctx context.Context, filter *repository.MailboxFilter) (int64, error) {
	return 0, nil
}
func (m *mockMailboxRepository) CountByUser(ctx context.Context, userID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mockMailboxRepository) SetDefault(ctx context.Context, id domain.ID) error       { return nil }
func (m *mockMailboxRepository) ClearDefault(ctx context.Context, userID domain.ID) error { return nil }
func (m *mockMailboxRepository) SetCatchAll(ctx context.Context, id domain.ID) error      { return nil }
func (m *mockMailboxRepository) ClearCatchAll(ctx context.Context, id domain.ID) error    { return nil }
func (m *mockMailboxRepository) UpdateStats(ctx context.Context, id domain.ID, stats *repository.MailboxStatsUpdate) error {
	return nil
}
func (m *mockMailboxRepository) IncrementMessageCount(ctx context.Context, id domain.ID, size int64) error {
	return nil
}
func (m *mockMailboxRepository) DecrementMessageCount(ctx context.Context, id domain.ID, size int64, wasUnread bool) error {
	return nil
}
func (m *mockMailboxRepository) UpdateUnreadCount(ctx context.Context, id domain.ID, delta int) error {
	return nil
}
func (m *mockMailboxRepository) RecalculateStats(ctx context.Context, id domain.ID) error { return nil }
func (m *mockMailboxRepository) GetStats(ctx context.Context, id domain.ID) (*domain.MailboxStats, error) {
	return nil, nil
}
func (m *mockMailboxRepository) GetStatsByUser(ctx context.Context, userID domain.ID) (*domain.MailboxStats, error) {
	return nil, nil
}
func (m *mockMailboxRepository) GetTotalStats(ctx context.Context) (*domain.MailboxStats, error) {
	return nil, nil
}
func (m *mockMailboxRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}
func (m *mockMailboxRepository) GetMailboxesWithMessages(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}
func (m *mockMailboxRepository) GetMailboxesWithUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}
func (m *mockMailboxRepository) TransferOwnership(ctx context.Context, fromUserID, toUserID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mockMailboxRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockMailboxRepository) GetDomains(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockMailboxRepository) GetMailboxesByDomain(ctx context.Context, domainName string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}

// mockMessageRepository is a mock implementation of MessageRepository for testing.
type mockMessageRepository struct {
	messages  []*domain.Message
	createErr error
}

func newMockMessageRepository() *mockMessageRepository {
	return &mockMessageRepository{
		messages: make([]*domain.Message, 0),
	}
}

func (m *mockMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.messages = append(m.messages, message)
	return nil
}

func (m *mockMessageRepository) setCreateError(err error) {
	m.createErr = err
}

func (m *mockMessageRepository) GetMessages() []*domain.Message {
	return m.messages
}

// Stub implementations for the rest of MessageRepository interface
func (m *mockMessageRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Message, error) {
	return nil, nil
}
func (m *mockMessageRepository) GetByMessageID(ctx context.Context, messageID string) (*domain.Message, error) {
	return nil, nil
}
func (m *mockMessageRepository) GetWithAttachments(ctx context.Context, id domain.ID) (*domain.Message, []*domain.Attachment, error) {
	return nil, nil, nil
}
func (m *mockMessageRepository) List(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) ListByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) ListSummaries(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	return nil, nil
}
func (m *mockMessageRepository) Update(ctx context.Context, message *domain.Message) error {
	return nil
}
func (m *mockMessageRepository) Delete(ctx context.Context, id domain.ID) error { return nil }
func (m *mockMessageRepository) DeleteByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mockMessageRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *mockMessageRepository) ExistsByMessageID(ctx context.Context, messageID string) (bool, error) {
	return false, nil
}
func (m *mockMessageRepository) Count(ctx context.Context, filter *repository.MessageFilter) (int64, error) {
	return 0, nil
}
func (m *mockMessageRepository) CountByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mockMessageRepository) CountUnreadByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mockMessageRepository) MarkAsRead(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *mockMessageRepository) MarkAsUnread(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *mockMessageRepository) MarkAllAsRead(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mockMessageRepository) ToggleStar(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *mockMessageRepository) Star(ctx context.Context, id domain.ID) error   { return nil }
func (m *mockMessageRepository) Unstar(ctx context.Context, id domain.ID) error { return nil }
func (m *mockMessageRepository) MarkAsSpam(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *mockMessageRepository) MarkAsNotSpam(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *mockMessageRepository) MoveToMailbox(ctx context.Context, id domain.ID, targetMailboxID domain.ID) error {
	return nil
}
func (m *mockMessageRepository) Search(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) SearchSummaries(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetThread(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	return nil, nil
}
func (m *mockMessageRepository) GetReplies(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	return nil, nil
}
func (m *mockMessageRepository) GetStarred(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetStarredByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetSpam(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetUnreadByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetMessagesWithAttachments(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetRecent(ctx context.Context, hours int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetByDateRange(ctx context.Context, dateRange *repository.DateRangeFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetBySender(ctx context.Context, senderAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetByRecipient(ctx context.Context, recipientAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetOldMessages(ctx context.Context, olderThanDays int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) GetLargeMessages(ctx context.Context, minSize int64, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (m *mockMessageRepository) DeleteOldMessages(ctx context.Context, olderThanDays int) (int64, error) {
	return 0, nil
}
func (m *mockMessageRepository) DeleteSpam(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockMessageRepository) BulkMarkAsRead(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockMessageRepository) BulkMarkAsUnread(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockMessageRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockMessageRepository) BulkMove(ctx context.Context, ids []domain.ID, targetMailboxID domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockMessageRepository) BulkStar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockMessageRepository) BulkUnstar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockMessageRepository) GetSizeByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mockMessageRepository) GetTotalSize(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockMessageRepository) GetDailyCounts(ctx context.Context, dateRange *repository.DateRangeFilter) ([]repository.DateCount, error) {
	return nil, nil
}
func (m *mockMessageRepository) GetSenderCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	return nil, nil
}
func (m *mockMessageRepository) GetRecipientCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	return nil, nil
}
func (m *mockMessageRepository) StoreRawBody(ctx context.Context, id domain.ID, rawBody []byte) error {
	return nil
}
func (m *mockMessageRepository) GetRawBody(ctx context.Context, id domain.ID) ([]byte, error) {
	return nil, nil
}

func createTestServer(t *testing.T, opts ...ServerOption) *Server {
	t.Helper()

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	logger := zerolog.New(io.Discard)
	cfg := &Config{
		Host:              "127.0.0.1",
		Port:              port,
		Domain:            "test.example.com",
		MaxMessageSize:    10 * 1024 * 1024,
		MaxRecipients:     100,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		GracefulTimeout:   5 * time.Second,
		AllowInsecureAuth: true,
	}

	server, err := New(cfg, logger, opts...)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return server
}

func TestBackendWithMailboxRepository(t *testing.T) {
	mailboxRepo := newMockMailboxRepository()
	mailbox := &domain.Mailbox{
		ID:      "mailbox-1",
		UserID:  "user-1",
		Name:    "Inbox",
		Address: "test@example.com",
	}
	mailboxRepo.addMailbox(mailbox)

	server := createTestServer(t, WithMailboxRepo(mailboxRepo))

	if server.Backend() == nil {
		t.Fatal("backend should not be nil")
	}

	// Test that backend has the mailbox repository
	ctx := context.Background()

	// Valid recipient should succeed
	err := server.Backend().validateRecipient(ctx, "test@example.com")
	if err != nil {
		t.Errorf("expected valid recipient to succeed, got error: %v", err)
	}

	// Invalid recipient should fail
	err = server.Backend().validateRecipient(ctx, "invalid@example.com")
	if err == nil {
		t.Error("expected invalid recipient to fail")
	}
}

func TestBackendWithoutRepository(t *testing.T) {
	// Server without repository should accept all recipients
	server := createTestServer(t)

	if server.Backend() == nil {
		t.Fatal("backend should not be nil")
	}

	ctx := context.Background()

	// Any recipient should succeed when no repository is configured
	err := server.Backend().validateRecipient(ctx, "any@example.com")
	if err != nil {
		t.Errorf("expected any recipient to succeed without repository, got error: %v", err)
	}
}

func TestBackendMaxValues(t *testing.T) {
	server := createTestServer(t)

	if server.Backend().MaxMessageSize() != 10*1024*1024 {
		t.Errorf("expected MaxMessageSize = 10MB, got %d", server.Backend().MaxMessageSize())
	}

	if server.Backend().MaxRecipients() != 100 {
		t.Errorf("expected MaxRecipients = 100, got %d", server.Backend().MaxRecipients())
	}

	if server.Backend().AuthRequired() != false {
		t.Error("expected AuthRequired = false")
	}
}

func TestBackendCatchAllMailbox(t *testing.T) {
	mailboxRepo := newMockMailboxRepository()
	catchAllMailbox := &domain.Mailbox{
		ID:         "catchall-1",
		UserID:     "user-1",
		Name:       "Catch-All",
		Address:    "*@example.com",
		IsCatchAll: true,
	}
	mailboxRepo.addMailbox(catchAllMailbox)

	server := createTestServer(t, WithMailboxRepo(mailboxRepo))
	ctx := context.Background()

	// Any address should match the catch-all
	err := server.Backend().validateRecipient(ctx, "anyuser@example.com")
	if err != nil {
		t.Errorf("expected catch-all to accept any address, got error: %v", err)
	}
}
