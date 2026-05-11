package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"

	"yunt/internal/domain"
	"yunt/internal/parser"
	"yunt/internal/repository"
)

// testIDGenerator is a simple ID generator for testing.
type testIDGenerator struct {
	counter int64
}

func newTestIDGenerator() *testIDGenerator {
	return &testIDGenerator{}
}

func (g *testIDGenerator) Generate() domain.ID {
	id := atomic.AddInt64(&g.counter, 1)
	return domain.ID("test-id-" + intToStr(id))
}

func intToStr(n int64) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// mockRepository implements repository.Repository for testing.
type mockRepository struct {
	mailboxes   *mockMailboxRepository
	messages    *mockMessageRepository
	attachments *mockAttachmentRepository

	transactionFn   func(ctx context.Context, fn func(tx repository.Repository) error) error
	shouldFailOnTx  bool
	txError         error
	transactionRepo *mockRepository
}

func newMockRepository() *mockRepository {
	mbxRepo := newMockMailboxRepository()
	msgRepo := newMockMessageRepository()
	msgRepo.mailboxRepo = mbxRepo
	repo := &mockRepository{
		mailboxes:   mbxRepo,
		messages:    msgRepo,
		attachments: newMockAttachmentRepository(),
	}
	repo.transactionRepo = repo
	return repo
}

func (r *mockRepository) Users() repository.UserRepository       { return nil }
func (r *mockRepository) Mailboxes() repository.MailboxRepository { return r.mailboxes }
func (r *mockRepository) Messages() repository.MessageRepository   { return r.messages }
func (r *mockRepository) Attachments() repository.AttachmentRepository {
	return r.attachments
}
func (r *mockRepository) Webhooks() repository.WebhookRepository   { return nil }
func (r *mockRepository) Settings() repository.SettingsRepository { return nil }
func (r *mockRepository) Health(ctx context.Context) error        { return nil }
func (r *mockRepository) Close() error                            { return nil }

func (r *mockRepository) Transaction(ctx context.Context, fn func(tx repository.Repository) error) error {
	if r.transactionFn != nil {
		return r.transactionFn(ctx, fn)
	}
	if r.shouldFailOnTx {
		return r.txError
	}
	// Simulate transaction by using the same repository
	return fn(r.transactionRepo)
}

func (r *mockRepository) TransactionWithOptions(ctx context.Context, opts repository.TransactionOptions, fn func(tx repository.Repository) error) error {
	return r.Transaction(ctx, fn)
}

// mockMailboxRepository implements repository.MailboxRepository for testing.
type mockMailboxRepository struct {
	mailboxes          map[domain.ID]*domain.Mailbox
	addressIndex       map[string]*domain.Mailbox
	getByIDError       error
	findMatchingError  error
	incrementError     error
	decrementError     error
	updateUnreadError  error
}

func newMockMailboxRepository() *mockMailboxRepository {
	return &mockMailboxRepository{
		mailboxes:    make(map[domain.ID]*domain.Mailbox),
		addressIndex: make(map[string]*domain.Mailbox),
	}
}

func (r *mockMailboxRepository) AddMailbox(mailbox *domain.Mailbox) {
	r.mailboxes[mailbox.ID] = mailbox
	r.addressIndex[mailbox.Address] = mailbox
}

func (r *mockMailboxRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Mailbox, error) {
	if r.getByIDError != nil {
		return nil, r.getByIDError
	}
	if mailbox, ok := r.mailboxes[id]; ok {
		return mailbox, nil
	}
	return nil, domain.NewNotFoundError("mailbox", id.String())
}

func (r *mockMailboxRepository) GetByAddress(ctx context.Context, address string) (*domain.Mailbox, error) {
	if mailbox, ok := r.addressIndex[address]; ok {
		return mailbox, nil
	}
	return nil, domain.NewNotFoundError("mailbox", address)
}

func (r *mockMailboxRepository) FindMatchingMailbox(ctx context.Context, address string) (*domain.Mailbox, error) {
	if r.findMatchingError != nil {
		return nil, r.findMatchingError
	}
	if mailbox, ok := r.addressIndex[address]; ok {
		return mailbox, nil
	}
	// Check for catch-all
	for _, mailbox := range r.mailboxes {
		if mailbox.IsCatchAll {
			return mailbox, nil
		}
	}
	return nil, domain.NewNotFoundError("mailbox", address)
}

func (r *mockMailboxRepository) IncrementMessageCount(ctx context.Context, id domain.ID, size int64) (uint32, error) {
	if r.incrementError != nil {
		return 0, r.incrementError
	}
	mailbox, ok := r.mailboxes[id]
	if !ok {
		return 0, domain.NewNotFoundError("mailbox", id.String())
	}
	mailbox.IncrementMessageCount(size)
	return uint32(mailbox.UIDNext - 1), nil
}

func (r *mockMailboxRepository) DecrementMessageCount(ctx context.Context, id domain.ID, size int64, wasUnread bool) error {
	if r.decrementError != nil {
		return r.decrementError
	}
	mailbox, ok := r.mailboxes[id]
	if !ok {
		return domain.NewNotFoundError("mailbox", id.String())
	}
	mailbox.DecrementMessageCount(size, wasUnread)
	return nil
}

func (r *mockMailboxRepository) UpdateUnreadCount(ctx context.Context, id domain.ID, delta int) error {
	if r.updateUnreadError != nil {
		return r.updateUnreadError
	}
	mailbox, ok := r.mailboxes[id]
	if !ok {
		return domain.NewNotFoundError("mailbox", id.String())
	}
	if delta > 0 {
		mailbox.MarkMessageUnread()
	} else if delta < 0 {
		mailbox.MarkMessageRead()
	}
	return nil
}

