package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"yunt/internal/api/middleware"
	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// --- mockFullRepo: implements repository.Repository ---

type mockFullRepo struct {
	users    *mockUsersRepo
	messages *mockMsgRepo
	mboxes   *mockMbxRepo
	webhooks *mockWhRepo
	attachs  *mockAttRepo
	settings *mockSettRepo
}

var _ repository.Repository = (*mockFullRepo)(nil)

func newMockFullRepo() *mockFullRepo {
	return &mockFullRepo{
		users:    newMockUsersRepo(),
		messages: newMockMsgRepo(),
		mboxes:   newMockMbxRepo(),
		webhooks: newMockWhRepo(),
		attachs:  newMockAttRepo(),
		settings: &mockSettRepo{},
	}
}

func (r *mockFullRepo) Users() repository.UserRepository           { return r.users }
func (r *mockFullRepo) Messages() repository.MessageRepository     { return r.messages }
func (r *mockFullRepo) Mailboxes() repository.MailboxRepository    { return r.mboxes }
func (r *mockFullRepo) Webhooks() repository.WebhookRepository     { return r.webhooks }
func (r *mockFullRepo) Attachments() repository.AttachmentRepository { return r.attachs }
func (r *mockFullRepo) Settings() repository.SettingsRepository     { return r.settings }
func (r *mockFullRepo) JMAP() repository.JMAPRepository             { return nil }
func (r *mockFullRepo) Health(_ context.Context) error              { return nil }
func (r *mockFullRepo) Close() error                                { return nil }
func (r *mockFullRepo) Transaction(_ context.Context, fn func(tx repository.Repository) error) error {
	return fn(r)
}
func (r *mockFullRepo) TransactionWithOptions(_ context.Context, _ repository.TransactionOptions, fn func(tx repository.Repository) error) error {
	return fn(r)
}

// --- setupFullTest ---

type fullTestEnv struct {
	echo       *echo.Echo
	repo       *mockFullRepo
	authSvc    *service.AuthService
	msgSvc     *service.MessageService
	mbxSvc     *service.MailboxService
	webhookSvc *service.WebhookService
	userSvc    *service.UserService
}

func setupFullTest() *fullTestEnv {
	repo := newMockFullRepo()

	cfg := config.AuthConfig{
		JWTSecret:         "test-secret-key-for-testing-purposes",
		JWTExpiration:     15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
		BCryptCost:        bcrypt.MinCost,
	}

	sessionStore := service.NewInMemorySessionStore()
	authSvc := service.NewAuthService(cfg, repo.users, sessionStore)
	userSvc := service.NewUserService(cfg, repo.users)
	idGen := &mockIDGen{}
	msgSvc := service.NewMessageService(repo, idGen)
	mbxSvc := service.NewMailboxService(repo, idGen)
	webhookSvc := service.NewWebhookService(repo, idGen)

	e := echo.New()
	v1 := e.Group("/api/v1")

	authHandler := NewAuthHandler(authSvc)
	authHandler.RegisterRoutes(v1)

	usersHandler := NewUsersHandler(userSvc, authSvc)
	usersHandler.RegisterRoutes(v1, authSvc)

	msgHandler := NewMessageHandler(msgSvc, mbxSvc, authSvc)
	msgHandler.RegisterRoutes(v1)

	mbxHandler := NewMailboxHandler(mbxSvc, authSvc)
	mbxHandler.RegisterRoutes(v1)

	whHandler := NewWebhookHandler(webhookSvc, authSvc)
	whHandler.RegisterRoutes(v1)

	searchHandler := NewSearchHandler(msgSvc, authSvc)
	searchHandler.RegisterRoutes(v1)

	attachHandler := NewAttachmentHandler(msgSvc, authSvc)
	attachHandler.RegisterRoutes(v1)

	systemHandler := NewSystemHandler(SystemHandlerConfig{
		Repo:           repo,
		AuthService:    authSvc,
		MessageService: msgSvc,
		Config:         &config.Config{},
		Version:        "test-v1",
	})
	systemHandler.RegisterRoutes(v1)

	return &fullTestEnv{
		echo:       e,
		repo:       repo,
		authSvc:    authSvc,
		msgSvc:     msgSvc,
		mbxSvc:     mbxSvc,
		webhookSvc: webhookSvc,
		userSvc:    userSvc,
	}
}

func (env *fullTestEnv) loginAdmin(t *testing.T) string {
	t.Helper()
	admin := makeAdmin("admin-ft", "ftadmin", "password123")
	env.repo.users.addUser(admin)
	defaultMbx := createTestMailbox("mbx-default", admin.ID, "inbox@localhost")
	defaultMbx.IsDefault = true
	env.repo.mboxes.add(defaultMbx)
	return loginForToken(t, env.echo, "ftadmin", "password123")
}

// --- Factory helpers ---

func createTestMessage(id string, mailboxID domain.ID) *domain.Message {
	return &domain.Message{
		ID:        domain.ID(id),
		MailboxID: mailboxID,
		MessageID: id + "@test.local",
		Subject:   "Test Subject " + id,
		From:      domain.EmailAddress{Name: "Sender", Address: "sender@test.com"},
		To:        []domain.EmailAddress{{Address: "inbox@localhost"}},
		TextBody:  "Text body for " + id,
		HTMLBody:  "<p>HTML body for " + id + "</p>",
		RawBody:   []byte("raw body"),
		Size:      1024,
		Status:    domain.MessageUnread,
		CreatedAt: domain.Now(),
	}
}

