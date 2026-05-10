package imap

import (
	"context"
	"testing"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// mailboxMockRepository is a mock implementation of MailboxRepository for testing.
type mailboxMockRepository struct {
	mailboxes map[domain.ID]*domain.Mailbox
	byUser    map[domain.ID][]*domain.Mailbox
}

func newMailboxMockRepository() *mailboxMockRepository {
	return &mailboxMockRepository{
		mailboxes: make(map[domain.ID]*domain.Mailbox),
		byUser:    make(map[domain.ID][]*domain.Mailbox),
	}
}

func (m *mailboxMockRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Mailbox, error) {
	if mb, ok := m.mailboxes[id]; ok {
		return mb, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mailboxMockRepository) ListByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	mailboxes := m.byUser[userID]
	return &repository.ListResult[*domain.Mailbox]{
		Items: mailboxes,
		Total: int64(len(mailboxes)),
	}, nil
}

func (m *mailboxMockRepository) Create(ctx context.Context, mailbox *domain.Mailbox) error {
	// Check if name already exists for user
	for _, mb := range m.byUser[mailbox.UserID] {
		if NormalizeMailboxName(mb.Name) == NormalizeMailboxName(mailbox.Name) {
			return domain.ErrAlreadyExists
		}
	}
	m.mailboxes[mailbox.ID] = mailbox
	m.byUser[mailbox.UserID] = append(m.byUser[mailbox.UserID], mailbox)
	return nil
}

func (m *mailboxMockRepository) Update(ctx context.Context, mailbox *domain.Mailbox) error {
	if _, ok := m.mailboxes[mailbox.ID]; !ok {
		return domain.ErrNotFound
	}
	m.mailboxes[mailbox.ID] = mailbox
	return nil
}

func (m *mailboxMockRepository) Delete(ctx context.Context, id domain.ID) error {
	mb, ok := m.mailboxes[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(m.mailboxes, id)
	// Remove from user's list
	userMailboxes := m.byUser[mb.UserID]
	for i, userMb := range userMailboxes {
		if userMb.ID == id {
			m.byUser[mb.UserID] = append(userMailboxes[:i], userMailboxes[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mailboxMockRepository) DeleteWithMessages(ctx context.Context, id domain.ID) error {
	return m.Delete(ctx, id)
}

// Stub implementations for unused methods
func (m *mailboxMockRepository) GetByAddress(ctx context.Context, address string) (*domain.Mailbox, error) {
	return nil, domain.ErrNotFound
}
func (m *mailboxMockRepository) GetCatchAll(ctx context.Context, domainName string) (*domain.Mailbox, error) {
	return nil, domain.ErrNotFound
}
func (m *mailboxMockRepository) GetDefault(ctx context.Context, userID domain.ID) (*domain.Mailbox, error) {
	return nil, domain.ErrNotFound
}
func (m *mailboxMockRepository) List(ctx context.Context, filter *repository.MailboxFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}
func (m *mailboxMockRepository) DeleteByUser(ctx context.Context, userID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mailboxMockRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	_, ok := m.mailboxes[id]
	return ok, nil
}
func (m *mailboxMockRepository) ExistsByAddress(ctx context.Context, address string) (bool, error) {
	return false, nil
}
func (m *mailboxMockRepository) Count(ctx context.Context, filter *repository.MailboxFilter) (int64, error) {
	return 0, nil
}
func (m *mailboxMockRepository) CountByUser(ctx context.Context, userID domain.ID) (int64, error) {
	return int64(len(m.byUser[userID])), nil
}
func (m *mailboxMockRepository) SetDefault(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *mailboxMockRepository) ClearDefault(ctx context.Context, userID domain.ID) error {
	return nil
}
func (m *mailboxMockRepository) SetCatchAll(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *mailboxMockRepository) ClearCatchAll(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *mailboxMockRepository) UpdateStats(ctx context.Context, id domain.ID, stats *repository.MailboxStatsUpdate) error {
	return nil
}
func (m *mailboxMockRepository) IncrementMessageCount(ctx context.Context, id domain.ID, size int64) error {
	return nil
}
func (m *mailboxMockRepository) DecrementMessageCount(ctx context.Context, id domain.ID, size int64, wasUnread bool) error {
	return nil
}
func (m *mailboxMockRepository) UpdateUnreadCount(ctx context.Context, id domain.ID, delta int) error {
	return nil
}
func (m *mailboxMockRepository) RecalculateStats(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *mailboxMockRepository) GetStats(ctx context.Context, id domain.ID) (*domain.MailboxStats, error) {
	return nil, nil
}
func (m *mailboxMockRepository) GetStatsByUser(ctx context.Context, userID domain.ID) (*domain.MailboxStats, error) {
	return nil, nil
}
func (m *mailboxMockRepository) GetTotalStats(ctx context.Context) (*domain.MailboxStats, error) {
	return nil, nil
}
func (m *mailboxMockRepository) FindMatchingMailbox(ctx context.Context, address string) (*domain.Mailbox, error) {
	return nil, domain.ErrNotFound
}
func (m *mailboxMockRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}
func (m *mailboxMockRepository) GetMailboxesWithMessages(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}
func (m *mailboxMockRepository) GetMailboxesWithUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}
func (m *mailboxMockRepository) TransferOwnership(ctx context.Context, fromUserID, toUserID domain.ID) (int64, error) {
	return 0, nil
}
func (m *mailboxMockRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mailboxMockRepository) GetDomains(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *mailboxMockRepository) GetMailboxesByDomain(ctx context.Context, domainName string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}

// mailboxOpsMockRepository wraps mock repositories for testing.
type mailboxOpsMockRepository struct {
	mailboxRepo *mailboxMockRepository
	messageRepo *messageMockRepository
}

func newMailboxOpsMockRepository() *mailboxOpsMockRepository {
	return &mailboxOpsMockRepository{
		mailboxRepo: newMailboxMockRepository(),
		messageRepo: newMessageMockRepository(),
	}
}

func (r *mailboxOpsMockRepository) Mailboxes() repository.MailboxRepository {
	return r.mailboxRepo
}

func (r *mailboxOpsMockRepository) Messages() repository.MessageRepository {
	return r.messageRepo
}

func (r *mailboxOpsMockRepository) Users() repository.UserRepository             { return nil }
func (r *mailboxOpsMockRepository) Attachments() repository.AttachmentRepository { return nil }
func (r *mailboxOpsMockRepository) Webhooks() repository.WebhookRepository       { return nil }
func (r *mailboxOpsMockRepository) Settings() repository.SettingsRepository      { return nil }
func (r *mailboxOpsMockRepository) Transaction(ctx context.Context, fn func(tx repository.Repository) error) error {
	return fn(r)
}
func (r *mailboxOpsMockRepository) TransactionWithOptions(ctx context.Context, opts repository.TransactionOptions, fn func(tx repository.Repository) error) error {
	return fn(r)
}
func (r *mailboxOpsMockRepository) Health(ctx context.Context) error { return nil }
func (r *mailboxOpsMockRepository) Close() error                     { return nil }

// messageMockRepository for message operations
type messageMockRepository struct {
	messages map[domain.ID]*domain.Message
}

func newMessageMockRepository() *messageMockRepository {
	return &messageMockRepository{
		messages: make(map[domain.ID]*domain.Message),
	}
}

func (m *messageMockRepository) ListByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	var msgs []*domain.Message
	for _, msg := range m.messages {
		if msg.MailboxID == mailboxID {
			msgs = append(msgs, msg)
		}
	}
	return &repository.ListResult[*domain.Message]{
		Items: msgs,
		Total: int64(len(msgs)),
	}, nil
}

func (m *messageMockRepository) MoveToMailbox(ctx context.Context, id domain.ID, targetMailboxID domain.ID) error {
	if msg, ok := m.messages[id]; ok {
		msg.MailboxID = targetMailboxID
		return nil
	}
	return domain.ErrNotFound
}

// Stub implementations for unused message methods
func (m *messageMockRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Message, error) {
	return nil, domain.ErrNotFound
}
func (m *messageMockRepository) GetByMessageID(ctx context.Context, messageID string) (*domain.Message, error) {
	return nil, domain.ErrNotFound
}
func (m *messageMockRepository) GetWithAttachments(ctx context.Context, id domain.ID) (*domain.Message, []*domain.Attachment, error) {
	return nil, nil, domain.ErrNotFound
}
func (m *messageMockRepository) List(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) ListSummaries(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	return &repository.ListResult[*domain.MessageSummary]{}, nil
}
func (m *messageMockRepository) Create(ctx context.Context, message *domain.Message) error {
	return nil
}
func (m *messageMockRepository) Update(ctx context.Context, message *domain.Message) error {
	return nil
}
func (m *messageMockRepository) Delete(ctx context.Context, id domain.ID) error { return nil }
func (m *messageMockRepository) DeleteByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *messageMockRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *messageMockRepository) ExistsByMessageID(ctx context.Context, messageID string) (bool, error) {
	return false, nil
}
func (m *messageMockRepository) Count(ctx context.Context, filter *repository.MessageFilter) (int64, error) {
	return 0, nil
}
func (m *messageMockRepository) CountByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *messageMockRepository) CountUnreadByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *messageMockRepository) MarkAsRead(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *messageMockRepository) MarkAsUnread(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *messageMockRepository) MarkAllAsRead(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *messageMockRepository) ToggleStar(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *messageMockRepository) Star(ctx context.Context, id domain.ID) error   { return nil }
func (m *messageMockRepository) Unstar(ctx context.Context, id domain.ID) error { return nil }
func (m *messageMockRepository) MarkAsSpam(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *messageMockRepository) MarkAsNotSpam(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *messageMockRepository) MarkAsDeleted(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *messageMockRepository) UnmarkAsDeleted(ctx context.Context, id domain.ID) error {
	return nil
}
func (m *messageMockRepository) Search(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) SearchSummaries(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	return &repository.ListResult[*domain.MessageSummary]{}, nil
}
func (m *messageMockRepository) GetThread(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	return nil, nil
}
func (m *messageMockRepository) GetReplies(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	return nil, nil
}
func (m *messageMockRepository) GetStarred(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetStarredByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetSpam(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetUnreadByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetMessagesWithAttachments(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetRecent(ctx context.Context, hours int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetByDateRange(ctx context.Context, dateRange *repository.DateRangeFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetBySender(ctx context.Context, senderAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetByRecipient(ctx context.Context, recipientAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetOldMessages(ctx context.Context, olderThanDays int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) GetLargeMessages(ctx context.Context, minSize int64, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (m *messageMockRepository) DeleteOldMessages(ctx context.Context, olderThanDays int) (int64, error) {
	return 0, nil
}
func (m *messageMockRepository) DeleteSpam(ctx context.Context) (int64, error) { return 0, nil }
func (m *messageMockRepository) BulkMarkAsRead(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *messageMockRepository) BulkMarkAsUnread(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *messageMockRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *messageMockRepository) BulkMove(ctx context.Context, ids []domain.ID, targetMailboxID domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *messageMockRepository) BulkStar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *messageMockRepository) BulkUnstar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *messageMockRepository) GetSizeByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (m *messageMockRepository) GetTotalSize(ctx context.Context) (int64, error) { return 0, nil }
func (m *messageMockRepository) GetDailyCounts(ctx context.Context, dateRange *repository.DateRangeFilter) ([]repository.DateCount, error) {
	return nil, nil
}
func (m *messageMockRepository) GetSenderCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	return nil, nil
}
func (m *messageMockRepository) GetRecipientCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	return nil, nil
}
func (m *messageMockRepository) StoreRawBody(ctx context.Context, id domain.ID, rawBody []byte) error {
	return nil
}
func (m *messageMockRepository) GetRawBody(ctx context.Context, id domain.ID) ([]byte, error) {
	return nil, domain.ErrNotFound
}

// Tests

func TestMailboxOperator_Create(t *testing.T) {
	ctx := context.Background()
	userID := domain.ID("user-123")
	repo := newMailboxOpsMockRepository()
	op := NewMailboxOperator(repo, userID)

	t.Run("create custom mailbox", func(t *testing.T) {
		err := op.Create(ctx, "Work", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify mailbox was created
		mailboxes := repo.mailboxRepo.byUser[userID]
		found := false
		for _, mb := range mailboxes {
			if mb.Name == "Work" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected mailbox 'Work' to be created")
		}
	})

	t.Run("cannot create system mailbox", func(t *testing.T) {
		err := op.Create(ctx, "INBOX", nil)
		if err == nil {
			t.Error("Expected error when creating system mailbox")
		}
		imapErr, ok := err.(*imap.Error)
		if !ok {
			t.Errorf("Expected imap.Error, got %T", err)
		} else if imapErr.Type != imap.StatusResponseTypeNo {
			t.Errorf("Expected NO response, got %v", imapErr.Type)
		}
	})

	t.Run("cannot create duplicate mailbox", func(t *testing.T) {
		// First creation should succeed
		err := op.Create(ctx, "Duplicate", nil)
		if err != nil {
			t.Errorf("First creation failed: %v", err)
		}

		// Second creation should fail
		err = op.Create(ctx, "Duplicate", nil)
		if err == nil {
			t.Error("Expected error when creating duplicate mailbox")
		}
		imapErr, ok := err.(*imap.Error)
		if !ok {
			t.Errorf("Expected imap.Error, got %T", err)
		} else if imapErr.Code != imap.ResponseCodeAlreadyExists {
			t.Errorf("Expected ALREADYEXISTS code, got %v", imapErr.Code)
		}
	})

	t.Run("invalid mailbox name", func(t *testing.T) {
		err := op.Create(ctx, "Invalid*Name", nil)
		if err == nil {
			t.Error("Expected error for invalid mailbox name")
		}
	})
}

func TestMailboxOperator_Delete(t *testing.T) {
	ctx := context.Background()
	userID := domain.ID("user-123")

	t.Run("delete custom mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		// Create a mailbox first
		err := op.Create(ctx, "ToDelete", nil)
		if err != nil {
			t.Fatalf("Failed to create mailbox: %v", err)
		}

		// Delete it
		err = op.Delete(ctx, "ToDelete")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify it's gone
		mailboxes := repo.mailboxRepo.byUser[userID]
		for _, mb := range mailboxes {
			if mb.Name == "ToDelete" {
				t.Error("Mailbox should have been deleted")
			}
		}
	})

	t.Run("cannot delete system mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		// Create INBOX (simulating system mailbox)
		inbox := domain.NewMailbox("inbox-id", userID, "INBOX", "")
		repo.mailboxRepo.mailboxes[inbox.ID] = inbox
		repo.mailboxRepo.byUser[userID] = []*domain.Mailbox{inbox}

		err := op.Delete(ctx, "INBOX")
		if err == nil {
			t.Error("Expected error when deleting system mailbox")
		}
	})

	t.Run("delete non-existent mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		err := op.Delete(ctx, "NonExistent")
		if err == nil {
			t.Error("Expected error when deleting non-existent mailbox")
		}
		imapErr, ok := err.(*imap.Error)
		if !ok {
			t.Errorf("Expected imap.Error, got %T", err)
		} else if imapErr.Code != imap.ResponseCodeNonExistent {
			t.Errorf("Expected NONEXISTENT code, got %v", imapErr.Code)
		}
	})
}

func TestMailboxOperator_Select(t *testing.T) {
	ctx := context.Background()
	userID := domain.ID("user-123")

	t.Run("select existing mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		// Create a mailbox
		inbox := domain.NewMailbox("inbox-id", userID, "INBOX", "")
		inbox.MessageCount = 10
		inbox.UnreadCount = 3
		repo.mailboxRepo.mailboxes[inbox.ID] = inbox
		repo.mailboxRepo.byUser[userID] = []*domain.Mailbox{inbox}

		mailbox, selectData, err := op.Select(ctx, "INBOX", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if mailbox == nil {
			t.Error("Expected mailbox to be returned")
		}
		if selectData == nil {
			t.Error("Expected select data to be returned")
		}
		if selectData.NumMessages != 10 {
			t.Errorf("Expected 10 messages, got %d", selectData.NumMessages)
		}
	})

	t.Run("select non-existent mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		_, _, err := op.Select(ctx, "NonExistent", nil)
		if err == nil {
			t.Error("Expected error when selecting non-existent mailbox")
		}
		imapErr, ok := err.(*imap.Error)
		if !ok {
			t.Errorf("Expected imap.Error, got %T", err)
		} else if imapErr.Code != imap.ResponseCodeNonExistent {
			t.Errorf("Expected NONEXISTENT code, got %v", imapErr.Code)
		}
	})
}

func TestMailboxOperator_Status(t *testing.T) {
	ctx := context.Background()
	userID := domain.ID("user-123")

	t.Run("get status of existing mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		// Create a mailbox
		work := domain.NewMailbox("work-id", userID, "Work", "")
		work.MessageCount = 25
		work.UnreadCount = 5
		repo.mailboxRepo.mailboxes[work.ID] = work
		repo.mailboxRepo.byUser[userID] = []*domain.Mailbox{work}

		statusData, err := op.Status(ctx, "Work", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if statusData == nil {
			t.Error("Expected status data to be returned")
		}
		if statusData.Mailbox != "Work" {
			t.Errorf("Expected mailbox name Work, got %s", statusData.Mailbox)
		}
		if statusData.NumMessages == nil || *statusData.NumMessages != 25 {
			t.Errorf("Expected 25 messages")
		}
		if statusData.NumUnseen == nil || *statusData.NumUnseen != 5 {
			t.Errorf("Expected 5 unseen")
		}
	})

	t.Run("status of non-existent mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		_, err := op.Status(ctx, "NonExistent", nil)
		if err == nil {
			t.Error("Expected error for non-existent mailbox")
		}
	})
}

func TestMailboxOperator_Rename(t *testing.T) {
	ctx := context.Background()
	userID := domain.ID("user-123")

	t.Run("rename custom mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		// Create a mailbox
		old := domain.NewMailbox("old-id", userID, "OldName", "")
		repo.mailboxRepo.mailboxes[old.ID] = old
		repo.mailboxRepo.byUser[userID] = []*domain.Mailbox{old}

		err := op.Rename(ctx, "OldName", "NewName", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify the name was changed
		if old.Name != "NewName" {
			t.Errorf("Expected name to be NewName, got %s", old.Name)
		}
	})

	t.Run("cannot rename to system mailbox name", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		// Create a mailbox
		custom := domain.NewMailbox("custom-id", userID, "Custom", "")
		repo.mailboxRepo.mailboxes[custom.ID] = custom
		repo.mailboxRepo.byUser[userID] = []*domain.Mailbox{custom}

		err := op.Rename(ctx, "Custom", "Sent", nil)
		if err == nil {
			t.Error("Expected error when renaming to system mailbox name")
		}
	})

	t.Run("cannot rename system mailbox", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		// Create Sent (system mailbox)
		sent := domain.NewMailbox("sent-id", userID, "Sent", "")
		repo.mailboxRepo.mailboxes[sent.ID] = sent
		repo.mailboxRepo.byUser[userID] = []*domain.Mailbox{sent}

		err := op.Rename(ctx, "Sent", "OldMail", nil)
		if err == nil {
			t.Error("Expected error when renaming system mailbox")
		}
	})

	t.Run("rename to already existing name", func(t *testing.T) {
		repo := newMailboxOpsMockRepository()
		op := NewMailboxOperator(repo, userID)

		// Create two mailboxes
		mb1 := domain.NewMailbox("mb1-id", userID, "Mailbox1", "")
		mb2 := domain.NewMailbox("mb2-id", userID, "Mailbox2", "")
		repo.mailboxRepo.mailboxes[mb1.ID] = mb1
		repo.mailboxRepo.mailboxes[mb2.ID] = mb2
		repo.mailboxRepo.byUser[userID] = []*domain.Mailbox{mb1, mb2}

		err := op.Rename(ctx, "Mailbox1", "Mailbox2", nil)
		if err == nil {
			t.Error("Expected error when renaming to existing name")
		}
		imapErr, ok := err.(*imap.Error)
		if !ok {
			t.Errorf("Expected imap.Error, got %T", err)
		} else if imapErr.Code != imap.ResponseCodeAlreadyExists {
			t.Errorf("Expected ALREADYEXISTS code, got %v", imapErr.Code)
		}
	})
}
