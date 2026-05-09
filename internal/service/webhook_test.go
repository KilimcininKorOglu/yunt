package service

import (
	"context"
	"errors"
	"testing"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// mockWebhookRepository implements repository.WebhookRepository for testing.
type mockWebhookRepository struct {
	webhooks   map[domain.ID]*domain.Webhook
	deliveries map[domain.ID]*domain.WebhookDelivery

	// Controllable errors
	createError        error
	getByIDError       error
	updateError        error
	deleteError        error
	existsByURLError   error
	listError          error
	listByUserError    error
	listActiveByEvent  error
	activateError      error
	deactivateError    error
	createDeliveryError error
	recordSuccessError  error
	recordFailureError  error
	listDeliveriesError error
	deliveryStatsError  error

	// Controllable return values
	existsByURLResult bool
}

func newMockWebhookRepository() *mockWebhookRepository {
	return &mockWebhookRepository{
		webhooks:   make(map[domain.ID]*domain.Webhook),
		deliveries: make(map[domain.ID]*domain.WebhookDelivery),
	}
}

func (r *mockWebhookRepository) addWebhook(wh *domain.Webhook) {
	r.webhooks[wh.ID] = wh
}

func (r *mockWebhookRepository) GetByID(_ context.Context, id domain.ID) (*domain.Webhook, error) {
	if r.getByIDError != nil {
		return nil, r.getByIDError
	}
	wh, ok := r.webhooks[id]
	if !ok {
		return nil, domain.NewNotFoundError("webhook", id.String())
	}
	return wh, nil
}

func (r *mockWebhookRepository) List(_ context.Context, _ *repository.WebhookFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	if r.listError != nil {
		return nil, r.listError
	}
	items := make([]*domain.Webhook, 0, len(r.webhooks))
	for _, wh := range r.webhooks {
		items = append(items, wh)
	}
	return &repository.ListResult[*domain.Webhook]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWebhookRepository) ListByUser(_ context.Context, userID domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	if r.listByUserError != nil {
		return nil, r.listByUserError
	}
	var items []*domain.Webhook
	for _, wh := range r.webhooks {
		if wh.UserID == userID {
			items = append(items, wh)
		}
	}
	return &repository.ListResult[*domain.Webhook]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWebhookRepository) ListByEvent(_ context.Context, event domain.WebhookEvent) ([]*domain.Webhook, error) {
	var items []*domain.Webhook
	for _, wh := range r.webhooks {
		if wh.SubscribesToEvent(event) {
			items = append(items, wh)
		}
	}
	return items, nil
}

func (r *mockWebhookRepository) ListActiveByEvent(_ context.Context, event domain.WebhookEvent) ([]*domain.Webhook, error) {
	if r.listActiveByEvent != nil {
		return nil, r.listActiveByEvent
	}
	var items []*domain.Webhook
	for _, wh := range r.webhooks {
		if wh.IsActive() && wh.SubscribesToEvent(event) {
			items = append(items, wh)
		}
	}
	return items, nil
}

func (r *mockWebhookRepository) Create(_ context.Context, wh *domain.Webhook) error {
	if r.createError != nil {
		return r.createError
	}
	r.webhooks[wh.ID] = wh
	return nil
}

func (r *mockWebhookRepository) Update(_ context.Context, wh *domain.Webhook) error {
	if r.updateError != nil {
		return r.updateError
	}
	if _, ok := r.webhooks[wh.ID]; !ok {
		return domain.NewNotFoundError("webhook", wh.ID.String())
	}
	r.webhooks[wh.ID] = wh
	return nil
}

func (r *mockWebhookRepository) Delete(_ context.Context, id domain.ID) error {
	if r.deleteError != nil {
		return r.deleteError
	}
	if _, ok := r.webhooks[id]; !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	delete(r.webhooks, id)
	return nil
}

func (r *mockWebhookRepository) DeleteByUser(_ context.Context, userID domain.ID) (int64, error) {
	var count int64
	for id, wh := range r.webhooks {
		if wh.UserID == userID {
			delete(r.webhooks, id)
			count++
		}
	}
	return count, nil
}

func (r *mockWebhookRepository) Exists(_ context.Context, id domain.ID) (bool, error) {
	_, ok := r.webhooks[id]
	return ok, nil
}

func (r *mockWebhookRepository) ExistsByURL(_ context.Context, _ domain.ID, _ string) (bool, error) {
	if r.existsByURLError != nil {
		return false, r.existsByURLError
	}
	return r.existsByURLResult, nil
}

func (r *mockWebhookRepository) Count(_ context.Context, _ *repository.WebhookFilter) (int64, error) {
	return int64(len(r.webhooks)), nil
}

func (r *mockWebhookRepository) CountByUser(_ context.Context, userID domain.ID) (int64, error) {
	var count int64
	for _, wh := range r.webhooks {
		if wh.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *mockWebhookRepository) CountByStatus(_ context.Context) (map[domain.WebhookStatus]int64, error) {
	counts := make(map[domain.WebhookStatus]int64)
	for _, wh := range r.webhooks {
		counts[wh.Status]++
	}
	return counts, nil
}

func (r *mockWebhookRepository) Activate(_ context.Context, id domain.ID) error {
	if r.activateError != nil {
		return r.activateError
	}
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.Status = domain.WebhookStatusActive
	return nil
}

func (r *mockWebhookRepository) Deactivate(_ context.Context, id domain.ID) error {
	if r.deactivateError != nil {
		return r.deactivateError
	}
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.Status = domain.WebhookStatusInactive
	return nil
}

func (r *mockWebhookRepository) MarkAsFailed(_ context.Context, id domain.ID, msg string) error {
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.MarkAsFailed(msg)
	return nil
}

func (r *mockWebhookRepository) UpdateStatus(_ context.Context, id domain.ID, status domain.WebhookStatus) error {
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.Status = status
	return nil
}

func (r *mockWebhookRepository) UpdateSecret(_ context.Context, id domain.ID, secret string) error {
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.Secret = secret
	return nil
}

func (r *mockWebhookRepository) AddEvent(_ context.Context, id domain.ID, event domain.WebhookEvent) (bool, error) {
	wh, ok := r.webhooks[id]
	if !ok {
		return false, domain.NewNotFoundError("webhook", id.String())
	}
	return wh.AddEvent(event), nil
}

func (r *mockWebhookRepository) RemoveEvent(_ context.Context, id domain.ID, event domain.WebhookEvent) (bool, error) {
	wh, ok := r.webhooks[id]
	if !ok {
		return false, domain.NewNotFoundError("webhook", id.String())
	}
	return wh.RemoveEvent(event), nil
}

func (r *mockWebhookRepository) SetEvents(_ context.Context, id domain.ID, events []domain.WebhookEvent) error {
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.Events = events
	return nil
}

func (r *mockWebhookRepository) RecordSuccess(_ context.Context, id domain.ID) error {
	if r.recordSuccessError != nil {
		return r.recordSuccessError
	}
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.RecordSuccess()
	return nil
}

func (r *mockWebhookRepository) RecordFailure(_ context.Context, id domain.ID, msg string) error {
	if r.recordFailureError != nil {
		return r.recordFailureError
	}
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.RecordFailure(msg)
	return nil
}

func (r *mockWebhookRepository) ResetRetryCount(_ context.Context, id domain.ID) error {
	wh, ok := r.webhooks[id]
	if !ok {
		return domain.NewNotFoundError("webhook", id.String())
	}
	wh.ResetRetryCount()
	return nil
}

func (r *mockWebhookRepository) GetActiveWebhooks(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	var items []*domain.Webhook
	for _, wh := range r.webhooks {
		if wh.IsActive() {
			items = append(items, wh)
		}
	}
	return &repository.ListResult[*domain.Webhook]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWebhookRepository) GetFailedWebhooks(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	var items []*domain.Webhook
	for _, wh := range r.webhooks {
		if wh.Status == domain.WebhookStatusFailed {
			items = append(items, wh)
		}
	}
	return &repository.ListResult[*domain.Webhook]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWebhookRepository) GetWebhooksNeedingRetry(_ context.Context) ([]*domain.Webhook, error) {
	var items []*domain.Webhook
	for _, wh := range r.webhooks {
		if wh.ShouldRetry() {
			items = append(items, wh)
		}
	}
	return items, nil
}

func (r *mockWebhookRepository) Search(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	return &repository.ListResult[*domain.Webhook]{}, nil
}

func (r *mockWebhookRepository) BulkActivate(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	for _, id := range ids {
		if wh, ok := r.webhooks[id]; ok {
			wh.Status = domain.WebhookStatusActive
			op.AddSuccess()
		} else {
			op.AddFailure(id.String(), domain.NewNotFoundError("webhook", id.String()))
		}
	}
	return op, nil
}

func (r *mockWebhookRepository) BulkDeactivate(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	for _, id := range ids {
		if wh, ok := r.webhooks[id]; ok {
			wh.Status = domain.WebhookStatusInactive
			op.AddSuccess()
		} else {
			op.AddFailure(id.String(), domain.NewNotFoundError("webhook", id.String()))
		}
	}
	return op, nil
}

func (r *mockWebhookRepository) BulkDelete(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	for _, id := range ids {
		if _, ok := r.webhooks[id]; ok {
			delete(r.webhooks, id)
			op.AddSuccess()
		} else {
			op.AddFailure(id.String(), domain.NewNotFoundError("webhook", id.String()))
		}
	}
	return op, nil
}

func (r *mockWebhookRepository) CreateDelivery(_ context.Context, d *domain.WebhookDelivery) error {
	if r.createDeliveryError != nil {
		return r.createDeliveryError
	}
	r.deliveries[d.ID] = d
	return nil
}

func (r *mockWebhookRepository) GetDelivery(_ context.Context, id domain.ID) (*domain.WebhookDelivery, error) {
	d, ok := r.deliveries[id]
	if !ok {
		return nil, domain.NewNotFoundError("delivery", id.String())
	}
	return d, nil
}

func (r *mockWebhookRepository) ListDeliveries(_ context.Context, webhookID domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	if r.listDeliveriesError != nil {
		return nil, r.listDeliveriesError
	}
	var items []*domain.WebhookDelivery
	for _, d := range r.deliveries {
		if d.WebhookID == webhookID {
			items = append(items, d)
		}
	}
	return &repository.ListResult[*domain.WebhookDelivery]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWebhookRepository) ListDeliveriesByEvent(_ context.Context, webhookID domain.ID, event domain.WebhookEvent, _ *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	var items []*domain.WebhookDelivery
	for _, d := range r.deliveries {
		if d.WebhookID == webhookID && d.Event == event {
			items = append(items, d)
		}
	}
	return &repository.ListResult[*domain.WebhookDelivery]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWebhookRepository) ListRecentDeliveries(_ context.Context, webhookID domain.ID, _ int) ([]*domain.WebhookDelivery, error) {
	var items []*domain.WebhookDelivery
	for _, d := range r.deliveries {
		if d.WebhookID == webhookID {
			items = append(items, d)
		}
	}
	return items, nil
}

func (r *mockWebhookRepository) ListFailedDeliveries(_ context.Context, webhookID domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	var items []*domain.WebhookDelivery
	for _, d := range r.deliveries {
		if d.WebhookID == webhookID && !d.Success {
			items = append(items, d)
		}
	}
	return &repository.ListResult[*domain.WebhookDelivery]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockWebhookRepository) DeleteDeliveries(_ context.Context, webhookID domain.ID) (int64, error) {
	var count int64
	for id, d := range r.deliveries {
		if d.WebhookID == webhookID {
			delete(r.deliveries, id)
			count++
		}
	}
	return count, nil
}

func (r *mockWebhookRepository) DeleteOldDeliveries(_ context.Context, _ int) (int64, error) {
	return 0, nil
}

func (r *mockWebhookRepository) GetDeliveryStats(_ context.Context, _ domain.ID) (*repository.WebhookDeliveryStats, error) {
	if r.deliveryStatsError != nil {
		return nil, r.deliveryStatsError
	}
	return &repository.WebhookDeliveryStats{}, nil
}

func (r *mockWebhookRepository) GetDeliveryStatsByDateRange(_ context.Context, _ domain.ID, _ *repository.DateRangeFilter) (*repository.WebhookDeliveryStats, error) {
	return &repository.WebhookDeliveryStats{}, nil
}

func (r *mockWebhookRepository) GetDailyDeliveryCounts(_ context.Context, _ domain.ID, _ *repository.DateRangeFilter) ([]repository.DateCount, error) {
	return nil, nil
}

func (r *mockWebhookRepository) GetEventDeliveryCounts(_ context.Context, _ domain.ID) ([]repository.EventCount, error) {
	return nil, nil
}

// mockWebhookRepository needs DateCount — check if it exists or define it.

// webhookMockRepo wraps mockWebhookRepository in a repository.Repository for WebhookService.
type webhookMockRepo struct {
	webhooks *mockWebhookRepository
}

func newWebhookMockRepo() *webhookMockRepo {
	return &webhookMockRepo{
		webhooks: newMockWebhookRepository(),
	}
}

func (r *webhookMockRepo) Users() repository.UserRepository             { return nil }
func (r *webhookMockRepo) Mailboxes() repository.MailboxRepository       { return nil }
func (r *webhookMockRepo) Messages() repository.MessageRepository        { return nil }
func (r *webhookMockRepo) Attachments() repository.AttachmentRepository  { return nil }
func (r *webhookMockRepo) Webhooks() repository.WebhookRepository        { return r.webhooks }
func (r *webhookMockRepo) Settings() repository.SettingsRepository       { return nil }
func (r *webhookMockRepo) Health(_ context.Context) error                { return nil }
func (r *webhookMockRepo) Close() error                                  { return nil }

func (r *webhookMockRepo) Transaction(_ context.Context, fn func(tx repository.Repository) error) error {
	return fn(r)
}

func (r *webhookMockRepo) TransactionWithOptions(_ context.Context, _ repository.TransactionOptions, fn func(tx repository.Repository) error) error {
	return fn(r)
}

// newTestWebhookService creates a WebhookService wired to a mock repository.
func newTestWebhookService() (*WebhookService, *webhookMockRepo) {
	repo := newWebhookMockRepo()
	idGen := newTestIDGenerator()
	svc := NewWebhookService(repo, idGen)
	return svc, repo
}

// --- helpers ---

func makeCreateInput(name, rawURL string, events []domain.WebhookEvent) *domain.WebhookCreateInput {
	return &domain.WebhookCreateInput{
		Name:   name,
		URL:    rawURL,
		Events: events,
	}
}

const (
	whTestUserID    = domain.ID("user-1")
	whTestWebhookID = domain.ID("wh-1")
	whTestURL       = "https://example.com/hook"
)

// --- CreateWebhook tests ---

func TestWebhookService_CreateWebhook(t *testing.T) {
	tests := []struct {
		name          string
		userID        domain.ID
		input         *domain.WebhookCreateInput
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name:   "successful creation",
			userID: whTestUserID,
			input:  makeCreateInput("My Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}),
			wantErr: false,
		},
		{
			name:          "empty user ID",
			userID:        domain.ID(""),
			input:         makeCreateInput("My Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}),
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name:          "invalid input — missing name",
			userID:        whTestUserID,
			input:         makeCreateInput("", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}),
			wantErr:       true,
		},
		{
			name:          "invalid input — missing events",
			userID:        whTestUserID,
			input:         makeCreateInput("My Hook", whTestURL, nil),
			wantErr:       true,
		},
		{
			name:          "duplicate URL",
			userID:        whTestUserID,
			input:         makeCreateInput("My Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}),
			setupMock: func(r *mockWebhookRepository) {
				r.existsByURLResult = true
			},
			wantErr:       true,
			wantErrTarget: domain.ErrAlreadyExists,
		},
		{
			name:   "repository ExistsByURL error",
			userID: whTestUserID,
			input:  makeCreateInput("My Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}),
			setupMock: func(r *mockWebhookRepository) {
				r.existsByURLError = errors.New("db error")
			},
			wantErr: true,
		},
		{
			name:   "repository Create error",
			userID: whTestUserID,
			input:  makeCreateInput("My Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}),
			setupMock: func(r *mockWebhookRepository) {
				r.createError = errors.New("db error")
			},
			wantErr: true,
		},
		{
			name:   "optional fields applied",
			userID: whTestUserID,
			input: func() *domain.WebhookCreateInput {
				maxRetries := 5
				timeout := 10
				return &domain.WebhookCreateInput{
					Name:           "Full Hook",
					URL:            whTestURL,
					Events:         []domain.WebhookEvent{domain.WebhookEventMessageReceived},
					Secret:         "s3cr3t",
					Headers:        map[string]string{"X-Custom": "value"},
					MaxRetries:     &maxRetries,
					TimeoutSeconds: &timeout,
				}
			}(),
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			got, err := svc.CreateWebhook(ctx, tc.userID, tc.input)

			if (err != nil) != tc.wantErr {
				t.Errorf("CreateWebhook() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("CreateWebhook() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}

			if !tc.wantErr {
				if got == nil {
					t.Fatal("CreateWebhook() returned nil webhook on success")
				}
				if got.UserID != tc.userID {
					t.Errorf("CreateWebhook() UserID = %v, want %v", got.UserID, tc.userID)
				}
				if got.URL != tc.input.URL {
					t.Errorf("CreateWebhook() URL = %v, want %v", got.URL, tc.input.URL)
				}
				if got.ID.IsEmpty() {
					t.Error("CreateWebhook() returned webhook with empty ID")
				}
			}
		})
	}
}

func TestWebhookService_CreateWebhook_OptionalFields(t *testing.T) {
	svc, _ := newTestWebhookService()
	maxRetries := 5
	timeout := 10
	input := &domain.WebhookCreateInput{
		Name:           "Full Hook",
		URL:            whTestURL,
		Events:         []domain.WebhookEvent{domain.WebhookEventMessageReceived},
		Secret:         "s3cr3t",
		Headers:        map[string]string{"X-Custom": "value"},
		MaxRetries:     &maxRetries,
		TimeoutSeconds: &timeout,
	}

	ctx := context.Background()
	got, err := svc.CreateWebhook(ctx, whTestUserID, input)
	if err != nil {
		t.Fatalf("CreateWebhook() unexpected error: %v", err)
	}

	if got.Secret != "s3cr3t" {
		t.Errorf("CreateWebhook() Secret = %v, want s3cr3t", got.Secret)
	}
	if got.MaxRetries != 5 {
		t.Errorf("CreateWebhook() MaxRetries = %v, want 5", got.MaxRetries)
	}
	if got.TimeoutSeconds != 10 {
		t.Errorf("CreateWebhook() TimeoutSeconds = %v, want 10", got.TimeoutSeconds)
	}
	if got.Headers["X-Custom"] != "value" {
		t.Errorf("CreateWebhook() Headers[X-Custom] = %v, want value", got.Headers["X-Custom"])
	}
}

// --- GetWebhook tests ---

func TestWebhookService_GetWebhook(t *testing.T) {
	tests := []struct {
		name          string
		id            domain.ID
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name: "found",
			id:   whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				wh := domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived})
				r.addWebhook(wh)
			},
			wantErr: false,
		},
		{
			name:          "empty ID",
			id:            domain.ID(""),
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name:          "not found",
			id:            domain.ID("missing"),
			wantErr:       true,
			wantErrTarget: domain.ErrNotFound,
		},
		{
			name: "repository error",
			id:   whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				r.getByIDError = errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			got, err := svc.GetWebhook(ctx, tc.id)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetWebhook() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("GetWebhook() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}
			if !tc.wantErr && got == nil {
				t.Error("GetWebhook() returned nil on success")
			}
		})
	}
}

// --- GetWebhookForUser tests ---

func TestWebhookService_GetWebhookForUser(t *testing.T) {
	tests := []struct {
		name          string
		webhookID     domain.ID
		userID        domain.ID
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name:      "owner match",
			webhookID: whTestWebhookID,
			userID:    whTestUserID,
			setupMock: func(r *mockWebhookRepository) {
				wh := domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived})
				r.addWebhook(wh)
			},
			wantErr: false,
		},
		{
			name:      "owner mismatch",
			webhookID: whTestWebhookID,
			userID:    domain.ID("other-user"),
			setupMock: func(r *mockWebhookRepository) {
				wh := domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived})
				r.addWebhook(wh)
			},
			wantErr:       true,
			wantErrTarget: domain.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			got, err := svc.GetWebhookForUser(ctx, tc.webhookID, tc.userID)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetWebhookForUser() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("GetWebhookForUser() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}
			if !tc.wantErr && got == nil {
				t.Error("GetWebhookForUser() returned nil on success")
			}
		})
	}
}

