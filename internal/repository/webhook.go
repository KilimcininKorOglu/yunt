package repository

import (
	"context"

	"yunt/internal/domain"
)

// WebhookRepository provides data access operations for Webhook entities.
// It supports CRUD operations, event subscription management, and delivery tracking.
type WebhookRepository interface {
	// GetByID retrieves a webhook by its unique identifier.
	// Returns domain.ErrNotFound if the webhook does not exist.
	GetByID(ctx context.Context, id domain.ID) (*domain.Webhook, error)

	// List retrieves webhooks with optional filtering, sorting, and pagination.
	// Returns an empty slice if no webhooks match the criteria.
	List(ctx context.Context, filter *WebhookFilter, opts *ListOptions) (*ListResult[*domain.Webhook], error)

	// ListByUser retrieves all webhooks owned by a specific user.
	ListByUser(ctx context.Context, userID domain.ID, opts *ListOptions) (*ListResult[*domain.Webhook], error)

	// ListByEvent retrieves all webhooks subscribed to a specific event.
	// Only returns active webhooks.
	ListByEvent(ctx context.Context, event domain.WebhookEvent) ([]*domain.Webhook, error)

	// ListActiveByEvent retrieves all active webhooks subscribed to a specific event.
	ListActiveByEvent(ctx context.Context, event domain.WebhookEvent) ([]*domain.Webhook, error)

	// Create creates a new webhook.
	// Returns domain.ErrAlreadyExists if a webhook with the same URL for the user exists.
	Create(ctx context.Context, webhook *domain.Webhook) error

	// Update updates an existing webhook.
	// Returns domain.ErrNotFound if the webhook does not exist.
	Update(ctx context.Context, webhook *domain.Webhook) error

	// Delete permanently removes a webhook by its ID.
	// This also removes all associated delivery records.
	// Returns domain.ErrNotFound if the webhook does not exist.
	Delete(ctx context.Context, id domain.ID) error

	// DeleteByUser removes all webhooks owned by a user.
	// Returns the number of deleted webhooks.
	DeleteByUser(ctx context.Context, userID domain.ID) (int64, error)

	// Exists checks if a webhook with the given ID exists.
	Exists(ctx context.Context, id domain.ID) (bool, error)

	// ExistsByURL checks if a webhook with the given URL exists for a user.
	ExistsByURL(ctx context.Context, userID domain.ID, url string) (bool, error)

	// Count returns the total number of webhooks matching the filter.
	Count(ctx context.Context, filter *WebhookFilter) (int64, error)

	// CountByUser returns the number of webhooks owned by a user.
	CountByUser(ctx context.Context, userID domain.ID) (int64, error)

	// CountByStatus returns webhook counts grouped by status.
	CountByStatus(ctx context.Context) (map[domain.WebhookStatus]int64, error)

	// Activate activates a webhook.
	// Returns domain.ErrNotFound if the webhook does not exist.
	Activate(ctx context.Context, id domain.ID) error

	// Deactivate deactivates a webhook.
	// Returns domain.ErrNotFound if the webhook does not exist.
	Deactivate(ctx context.Context, id domain.ID) error

	// MarkAsFailed marks a webhook as failed.
	// Returns domain.ErrNotFound if the webhook does not exist.
	MarkAsFailed(ctx context.Context, id domain.ID, errorMsg string) error

	// UpdateStatus updates a webhook's status.
	// Returns domain.ErrNotFound if the webhook does not exist.
	UpdateStatus(ctx context.Context, id domain.ID, status domain.WebhookStatus) error

	// UpdateSecret updates a webhook's secret.
	// Returns domain.ErrNotFound if the webhook does not exist.
	UpdateSecret(ctx context.Context, id domain.ID, secret string) error

	// AddEvent adds an event subscription to a webhook.
	// Returns true if the event was added, false if already subscribed.
	// Returns domain.ErrNotFound if the webhook does not exist.
	AddEvent(ctx context.Context, id domain.ID, event domain.WebhookEvent) (bool, error)

	// RemoveEvent removes an event subscription from a webhook.
	// Returns true if the event was removed, false if not subscribed.
	// Returns domain.ErrNotFound if the webhook does not exist.
	RemoveEvent(ctx context.Context, id domain.ID, event domain.WebhookEvent) (bool, error)

	// SetEvents replaces all event subscriptions for a webhook.
	// Returns domain.ErrNotFound if the webhook does not exist.
	SetEvents(ctx context.Context, id domain.ID, events []domain.WebhookEvent) error

	// RecordSuccess records a successful delivery for a webhook.
	// Returns domain.ErrNotFound if the webhook does not exist.
	RecordSuccess(ctx context.Context, id domain.ID) error

	// RecordFailure records a failed delivery for a webhook.
	// Returns domain.ErrNotFound if the webhook does not exist.
	RecordFailure(ctx context.Context, id domain.ID, errorMsg string) error

	// ResetRetryCount resets the retry counter for a webhook.
	// Returns domain.ErrNotFound if the webhook does not exist.
	ResetRetryCount(ctx context.Context, id domain.ID) error

	// GetActiveWebhooks retrieves all active webhooks.
	GetActiveWebhooks(ctx context.Context, opts *ListOptions) (*ListResult[*domain.Webhook], error)

	// GetFailedWebhooks retrieves all webhooks in failed status.
	GetFailedWebhooks(ctx context.Context, opts *ListOptions) (*ListResult[*domain.Webhook], error)

	// GetWebhooksNeedingRetry retrieves webhooks that should be retried.
	// These are webhooks that have failed but haven't exceeded max retries.
	GetWebhooksNeedingRetry(ctx context.Context) ([]*domain.Webhook, error)

	// Search performs a text search across webhook fields.
	// Searches in name and URL.
	Search(ctx context.Context, query string, opts *ListOptions) (*ListResult[*domain.Webhook], error)

	// BulkActivate activates multiple webhooks.
	BulkActivate(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// BulkDeactivate deactivates multiple webhooks.
	BulkDeactivate(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// BulkDelete permanently removes multiple webhooks.
	BulkDelete(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// CreateDelivery creates a new webhook delivery record.
	CreateDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error

	// GetDelivery retrieves a delivery record by its ID.
	// Returns domain.ErrNotFound if the delivery does not exist.
	GetDelivery(ctx context.Context, id domain.ID) (*domain.WebhookDelivery, error)

	// ListDeliveries retrieves delivery records for a webhook.
	ListDeliveries(ctx context.Context, webhookID domain.ID, opts *ListOptions) (*ListResult[*domain.WebhookDelivery], error)

	// ListDeliveriesByEvent retrieves delivery records filtered by event type.
	ListDeliveriesByEvent(ctx context.Context, webhookID domain.ID, event domain.WebhookEvent, opts *ListOptions) (*ListResult[*domain.WebhookDelivery], error)

	// ListRecentDeliveries retrieves recent delivery records (last N hours).
	ListRecentDeliveries(ctx context.Context, webhookID domain.ID, hours int) ([]*domain.WebhookDelivery, error)

	// ListFailedDeliveries retrieves failed delivery records for a webhook.
	ListFailedDeliveries(ctx context.Context, webhookID domain.ID, opts *ListOptions) (*ListResult[*domain.WebhookDelivery], error)

	// DeleteDeliveries removes all delivery records for a webhook.
	DeleteDeliveries(ctx context.Context, webhookID domain.ID) (int64, error)

	// DeleteOldDeliveries removes delivery records older than the specified days.
	// Returns the number of deleted records.
	DeleteOldDeliveries(ctx context.Context, olderThanDays int) (int64, error)

	// GetDeliveryStats retrieves delivery statistics for a webhook.
	GetDeliveryStats(ctx context.Context, webhookID domain.ID) (*WebhookDeliveryStats, error)

	// GetDeliveryStatsByDateRange retrieves delivery statistics within a date range.
	GetDeliveryStatsByDateRange(ctx context.Context, webhookID domain.ID, dateRange *DateRangeFilter) (*WebhookDeliveryStats, error)

	// GetDailyDeliveryCounts retrieves delivery counts grouped by day.
	GetDailyDeliveryCounts(ctx context.Context, webhookID domain.ID, dateRange *DateRangeFilter) ([]DateCount, error)

	// GetEventDeliveryCounts retrieves delivery counts grouped by event type.
	GetEventDeliveryCounts(ctx context.Context, webhookID domain.ID) ([]EventCount, error)
}

// WebhookFilter provides filtering options for webhook queries.
type WebhookFilter struct {
	// IDs filters by specific webhook IDs.
	IDs []domain.ID

	// UserID filters by owner user ID.
	UserID *domain.ID

	// UserIDs filters by multiple owner user IDs (OR condition).
	UserIDs []domain.ID

	// Status filters by webhook status.
	Status *domain.WebhookStatus

	// Statuses filters by multiple statuses (OR condition).
	Statuses []domain.WebhookStatus

	// Event filters by subscribed event.
	Event *domain.WebhookEvent

	// Events filters by any of the specified events (OR condition).
	Events []domain.WebhookEvent

	// URL filters by exact URL match.
	URL string

	// URLContains filters by partial URL match.
	URLContains string

	// Name filters by exact name match.
	Name string

	// NameContains filters by partial name match.
	NameContains string

	// Search performs text search on name and URL.
	Search string

	// HasFailures filters webhooks that have had failures.
	HasFailures *bool

	// LastTriggeredAfter filters webhooks triggered after this timestamp.
	LastTriggeredAfter *domain.Timestamp

	// LastTriggeredBefore filters webhooks triggered before this timestamp.
	LastTriggeredBefore *domain.Timestamp

	// NeverTriggered filters webhooks that have never been triggered.
	NeverTriggered *bool

	// CreatedAfter filters webhooks created after this timestamp.
	CreatedAfter *domain.Timestamp

	// CreatedBefore filters webhooks created before this timestamp.
	CreatedBefore *domain.Timestamp
}

// IsEmpty returns true if no filter criteria are set.
func (f *WebhookFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.IDs) == 0 &&
		f.UserID == nil &&
		len(f.UserIDs) == 0 &&
		f.Status == nil &&
		len(f.Statuses) == 0 &&
		f.Event == nil &&
		len(f.Events) == 0 &&
		f.URL == "" &&
		f.URLContains == "" &&
		f.Name == "" &&
		f.NameContains == "" &&
		f.Search == "" &&
		f.HasFailures == nil &&
		f.LastTriggeredAfter == nil &&
		f.LastTriggeredBefore == nil &&
		f.NeverTriggered == nil &&
		f.CreatedAfter == nil &&
		f.CreatedBefore == nil
}

// WebhookSortField represents the available fields for sorting webhooks.
type WebhookSortField string

const (
	// WebhookSortByName sorts by name.
	WebhookSortByName WebhookSortField = "name"

	// WebhookSortByURL sorts by URL.
	WebhookSortByURL WebhookSortField = "url"

	// WebhookSortByStatus sorts by status.
	WebhookSortByStatus WebhookSortField = "status"

	// WebhookSortBySuccessCount sorts by success count.
	WebhookSortBySuccessCount WebhookSortField = "successCount"

	// WebhookSortByFailureCount sorts by failure count.
	WebhookSortByFailureCount WebhookSortField = "failureCount"

	// WebhookSortByLastTriggered sorts by last triggered timestamp.
	WebhookSortByLastTriggered WebhookSortField = "lastTriggeredAt"

	// WebhookSortByCreatedAt sorts by creation timestamp.
	WebhookSortByCreatedAt WebhookSortField = "createdAt"

	// WebhookSortByUpdatedAt sorts by update timestamp.
	WebhookSortByUpdatedAt WebhookSortField = "updatedAt"
)

// IsValid returns true if the sort field is a recognized value.
func (f WebhookSortField) IsValid() bool {
	switch f {
	case WebhookSortByName, WebhookSortByURL, WebhookSortByStatus,
		WebhookSortBySuccessCount, WebhookSortByFailureCount,
		WebhookSortByLastTriggered, WebhookSortByCreatedAt, WebhookSortByUpdatedAt:
		return true
	default:
		return false
	}
}

// String returns the string representation of the sort field.
func (f WebhookSortField) String() string {
	return string(f)
}

// WebhookDeliveryStats represents statistics for webhook deliveries.
type WebhookDeliveryStats struct {
	// TotalDeliveries is the total number of delivery attempts.
	TotalDeliveries int64

	// SuccessfulDeliveries is the number of successful deliveries.
	SuccessfulDeliveries int64

	// FailedDeliveries is the number of failed deliveries.
	FailedDeliveries int64

	// SuccessRate is the percentage of successful deliveries.
	SuccessRate float64

	// AverageDuration is the average delivery duration in milliseconds.
	AverageDuration float64

	// MaxDuration is the maximum delivery duration in milliseconds.
	MaxDuration int64

	// MinDuration is the minimum delivery duration in milliseconds.
	MinDuration int64

	// LastDeliveryAt is the timestamp of the most recent delivery.
	LastDeliveryAt *domain.Timestamp

	// LastSuccessAt is the timestamp of the most recent successful delivery.
	LastSuccessAt *domain.Timestamp

	// LastFailureAt is the timestamp of the most recent failed delivery.
	LastFailureAt *domain.Timestamp
}

// EventCount represents a count grouped by webhook event.
type EventCount struct {
	// Event is the webhook event type.
	Event domain.WebhookEvent

	// Count is the number of deliveries for this event.
	Count int64

	// SuccessCount is the number of successful deliveries.
	SuccessCount int64

	// FailureCount is the number of failed deliveries.
	FailureCount int64
}