func createTestMailbox(id string, userID domain.ID, address string) *domain.Mailbox {
	return &domain.Mailbox{
		ID:      domain.ID(id),
		UserID:  userID,
		Name:    "Mailbox " + id,
		Address: address,
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
}

func createTestWebhook(id string, userID domain.ID) *domain.Webhook {
	return &domain.Webhook{
		ID:     domain.ID(id),
		UserID: userID,
		Name:   "Webhook " + id,
		URL:    "https://example.com/webhook/" + id,
		Events: []domain.WebhookEvent{domain.WebhookEventMessageReceived},
		Status: domain.WebhookStatusActive,
		Secret: "secret-" + id,
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
}

func createTestAttachment(id string, messageID domain.ID) *domain.Attachment {
	return &domain.Attachment{
		ID:          domain.ID(id),
		MessageID:   messageID,
		Filename:    "file-" + id + ".txt",
		ContentType: "text/plain",
		Size:        256,
		CreatedAt:   domain.Now(),
	}
}

// --- mockMsgRepo: implements repository.MessageRepository ---

type mockMsgRepo struct {
	messages map[domain.ID]*domain.Message
	rawBodies map[domain.ID][]byte
}

var _ repository.MessageRepository = (*mockMsgRepo)(nil)

func newMockMsgRepo() *mockMsgRepo {
	return &mockMsgRepo{
		messages:  make(map[domain.ID]*domain.Message),
		rawBodies: make(map[domain.ID][]byte),
	}
}

func (r *mockMsgRepo) add(m *domain.Message) {
	r.messages[m.ID] = m
	if len(m.RawBody) > 0 {
		r.rawBodies[m.ID] = m.RawBody
	}
}

func (r *mockMsgRepo) GetByID(_ context.Context, id domain.ID) (*domain.Message, error) {
	if m, ok := r.messages[id]; ok {
		return m, nil
	}
	return nil, domain.NewNotFoundError("message", id.String())
}

func (r *mockMsgRepo) GetByMessageID(_ context.Context, msgID string) (*domain.Message, error) {
	for _, m := range r.messages {
		if m.MessageID == msgID {
			return m, nil
		}
	}
	return nil, domain.NewNotFoundError("message", msgID)
}

func (r *mockMsgRepo) GetWithAttachments(_ context.Context, id domain.ID) (*domain.Message, []*domain.Attachment, error) {
	if m, ok := r.messages[id]; ok {
		return m, nil, nil
	}
	return nil, nil, domain.NewNotFoundError("message", id.String())
}

func (r *mockMsgRepo) List(_ context.Context, filter *repository.MessageFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	var items []*domain.Message
	for _, m := range r.messages {
		if filter != nil && filter.MailboxID != nil && m.MailboxID != *filter.MailboxID {
			continue
		}
		if filter != nil && len(filter.MailboxIDs) > 0 {
			found := false
			for _, mid := range filter.MailboxIDs {
				if m.MailboxID == mid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		items = append(items, m)
	}
	return &repository.ListResult[*domain.Message]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockMsgRepo) ListByMailbox(_ context.Context, mailboxID domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	var items []*domain.Message
	for _, m := range r.messages {
		if m.MailboxID == mailboxID {
			items = append(items, m)
		}
	}
	return &repository.ListResult[*domain.Message]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockMsgRepo) ListSummaries(_ context.Context, _ *repository.MessageFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	return &repository.ListResult[*domain.MessageSummary]{}, nil
}

func (r *mockMsgRepo) Create(_ context.Context, m *domain.Message) error {
	r.messages[m.ID] = m
	return nil
}

func (r *mockMsgRepo) Update(_ context.Context, m *domain.Message) error {
	r.messages[m.ID] = m
	return nil
}

func (r *mockMsgRepo) Delete(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	delete(r.messages, id)
	return nil
}

func (r *mockMsgRepo) DeleteByMailbox(_ context.Context, mailboxID domain.ID) (int64, error) {
	var count int64
	for id, m := range r.messages {
		if m.MailboxID == mailboxID {
			delete(r.messages, id)
			count++
		}
	}
	return count, nil
}

func (r *mockMsgRepo) Exists(_ context.Context, id domain.ID) (bool, error) {
	_, ok := r.messages[id]
	return ok, nil
}

func (r *mockMsgRepo) ExistsByMessageID(_ context.Context, msgID string) (bool, error) {
	for _, m := range r.messages {
		if m.MessageID == msgID {
			return true, nil
		}
	}
	return false, nil
}

func (r *mockMsgRepo) Count(_ context.Context, filter *repository.MessageFilter) (int64, error) {
	if filter == nil || filter.IsEmpty() {
		return int64(len(r.messages)), nil
	}
	var count int64
	for _, m := range r.messages {
		if filter.MailboxID != nil && m.MailboxID != *filter.MailboxID {
			continue
		}
		count++
	}
	return count, nil
}

func (r *mockMsgRepo) CountByMailbox(_ context.Context, mailboxID domain.ID) (int64, error) {
	var count int64
	for _, m := range r.messages {
		if m.MailboxID == mailboxID {
			count++
		}
	}
	return count, nil
}

func (r *mockMsgRepo) CountUnreadByMailbox(_ context.Context, _ domain.ID) (int64, error) { return 0, nil }

func (r *mockMsgRepo) MarkAsRead(_ context.Context, id domain.ID) (bool, error) {
	if m, ok := r.messages[id]; ok {
		m.Status = domain.MessageRead
		return true, nil
	}
	return false, domain.NewNotFoundError("message", id.String())
}

func (r *mockMsgRepo) MarkAsUnread(_ context.Context, id domain.ID) (bool, error) {
	if m, ok := r.messages[id]; ok {
		m.Status = domain.MessageUnread
		return true, nil
	}
	return false, domain.NewNotFoundError("message", id.String())
}

func (r *mockMsgRepo) MarkAllAsRead(_ context.Context, _ domain.ID) (int64, error)     { return 0, nil }
func (r *mockMsgRepo) ToggleStar(_ context.Context, _ domain.ID) (bool, error)          { return false, nil }

func (r *mockMsgRepo) Star(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) Unstar(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) MarkAsSpam(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) MarkAsNotSpam(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) MarkAsDeleted(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) UnmarkAsDeleted(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) MarkAsDraft(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) UnmarkAsDraft(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) MarkAsAnswered(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) UnmarkAsAnswered(_ context.Context, id domain.ID) error {
	if _, ok := r.messages[id]; !ok {
		return domain.NewNotFoundError("message", id.String())
	}
	return nil
}

func (r *mockMsgRepo) MoveToMailbox(_ context.Context, id domain.ID, targetID domain.ID) error {
	if m, ok := r.messages[id]; ok {
		m.MailboxID = targetID
		return nil
	}
	return domain.NewNotFoundError("message", id.String())
}

func (r *mockMsgRepo) Search(_ context.Context, _ *repository.SearchOptions, filter *repository.MessageFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return r.List(context.Background(), filter, nil)
}

func (r *mockMsgRepo) GetThread(_ context.Context, _ domain.ID) ([]*domain.Message, error)   { return nil, nil }
func (r *mockMsgRepo) GetReplies(_ context.Context, _ domain.ID) ([]*domain.Message, error)  { return nil, nil }
func (r *mockMsgRepo) GetStarred(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetStarredByUser(_ context.Context, _ domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetSpam(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetUnread(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetUnreadByMailbox(_ context.Context, _ domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetMessagesWithAttachments(_ context.Context, _ *repository.MessageFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetRecent(_ context.Context, _ int, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetByDateRange(_ context.Context, _ *repository.DateRangeFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetBySender(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetByRecipient(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetOldMessages(_ context.Context, _ int, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) GetLargeMessages(_ context.Context, _ int64, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) DeleteOldMessages(_ context.Context, _ int) (int64, error) { return 0, nil }
func (r *mockMsgRepo) DeleteSpam(_ context.Context) (int64, error)                { return 0, nil }

func (r *mockMsgRepo) BulkMarkAsRead(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockMsgRepo) BulkMarkAsUnread(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockMsgRepo) BulkDelete(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockMsgRepo) BulkMove(_ context.Context, ids []domain.ID, _ domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockMsgRepo) BulkStar(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockMsgRepo) BulkUnstar(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockMsgRepo) GetSizeByMailbox(_ context.Context, _ domain.ID) (int64, error) { return 0, nil }
func (r *mockMsgRepo) GetTotalSize(_ context.Context) (int64, error)                   { return 0, nil }
func (r *mockMsgRepo) GetDailyCounts(_ context.Context, _ *repository.DateRangeFilter) ([]repository.DateCount, error) {
	return nil, nil
}
func (r *mockMsgRepo) GetHourlyCounts(_ context.Context, _ *repository.DateRangeFilter) ([]repository.HourCount, error) {
	return nil, nil
}
func (r *mockMsgRepo) GetSenderCounts(_ context.Context, _ int) ([]repository.AddressCount, error) {
	return nil, nil
}
func (r *mockMsgRepo) GetRecipientCounts(_ context.Context, _ int) ([]repository.AddressCount, error) {
	return nil, nil
}

func (r *mockMsgRepo) StoreRawBody(_ context.Context, id domain.ID, rawBody []byte) error {
	r.rawBodies[id] = rawBody
	return nil
}

func (r *mockMsgRepo) GetRawBody(_ context.Context, id domain.ID) ([]byte, error) {
	if data, ok := r.rawBodies[id]; ok {
		return data, nil
	}
	if m, ok := r.messages[id]; ok && len(m.RawBody) > 0 {
		return m.RawBody, nil
	}
	return nil, domain.NewNotFoundError("raw_body", id.String())
}
func (r *mockMsgRepo) GetByIMAPUID(_ context.Context, _ domain.ID, _ uint32) (*domain.Message, error) {
	return nil, domain.NewNotFoundError("message", "imap_uid")
}
func (r *mockMsgRepo) GetByMessageIDs(_ context.Context, _ []string) ([]*domain.Message, error) {
	return nil, nil
}
func (r *mockMsgRepo) GetByThreadID(_ context.Context, _ domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	return &repository.ListResult[*domain.Message]{}, nil
}
func (r *mockMsgRepo) UpdateThreadID(_ context.Context, _, _ domain.ID) error { return nil }
func (r *mockMsgRepo) GetByBlobID(_ context.Context, _ string) (*domain.Message, error) {
	return nil, domain.NewNotFoundError("message", "blob")
}

// --- mockMbxRepo: implements repository.MailboxRepository ---

type mockMbxRepo struct {
	mailboxes map[domain.ID]*domain.Mailbox
}

var _ repository.MailboxRepository = (*mockMbxRepo)(nil)

func newMockMbxRepo() *mockMbxRepo {
	return &mockMbxRepo{mailboxes: make(map[domain.ID]*domain.Mailbox)}
}

func (r *mockMbxRepo) add(m *domain.Mailbox) { r.mailboxes[m.ID] = m }

func (r *mockMbxRepo) GetByID(_ context.Context, id domain.ID) (*domain.Mailbox, error) {
	if m, ok := r.mailboxes[id]; ok {
		return m, nil
	}
	return nil, domain.NewNotFoundError("mailbox", id.String())
}

func (r *mockMbxRepo) GetByAddress(_ context.Context, addr string) (*domain.Mailbox, error) {
	for _, m := range r.mailboxes {
		if m.Address == addr {
			return m, nil
		}
	}
	return nil, domain.NewNotFoundError("mailbox", addr)
}

func (r *mockMbxRepo) GetCatchAll(_ context.Context, _ string) (*domain.Mailbox, error) {
	return nil, domain.NewNotFoundError("mailbox", "catch-all")
}

func (r *mockMbxRepo) GetDefault(_ context.Context, userID domain.ID) (*domain.Mailbox, error) {
	for _, m := range r.mailboxes {
		if m.UserID == userID && m.IsDefault {
			return m, nil
		}
	}
	return nil, domain.NewNotFoundError("mailbox", "default")
}

func (r *mockMbxRepo) List(_ context.Context, _ *repository.MailboxFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	var items []*domain.Mailbox
	for _, m := range r.mailboxes {
		items = append(items, m)
	}
	return &repository.ListResult[*domain.Mailbox]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockMbxRepo) ListByUser(_ context.Context, userID domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	var items []*domain.Mailbox
	for _, m := range r.mailboxes {
		if m.UserID == userID {
			items = append(items, m)
		}
	}
	return &repository.ListResult[*domain.Mailbox]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockMbxRepo) Create(_ context.Context, m *domain.Mailbox) error {
	r.mailboxes[m.ID] = m
	return nil
}

func (r *mockMbxRepo) Update(_ context.Context, m *domain.Mailbox) error {
	r.mailboxes[m.ID] = m
	return nil
}

func (r *mockMbxRepo) Delete(_ context.Context, id domain.ID) error {
	if _, ok := r.mailboxes[id]; !ok {
		return domain.NewNotFoundError("mailbox", id.String())
	}
	delete(r.mailboxes, id)
	return nil
}

func (r *mockMbxRepo) DeleteWithMessages(_ context.Context, id domain.ID) error {
	return r.Delete(context.Background(), id)
}

func (r *mockMbxRepo) DeleteByUser(_ context.Context, _ domain.ID) (int64, error) { return 0, nil }
func (r *mockMbxRepo) Exists(_ context.Context, id domain.ID) (bool, error) {
	_, ok := r.mailboxes[id]
	return ok, nil
}
func (r *mockMbxRepo) ExistsByAddress(_ context.Context, addr string) (bool, error) {
	for _, m := range r.mailboxes {
		if m.Address == addr {
			return true, nil
		}
	}
	return false, nil
}
func (r *mockMbxRepo) Count(_ context.Context, _ *repository.MailboxFilter) (int64, error) {
	return int64(len(r.mailboxes)), nil
}
func (r *mockMbxRepo) CountByUser(_ context.Context, userID domain.ID) (int64, error) {
	var count int64
	for _, m := range r.mailboxes {
		if m.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *mockMbxRepo) SetDefault(_ context.Context, id domain.ID) error {
	if m, ok := r.mailboxes[id]; ok {
		for _, other := range r.mailboxes {
			if other.UserID == m.UserID {
				other.IsDefault = false
			}
		}
		m.IsDefault = true
		return nil
	}
	return domain.NewNotFoundError("mailbox", id.String())
}

func (r *mockMbxRepo) ClearDefault(_ context.Context, userID domain.ID) error {
	for _, m := range r.mailboxes {
		if m.UserID == userID {
			m.IsDefault = false
		}
	}
	return nil
}

func (r *mockMbxRepo) SetCatchAll(_ context.Context, _ domain.ID) error   { return nil }
func (r *mockMbxRepo) ClearCatchAll(_ context.Context, _ domain.ID) error { return nil }

func (r *mockMbxRepo) UpdateStats(_ context.Context, _ domain.ID, _ *repository.MailboxStatsUpdate) error {
	return nil
}
func (r *mockMbxRepo) IncrementMessageCount(_ context.Context, _ domain.ID, _ int64, _ bool) (uint32, error) {
	return 1, nil
}
func (r *mockMbxRepo) DecrementMessageCount(_ context.Context, _ domain.ID, _ int64, _ bool) error {
	return nil
}
func (r *mockMbxRepo) UpdateUnreadCount(_ context.Context, _ domain.ID, _ int) error { return nil }
func (r *mockMbxRepo) RecalculateStats(_ context.Context, _ domain.ID) error          { return nil }

func (r *mockMbxRepo) GetStats(_ context.Context, id domain.ID) (*domain.MailboxStats, error) {
	if _, ok := r.mailboxes[id]; !ok {
		return nil, domain.NewNotFoundError("mailbox", id.String())
	}
	return &domain.MailboxStats{}, nil
}

func (r *mockMbxRepo) GetStatsByUser(_ context.Context, _ domain.ID) (*domain.MailboxStats, error) {
	return &domain.MailboxStats{}, nil
}

func (r *mockMbxRepo) GetTotalStats(_ context.Context) (*domain.MailboxStats, error) {
	return &domain.MailboxStats{}, nil
}

func (r *mockMbxRepo) FindMatchingMailbox(_ context.Context, addr string) (*domain.Mailbox, error) {
	for _, m := range r.mailboxes {
		if m.Address == addr {
			return m, nil
		}
	}
	return nil, domain.NewNotFoundError("mailbox", addr)
}

func (r *mockMbxRepo) Search(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}
func (r *mockMbxRepo) GetMailboxesWithMessages(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}
func (r *mockMbxRepo) GetMailboxesWithUnread(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}
func (r *mockMbxRepo) TransferOwnership(_ context.Context, _, _ domain.ID) (int64, error) { return 0, nil }
func (r *mockMbxRepo) BulkDelete(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}
func (r *mockMbxRepo) GetDomains(_ context.Context) ([]string, error) { return nil, nil }
func (r *mockMbxRepo) GetMailboxesByDomain(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	return &repository.ListResult[*domain.Mailbox]{}, nil
}

// --- mockWhRepo: implements repository.WebhookRepository ---

type mockWhRepo struct {
	webhooks   map[domain.ID]*domain.Webhook
	deliveries map[domain.ID]*domain.WebhookDelivery
}

var _ repository.WebhookRepository = (*mockWhRepo)(nil)

func newMockWhRepo() *mockWhRepo {
	return &mockWhRepo{
		webhooks:   make(map[domain.ID]*domain.Webhook),
		deliveries: make(map[domain.ID]*domain.WebhookDelivery),
	}
}

func (r *mockWhRepo) add(w *domain.Webhook) { r.webhooks[w.ID] = w }

func (r *mockWhRepo) GetByID(_ context.Context, id domain.ID) (*domain.Webhook, error) {
	if w, ok := r.webhooks[id]; ok {
		return w, nil
	}
	return nil, domain.NewNotFoundError("webhook", id.String())
}

func (r *mockWhRepo) List(_ context.Context, filter *repository.WebhookFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	var items []*domain.Webhook
	for _, w := range r.webhooks {
		if filter != nil && filter.UserID != nil && w.UserID != *filter.UserID {
			continue
		}
		items = append(items, w)
	}
	return &repository.ListResult[*domain.Webhook]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWhRepo) ListByUser(_ context.Context, userID domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	var items []*domain.Webhook
	for _, w := range r.webhooks {
		if w.UserID == userID {
			items = append(items, w)
		}
	}
	return &repository.ListResult[*domain.Webhook]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWhRepo) ListByEvent(_ context.Context, _ domain.WebhookEvent) ([]*domain.Webhook, error) { return nil, nil }
func (r *mockWhRepo) ListActiveByEvent(_ context.Context, _ domain.WebhookEvent) ([]*domain.Webhook, error) { return nil, nil }

func (r *mockWhRepo) Create(_ context.Context, w *domain.Webhook) error {
	r.webhooks[w.ID] = w
	return nil
}

func (r *mockWhRepo) Update(_ context.Context, w *domain.Webhook) error {
	r.webhooks[w.ID] = w
	return nil
}

func (r *mockWhRepo) Delete(_ context.Context, id domain.ID) error {
	if _, ok := r.webhooks[id]; !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	delete(r.webhooks, id)
	return nil
}

func (r *mockWhRepo) DeleteByUser(_ context.Context, _ domain.ID) (int64, error) { return 0, nil }
func (r *mockWhRepo) Exists(_ context.Context, id domain.ID) (bool, error) {
	_, ok := r.webhooks[id]
	return ok, nil
}
func (r *mockWhRepo) ExistsByURL(_ context.Context, _ domain.ID, _ string) (bool, error) { return false, nil }
func (r *mockWhRepo) Count(_ context.Context, _ *repository.WebhookFilter) (int64, error) {
	return int64(len(r.webhooks)), nil
}
func (r *mockWhRepo) CountByUser(_ context.Context, userID domain.ID) (int64, error) {
	var count int64
	for _, w := range r.webhooks {
		if w.UserID == userID {
			count++
		}
	}
	return count, nil
}
func (r *mockWhRepo) CountByStatus(_ context.Context) (map[domain.WebhookStatus]int64, error) {
	return make(map[domain.WebhookStatus]int64), nil
}

func (r *mockWhRepo) Activate(_ context.Context, id domain.ID) error {
	if w, ok := r.webhooks[id]; ok {
		w.Status = domain.WebhookStatusActive
		return nil
	}
	return domain.NewNotFoundError("webhook", id.String())
}

func (r *mockWhRepo) Deactivate(_ context.Context, id domain.ID) error {
	if w, ok := r.webhooks[id]; ok {
		w.Status = domain.WebhookStatusInactive
		return nil
	}
	return domain.NewNotFoundError("webhook", id.String())
}

func (r *mockWhRepo) MarkAsFailed(_ context.Context, _ domain.ID, _ string) error  { return nil }
func (r *mockWhRepo) UpdateStatus(_ context.Context, _ domain.ID, _ domain.WebhookStatus) error { return nil }
func (r *mockWhRepo) UpdateSecret(_ context.Context, _ domain.ID, _ string) error  { return nil }
func (r *mockWhRepo) AddEvent(_ context.Context, _ domain.ID, _ domain.WebhookEvent) (bool, error) { return false, nil }
func (r *mockWhRepo) RemoveEvent(_ context.Context, _ domain.ID, _ domain.WebhookEvent) (bool, error) { return false, nil }
func (r *mockWhRepo) SetEvents(_ context.Context, _ domain.ID, _ []domain.WebhookEvent) error { return nil }
func (r *mockWhRepo) RecordSuccess(_ context.Context, _ domain.ID) error            { return nil }
func (r *mockWhRepo) RecordFailure(_ context.Context, _ domain.ID, _ string) error  { return nil }
func (r *mockWhRepo) ResetRetryCount(_ context.Context, _ domain.ID) error          { return nil }
func (r *mockWhRepo) GetActiveWebhooks(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	return &repository.ListResult[*domain.Webhook]{}, nil
}
func (r *mockWhRepo) GetFailedWebhooks(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	return &repository.ListResult[*domain.Webhook]{}, nil
}
func (r *mockWhRepo) GetWebhooksNeedingRetry(_ context.Context) ([]*domain.Webhook, error) { return nil, nil }
func (r *mockWhRepo) Search(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	return &repository.ListResult[*domain.Webhook]{}, nil
}
func (r *mockWhRepo) BulkActivate(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}
func (r *mockWhRepo) BulkDeactivate(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}
func (r *mockWhRepo) BulkDelete(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockWhRepo) CreateDelivery(_ context.Context, d *domain.WebhookDelivery) error {
	r.deliveries[d.ID] = d
	return nil
}

func (r *mockWhRepo) GetDelivery(_ context.Context, id domain.ID) (*domain.WebhookDelivery, error) {
	if d, ok := r.deliveries[id]; ok {
		return d, nil
	}
	return nil, domain.NewNotFoundError("delivery", id.String())
}

func (r *mockWhRepo) ListDeliveries(_ context.Context, _ domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	var items []*domain.WebhookDelivery
	for _, d := range r.deliveries {
		items = append(items, d)
	}
	return &repository.ListResult[*domain.WebhookDelivery]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWhRepo) ListDeliveriesByEvent(_ context.Context, _ domain.ID, _ domain.WebhookEvent, _ *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	return &repository.ListResult[*domain.WebhookDelivery]{}, nil
}
func (r *mockWhRepo) ListRecentDeliveries(_ context.Context, _ domain.ID, _ int) ([]*domain.WebhookDelivery, error) {
	return nil, nil
}
func (r *mockWhRepo) ListFailedDeliveries(_ context.Context, _ domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	return &repository.ListResult[*domain.WebhookDelivery]{}, nil
}
func (r *mockWhRepo) DeleteDeliveries(_ context.Context, _ domain.ID) (int64, error) { return 0, nil }
func (r *mockWhRepo) DeleteOldDeliveries(_ context.Context, _ int) (int64, error)     { return 0, nil }

func (r *mockWhRepo) GetDeliveryStats(_ context.Context, _ domain.ID) (*repository.WebhookDeliveryStats, error) {
	return &repository.WebhookDeliveryStats{}, nil
}

func (r *mockWhRepo) GetDeliveryStatsByDateRange(_ context.Context, _ domain.ID, _ *repository.DateRangeFilter) (*repository.WebhookDeliveryStats, error) {
	return &repository.WebhookDeliveryStats{}, nil
}
func (r *mockWhRepo) GetDailyDeliveryCounts(_ context.Context, _ domain.ID, _ *repository.DateRangeFilter) ([]repository.DateCount, error) {
	return nil, nil
}
func (r *mockWhRepo) GetEventDeliveryCounts(_ context.Context, _ domain.ID) ([]repository.EventCount, error) {
	return nil, nil
}

// --- mockAttRepo: implements repository.AttachmentRepository ---

type mockAttRepo struct {
	attachments map[domain.ID]*domain.Attachment
	contents    map[domain.ID][]byte
}

var _ repository.AttachmentRepository = (*mockAttRepo)(nil)

func newMockAttRepo() *mockAttRepo {
	return &mockAttRepo{
		attachments: make(map[domain.ID]*domain.Attachment),
		contents:    make(map[domain.ID][]byte),
	}
}

func (r *mockAttRepo) add(a *domain.Attachment, content []byte) {
	r.attachments[a.ID] = a
	if content != nil {
		r.contents[a.ID] = content
	}
}

func (r *mockAttRepo) GetByID(_ context.Context, id domain.ID) (*domain.Attachment, error) {
	if a, ok := r.attachments[id]; ok {
		return a, nil
	}
	return nil, domain.NewNotFoundError("attachment", id.String())
}

func (r *mockAttRepo) GetByContentID(_ context.Context, _ string) (*domain.Attachment, error) {
	return nil, domain.NewNotFoundError("attachment", "content-id")
}

func (r *mockAttRepo) List(_ context.Context, _ *repository.AttachmentFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	var items []*domain.Attachment
	for _, a := range r.attachments {
		items = append(items, a)
	}
	return &repository.ListResult[*domain.Attachment]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockAttRepo) ListByMessage(_ context.Context, msgID domain.ID) ([]*domain.Attachment, error) {
	var items []*domain.Attachment
	for _, a := range r.attachments {
		if a.MessageID == msgID {
			items = append(items, a)
		}
	}
	return items, nil
}

func (r *mockAttRepo) ListByMessages(_ context.Context, _ []domain.ID) (map[domain.ID][]*domain.Attachment, error) {
	return make(map[domain.ID][]*domain.Attachment), nil
}

func (r *mockAttRepo) ListSummaries(_ context.Context, _ *repository.AttachmentFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.AttachmentSummary], error) {
	return &repository.ListResult[*domain.AttachmentSummary]{}, nil
}

func (r *mockAttRepo) ListSummariesByMessage(_ context.Context, _ domain.ID) ([]*domain.AttachmentSummary, error) {
	return nil, nil
}

func (r *mockAttRepo) Create(_ context.Context, a *domain.Attachment) error {
	r.attachments[a.ID] = a
	return nil
}

func (r *mockAttRepo) CreateWithContent(_ context.Context, a *domain.Attachment, content io.Reader) error {
	r.attachments[a.ID] = a
	if content != nil {
		data, _ := io.ReadAll(content)
		r.contents[a.ID] = data
	}
	return nil
}

func (r *mockAttRepo) Update(_ context.Context, a *domain.Attachment) error {
	r.attachments[a.ID] = a
	return nil
}

func (r *mockAttRepo) Delete(_ context.Context, id domain.ID) error {
	delete(r.attachments, id)
	delete(r.contents, id)
	return nil
}

func (r *mockAttRepo) DeleteByMessage(_ context.Context, _ domain.ID) (int64, error) { return 0, nil }
func (r *mockAttRepo) DeleteByMessages(_ context.Context, _ []domain.ID) (int64, error) { return 0, nil }
func (r *mockAttRepo) Exists(_ context.Context, id domain.ID) (bool, error) {
	_, ok := r.attachments[id]
	return ok, nil
}
func (r *mockAttRepo) ExistsByContentID(_ context.Context, _ string) (bool, error) { return false, nil }
func (r *mockAttRepo) Count(_ context.Context, _ *repository.AttachmentFilter) (int64, error) {
	return int64(len(r.attachments)), nil
}
func (r *mockAttRepo) CountByMessage(_ context.Context, _ domain.ID) (int64, error) { return 0, nil }
func (r *mockAttRepo) StoreContent(_ context.Context, id domain.ID, content io.Reader) error {
	data, _ := io.ReadAll(content)
	r.contents[id] = data
	return nil
}

func (r *mockAttRepo) GetContent(_ context.Context, id domain.ID) (io.ReadCloser, error) {
	if data, ok := r.contents[id]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, domain.NewNotFoundError("content", id.String())
}

func (r *mockAttRepo) GetContentWithMetadata(_ context.Context, id domain.ID) (*domain.Attachment, io.ReadCloser, error) {
	a, ok := r.attachments[id]
	if !ok {
		return nil, nil, domain.NewNotFoundError("attachment", id.String())
	}
	data, ok := r.contents[id]
	if !ok {
		return a, nil, domain.NewNotFoundError("content", id.String())
	}
	return a, io.NopCloser(bytes.NewReader(data)), nil
}

func (r *mockAttRepo) GetContentSize(_ context.Context, id domain.ID) (int64, error) {
	if data, ok := r.contents[id]; ok {
		return int64(len(data)), nil
	}
	return 0, nil
}

func (r *mockAttRepo) VerifyContent(_ context.Context, _ domain.ID) (bool, error) { return true, nil }
func (r *mockAttRepo) GetTotalSize(_ context.Context) (int64, error)                { return 0, nil }
func (r *mockAttRepo) GetTotalSizeByMessage(_ context.Context, _ domain.ID) (int64, error) { return 0, nil }
func (r *mockAttRepo) GetByChecksum(_ context.Context, _ string) ([]*domain.Attachment, error) { return nil, nil }
func (r *mockAttRepo) GetInlineAttachments(_ context.Context, _ domain.ID) ([]*domain.Attachment, error) { return nil, nil }
func (r *mockAttRepo) GetNonInlineAttachments(_ context.Context, _ domain.ID) ([]*domain.Attachment, error) { return nil, nil }
func (r *mockAttRepo) GetByContentType(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return &repository.ListResult[*domain.Attachment]{}, nil
}
func (r *mockAttRepo) GetImages(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return &repository.ListResult[*domain.Attachment]{}, nil
}
func (r *mockAttRepo) GetLargeAttachments(_ context.Context, _ int64, _ *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return &repository.ListResult[*domain.Attachment]{}, nil
}
func (r *mockAttRepo) Search(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	return &repository.ListResult[*domain.Attachment]{}, nil
}
func (r *mockAttRepo) BulkDelete(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}
func (r *mockAttRepo) CleanupOrphaned(_ context.Context) (int64, error) { return 0, nil }
func (r *mockAttRepo) GetStorageStats(_ context.Context) (*repository.AttachmentStorageStats, error) {
	return &repository.AttachmentStorageStats{}, nil
}
func (r *mockAttRepo) GetContentTypeStats(_ context.Context) ([]repository.ContentTypeStats, error) {
	return nil, nil
}

// --- mockSettRepo: implements repository.SettingsRepository ---

type mockSettRepo struct{}

var _ repository.SettingsRepository = (*mockSettRepo)(nil)

func (r *mockSettRepo) Get(_ context.Context) (*domain.Settings, error)                    { return &domain.Settings{}, nil }
func (r *mockSettRepo) GetByID(_ context.Context, _ domain.ID) (*domain.Settings, error)   { return &domain.Settings{}, nil }
func (r *mockSettRepo) Save(_ context.Context, _ *domain.Settings) error                   { return nil }
func (r *mockSettRepo) Update(_ context.Context, _ domain.ID, _ *domain.SettingsUpdateInput) error { return nil }
func (r *mockSettRepo) Reset(_ context.Context) error                                       { return nil }
func (r *mockSettRepo) Exists(_ context.Context) (bool, error)                              { return true, nil }
func (r *mockSettRepo) GetSMTP(_ context.Context) (*domain.SMTPSettings, error)             { return &domain.SMTPSettings{}, nil }
func (r *mockSettRepo) UpdateSMTP(_ context.Context, _ *domain.SMTPSettingsUpdate) error    { return nil }
func (r *mockSettRepo) GetIMAP(_ context.Context) (*domain.IMAPSettings, error)             { return &domain.IMAPSettings{}, nil }
func (r *mockSettRepo) UpdateIMAP(_ context.Context, _ *domain.IMAPSettingsUpdate) error    { return nil }
func (r *mockSettRepo) GetWebUI(_ context.Context) (*domain.WebUISettings, error)           { return &domain.WebUISettings{}, nil }
func (r *mockSettRepo) UpdateWebUI(_ context.Context, _ *domain.WebUISettingsUpdate) error  { return nil }
func (r *mockSettRepo) GetStorage(_ context.Context) (*domain.StorageSettings, error)       { return &domain.StorageSettings{}, nil }
func (r *mockSettRepo) UpdateStorage(_ context.Context, _ *domain.StorageSettingsUpdate) error { return nil }
func (r *mockSettRepo) GetSecurity(_ context.Context) (*domain.SecuritySettings, error)     { return &domain.SecuritySettings{}, nil }
func (r *mockSettRepo) UpdateSecurity(_ context.Context, _ *domain.SecuritySettingsUpdate) error { return nil }
func (r *mockSettRepo) GetRetention(_ context.Context) (*domain.RetentionSettings, error)   { return &domain.RetentionSettings{}, nil }
func (r *mockSettRepo) UpdateRetention(_ context.Context, _ *domain.RetentionSettingsUpdate) error { return nil }
func (r *mockSettRepo) GetNotifications(_ context.Context) (*domain.NotificationSettings, error) { return &domain.NotificationSettings{}, nil }
func (r *mockSettRepo) UpdateNotifications(_ context.Context, _ *domain.NotificationSettingsUpdate) error { return nil }
func (r *mockSettRepo) GetSettingValue(_ context.Context, _ string) (interface{}, error)     { return nil, nil }
func (r *mockSettRepo) SetSettingValue(_ context.Context, _ string, _ interface{}) error     { return nil }
func (r *mockSettRepo) GetHistory(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*repository.SettingsChange], error) {
	return &repository.ListResult[*repository.SettingsChange]{}, nil
}
func (r *mockSettRepo) GetHistoryByField(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*repository.SettingsChange], error) {
	return &repository.ListResult[*repository.SettingsChange]{}, nil
}
func (r *mockSettRepo) Revert(_ context.Context, _ domain.ID) error                  { return nil }
func (r *mockSettRepo) Export(_ context.Context) (*repository.SettingsExport, error)  { return &repository.SettingsExport{}, nil }
func (r *mockSettRepo) Import(_ context.Context, _ *repository.SettingsExport, _ bool) error { return nil }
func (r *mockSettRepo) Validate(_ context.Context) ([]*repository.SettingsValidationError, error) { return nil, nil }
func (r *mockSettRepo) GetDatabaseInfo(_ context.Context) (*repository.DatabaseInfo, error) { return &repository.DatabaseInfo{}, nil }
func (r *mockSettRepo) TestSMTPConnection(_ context.Context) error                    { return nil }
func (r *mockSettRepo) TestDatabaseConnection(_ context.Context) error                { return nil }

// --- mockIDGen: implements service.IDGenerator ---

type mockIDGen struct {
	counter int
}

func (g *mockIDGen) Generate() domain.ID {
	g.counter++
	return domain.ID(fmt.Sprintf("gen-%d", g.counter))
}

// --- Shared helpers ---

// loginForToken is defined in users_test.go and reused here.
// makeAuthReq is defined in users_test.go and reused here.
// makeAdmin is defined in users_test.go and reused here.
// createTestUser is defined in auth_test.go and reused here.

// seedAdminWithMailbox creates an admin user and a default mailbox, returns the token.
func seedAdminWithMailbox(env *fullTestEnv, t *testing.T) string {
	t.Helper()
	admin := makeAdmin("admin-seed", "seedadmin", "password123")
	env.repo.users.addUser(admin)
	mbx := createTestMailbox("mbx-seed", admin.ID, "inbox@localhost")
	mbx.IsDefault = true
	env.repo.mboxes.add(mbx)
	return loginForToken(t, env.echo, "seedadmin", "password123")
}

// loginForToken is already defined in users_test.go (same package).
// We use it directly. The _ import of middleware is needed for route registration.
var _ = middleware.Auth