// --- ListWebhooks tests ---

func TestWebhookService_ListWebhooks(t *testing.T) {
	svc, repo := newTestWebhookService()
	wh1 := domain.NewWebhook(domain.ID("wh-1"), whTestUserID, "Hook 1", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived})
	wh2 := domain.NewWebhook(domain.ID("wh-2"), whTestUserID, "Hook 2", "https://other.example.com/hook", []domain.WebhookEvent{domain.WebhookEventMailboxCreated})
	repo.webhooks.addWebhook(wh1)
	repo.webhooks.addWebhook(wh2)

	ctx := context.Background()
	result, err := svc.ListWebhooks(ctx, &repository.WebhookFilter{}, nil)
	if err != nil {
		t.Fatalf("ListWebhooks() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("ListWebhooks() returned nil result")
	}
	if result.Total != 2 {
		t.Errorf("ListWebhooks() Total = %v, want 2", result.Total)
	}
}

func TestWebhookService_ListWebhooks_RepoError(t *testing.T) {
	svc, repo := newTestWebhookService()
	repo.webhooks.listError = errors.New("db error")

	ctx := context.Background()
	_, err := svc.ListWebhooks(ctx, nil, nil)
	if err == nil {
		t.Error("ListWebhooks() should return error when repo fails")
	}
}