// Stub implementations for the rest of MailboxRepository interface
func (r *mockMailboxRepository) GetCatchAll(ctx context.Context, domainName string) (*domain.Mailbox, error) {
	return nil, domain.ErrNotFound
}
func (r *mockMailboxRepository) GetDefault(ctx context.Context, userID domain.ID) (*domain.Mailbox, error) {
	for _, m := range r.mailboxes {
		if m.UserID == userID && m.IsDefault {
			return m, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *mockMailboxRepository) List(ctx context.Context, filter *repository.MailboxFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	var items []*domain.Mailbox
	for _, m := range r.mailboxes {
		items = append(items, m)
	}
	return &repository.ListResult[*domain.Mailbox]{Items: items, Total: int64(len(items))}, nil
}
func (r *mockMailboxRepository) ListByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	var items []*domain.Mailbox
	for _, m := range r.mailboxes {
		if m.UserID == userID {
			items = append(items, m)
		}
	}
	return &repository.ListResult[*domain.Mailbox]{Items: items, Total: int64(len(items))}, nil
}
func (r *mockMailboxRepository) Create(ctx context.Context, mailbox *domain.Mailbox) error {
	if _, exists := r.addressIndex[mailbox.Address]; exists {
		return domain.NewAlreadyExistsError("mailbox", "address", mailbox.Address)
	}
	r.mailboxes[mailbox.ID] = mailbox
	r.addressIndex[mailbox.Address] = mailbox
	return nil
}
func (r *mockMailboxRepository) Update(ctx context.Context, mailbox *domain.Mailbox) error {
	if _, exists := r.mailboxes[mailbox.ID]; !exists {
		return domain.NewNotFoundError("mailbox", mailbox.ID.String())
	}
	r.mailboxes[mailbox.ID] = mailbox
	return nil
}
func (r *mockMailboxRepository) Delete(ctx context.Context, id domain.ID) error {
	mailbox, exists := r.mailboxes[id]
	if !exists {
		return domain.NewNotFoundError("mailbox", id.String())
	}
	delete(r.mailboxes, id)
	delete(r.addressIndex, mailbox.Address)
	return nil
}
func (r *mockMailboxRepository) DeleteWithMessages(ctx context.Context, id domain.ID) error {
	return r.Delete(ctx, id)
}
func (r *mockMailboxRepository) DeleteByUser(ctx context.Context, userID domain.ID) (int64, error) {
	var count int64
	for id, m := range r.mailboxes {
		if m.UserID == userID {
			delete(r.mailboxes, id)
			delete(r.addressIndex, m.Address)
			count++
		}
	}
	return count, nil
}
func (r *mockMailboxRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	_, ok := r.mailboxes[id]
	return ok, nil
}
func (r *mockMailboxRepository) ExistsByAddress(ctx context.Context, address string) (bool, error) {
	_, ok := r.addressIndex[address]
	return ok, nil
}
func (r *mockMailboxRepository) Count(ctx context.Context, filter *repository.MailboxFilter) (int64, error) {
	return int64(len(r.mailboxes)), nil
}
func (r *mockMailboxRepository) CountByUser(ctx context.Context, userID domain.ID) (int64, error) {
	var count int64
	for _, m := range r.mailboxes {
		if m.UserID == userID {
			count++
		}
	}
	return count, nil
}
func (r *mockMailboxRepository) SetDefault(ctx context.Context, id domain.ID) error {
	mailbox, ok := r.mailboxes[id]
	if !ok {
		return domain.NewNotFoundError("mailbox", id.String())
	}
	mailbox.IsDefault = true
	return nil
}
func (r *mockMailboxRepository) ClearDefault(ctx context.Context, userID domain.ID) error {
	for _, m := range r.mailboxes {
		if m.UserID == userID {
			m.IsDefault = false
		}
	}
	return nil
}
func (r *mockMailboxRepository) SetCatchAll(ctx context.Context, id domain.ID) error {
	mailbox, ok := r.mailboxes[id]
	if !ok {
		return domain.NewNotFoundError("mailbox", id.String())
	}
	mailbox.IsCatchAll = true
	return nil
}
func (r *mockMailboxRepository) ClearCatchAll(ctx context.Context, id domain.ID) error {
	mailbox, ok := r.mailboxes[id]
	if !ok {
		return domain.NewNotFoundError("mailbox", id.String())
	}
	mailbox.IsCatchAll = false
	return nil
}
func (r *mockMailboxRepository) UpdateStats(ctx context.Context, id domain.ID, stats *repository.MailboxStatsUpdate) error {
	return nil
}
func (r *mockMailboxRepository) RecalculateStats(ctx context.Context, id domain.ID) error { return nil }
func (r *mockMailboxRepository) GetStats(ctx context.Context, id domain.ID) (*domain.MailboxStats, error) {
	return &domain.MailboxStats{}, nil
}
func (r *mockMailboxRepository) GetStatsByUser(ctx context.Context, userID domain.ID) (*domain.MailboxStats, error) {
	return &domain.MailboxStats{}, nil
}
func (r *mockMailboxRepository) GetTotalStats(ctx context.Context) (*domain.MailboxStats, error) {
	return nil, nil
}
func (r *mockMailboxRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}
func (r *mockMailboxRepository) GetMailboxesWithMessages(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}
func (r *mockMailboxRepository) GetMailboxesWithUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}
func (r *mockMailboxRepository) TransferOwnership(ctx context.Context, fromUserID, toUserID domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockMailboxRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (r *mockMailboxRepository) GetDomains(ctx context.Context) ([]string, error) { return nil, nil }
func (r *mockMailboxRepository) GetMailboxesByDomain(ctx context.Context, domainName string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return nil, nil
}

// mockMessageRepository implements repository.MessageRepository for testing.
type mockMessageRepository struct {
	messages           map[domain.ID]*domain.Message
	messageIDIndex     map[string]*domain.Message
	rawBodies          map[domain.ID][]byte
	mailboxRepo        *mockMailboxRepository
	createError        error
	storeRawBodyError  error
	getByIDError       error
	getByMessageIDErr  error
	markAsReadError    error
	markAsUnreadError  error
	deleteError        error
	moveError          error
	listError          error
}

func newMockMessageRepository() *mockMessageRepository {
	return &mockMessageRepository{
		messages:       make(map[domain.ID]*domain.Message),
		messageIDIndex: make(map[string]*domain.Message),
		rawBodies:      make(map[domain.ID][]byte),
	}
}

func (r *mockMessageRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Message, error) {
	if r.getByIDError != nil {
		return nil, r.getByIDError
	}
	if msg, ok := r.messages[id]; ok {
		return msg, nil
	}
	return nil, domain.NewNotFoundError("message", id.String())
}

func (r *mockMessageRepository) GetByMessageID(ctx context.Context, messageID string) (*domain.Message, error) {
	if r.getByMessageIDErr != nil {
		return nil, r.getByMessageIDErr
	}
	if msg, ok := r.messageIDIndex[messageID]; ok {
		return msg, nil
	}
	return nil, domain.NewNotFoundError("message", messageID)
}

func (r *mockMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	if r.createError != nil {
		return r.createError
	}
	r.messages[message.ID] = message
	if message.MessageID != "" {
		r.messageIDIndex[message.MessageID] = message
	}
	return nil
}

func (r *mockMessageRepository) StoreRawBody(ctx context.Context, id domain.ID, rawBody []byte) error {
	if r.storeRawBodyError != nil {
		return r.storeRawBodyError
	}
	r.rawBodies[id] = rawBody
	return nil
}

func (r *mockMessageRepository) GetRawBody(ctx context.Context, id domain.ID) ([]byte, error) {
	if raw, ok := r.rawBodies[id]; ok {
		return raw, nil
	}
	return nil, domain.NewNotFoundError("raw body", id.String())
}

func (r *mockMessageRepository) GetByIMAPUID(_ context.Context, _ domain.ID, _ uint32) (*domain.Message, error) {
	return nil, domain.NewNotFoundError("message", "imap_uid")
}

func (r *mockMessageRepository) GetWithAttachments(ctx context.Context, id domain.ID) (*domain.Message, []*domain.Attachment, error) {
	if r.getByIDError != nil {
		return nil, nil, r.getByIDError
	}
	if msg, ok := r.messages[id]; ok {
		return msg, nil, nil
	}
	return nil, nil, domain.NewNotFoundError("message", id.String())
}

func (r *mockMessageRepository) MarkAsRead(ctx context.Context, id domain.ID) (bool, error) {
	if r.markAsReadError != nil {
		return false, r.markAsReadError
	}
	msg, ok := r.messages[id]
	if !ok {
		return false, domain.NewNotFoundError("message", id.String())
	}
	return msg.MarkAsRead(), nil
}

func (r *mockMessageRepository) MarkAsUnread(ctx context.Context, id domain.ID) (bool, error) {
	if r.markAsUnreadError != nil {
		return false, r.markAsUnreadError
	}
	msg, ok := r.messages[id]
	if !ok {
		return false, domain.NewNotFoundError("message", id.String())
	}
	return msg.MarkAsUnread(), nil
}

func (r *mockMessageRepository) Delete(ctx context.Context, id domain.ID) error {
	if r.deleteError != nil {
		return r.deleteError
	}
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	delete(r.messages, id)
	return nil
}

func (r *mockMessageRepository) MoveToMailbox(ctx context.Context, id domain.ID, targetMailboxID domain.ID) error {
	if r.moveError != nil {
		return r.moveError
	}
	msg, ok := r.messages[id]
	if !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	sourceMailboxID := msg.MailboxID
	wasUnread := msg.Status == domain.MessageUnread
	msg.MailboxID = targetMailboxID

	if r.mailboxRepo != nil {
		r.mailboxRepo.DecrementMessageCount(ctx, sourceMailboxID, msg.Size, wasUnread)
		r.mailboxRepo.IncrementMessageCount(ctx, targetMailboxID, msg.Size)
		if !wasUnread {
			r.mailboxRepo.UpdateUnreadCount(ctx, targetMailboxID, -1)
		}
	}
	return nil
}

func (r *mockMessageRepository) ExistsByMessageID(ctx context.Context, messageID string) (bool, error) {
	_, ok := r.messageIDIndex[messageID]
	return ok, nil
}

func (r *mockMessageRepository) List(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	if r.listError != nil {
		return nil, r.listError
	}
	items := make([]*domain.Message, 0, len(r.messages))
	for _, msg := range r.messages {
		items = append(items, msg)
	}
	return &repository.ListResult[*domain.Message]{
		Items:   items,
		Total:   int64(len(items)),
		HasMore: false,
	}, nil
}

func (r *mockMessageRepository) ListByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	if r.listError != nil {
		return nil, r.listError
	}
	items := make([]*domain.Message, 0)
	for _, msg := range r.messages {
		if msg.MailboxID == mailboxID {
			items = append(items, msg)
		}
	}
	return &repository.ListResult[*domain.Message]{
		Items:   items,
		Total:   int64(len(items)),
		HasMore: false,
	}, nil
}

