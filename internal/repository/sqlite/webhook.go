package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// WebhookRepository implements the repository.WebhookRepository interface for SQLite.
type WebhookRepository struct {
	repo *Repository
}

// webhookRow is the database representation of a webhook.
type webhookRow struct {
	ID              string         `db:"id"`
	UserID          string         `db:"user_id"`
	Name            string         `db:"name"`
	URL             string         `db:"url"`
	Secret          sql.NullString `db:"secret"`
	Events          string         `db:"events"`
	Status          string         `db:"status"`
	Headers         sql.NullString `db:"headers"`
	RetryCount      int            `db:"retry_count"`
	MaxRetries      int            `db:"max_retries"`
	TimeoutSeconds  int            `db:"timeout_seconds"`
	LastTriggeredAt sql.NullTime   `db:"last_triggered_at"`
	LastSuccessAt   sql.NullTime   `db:"last_success_at"`
	LastFailureAt   sql.NullTime   `db:"last_failure_at"`
	LastError       sql.NullString `db:"last_error"`
	SuccessCount    int64          `db:"success_count"`
	FailureCount    int64          `db:"failure_count"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

// webhookDeliveryRow is the database representation of a webhook delivery.
type webhookDeliveryRow struct {
	ID            string         `db:"id"`
	WebhookID     string         `db:"webhook_id"`
	Event         string         `db:"event"`
	Payload       string         `db:"payload"`
	StatusCode    sql.NullInt32  `db:"status_code"`
	Response      sql.NullString `db:"response"`
	Error         sql.NullString `db:"error"`
	Success       bool           `db:"success"`
	Duration      int64          `db:"duration"`
	AttemptNumber int            `db:"attempt_number"`
	CreatedAt     time.Time      `db:"created_at"`
}

// NewWebhookRepository creates a new SQLite webhook repository.
func NewWebhookRepository(repo *Repository) *WebhookRepository {
	return &WebhookRepository{repo: repo}
}

// toWebhook converts a webhookRow to a domain.Webhook.
func (r *webhookRow) toWebhook() *domain.Webhook {
	webhook := &domain.Webhook{
		ID:             domain.ID(r.ID),
		UserID:         domain.ID(r.UserID),
		Name:           r.Name,
		URL:            r.URL,
		Status:         domain.WebhookStatus(r.Status),
		RetryCount:     r.RetryCount,
		MaxRetries:     r.MaxRetries,
		TimeoutSeconds: r.TimeoutSeconds,
		SuccessCount:   r.SuccessCount,
		FailureCount:   r.FailureCount,
		CreatedAt:      domain.Timestamp{Time: r.CreatedAt},
		UpdatedAt:      domain.Timestamp{Time: r.UpdatedAt},
		Headers:        make(map[string]string),
		Events:         make([]domain.WebhookEvent, 0),
	}

	if r.Secret.Valid {
		webhook.Secret = r.Secret.String
	}
	if r.LastError.Valid {
		webhook.LastError = r.LastError.String
	}
	if r.LastTriggeredAt.Valid {
		ts := domain.Timestamp{Time: r.LastTriggeredAt.Time}
		webhook.LastTriggeredAt = &ts
	}
	if r.LastSuccessAt.Valid {
		ts := domain.Timestamp{Time: r.LastSuccessAt.Time}
		webhook.LastSuccessAt = &ts
	}
	if r.LastFailureAt.Valid {
		ts := domain.Timestamp{Time: r.LastFailureAt.Time}
		webhook.LastFailureAt = &ts
	}
	if r.Headers.Valid && r.Headers.String != "" {
		_ = json.Unmarshal([]byte(r.Headers.String), &webhook.Headers)
	}
	if r.Events != "" {
		var events []string
		_ = json.Unmarshal([]byte(r.Events), &events)
		for _, e := range events {
			webhook.Events = append(webhook.Events, domain.WebhookEvent(e))
		}
	}

	return webhook
}

// toWebhookDelivery converts a webhookDeliveryRow to a domain.WebhookDelivery.
func (r *webhookDeliveryRow) toWebhookDelivery() *domain.WebhookDelivery {
	delivery := &domain.WebhookDelivery{
		ID:            domain.ID(r.ID),
		WebhookID:     domain.ID(r.WebhookID),
		Event:         domain.WebhookEvent(r.Event),
		Payload:       r.Payload,
		Success:       r.Success,
		Duration:      r.Duration,
		AttemptNumber: r.AttemptNumber,
		CreatedAt:     domain.Timestamp{Time: r.CreatedAt},
	}

	if r.StatusCode.Valid {
		delivery.StatusCode = int(r.StatusCode.Int32)
	}
	if r.Response.Valid {
		delivery.Response = r.Response.String
	}
	if r.Error.Valid {
		delivery.Error = r.Error.String
	}

	return delivery
}

// GetByID retrieves a webhook by its unique identifier.
func (w *WebhookRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Webhook, error) {
	query := `SELECT id, user_id, name, url, secret, events, status, headers, 
		retry_count, max_retries, timeout_seconds, last_triggered_at, last_success_at, 
		last_failure_at, last_error, success_count, failure_count, created_at, updated_at 
		FROM webhooks WHERE id = ?`

	var row webhookRow
	if err := w.repo.db().GetContext(ctx, &row, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("webhook", string(id))
		}
		return nil, fmt.Errorf("failed to get webhook by ID: %w", err)
	}

	return row.toWebhook(), nil
}

// List retrieves webhooks with optional filtering, sorting, and pagination.
func (w *WebhookRepository) List(ctx context.Context, filter *repository.WebhookFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	query, args := w.buildListQuery(filter, opts, false)
	countQuery, countArgs := w.buildListQuery(filter, opts, true)

	var total int64
	if err := w.repo.db().GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, fmt.Errorf("failed to count webhooks: %w", err)
	}

	var rows []webhookRow
	if err := w.repo.db().SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	webhooks := make([]*domain.Webhook, len(rows))
	for i, row := range rows {
		webhooks[i] = row.toWebhook()
	}

	result := &repository.ListResult[*domain.Webhook]{
		Items: webhooks,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// buildListQuery builds the SQL query for listing webhooks.
func (w *WebhookRepository) buildListQuery(filter *repository.WebhookFilter, opts *repository.ListOptions, countOnly bool) (string, []interface{}) {
	var sb strings.Builder
	args := make([]interface{}, 0)

	if countOnly {
		sb.WriteString("SELECT COUNT(*) FROM webhooks WHERE 1=1")
	} else {
		sb.WriteString(`SELECT id, user_id, name, url, secret, events, status, headers, 
			retry_count, max_retries, timeout_seconds, last_triggered_at, last_success_at, 
			last_failure_at, last_error, success_count, failure_count, created_at, updated_at 
			FROM webhooks WHERE 1=1`)
	}

	if filter != nil {
		if len(filter.IDs) > 0 {
			placeholders := make([]string, len(filter.IDs))
			for i, id := range filter.IDs {
				placeholders[i] = "?"
				args = append(args, string(id))
			}
			sb.WriteString(fmt.Sprintf(" AND id IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.UserID != nil {
			sb.WriteString(" AND user_id = ?")
			args = append(args, string(*filter.UserID))
		}

		if len(filter.UserIDs) > 0 {
			placeholders := make([]string, len(filter.UserIDs))
			for i, id := range filter.UserIDs {
				placeholders[i] = "?"
				args = append(args, string(id))
			}
			sb.WriteString(fmt.Sprintf(" AND user_id IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.Status != nil {
			sb.WriteString(" AND status = ?")
			args = append(args, string(*filter.Status))
		}

		if len(filter.Statuses) > 0 {
			placeholders := make([]string, len(filter.Statuses))
			for i, status := range filter.Statuses {
				placeholders[i] = "?"
				args = append(args, string(status))
			}
			sb.WriteString(fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.Event != nil {
			sb.WriteString(" AND events LIKE ?")
			args = append(args, "%"+string(*filter.Event)+"%")
		}

		if len(filter.Events) > 0 {
			conditions := make([]string, len(filter.Events))
			for i, event := range filter.Events {
				conditions[i] = "events LIKE ?"
				args = append(args, "%"+string(event)+"%")
			}
			sb.WriteString(fmt.Sprintf(" AND (%s)", strings.Join(conditions, " OR ")))
		}

		if filter.URL != "" {
			sb.WriteString(" AND url = ?")
			args = append(args, filter.URL)
		}

		if filter.URLContains != "" {
			sb.WriteString(" AND url LIKE ?")
			args = append(args, "%"+filter.URLContains+"%")
		}

		if filter.Name != "" {
			sb.WriteString(" AND name = ?")
			args = append(args, filter.Name)
		}

		if filter.NameContains != "" {
			sb.WriteString(" AND name LIKE ?")
			args = append(args, "%"+filter.NameContains+"%")
		}

		if filter.Search != "" {
			sb.WriteString(" AND (name LIKE ? OR url LIKE ?)")
			pattern := "%" + filter.Search + "%"
			args = append(args, pattern, pattern)
		}

		if filter.HasFailures != nil {
			if *filter.HasFailures {
				sb.WriteString(" AND failure_count > 0")
			} else {
				sb.WriteString(" AND failure_count = 0")
			}
		}

		if filter.LastTriggeredAfter != nil {
			sb.WriteString(" AND last_triggered_at > ?")
			args = append(args, filter.LastTriggeredAfter.Time)
		}

		if filter.LastTriggeredBefore != nil {
			sb.WriteString(" AND last_triggered_at < ?")
			args = append(args, filter.LastTriggeredBefore.Time)
		}

		if filter.NeverTriggered != nil && *filter.NeverTriggered {
			sb.WriteString(" AND last_triggered_at IS NULL")
		}

		if filter.CreatedAfter != nil {
			sb.WriteString(" AND created_at > ?")
			args = append(args, filter.CreatedAfter.Time)
		}

		if filter.CreatedBefore != nil {
			sb.WriteString(" AND created_at < ?")
			args = append(args, filter.CreatedBefore.Time)
		}
	}

	if !countOnly {
		if opts != nil && opts.Sort != nil {
			field := w.mapSortField(opts.Sort.Field)
			order := "ASC"
			if opts.Sort.Order == domain.SortDesc {
				order = "DESC"
			}
			sb.WriteString(fmt.Sprintf(" ORDER BY %s %s", field, order))
		} else {
			sb.WriteString(" ORDER BY created_at DESC")
		}

		if opts != nil && opts.Pagination != nil {
			opts.Pagination.Normalize()
			sb.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset()))
		}
	}

	return sb.String(), args
}

// mapSortField maps repository sort field to database column.
func (w *WebhookRepository) mapSortField(field string) string {
	switch field {
	case "name":
		return "name"
	case "url":
		return "url"
	case "status":
		return "status"
	case "successCount":
		return "success_count"
	case "failureCount":
		return "failure_count"
	case "lastTriggeredAt":
		return "last_triggered_at"
	case "createdAt":
		return "created_at"
	case "updatedAt":
		return "updated_at"
	default:
		return "created_at"
	}
}

// ListByUser retrieves all webhooks owned by a specific user.
func (w *WebhookRepository) ListByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	filter := &repository.WebhookFilter{UserID: &userID}
	return w.List(ctx, filter, opts)
}

// ListByEvent retrieves all webhooks subscribed to a specific event.
func (w *WebhookRepository) ListByEvent(ctx context.Context, event domain.WebhookEvent) ([]*domain.Webhook, error) {
	filter := &repository.WebhookFilter{Event: &event}
	result, err := w.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// ListActiveByEvent retrieves all active webhooks subscribed to a specific event.
func (w *WebhookRepository) ListActiveByEvent(ctx context.Context, event domain.WebhookEvent) ([]*domain.Webhook, error) {
	status := domain.WebhookStatusActive
	filter := &repository.WebhookFilter{
		Event:  &event,
		Status: &status,
	}
	result, err := w.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// Create creates a new webhook.
func (w *WebhookRepository) Create(ctx context.Context, webhook *domain.Webhook) error {
	exists, err := w.ExistsByURL(ctx, webhook.UserID, webhook.URL)
	if err != nil {
		return err
	}
	if exists {
		return domain.NewAlreadyExistsError("webhook", "url", webhook.URL)
	}

	query := `INSERT INTO webhooks (id, user_id, name, url, secret, events, status, 
		headers, retry_count, max_retries, timeout_seconds, last_triggered_at, 
		last_success_at, last_failure_at, last_error, success_count, failure_count, 
		created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var secret, headers, lastError sql.NullString
	if webhook.Secret != "" {
		secret = sql.NullString{String: webhook.Secret, Valid: true}
	}
	if len(webhook.Headers) > 0 {
		data, _ := json.Marshal(webhook.Headers)
		headers = sql.NullString{String: string(data), Valid: true}
	}
	if webhook.LastError != "" {
		lastError = sql.NullString{String: webhook.LastError, Valid: true}
	}

	// Serialize events
	events := make([]string, len(webhook.Events))
	for i, e := range webhook.Events {
		events[i] = string(e)
	}
	eventsJSON, _ := json.Marshal(events)

	var lastTriggeredAt, lastSuccessAt, lastFailureAt sql.NullTime
	if webhook.LastTriggeredAt != nil {
		lastTriggeredAt = sql.NullTime{Time: webhook.LastTriggeredAt.Time, Valid: true}
	}
	if webhook.LastSuccessAt != nil {
		lastSuccessAt = sql.NullTime{Time: webhook.LastSuccessAt.Time, Valid: true}
	}
	if webhook.LastFailureAt != nil {
		lastFailureAt = sql.NullTime{Time: webhook.LastFailureAt.Time, Valid: true}
	}

	_, err = w.repo.db().ExecContext(ctx, query,
		string(webhook.ID),
		string(webhook.UserID),
		webhook.Name,
		webhook.URL,
		secret,
		string(eventsJSON),
		string(webhook.Status),
		headers,
		webhook.RetryCount,
		webhook.MaxRetries,
		webhook.TimeoutSeconds,
		lastTriggeredAt,
		lastSuccessAt,
		lastFailureAt,
		lastError,
		webhook.SuccessCount,
		webhook.FailureCount,
		webhook.CreatedAt.Time,
		webhook.UpdatedAt.Time,
	)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil
}

// Update updates an existing webhook.
func (w *WebhookRepository) Update(ctx context.Context, webhook *domain.Webhook) error {
	exists, err := w.Exists(ctx, webhook.ID)
	if err != nil {
		return err
	}
	if !exists {
		return domain.NewNotFoundError("webhook", string(webhook.ID))
	}

	query := `UPDATE webhooks SET user_id = ?, name = ?, url = ?, secret = ?, 
		events = ?, status = ?, headers = ?, retry_count = ?, max_retries = ?, 
		timeout_seconds = ?, last_triggered_at = ?, last_success_at = ?, 
		last_failure_at = ?, last_error = ?, success_count = ?, failure_count = ?, 
		updated_at = ? WHERE id = ?`

	var secret, headers, lastError sql.NullString
	if webhook.Secret != "" {
		secret = sql.NullString{String: webhook.Secret, Valid: true}
	}
	if len(webhook.Headers) > 0 {
		data, _ := json.Marshal(webhook.Headers)
		headers = sql.NullString{String: string(data), Valid: true}
	}
	if webhook.LastError != "" {
		lastError = sql.NullString{String: webhook.LastError, Valid: true}
	}

	events := make([]string, len(webhook.Events))
	for i, e := range webhook.Events {
		events[i] = string(e)
	}
	eventsJSON, _ := json.Marshal(events)

	var lastTriggeredAt, lastSuccessAt, lastFailureAt sql.NullTime
	if webhook.LastTriggeredAt != nil {
		lastTriggeredAt = sql.NullTime{Time: webhook.LastTriggeredAt.Time, Valid: true}
	}
	if webhook.LastSuccessAt != nil {
		lastSuccessAt = sql.NullTime{Time: webhook.LastSuccessAt.Time, Valid: true}
	}
	if webhook.LastFailureAt != nil {
		lastFailureAt = sql.NullTime{Time: webhook.LastFailureAt.Time, Valid: true}
	}

	_, err = w.repo.db().ExecContext(ctx, query,
		string(webhook.UserID),
		webhook.Name,
		webhook.URL,
		secret,
		string(eventsJSON),
		string(webhook.Status),
		headers,
		webhook.RetryCount,
		webhook.MaxRetries,
		webhook.TimeoutSeconds,
		lastTriggeredAt,
		lastSuccessAt,
		lastFailureAt,
		lastError,
		webhook.SuccessCount,
		webhook.FailureCount,
		time.Now().UTC(),
		string(webhook.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	return nil
}

// Delete permanently removes a webhook by its ID.
func (w *WebhookRepository) Delete(ctx context.Context, id domain.ID) error {
	result, err := w.repo.db().ExecContext(ctx, "DELETE FROM webhooks WHERE id = ?", string(id))
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// DeleteByUser removes all webhooks owned by a user.
func (w *WebhookRepository) DeleteByUser(ctx context.Context, userID domain.ID) (int64, error) {
	result, err := w.repo.db().ExecContext(ctx, "DELETE FROM webhooks WHERE user_id = ?", string(userID))
	if err != nil {
		return 0, fmt.Errorf("failed to delete webhooks by user: %w", err)
	}

	return result.RowsAffected()
}

// Exists checks if a webhook with the given ID exists.
func (w *WebhookRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM webhooks WHERE id = ?)`

	var exists bool
	if err := w.repo.db().GetContext(ctx, &exists, query, string(id)); err != nil {
		return false, fmt.Errorf("failed to check webhook existence: %w", err)
	}

	return exists, nil
}

// ExistsByURL checks if a webhook with the given URL exists for a user.
func (w *WebhookRepository) ExistsByURL(ctx context.Context, userID domain.ID, url string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM webhooks WHERE user_id = ? AND url = ?)`

	var exists bool
	if err := w.repo.db().GetContext(ctx, &exists, query, string(userID), url); err != nil {
		return false, fmt.Errorf("failed to check URL existence: %w", err)
	}

	return exists, nil
}

// Count returns the total number of webhooks matching the filter.
func (w *WebhookRepository) Count(ctx context.Context, filter *repository.WebhookFilter) (int64, error) {
	query, args := w.buildListQuery(filter, nil, true)

	var count int64
	if err := w.repo.db().GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count webhooks: %w", err)
	}

	return count, nil
}

// CountByUser returns the number of webhooks owned by a user.
func (w *WebhookRepository) CountByUser(ctx context.Context, userID domain.ID) (int64, error) {
	query := `SELECT COUNT(*) FROM webhooks WHERE user_id = ?`

	var count int64
	if err := w.repo.db().GetContext(ctx, &count, query, string(userID)); err != nil {
		return 0, fmt.Errorf("failed to count webhooks by user: %w", err)
	}

	return count, nil
}

// CountByStatus returns webhook counts grouped by status.
func (w *WebhookRepository) CountByStatus(ctx context.Context) (map[domain.WebhookStatus]int64, error) {
	query := `SELECT status, COUNT(*) as count FROM webhooks GROUP BY status`

	type statusCount struct {
		Status string `db:"status"`
		Count  int64  `db:"count"`
	}

	var counts []statusCount
	if err := w.repo.db().SelectContext(ctx, &counts, query); err != nil {
		return nil, fmt.Errorf("failed to count webhooks by status: %w", err)
	}

	result := make(map[domain.WebhookStatus]int64)
	for _, sc := range counts {
		result[domain.WebhookStatus(sc.Status)] = sc.Count
	}

	return result, nil
}

// Activate activates a webhook.
func (w *WebhookRepository) Activate(ctx context.Context, id domain.ID) error {
	return w.UpdateStatus(ctx, id, domain.WebhookStatusActive)
}

// Deactivate deactivates a webhook.
func (w *WebhookRepository) Deactivate(ctx context.Context, id domain.ID) error {
	return w.UpdateStatus(ctx, id, domain.WebhookStatusInactive)
}

// MarkAsFailed marks a webhook as failed.
func (w *WebhookRepository) MarkAsFailed(ctx context.Context, id domain.ID, errorMsg string) error {
	query := `UPDATE webhooks SET status = ?, last_error = ?, last_failure_at = ?, 
		updated_at = ? WHERE id = ?`

	now := time.Now().UTC()
	result, err := w.repo.db().ExecContext(ctx, query,
		string(domain.WebhookStatusFailed),
		errorMsg,
		now,
		now,
		string(id),
	)
	if err != nil {
		return fmt.Errorf("failed to mark webhook as failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// UpdateStatus updates a webhook's status.
func (w *WebhookRepository) UpdateStatus(ctx context.Context, id domain.ID, status domain.WebhookStatus) error {
	query := `UPDATE webhooks SET status = ?, updated_at = ? WHERE id = ?`

	result, err := w.repo.db().ExecContext(ctx, query, string(status), time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// UpdateSecret updates a webhook's secret.
func (w *WebhookRepository) UpdateSecret(ctx context.Context, id domain.ID, secret string) error {
	query := `UPDATE webhooks SET secret = ?, updated_at = ? WHERE id = ?`

	var secretVal sql.NullString
	if secret != "" {
		secretVal = sql.NullString{String: secret, Valid: true}
	}

	result, err := w.repo.db().ExecContext(ctx, query, secretVal, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// AddEvent adds an event subscription to a webhook.
func (w *WebhookRepository) AddEvent(ctx context.Context, id domain.ID, event domain.WebhookEvent) (bool, error) {
	webhook, err := w.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	if webhook.SubscribesToEvent(event) {
		return false, nil
	}

	webhook.AddEvent(event)
	return true, w.SetEvents(ctx, id, webhook.Events)
}

// RemoveEvent removes an event subscription from a webhook.
func (w *WebhookRepository) RemoveEvent(ctx context.Context, id domain.ID, event domain.WebhookEvent) (bool, error) {
	webhook, err := w.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	if !webhook.SubscribesToEvent(event) {
		return false, nil
	}

	webhook.RemoveEvent(event)
	return true, w.SetEvents(ctx, id, webhook.Events)
}

// SetEvents replaces all event subscriptions for a webhook.
func (w *WebhookRepository) SetEvents(ctx context.Context, id domain.ID, events []domain.WebhookEvent) error {
	eventStrings := make([]string, len(events))
	for i, e := range events {
		eventStrings[i] = string(e)
	}
	eventsJSON, _ := json.Marshal(eventStrings)

	query := `UPDATE webhooks SET events = ?, updated_at = ? WHERE id = ?`

	result, err := w.repo.db().ExecContext(ctx, query, string(eventsJSON), time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to set events: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// RecordSuccess records a successful delivery for a webhook.
func (w *WebhookRepository) RecordSuccess(ctx context.Context, id domain.ID) error {
	now := time.Now().UTC()
	query := `UPDATE webhooks SET success_count = success_count + 1, 
		last_triggered_at = ?, last_success_at = ?, retry_count = 0, 
		last_error = NULL, updated_at = ? WHERE id = ?`

	result, err := w.repo.db().ExecContext(ctx, query, now, now, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to record success: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// RecordFailure records a failed delivery for a webhook.
func (w *WebhookRepository) RecordFailure(ctx context.Context, id domain.ID, errorMsg string) error {
	now := time.Now().UTC()
	query := `UPDATE webhooks SET failure_count = failure_count + 1, 
		retry_count = retry_count + 1, last_triggered_at = ?, 
		last_failure_at = ?, last_error = ?, updated_at = ? WHERE id = ?`

	result, err := w.repo.db().ExecContext(ctx, query, now, now, errorMsg, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to record failure: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// ResetRetryCount resets the retry counter for a webhook.
func (w *WebhookRepository) ResetRetryCount(ctx context.Context, id domain.ID) error {
	query := `UPDATE webhooks SET retry_count = 0, updated_at = ? WHERE id = ?`

	result, err := w.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to reset retry count: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// GetActiveWebhooks retrieves all active webhooks.
func (w *WebhookRepository) GetActiveWebhooks(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	status := domain.WebhookStatusActive
	filter := &repository.WebhookFilter{Status: &status}
	return w.List(ctx, filter, opts)
}

// GetFailedWebhooks retrieves all webhooks in failed status.
func (w *WebhookRepository) GetFailedWebhooks(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	status := domain.WebhookStatusFailed
	filter := &repository.WebhookFilter{Status: &status}
	return w.List(ctx, filter, opts)
}

// GetWebhooksNeedingRetry retrieves webhooks that should be retried.
func (w *WebhookRepository) GetWebhooksNeedingRetry(ctx context.Context) ([]*domain.Webhook, error) {
	query := `SELECT id, user_id, name, url, secret, events, status, headers, 
		retry_count, max_retries, timeout_seconds, last_triggered_at, last_success_at, 
		last_failure_at, last_error, success_count, failure_count, created_at, updated_at 
		FROM webhooks WHERE status != ? AND retry_count < max_retries AND last_failure_at IS NOT NULL`

	var rows []webhookRow
	if err := w.repo.db().SelectContext(ctx, &rows, query, string(domain.WebhookStatusFailed)); err != nil {
		return nil, fmt.Errorf("failed to get webhooks needing retry: %w", err)
	}

	webhooks := make([]*domain.Webhook, len(rows))
	for i, row := range rows {
		webhooks[i] = row.toWebhook()
	}

	return webhooks, nil
}

// Search performs a text search across webhook fields.
func (w *WebhookRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	filter := &repository.WebhookFilter{Search: query}
	return w.List(ctx, filter, opts)
}

// BulkActivate activates multiple webhooks.
func (w *WebhookRepository) BulkActivate(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := w.Activate(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkDeactivate deactivates multiple webhooks.
func (w *WebhookRepository) BulkDeactivate(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := w.Deactivate(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkDelete permanently removes multiple webhooks.
func (w *WebhookRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := w.Delete(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// CreateDelivery creates a new webhook delivery record.
func (w *WebhookRepository) CreateDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error {
	query := `INSERT INTO webhook_deliveries (id, webhook_id, event, payload, 
		status_code, response, error, success, duration, attempt_number, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var statusCode sql.NullInt32
	if delivery.StatusCode != 0 {
		statusCode = sql.NullInt32{Int32: int32(delivery.StatusCode), Valid: true}
	}

	var response, errMsg sql.NullString
	if delivery.Response != "" {
		response = sql.NullString{String: delivery.Response, Valid: true}
	}
	if delivery.Error != "" {
		errMsg = sql.NullString{String: delivery.Error, Valid: true}
	}

	_, err := w.repo.db().ExecContext(ctx, query,
		string(delivery.ID),
		string(delivery.WebhookID),
		string(delivery.Event),
		delivery.Payload,
		statusCode,
		response,
		errMsg,
		delivery.Success,
		delivery.Duration,
		delivery.AttemptNumber,
		delivery.CreatedAt.Time,
	)
	if err != nil {
		return fmt.Errorf("failed to create delivery: %w", err)
	}

	return nil
}

// GetDelivery retrieves a delivery record by its ID.
func (w *WebhookRepository) GetDelivery(ctx context.Context, id domain.ID) (*domain.WebhookDelivery, error) {
	query := `SELECT id, webhook_id, event, payload, status_code, response, 
		error, success, duration, attempt_number, created_at 
		FROM webhook_deliveries WHERE id = ?`

	var row webhookDeliveryRow
	if err := w.repo.db().GetContext(ctx, &row, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("webhook delivery", string(id))
		}
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}

	return row.toWebhookDelivery(), nil
}

// ListDeliveries retrieves delivery records for a webhook.
func (w *WebhookRepository) ListDeliveries(ctx context.Context, webhookID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	countQuery := `SELECT COUNT(*) FROM webhook_deliveries WHERE webhook_id = ?`
	var total int64
	if err := w.repo.db().GetContext(ctx, &total, countQuery, string(webhookID)); err != nil {
		return nil, fmt.Errorf("failed to count deliveries: %w", err)
	}

	query := `SELECT id, webhook_id, event, payload, status_code, response, 
		error, success, duration, attempt_number, created_at 
		FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC`

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset())
	}

	var rows []webhookDeliveryRow
	if err := w.repo.db().SelectContext(ctx, &rows, query, string(webhookID)); err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(rows))
	for i, row := range rows {
		deliveries[i] = row.toWebhookDelivery()
	}

	result := &repository.ListResult[*domain.WebhookDelivery]{
		Items: deliveries,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// ListDeliveriesByEvent retrieves delivery records filtered by event type.
func (w *WebhookRepository) ListDeliveriesByEvent(ctx context.Context, webhookID domain.ID, event domain.WebhookEvent, opts *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	countQuery := `SELECT COUNT(*) FROM webhook_deliveries WHERE webhook_id = ? AND event = ?`
	var total int64
	if err := w.repo.db().GetContext(ctx, &total, countQuery, string(webhookID), string(event)); err != nil {
		return nil, fmt.Errorf("failed to count deliveries: %w", err)
	}

	query := `SELECT id, webhook_id, event, payload, status_code, response, 
		error, success, duration, attempt_number, created_at 
		FROM webhook_deliveries WHERE webhook_id = ? AND event = ? ORDER BY created_at DESC`

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset())
	}

	var rows []webhookDeliveryRow
	if err := w.repo.db().SelectContext(ctx, &rows, query, string(webhookID), string(event)); err != nil {
		return nil, fmt.Errorf("failed to list deliveries by event: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(rows))
	for i, row := range rows {
		deliveries[i] = row.toWebhookDelivery()
	}

	result := &repository.ListResult[*domain.WebhookDelivery]{
		Items: deliveries,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// ListRecentDeliveries retrieves recent delivery records.
func (w *WebhookRepository) ListRecentDeliveries(ctx context.Context, webhookID domain.ID, hours int) ([]*domain.WebhookDelivery, error) {
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)

	query := `SELECT id, webhook_id, event, payload, status_code, response, 
		error, success, duration, attempt_number, created_at 
		FROM webhook_deliveries WHERE webhook_id = ? AND created_at > ? ORDER BY created_at DESC`

	var rows []webhookDeliveryRow
	if err := w.repo.db().SelectContext(ctx, &rows, query, string(webhookID), since); err != nil {
		return nil, fmt.Errorf("failed to list recent deliveries: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(rows))
	for i, row := range rows {
		deliveries[i] = row.toWebhookDelivery()
	}

	return deliveries, nil
}

// ListFailedDeliveries retrieves failed delivery records for a webhook.
func (w *WebhookRepository) ListFailedDeliveries(ctx context.Context, webhookID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	countQuery := `SELECT COUNT(*) FROM webhook_deliveries WHERE webhook_id = ? AND success = 0`
	var total int64
	if err := w.repo.db().GetContext(ctx, &total, countQuery, string(webhookID)); err != nil {
		return nil, fmt.Errorf("failed to count failed deliveries: %w", err)
	}

	query := `SELECT id, webhook_id, event, payload, status_code, response, 
		error, success, duration, attempt_number, created_at 
		FROM webhook_deliveries WHERE webhook_id = ? AND success = 0 ORDER BY created_at DESC`

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset())
	}

	var rows []webhookDeliveryRow
	if err := w.repo.db().SelectContext(ctx, &rows, query, string(webhookID)); err != nil {
		return nil, fmt.Errorf("failed to list failed deliveries: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(rows))
	for i, row := range rows {
		deliveries[i] = row.toWebhookDelivery()
	}

	result := &repository.ListResult[*domain.WebhookDelivery]{
		Items: deliveries,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// DeleteDeliveries removes all delivery records for a webhook.
func (w *WebhookRepository) DeleteDeliveries(ctx context.Context, webhookID domain.ID) (int64, error) {
	result, err := w.repo.db().ExecContext(ctx, "DELETE FROM webhook_deliveries WHERE webhook_id = ?", string(webhookID))
	if err != nil {
		return 0, fmt.Errorf("failed to delete deliveries: %w", err)
	}

	return result.RowsAffected()
}

// DeleteOldDeliveries removes delivery records older than the specified days.
func (w *WebhookRepository) DeleteOldDeliveries(ctx context.Context, olderThanDays int) (int64, error) {
	before := time.Now().UTC().AddDate(0, 0, -olderThanDays)

	result, err := w.repo.db().ExecContext(ctx, "DELETE FROM webhook_deliveries WHERE created_at < ?", before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old deliveries: %w", err)
	}

	return result.RowsAffected()
}

// GetDeliveryStats retrieves delivery statistics for a webhook.
func (w *WebhookRepository) GetDeliveryStats(ctx context.Context, webhookID domain.ID) (*repository.WebhookDeliveryStats, error) {
	query := `SELECT 
		COUNT(*) as total_deliveries,
		SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_deliveries,
		SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_deliveries,
		AVG(duration) as average_duration,
		MAX(duration) as max_duration,
		MIN(duration) as min_duration,
		MAX(created_at) as last_delivery_at,
		MAX(CASE WHEN success = 1 THEN created_at END) as last_success_at,
		MAX(CASE WHEN success = 0 THEN created_at END) as last_failure_at
		FROM webhook_deliveries WHERE webhook_id = ?`

	var stats struct {
		TotalDeliveries      int64          `db:"total_deliveries"`
		SuccessfulDeliveries int64          `db:"successful_deliveries"`
		FailedDeliveries     int64          `db:"failed_deliveries"`
		AverageDuration      sql.NullFloat64 `db:"average_duration"`
		MaxDuration          sql.NullInt64  `db:"max_duration"`
		MinDuration          sql.NullInt64  `db:"min_duration"`
		LastDeliveryAt       sql.NullTime   `db:"last_delivery_at"`
		LastSuccessAt        sql.NullTime   `db:"last_success_at"`
		LastFailureAt        sql.NullTime   `db:"last_failure_at"`
	}

	if err := w.repo.db().GetContext(ctx, &stats, query, string(webhookID)); err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}

	result := &repository.WebhookDeliveryStats{
		TotalDeliveries:      stats.TotalDeliveries,
		SuccessfulDeliveries: stats.SuccessfulDeliveries,
		FailedDeliveries:     stats.FailedDeliveries,
	}

	if stats.TotalDeliveries > 0 {
		result.SuccessRate = float64(stats.SuccessfulDeliveries) / float64(stats.TotalDeliveries) * 100
	}
	if stats.AverageDuration.Valid {
		result.AverageDuration = stats.AverageDuration.Float64
	}
	if stats.MaxDuration.Valid {
		result.MaxDuration = stats.MaxDuration.Int64
	}
	if stats.MinDuration.Valid {
		result.MinDuration = stats.MinDuration.Int64
	}
	if stats.LastDeliveryAt.Valid {
		ts := domain.Timestamp{Time: stats.LastDeliveryAt.Time}
		result.LastDeliveryAt = &ts
	}
	if stats.LastSuccessAt.Valid {
		ts := domain.Timestamp{Time: stats.LastSuccessAt.Time}
		result.LastSuccessAt = &ts
	}
	if stats.LastFailureAt.Valid {
		ts := domain.Timestamp{Time: stats.LastFailureAt.Time}
		result.LastFailureAt = &ts
	}

	return result, nil
}

// GetDeliveryStatsByDateRange retrieves delivery statistics within a date range.
func (w *WebhookRepository) GetDeliveryStatsByDateRange(ctx context.Context, webhookID domain.ID, dateRange *repository.DateRangeFilter) (*repository.WebhookDeliveryStats, error) {
	var sb strings.Builder
	args := make([]interface{}, 0)

	sb.WriteString(`SELECT 
		COUNT(*) as total_deliveries,
		SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_deliveries,
		SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_deliveries,
		AVG(duration) as average_duration,
		MAX(duration) as max_duration,
		MIN(duration) as min_duration
		FROM webhook_deliveries WHERE webhook_id = ?`)
	args = append(args, string(webhookID))

	if dateRange != nil {
		if dateRange.From != nil {
			sb.WriteString(" AND created_at >= ?")
			args = append(args, dateRange.From.Time)
		}
		if dateRange.To != nil {
			sb.WriteString(" AND created_at <= ?")
			args = append(args, dateRange.To.Time)
		}
	}

	var stats struct {
		TotalDeliveries      int64          `db:"total_deliveries"`
		SuccessfulDeliveries int64          `db:"successful_deliveries"`
		FailedDeliveries     int64          `db:"failed_deliveries"`
		AverageDuration      sql.NullFloat64 `db:"average_duration"`
		MaxDuration          sql.NullInt64  `db:"max_duration"`
		MinDuration          sql.NullInt64  `db:"min_duration"`
	}

	if err := w.repo.db().GetContext(ctx, &stats, sb.String(), args...); err != nil {
		return nil, fmt.Errorf("failed to get delivery stats by date range: %w", err)
	}

	result := &repository.WebhookDeliveryStats{
		TotalDeliveries:      stats.TotalDeliveries,
		SuccessfulDeliveries: stats.SuccessfulDeliveries,
		FailedDeliveries:     stats.FailedDeliveries,
	}

	if stats.TotalDeliveries > 0 {
		result.SuccessRate = float64(stats.SuccessfulDeliveries) / float64(stats.TotalDeliveries) * 100
	}
	if stats.AverageDuration.Valid {
		result.AverageDuration = stats.AverageDuration.Float64
	}
	if stats.MaxDuration.Valid {
		result.MaxDuration = stats.MaxDuration.Int64
	}
	if stats.MinDuration.Valid {
		result.MinDuration = stats.MinDuration.Int64
	}

	return result, nil
}

// GetDailyDeliveryCounts retrieves delivery counts grouped by day.
func (w *WebhookRepository) GetDailyDeliveryCounts(ctx context.Context, webhookID domain.ID, dateRange *repository.DateRangeFilter) ([]repository.DateCount, error) {
	var sb strings.Builder
	args := make([]interface{}, 0)

	sb.WriteString(`SELECT DATE(created_at) as date, COUNT(*) as count 
		FROM webhook_deliveries WHERE webhook_id = ?`)
	args = append(args, string(webhookID))

	if dateRange != nil {
		if dateRange.From != nil {
			sb.WriteString(" AND created_at >= ?")
			args = append(args, dateRange.From.Time)
		}
		if dateRange.To != nil {
			sb.WriteString(" AND created_at <= ?")
			args = append(args, dateRange.To.Time)
		}
	}

	sb.WriteString(" GROUP BY DATE(created_at) ORDER BY date")

	var counts []repository.DateCount
	if err := w.repo.db().SelectContext(ctx, &counts, sb.String(), args...); err != nil {
		return nil, fmt.Errorf("failed to get daily delivery counts: %w", err)
	}

	return counts, nil
}

// GetEventDeliveryCounts retrieves delivery counts grouped by event type.
func (w *WebhookRepository) GetEventDeliveryCounts(ctx context.Context, webhookID domain.ID) ([]repository.EventCount, error) {
	query := `SELECT event, COUNT(*) as count,
		SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count,
		SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failure_count
		FROM webhook_deliveries WHERE webhook_id = ? GROUP BY event ORDER BY count DESC`

	type rawEventCount struct {
		Event        string `db:"event"`
		Count        int64  `db:"count"`
		SuccessCount int64  `db:"success_count"`
		FailureCount int64  `db:"failure_count"`
	}

	var rawCounts []rawEventCount
	if err := w.repo.db().SelectContext(ctx, &rawCounts, query, string(webhookID)); err != nil {
		return nil, fmt.Errorf("failed to get event delivery counts: %w", err)
	}

	counts := make([]repository.EventCount, len(rawCounts))
	for i, rc := range rawCounts {
		counts[i] = repository.EventCount{
			Event:        domain.WebhookEvent(rc.Event),
			Count:        rc.Count,
			SuccessCount: rc.SuccessCount,
			FailureCount: rc.FailureCount,
		}
	}

	return counts, nil
}

// Ensure WebhookRepository implements repository.WebhookRepository
var _ repository.WebhookRepository = (*WebhookRepository)(nil)