// --- ListWebhooksByUser tests ---

func TestWebhookService_ListWebhooksByUser(t *testing.T) {
	tests := []struct {
		name          string
		userID        domain.ID
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
		wantCount     int
	}{
		{
			name:   "returns user webhooks",
			userID: whTestUserID,
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(domain.ID("wh-1"), whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
				r.addWebhook(domain.NewWebhook(domain.ID("wh-2"), domain.ID("other"), "Hook 2", "https://other.example.com/hook", []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantCount: 1,
		},
		{
			name:          "empty user ID",
			userID:        domain.ID(""),
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name:   "repository error",
			userID: whTestUserID,
			setupMock: func(r *mockWebhookRepository) {
				r.listByUserError = errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			result, err := svc.ListWebhooksByUser(ctx, tc.userID, nil)

			if (err != nil) != tc.wantErr {
				t.Errorf("ListWebhooksByUser() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("ListWebhooksByUser() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}
			if !tc.wantErr && int(result.Total) != tc.wantCount {
				t.Errorf("ListWebhooksByUser() Total = %v, want %v", result.Total, tc.wantCount)
			}
		})
	}
}

// --- UpdateWebhook tests ---

func TestWebhookService_UpdateWebhook(t *testing.T) {
	newName := "Updated Name"
	newURL := "https://new.example.com/hook"

	tests := []struct {
		name          string
		id            domain.ID
		input         *domain.WebhookUpdateInput
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name:  "successful update",
			id:    whTestWebhookID,
			input: &domain.WebhookUpdateInput{Name: &newName},
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Old Name", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantErr: false,
		},
		{
			name:          "empty webhook ID",
			id:            domain.ID(""),
			input:         &domain.WebhookUpdateInput{Name: &newName},
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name:  "webhook not found",
			id:    domain.ID("missing"),
			input: &domain.WebhookUpdateInput{Name: &newName},
			wantErr:       true,
			wantErrTarget: domain.ErrNotFound,
		},
		{
			name:  "URL change — duplicate",
			id:    whTestWebhookID,
			input: &domain.WebhookUpdateInput{URL: &newURL},
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
				r.existsByURLResult = true
			},
			wantErr:       true,
			wantErrTarget: domain.ErrAlreadyExists,
		},
		{
			name: "URL change — same URL (no duplicate check needed)",
			id:   whTestWebhookID,
			input: func() *domain.WebhookUpdateInput {
				sameURL := string(whTestURL)
				return &domain.WebhookUpdateInput{URL: &sameURL}
			}(),
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantErr: false,
		},
		{
			name:  "repository update error",
			id:    whTestWebhookID,
			input: &domain.WebhookUpdateInput{Name: &newName},
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
				r.updateError = errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			got, err := svc.UpdateWebhook(ctx, tc.id, tc.input)

			if (err != nil) != tc.wantErr {
				t.Errorf("UpdateWebhook() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("UpdateWebhook() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}
			if !tc.wantErr {
				if got == nil {
					t.Fatal("UpdateWebhook() returned nil on success")
				}
				if tc.input.Name != nil && got.Name != *tc.input.Name {
					t.Errorf("UpdateWebhook() Name = %v, want %v", got.Name, *tc.input.Name)
				}
			}
		})
	}
}

// --- UpdateWebhookForUser tests ---

func TestWebhookService_UpdateWebhookForUser(t *testing.T) {
	newName := "New Name"

	tests := []struct {
		name      string
		webhookID domain.ID
		userID    domain.ID
		input     *domain.WebhookUpdateInput
		setupMock func(*mockWebhookRepository)
		wantErr   bool
	}{
		{
			name:      "owner update",
			webhookID: whTestWebhookID,
			userID:    whTestUserID,
			input:     &domain.WebhookUpdateInput{Name: &newName},
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantErr: false,
		},
		{
			name:      "non-owner update rejected",
			webhookID: whTestWebhookID,
			userID:    domain.ID("intruder"),
			input:     &domain.WebhookUpdateInput{Name: &newName},
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			_, err := svc.UpdateWebhookForUser(ctx, tc.webhookID, tc.userID, tc.input)

			if (err != nil) != tc.wantErr {
				t.Errorf("UpdateWebhookForUser() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// --- DeleteWebhook tests ---

func TestWebhookService_DeleteWebhook(t *testing.T) {
	tests := []struct {
		name          string
		id            domain.ID
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name: "successful delete",
			id:   whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantErr: false,
		},
		{
			name:          "empty ID",
			id:            domain.ID(""),
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name:          "not found",
			id:            domain.ID("missing"),
			wantErr:       true,
			wantErrTarget: domain.ErrNotFound,
		},
		{
			name: "repository delete error",
			id:   whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
				r.deleteError = errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			err := svc.DeleteWebhook(ctx, tc.id)

			if (err != nil) != tc.wantErr {
				t.Errorf("DeleteWebhook() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("DeleteWebhook() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}

			if !tc.wantErr {
				if _, exists := repo.webhooks.webhooks[tc.id]; exists {
					t.Error("DeleteWebhook() webhook should have been removed from repository")
				}
			}
		})
	}
}

// --- DeleteWebhookForUser tests ---

func TestWebhookService_DeleteWebhookForUser(t *testing.T) {
	tests := []struct {
		name      string
		webhookID domain.ID
		userID    domain.ID
		setupMock func(*mockWebhookRepository)
		wantErr   bool
	}{
		{
			name:      "owner delete",
			webhookID: whTestWebhookID,
			userID:    whTestUserID,
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantErr: false,
		},
		{
			name:      "non-owner delete rejected",
			webhookID: whTestWebhookID,
			userID:    domain.ID("intruder"),
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			err := svc.DeleteWebhookForUser(ctx, tc.webhookID, tc.userID)

			if (err != nil) != tc.wantErr {
				t.Errorf("DeleteWebhookForUser() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// --- ActivateWebhook / DeactivateWebhook tests ---

func TestWebhookService_ActivateWebhook(t *testing.T) {
	tests := []struct {
		name          string
		id            domain.ID
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name: "activates webhook",
			id:   whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				wh := domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived})
				wh.Status = domain.WebhookStatusInactive
				r.addWebhook(wh)
			},
			wantErr: false,
		},
		{
			name:          "empty ID",
			id:            domain.ID(""),
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name: "repository error",
			id:   whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				r.activateError = errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			err := svc.ActivateWebhook(ctx, tc.id)

			if (err != nil) != tc.wantErr {
				t.Errorf("ActivateWebhook() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("ActivateWebhook() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}

			if !tc.wantErr {
				wh := repo.webhooks.webhooks[tc.id]
				if wh.Status != domain.WebhookStatusActive {
					t.Errorf("ActivateWebhook() Status = %v, want active", wh.Status)
				}
			}
		})
	}
}

func TestWebhookService_DeactivateWebhook(t *testing.T) {
	tests := []struct {
		name          string
		id            domain.ID
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name: "deactivates webhook",
			id:   whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				r.addWebhook(domain.NewWebhook(whTestWebhookID, whTestUserID, "Hook", whTestURL, []domain.WebhookEvent{domain.WebhookEventMessageReceived}))
			},
			wantErr: false,
		},
		{
			name:          "empty ID",
			id:            domain.ID(""),
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name: "repository error",
			id:   whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				r.deactivateError = errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			err := svc.DeactivateWebhook(ctx, tc.id)

			if (err != nil) != tc.wantErr {
				t.Errorf("DeactivateWebhook() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("DeactivateWebhook() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}

			if !tc.wantErr {
				wh := repo.webhooks.webhooks[tc.id]
				if wh.Status != domain.WebhookStatusInactive {
					t.Errorf("DeactivateWebhook() Status = %v, want inactive", wh.Status)
				}
			}
		})
	}
}

// --- TriggerEvent tests ---

func TestWebhookService_TriggerEvent_NoSubscribers(t *testing.T) {
	svc, _ := newTestWebhookService()

	ctx := context.Background()
	err := svc.TriggerEvent(ctx, domain.WebhookEventMessageReceived, map[string]string{"key": "val"})
	if err != nil {
		t.Errorf("TriggerEvent() error = %v, want nil when no subscribers", err)
	}
}

func TestWebhookService_TriggerEvent_ListActiveError(t *testing.T) {
	svc, repo := newTestWebhookService()
	repo.webhooks.listActiveByEvent = errors.New("db error")

	ctx := context.Background()
	err := svc.TriggerEvent(ctx, domain.WebhookEventMessageReceived, nil)
	if err == nil {
		t.Error("TriggerEvent() should return error when listing active webhooks fails")
	}
}

// --- ListDeliveries tests ---

func TestWebhookService_ListDeliveries(t *testing.T) {
	tests := []struct {
		name          string
		webhookID     domain.ID
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name:      "returns deliveries",
			webhookID: whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				d := domain.NewWebhookDelivery(domain.ID("d-1"), whTestWebhookID, domain.WebhookEventMessageReceived, "{}", 1)
				r.deliveries[d.ID] = d
			},
			wantErr: false,
		},
		{
			name:          "empty webhook ID",
			webhookID:     domain.ID(""),
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name:      "repository error",
			webhookID: whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				r.listDeliveriesError = errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			result, err := svc.ListDeliveries(ctx, tc.webhookID, nil)

			if (err != nil) != tc.wantErr {
				t.Errorf("ListDeliveries() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("ListDeliveries() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}
			if !tc.wantErr && result == nil {
				t.Error("ListDeliveries() returned nil result on success")
			}
		})
	}
}

// --- GetDeliveryStats tests ---

func TestWebhookService_GetDeliveryStats(t *testing.T) {
	tests := []struct {
		name          string
		webhookID     domain.ID
		setupMock     func(*mockWebhookRepository)
		wantErr       bool
		wantErrTarget error
	}{
		{
			name:      "returns stats",
			webhookID: whTestWebhookID,
			wantErr:   false,
		},
		{
			name:          "empty webhook ID",
			webhookID:     domain.ID(""),
			wantErr:       true,
			wantErrTarget: domain.ErrInvalidInput,
		},
		{
			name:      "repository error",
			webhookID: whTestWebhookID,
			setupMock: func(r *mockWebhookRepository) {
				r.deliveryStatsError = errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestWebhookService()
			if tc.setupMock != nil {
				tc.setupMock(repo.webhooks)
			}

			ctx := context.Background()
			stats, err := svc.GetDeliveryStats(ctx, tc.webhookID)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetDeliveryStats() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErrTarget != nil && !errors.Is(err, tc.wantErrTarget) {
				t.Errorf("GetDeliveryStats() error = %v, wantErrTarget %v", err, tc.wantErrTarget)
			}
			if !tc.wantErr && stats == nil {
				t.Error("GetDeliveryStats() returned nil stats on success")
			}
		})
	}
}

// --- WebhookServiceError tests ---

func TestWebhookServiceError_Error(t *testing.T) {
	err := &WebhookServiceError{
		Op:      "create",
		Message: "something went wrong",
		Err:     errors.New("underlying"),
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("WebhookServiceError.Error() returned empty string")
	}

	if !errors.Is(err, errors.New("underlying")) {
		// errors.Is uses pointer comparison unless Unwrap/Is is implemented
	}

	// Unwrap should return inner error
	inner := errors.Unwrap(err)
	if inner == nil {
		t.Error("WebhookServiceError.Unwrap() returned nil")
	}
}

func TestWebhookServiceError_Is(t *testing.T) {
	inner := domain.ErrNotFound
	err := &WebhookServiceError{Op: "get", Message: "not found", Err: inner}

	if !errors.Is(err, domain.ErrNotFound) {
		t.Error("WebhookServiceError.Is() should match wrapped domain.ErrNotFound")
	}
	if errors.Is(err, domain.ErrAlreadyExists) {
		t.Error("WebhookServiceError.Is() should NOT match domain.ErrAlreadyExists")
	}
}

func TestWebhookServiceError_NilErr(t *testing.T) {
	err := &WebhookServiceError{Op: "test", Message: "no inner", Err: nil}

	// Is should return false when Err is nil
	if errors.Is(err, domain.ErrNotFound) {
		t.Error("WebhookServiceError.Is() should return false when Err is nil")
	}

	// Error() should still work
	if err.Error() == "" {
		t.Error("WebhookServiceError.Error() should not return empty string")
	}
}