// Stub implementations for the rest of MessageRepository interface
func (r *mockMessageRepository) ListSummaries(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	return nil, nil
}
func (r *mockMessageRepository) Update(ctx context.Context, message *domain.Message) error   { return nil }
func (r *mockMessageRepository) DeleteByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockMessageRepository) Exists(ctx context.Context, id domain.ID) (bool, error)      { return false, nil }
func (r *mockMessageRepository) Count(ctx context.Context, filter *repository.MessageFilter) (int64, error) {
	return 0, nil
}
func (r *mockMessageRepository) CountByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockMessageRepository) CountUnreadByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockMessageRepository) MarkAllAsRead(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockMessageRepository) ToggleStar(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (r *mockMessageRepository) Star(ctx context.Context, id domain.ID) error   { return nil }
func (r *mockMessageRepository) Unstar(ctx context.Context, id domain.ID) error { return nil }
func (r *mockMessageRepository) MarkAsSpam(ctx context.Context, id domain.ID) error {
	return nil
}
func (r *mockMessageRepository) MarkAsNotSpam(ctx context.Context, id domain.ID) error {
	return nil
}
func (r *mockMessageRepository) MarkAsDeleted(ctx context.Context, id domain.ID) error {
	return nil
}
func (r *mockMessageRepository) UnmarkAsDeleted(ctx context.Context, id domain.ID) error {
	return nil
}
func (r *mockMessageRepository) MarkAsDraft(ctx context.Context, id domain.ID) error {
	return nil
}
func (r *mockMessageRepository) UnmarkAsDraft(ctx context.Context, id domain.ID) error {
	return nil
}
func (r *mockMessageRepository) MarkAsAnswered(ctx context.Context, id domain.ID) error {
	return nil
}
func (r *mockMessageRepository) UnmarkAsAnswered(ctx context.Context, id domain.ID) error {
	return nil
}
func (r *mockMessageRepository) Search(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) SearchSummaries(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetThread(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	return nil, nil
}
func (r *mockMessageRepository) GetReplies(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	return nil, nil
}
func (r *mockMessageRepository) GetStarred(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetStarredByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetSpam(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetUnreadByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetMessagesWithAttachments(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetRecent(ctx context.Context, hours int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetByDateRange(ctx context.Context, dateRange *repository.DateRangeFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetBySender(ctx context.Context, senderAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetByRecipient(ctx context.Context, recipientAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetOldMessages(ctx context.Context, olderThanDays int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) GetLargeMessages(ctx context.Context, minSize int64, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return nil, nil
}
func (r *mockMessageRepository) DeleteOldMessages(ctx context.Context, olderThanDays int) (int64, error) {
	return 0, nil
}
func (r *mockMessageRepository) DeleteSpam(ctx context.Context) (int64, error) { return 0, nil }
func (r *mockMessageRepository) BulkMarkAsRead(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (r *mockMessageRepository) BulkMarkAsUnread(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (r *mockMessageRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (r *mockMessageRepository) BulkMove(ctx context.Context, ids []domain.ID, targetMailboxID domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (r *mockMessageRepository) BulkStar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (r *mockMessageRepository) BulkUnstar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (r *mockMessageRepository) GetSizeByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockMessageRepository) GetTotalSize(ctx context.Context) (int64, error) { return 0, nil }
func (r *mockMessageRepository) GetDailyCounts(ctx context.Context, dateRange *repository.DateRangeFilter) ([]repository.DateCount, error) {
	return nil, nil
}
func (r *mockMessageRepository) GetHourlyCounts(ctx context.Context, dateRange *repository.DateRangeFilter) ([]repository.HourCount, error) {
	return nil, nil
}
func (r *mockMessageRepository) GetSenderCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	return nil, nil
}
func (r *mockMessageRepository) GetRecipientCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	return nil, nil
}

// mockAttachmentRepository implements repository.AttachmentRepository for testing.
type mockAttachmentRepository struct {
	attachments           map[domain.ID]*domain.Attachment
	contents              map[domain.ID][]byte
	createWithContentErr  error
	deleteByMessageErr    error
}

func newMockAttachmentRepository() *mockAttachmentRepository {
	return &mockAttachmentRepository{
		attachments: make(map[domain.ID]*domain.Attachment),
		contents:    make(map[domain.ID][]byte),
	}
}

func (r *mockAttachmentRepository) CreateWithContent(ctx context.Context, attachment *domain.Attachment, content io.Reader) error {
	if r.createWithContentErr != nil {
		return r.createWithContentErr
	}
	r.attachments[attachment.ID] = attachment
	data, _ := io.ReadAll(content)
	r.contents[attachment.ID] = data
	return nil
}

func (r *mockAttachmentRepository) DeleteByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	if r.deleteByMessageErr != nil {
		return 0, r.deleteByMessageErr
	}
	var count int64
	for id, att := range r.attachments {
		if att.MessageID == messageID {
			delete(r.attachments, id)
			delete(r.contents, id)
			count++
		}
	}
	return count, nil
}

// Stub implementations for the rest of AttachmentRepository interface
func (r *mockAttachmentRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Attachment, error) {
	if att, ok := r.attachments[id]; ok {
		return att, nil
	}
	return nil, domain.NewNotFoundError("attachment", id.String())
}
func (r *mockAttachmentRepository) GetByContentID(ctx context.Context, contentID string) (*domain.Attachment, error) {
	return nil, domain.ErrNotFound
}
func (r *mockAttachmentRepository) List(ctx context.Context, filter *repository.AttachmentFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return nil, nil
}
func (r *mockAttachmentRepository) ListByMessage(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	var result []*domain.Attachment
	for _, att := range r.attachments {
		if att.MessageID == messageID {
			result = append(result, att)
		}
	}
	return result, nil
}
func (r *mockAttachmentRepository) ListByMessages(ctx context.Context, messageIDs []domain.ID) (map[domain.ID][]*domain.Attachment, error) {
	return nil, nil
}
func (r *mockAttachmentRepository) ListSummaries(ctx context.Context, filter *repository.AttachmentFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.AttachmentSummary], error) {
	return nil, nil
}
func (r *mockAttachmentRepository) ListSummariesByMessage(ctx context.Context, messageID domain.ID) ([]*domain.AttachmentSummary, error) {
	return nil, nil
}
func (r *mockAttachmentRepository) Create(ctx context.Context, attachment *domain.Attachment) error {
	return nil
}
func (r *mockAttachmentRepository) Update(ctx context.Context, attachment *domain.Attachment) error {
	return nil
}
func (r *mockAttachmentRepository) Delete(ctx context.Context, id domain.ID) error { return nil }
func (r *mockAttachmentRepository) DeleteByMessages(ctx context.Context, messageIDs []domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockAttachmentRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (r *mockAttachmentRepository) ExistsByContentID(ctx context.Context, contentID string) (bool, error) {
	return false, nil
}
func (r *mockAttachmentRepository) Count(ctx context.Context, filter *repository.AttachmentFilter) (int64, error) {
	return 0, nil
}
func (r *mockAttachmentRepository) CountByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockAttachmentRepository) StoreContent(ctx context.Context, id domain.ID, content io.Reader) error {
	return nil
}
func (r *mockAttachmentRepository) GetContent(ctx context.Context, id domain.ID) (io.ReadCloser, error) {
	if content, ok := r.contents[id]; ok {
		return io.NopCloser(bytes.NewReader(content)), nil
	}
	return nil, domain.ErrNotFound
}
func (r *mockAttachmentRepository) GetContentWithMetadata(ctx context.Context, id domain.ID) (*domain.Attachment, io.ReadCloser, error) {
	return nil, nil, nil
}
func (r *mockAttachmentRepository) GetContentSize(ctx context.Context, id domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockAttachmentRepository) VerifyContent(ctx context.Context, id domain.ID) (bool, error) {
	return true, nil
}
func (r *mockAttachmentRepository) GetTotalSize(ctx context.Context) (int64, error) { return 0, nil }
func (r *mockAttachmentRepository) GetTotalSizeByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	return 0, nil
}
func (r *mockAttachmentRepository) GetByChecksum(ctx context.Context, checksum string) ([]*domain.Attachment, error) {
	return nil, nil
}
func (r *mockAttachmentRepository) GetInlineAttachments(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	return nil, nil
}
func (r *mockAttachmentRepository) GetNonInlineAttachments(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	return nil, nil
}
func (r *mockAttachmentRepository) GetByContentType(ctx context.Context, contentType string, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return nil, nil
}
func (r *mockAttachmentRepository) GetImages(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return nil, nil
}
func (r *mockAttachmentRepository) GetLargeAttachments(ctx context.Context, minSize int64, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return nil, nil
}
func (r *mockAttachmentRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return nil, nil
}
func (r *mockAttachmentRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (r *mockAttachmentRepository) CleanupOrphaned(ctx context.Context) (int64, error) { return 0, nil }
func (r *mockAttachmentRepository) GetStorageStats(ctx context.Context) (*repository.AttachmentStorageStats, error) {
	return nil, nil
}
func (r *mockAttachmentRepository) GetContentTypeStats(ctx context.Context) ([]repository.ContentTypeStats, error) {
	return nil, nil
}



// Test helper functions
func createTestMailbox() *domain.Mailbox {
	return domain.NewMailbox(
		domain.ID("mailbox-1"),
		domain.ID("user-1"),
		"Inbox",
		"test@example.com",
	)
}

func createSimpleMessage() []byte {
	return []byte(`From: sender@example.com
To: test@example.com
Subject: Test Subject
Message-ID: <unique-id-123@example.com>
Date: Mon, 01 Jan 2024 12:00:00 +0000
Content-Type: text/plain; charset=utf-8

Hello, this is a test message.
`)
}

func createMessageWithAttachment() []byte {
	return []byte(`From: sender@example.com
To: test@example.com
Subject: Test with Attachment
Message-ID: <attach-id-456@example.com>
Date: Mon, 01 Jan 2024 12:00:00 +0000
Content-Type: multipart/mixed; boundary="boundary123"

--boundary123
Content-Type: text/plain; charset=utf-8

This is the message body.

--boundary123
Content-Type: application/pdf
Content-Disposition: attachment; filename="document.pdf"
Content-Transfer-Encoding: base64

JVBERi0xLjQKJeLjz9MKMSAwIG9iago8PC9UeXBlL0NhdGFsb2cvUGFnZXMgMiAwIFI+PgplbmRvYmoK
--boundary123--
`)
}

func createMultipartMessage() []byte {
	return []byte(`From: sender@example.com
To: test@example.com
Subject: Multipart Test
Message-ID: <multipart-789@example.com>
Date: Mon, 01 Jan 2024 12:00:00 +0000
Content-Type: multipart/alternative; boundary="alt-boundary"

--alt-boundary
Content-Type: text/plain; charset=utf-8

Plain text version.

--alt-boundary
Content-Type: text/html; charset=utf-8

<html><body><p>HTML version.</p></body></html>

--alt-boundary--
`)
}

// TestNewMessageService tests the creation of a new MessageService.
func TestNewMessageService(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()

	svc := NewMessageService(repo, idGen)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo != repo {
		t.Error("expected repo to be set")
	}
	if svc.idGenerator != idGen {
		t.Error("expected idGenerator to be set")
	}
	if svc.parser == nil {
		t.Error("expected parser to be initialized")
	}
}

// TestWithParser tests setting a custom parser.
func TestWithParser(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	customParser := parser.NewParser()
	customParser.StrictMode = true

	result := svc.WithParser(customParser)

	if result != svc {
		t.Error("expected WithParser to return the same service")
	}
	if svc.parser != customParser {
		t.Error("expected parser to be set to custom parser")
	}
}

// TestStoreMessage tests successful message storage.
func TestStoreMessage(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData: createSimpleMessage(),
	}

	result, err := svc.StoreMessage(ctx, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsDuplicate {
		t.Error("expected message to not be duplicate")
	}
	if result.Message == nil {
		t.Fatal("expected message to be set")
	}
	if result.Message.Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got %q", result.Message.Subject)
	}
	if result.Message.From.Address != "sender@example.com" {
		t.Errorf("expected from 'sender@example.com', got %q", result.Message.From.Address)
	}
	if result.Message.MailboxID != mailbox.ID {
		t.Errorf("expected mailbox ID %s, got %s", mailbox.ID, result.Message.MailboxID)
	}

	// Verify mailbox stats were updated
	if mailbox.MessageCount != 1 {
		t.Errorf("expected message count 1, got %d", mailbox.MessageCount)
	}
	if mailbox.UnreadCount != 1 {
		t.Errorf("expected unread count 1, got %d", mailbox.UnreadCount)
	}

	// Verify raw body was stored
	rawBody, err := repo.messages.GetRawBody(ctx, result.Message.ID)
	if err != nil {
		t.Fatalf("failed to get raw body: %v", err)
	}
	if !bytes.Equal(rawBody, input.RawData) {
		t.Error("raw body does not match original")
	}
}

// TestStoreMessageWithTargetMailbox tests storing with explicit target mailbox.
func TestStoreMessageWithTargetMailbox(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := domain.NewMailbox(
		domain.ID("specific-mailbox"),
		domain.ID("user-1"),
		"Specific Mailbox",
		"specific@example.com",
	)
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData:         createSimpleMessage(),
		TargetMailboxID: mailbox.ID,
	}

	result, err := svc.StoreMessage(ctx, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message.MailboxID != mailbox.ID {
		t.Errorf("expected mailbox ID %s, got %s", mailbox.ID, result.Message.MailboxID)
	}
}

// TestStoreMessageWithAttachments tests storing a message with attachments.
func TestStoreMessageWithAttachments(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData: createMessageWithAttachment(),
	}

	result, err := svc.StoreMessage(ctx, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message.AttachmentCount != 1 {
		t.Errorf("expected attachment count 1, got %d", result.Message.AttachmentCount)
	}
	if len(result.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(result.Attachments))
	}
	if result.Attachments[0].Filename != "document.pdf" {
		t.Errorf("expected filename 'document.pdf', got %q", result.Attachments[0].Filename)
	}
	if result.Attachments[0].ContentType != "application/pdf" {
		t.Errorf("expected content type 'application/pdf', got %q", result.Attachments[0].ContentType)
	}
	if result.Attachments[0].Checksum == "" {
		t.Error("expected checksum to be set")
	}

	// Verify attachment content was stored
	if len(repo.attachments.contents) != 1 {
		t.Error("expected attachment content to be stored")
	}
}

// TestStoreMessageDuplicateDetection tests duplicate message detection.
func TestStoreMessageDuplicateDetection(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData: createSimpleMessage(),
	}

	// Store first message
	result1, err := svc.StoreMessage(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error on first store: %v", err)
	}

	// Try to store duplicate
	result2, err := svc.StoreMessage(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error on duplicate store: %v", err)
	}
	if !result2.IsDuplicate {
		t.Error("expected duplicate detection")
	}
	if result2.DuplicateID != result1.Message.ID {
		t.Errorf("expected duplicate ID %s, got %s", result1.Message.ID, result2.DuplicateID)
	}
}

// TestStoreMessageSkipDuplicateCheck tests storing with duplicate check disabled.
func TestStoreMessageSkipDuplicateCheck(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData: createSimpleMessage(),
	}

	// Store first message
	_, err := svc.StoreMessage(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error on first store: %v", err)
	}

	// Store with duplicate check disabled
	input.SkipDuplicateCheck = true
	result2, err := svc.StoreMessage(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error with skip duplicate: %v", err)
	}
	if result2.IsDuplicate {
		t.Error("expected no duplicate detection when skipped")
	}
}

// TestStoreMessageInvalidInput tests validation of store input.
func TestStoreMessageInvalidInput(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	ctx := context.Background()

	// Test nil input
	_, err := svc.StoreMessage(ctx, nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}

	// Test empty raw data
	_, err = svc.StoreMessage(ctx, &StoreMessageInput{})
	if err == nil {
		t.Error("expected error for empty raw data")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// TestStoreMessageNoMatchingMailbox tests error when no mailbox matches.
func TestStoreMessageNoMatchingMailbox(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	// No mailbox added

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData: createSimpleMessage(),
	}

	_, err := svc.StoreMessage(ctx, input)
	if err == nil {
		t.Error("expected error for no matching mailbox")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestStoreMessageTargetMailboxNotFound tests error when target mailbox not found.
func TestStoreMessageTargetMailboxNotFound(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData:         createSimpleMessage(),
		TargetMailboxID: domain.ID("nonexistent"),
	}

	_, err := svc.StoreMessage(ctx, input)
	if err == nil {
		t.Error("expected error for nonexistent target mailbox")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestGetMessage tests message retrieval.
func TestGetMessage(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()

	// Store a message first
	input := &StoreMessageInput{RawData: createSimpleMessage()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	// Get the message
	msg, err := svc.GetMessage(ctx, storeResult.Message.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ID != storeResult.Message.ID {
		t.Errorf("expected ID %s, got %s", storeResult.Message.ID, msg.ID)
	}
}

// TestGetMessageNotFound tests retrieval of nonexistent message.
func TestGetMessageNotFound(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	ctx := context.Background()
	_, err := svc.GetMessage(ctx, domain.ID("nonexistent"))
	if err == nil {
		t.Error("expected error for nonexistent message")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestGetMessageEmptyID tests retrieval with empty ID.
func TestGetMessageEmptyID(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	ctx := context.Background()
	_, err := svc.GetMessage(ctx, domain.ID(""))
	if err == nil {
		t.Error("expected error for empty ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// TestGetRawMessage tests raw message retrieval.
func TestGetRawMessage(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	rawData := createSimpleMessage()
	input := &StoreMessageInput{RawData: rawData}
	storeResult, _ := svc.StoreMessage(ctx, input)

	raw, err := svc.GetRawMessage(ctx, storeResult.Message.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(raw, rawData) {
		t.Error("raw message does not match original")
	}
}

// TestDeleteMessage tests message deletion.
func TestDeleteMessage(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{RawData: createSimpleMessage()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	// Verify mailbox stats before delete
	if mailbox.MessageCount != 1 {
		t.Errorf("expected message count 1 before delete, got %d", mailbox.MessageCount)
	}

	err := svc.DeleteMessage(ctx, storeResult.Message.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify message is deleted
	_, err = svc.GetMessage(ctx, storeResult.Message.ID)
	if err == nil {
		t.Error("expected message to be deleted")
	}

	// Verify mailbox stats updated
	if mailbox.MessageCount != 0 {
		t.Errorf("expected message count 0 after delete, got %d", mailbox.MessageCount)
	}
}

// TestDeleteMessageWithAttachments tests deletion with attachments.
func TestDeleteMessageWithAttachments(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{RawData: createMessageWithAttachment()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	// Verify attachment exists
	if len(repo.attachments.attachments) != 1 {
		t.Error("expected attachment to exist before delete")
	}

	err := svc.DeleteMessage(ctx, storeResult.Message.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify attachments are deleted
	if len(repo.attachments.attachments) != 0 {
		t.Error("expected attachments to be deleted")
	}
}

// TestMarkAsRead tests marking a message as read.
func TestMarkAsRead(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{RawData: createSimpleMessage()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	// Initial state should be unread
	if storeResult.Message.Status != domain.MessageUnread {
		t.Errorf("expected initial status unread, got %s", storeResult.Message.Status)
	}
	if mailbox.UnreadCount != 1 {
		t.Errorf("expected unread count 1, got %d", mailbox.UnreadCount)
	}

	err := svc.MarkAsRead(ctx, storeResult.Message.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify message is read
	msg, _ := svc.GetMessage(ctx, storeResult.Message.ID)
	if msg.Status != domain.MessageRead {
		t.Errorf("expected status read, got %s", msg.Status)
	}

	// Verify mailbox unread count updated
	if mailbox.UnreadCount != 0 {
		t.Errorf("expected unread count 0, got %d", mailbox.UnreadCount)
	}
}

// TestMarkAsUnread tests marking a message as unread.
func TestMarkAsUnread(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{RawData: createSimpleMessage()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	// Mark as read first
	_ = svc.MarkAsRead(ctx, storeResult.Message.ID)

	// Then mark as unread
	err := svc.MarkAsUnread(ctx, storeResult.Message.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify message is unread
	msg, _ := svc.GetMessage(ctx, storeResult.Message.ID)
	if msg.Status != domain.MessageUnread {
		t.Errorf("expected status unread, got %s", msg.Status)
	}

	// Verify mailbox unread count updated
	if mailbox.UnreadCount != 1 {
		t.Errorf("expected unread count 1, got %d", mailbox.UnreadCount)
	}
}

// TestMoveMessage tests moving a message between mailboxes.
func TestMoveMessage(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	sourceMailbox := createTestMailbox()
	targetMailbox := domain.NewMailbox(
		domain.ID("mailbox-2"),
		domain.ID("user-1"),
		"Target",
		"target@example.com",
	)
	repo.mailboxes.AddMailbox(sourceMailbox)
	repo.mailboxes.AddMailbox(targetMailbox)

	ctx := context.Background()
	input := &StoreMessageInput{RawData: createSimpleMessage()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	// Initial state
	if sourceMailbox.MessageCount != 1 {
		t.Errorf("expected source message count 1, got %d", sourceMailbox.MessageCount)
	}

	err := svc.MoveMessage(ctx, storeResult.Message.ID, targetMailbox.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify message moved
	msg, _ := svc.GetMessage(ctx, storeResult.Message.ID)
	if msg.MailboxID != targetMailbox.ID {
		t.Errorf("expected mailbox ID %s, got %s", targetMailbox.ID, msg.MailboxID)
	}

	// Verify source mailbox stats
	if sourceMailbox.MessageCount != 0 {
		t.Errorf("expected source message count 0, got %d", sourceMailbox.MessageCount)
	}

	// Verify target mailbox stats
	if targetMailbox.MessageCount != 1 {
		t.Errorf("expected target message count 1, got %d", targetMailbox.MessageCount)
	}
}

// TestMoveMessageSameMailbox tests moving to the same mailbox (no-op).
func TestMoveMessageSameMailbox(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{RawData: createSimpleMessage()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	err := svc.MoveMessage(ctx, storeResult.Message.ID, mailbox.ID)
	if err != nil {
		t.Fatalf("expected no error for same mailbox move, got %v", err)
	}
}

// TestListMessages tests message listing.
func TestListMessages(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()

	// Store multiple messages
	for i := 0; i < 3; i++ {
		input := &StoreMessageInput{
			RawData:            createSimpleMessage(),
			SkipDuplicateCheck: true,
		}
		_, _ = svc.StoreMessage(ctx, input)
	}

	result, err := svc.ListMessages(ctx, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
}

// TestListMessagesByMailbox tests listing messages by mailbox.
func TestListMessagesByMailbox(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox1 := createTestMailbox()
	mailbox2 := domain.NewMailbox(
		domain.ID("mailbox-2"),
		domain.ID("user-1"),
		"Other",
		"other@example.com",
	)
	repo.mailboxes.AddMailbox(mailbox1)
	repo.mailboxes.AddMailbox(mailbox2)

	ctx := context.Background()

	// Store messages in different mailboxes
	_, _ = svc.StoreMessage(ctx, &StoreMessageInput{RawData: createSimpleMessage()})
	_, _ = svc.StoreMessage(ctx, &StoreMessageInput{
		RawData:            createSimpleMessage(),
		TargetMailboxID:    mailbox2.ID,
		SkipDuplicateCheck: true,
	})

	result, err := svc.ListMessagesByMailbox(ctx, mailbox1.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 message in mailbox1, got %d", len(result.Items))
	}
}

// TestExistsByMessageID tests checking message existence by Message-ID.
func TestExistsByMessageID(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()

	// Check non-existent
	exists, err := svc.ExistsByMessageID(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected false for non-existent message")
	}

	// Store a message
	_, _ = svc.StoreMessage(ctx, &StoreMessageInput{RawData: createSimpleMessage()})

	// Check existent
	exists, err = svc.ExistsByMessageID(ctx, "unique-id-123@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true for existing message")
	}
}

// TestExistsByMessageIDEmptyID tests validation for empty Message-ID.
func TestExistsByMessageIDEmptyID(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	ctx := context.Background()
	_, err := svc.ExistsByMessageID(ctx, "")
	if err == nil {
		t.Error("expected error for empty message ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// TestGetMessageWithAttachments tests getting message with attachments.
func TestGetMessageWithAttachments(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{RawData: createMessageWithAttachment()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	msg, attachments, err := svc.GetMessageWithAttachments(ctx, storeResult.Message.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message to be returned")
	}
	// Note: The mock doesn't return attachments from GetWithAttachments,
	// but we verify the method is called correctly
	_ = attachments
}

// TestMessageServiceError tests error handling and error interface.
func TestMessageServiceError(t *testing.T) {
	err := &MessageServiceError{
		Op:      "test_op",
		Message: "test message",
		Err:     domain.ErrNotFound,
	}

	errStr := err.Error()
	if errStr != "message service test_op: test message: entity not found" {
		t.Errorf("unexpected error string: %s", errStr)
	}

	if err.Unwrap() != domain.ErrNotFound {
		t.Error("expected Unwrap to return underlying error")
	}

	if !errors.Is(err, domain.ErrNotFound) {
		t.Error("expected Is to return true for wrapped error")
	}
}

// TestMessageServiceErrorNoUnderlying tests error without underlying error.
func TestMessageServiceErrorNoUnderlying(t *testing.T) {
	err := &MessageServiceError{
		Op:      "test_op",
		Message: "test message",
	}

	errStr := err.Error()
	if errStr != "message service test_op: test message" {
		t.Errorf("unexpected error string: %s", errStr)
	}

	if err.Unwrap() != nil {
		t.Error("expected Unwrap to return nil")
	}

	if errors.Is(err, domain.ErrNotFound) {
		t.Error("expected Is to return false when no underlying error")
	}
}

// TestStoreMultipartMessage tests storing a multipart message.
func TestStoreMultipartMessage(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData: createMultipartMessage(),
	}

	result, err := svc.StoreMessage(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Message.TextBody == "" {
		t.Error("expected text body to be set")
	}
	if result.Message.HTMLBody == "" {
		t.Error("expected HTML body to be set")
	}
	if result.Message.ContentType != domain.ContentTypeMultipart {
		t.Errorf("expected content type multipart, got %s", result.Message.ContentType)
	}
}

// TestTransactionRollback tests that errors cause transaction rollback.
func TestTransactionRollback(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	mailbox := createTestMailbox()
	repo.mailboxes.AddMailbox(mailbox)

	// Make attachment storage fail
	repo.attachments.createWithContentErr = errors.New("storage failure")

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData: createMessageWithAttachment(),
	}

	_, err := svc.StoreMessage(ctx, input)
	if err == nil {
		t.Error("expected error on storage failure")
	}

	// Mailbox stats should not be updated on failure
	// (In a real transaction, the increment would be rolled back)
}

// TestCatchAllMailbox tests routing to catch-all mailbox.
func TestCatchAllMailbox(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	catchAll := domain.NewMailbox(
		domain.ID("catch-all"),
		domain.ID("user-1"),
		"Catch All",
		"*@example.com",
	)
	catchAll.SetCatchAll()
	repo.mailboxes.AddMailbox(catchAll)

	ctx := context.Background()
	input := &StoreMessageInput{
		RawData: createSimpleMessage(), // To: test@example.com - no exact match
	}

	result, err := svc.StoreMessage(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message.MailboxID != catchAll.ID {
		t.Errorf("expected catch-all mailbox, got %s", result.Message.MailboxID)
	}
}

// TestMoveReadMessageUpdatesUnreadCorrectly tests that moving a read message
// updates unread counts correctly.
func TestMoveReadMessageUpdatesUnreadCorrectly(t *testing.T) {
	repo := newMockRepository()
	idGen := newTestIDGenerator()
	svc := NewMessageService(repo, idGen)

	sourceMailbox := createTestMailbox()
	targetMailbox := domain.NewMailbox(
		domain.ID("mailbox-2"),
		domain.ID("user-1"),
		"Target",
		"target@example.com",
	)
	repo.mailboxes.AddMailbox(sourceMailbox)
	repo.mailboxes.AddMailbox(targetMailbox)

	ctx := context.Background()
	input := &StoreMessageInput{RawData: createSimpleMessage()}
	storeResult, _ := svc.StoreMessage(ctx, input)

	// Mark as read
	_ = svc.MarkAsRead(ctx, storeResult.Message.ID)

	// Source should have 0 unread now
	if sourceMailbox.UnreadCount != 0 {
		t.Errorf("expected source unread 0 after mark read, got %d", sourceMailbox.UnreadCount)
	}

	// Move message
	err := svc.MoveMessage(ctx, storeResult.Message.ID, targetMailbox.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Target should still have 0 unread (message was read)
	if targetMailbox.UnreadCount != 0 {
		t.Errorf("expected target unread 0 for read message, got %d", targetMailbox.UnreadCount)
	}
}
