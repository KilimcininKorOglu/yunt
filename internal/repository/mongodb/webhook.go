package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// WebhookRepository implements the repository.WebhookRepository interface for MongoDB.
type WebhookRepository struct {
	repo *Repository
}

// webhookDocument is the MongoDB document representation of a webhook.
type webhookDocument struct {
	ID              string            `bson:"_id"`
	UserID          string            `bson:"userId"`
	Name            string            `bson:"name"`
	URL             string            `bson:"url"`
	Secret          string            `bson:"secret,omitempty"`
	Events          []string          `bson:"events"`
	Status          string            `bson:"status"`
	Headers         map[string]string `bson:"headers,omitempty"`
	RetryCount      int               `bson:"retryCount"`
	MaxRetries      int               `bson:"maxRetries"`
	TimeoutSeconds  int               `bson:"timeoutSeconds"`
	LastTriggeredAt *time.Time        `bson:"lastTriggeredAt,omitempty"`
	LastSuccessAt   *time.Time        `bson:"lastSuccessAt,omitempty"`
	LastFailureAt   *time.Time        `bson:"lastFailureAt,omitempty"`
	LastError       string            `bson:"lastError,omitempty"`
	SuccessCount    int64             `bson:"successCount"`
	FailureCount    int64             `bson:"failureCount"`
	CreatedAt       time.Time         `bson:"createdAt"`
	UpdatedAt       time.Time         `bson:"updatedAt"`
}

// webhookDeliveryDocument is the MongoDB document representation of a webhook delivery.
type webhookDeliveryDocument struct {
	ID            string    `bson:"_id"`
	WebhookID     string    `bson:"webhookId"`
	Event         string    `bson:"event"`
	Payload       string    `bson:"payload"`
	StatusCode    int       `bson:"statusCode"`
	Response      string    `bson:"response,omitempty"`
	Error         string    `bson:"error,omitempty"`
	Success       bool      `bson:"success"`
	Duration      int64     `bson:"duration"`
	AttemptNumber int       `bson:"attemptNumber"`
	CreatedAt     time.Time `bson:"createdAt"`
}

// NewWebhookRepository creates a new MongoDB webhook repository.
func NewWebhookRepository(repo *Repository) *WebhookRepository {
	return &WebhookRepository{repo: repo}
}

// collection returns the webhooks collection.
func (w *WebhookRepository) collection() mongoCollection {
	return w.repo.collection(CollectionWebhooks)
}

// deliveriesCollection returns the webhook deliveries collection.
func (w *WebhookRepository) deliveriesCollection() mongoCollection {
	return w.repo.collection(CollectionWebhookDeliveries)
}

// toDocument converts a domain.Webhook to a MongoDB document.
func (w *WebhookRepository) toDocument(webhook *domain.Webhook) *webhookDocument {
	events := make([]string, len(webhook.Events))
	for i, e := range webhook.Events {
		events[i] = string(e)
	}

	doc := &webhookDocument{
		ID:             string(webhook.ID),
		UserID:         string(webhook.UserID),
		Name:           webhook.Name,
		URL:            webhook.URL,
		Secret:         webhook.Secret,
		Events:         events,
		Status:         string(webhook.Status),
		Headers:        webhook.Headers,
		RetryCount:     webhook.RetryCount,
		MaxRetries:     webhook.MaxRetries,
		TimeoutSeconds: webhook.TimeoutSeconds,
		LastError:      webhook.LastError,
		SuccessCount:   webhook.SuccessCount,
		FailureCount:   webhook.FailureCount,
		CreatedAt:      webhook.CreatedAt.Time,
		UpdatedAt:      webhook.UpdatedAt.Time,
	}

	if webhook.LastTriggeredAt != nil {
		t := webhook.LastTriggeredAt.Time
		doc.LastTriggeredAt = &t
	}
	if webhook.LastSuccessAt != nil {
		t := webhook.LastSuccessAt.Time
		doc.LastSuccessAt = &t
	}
	if webhook.LastFailureAt != nil {
		t := webhook.LastFailureAt.Time
		doc.LastFailureAt = &t
	}

	return doc
}

// toDomain converts a MongoDB document to a domain.Webhook.
func (w *WebhookRepository) toDomain(doc *webhookDocument) *domain.Webhook {
	events := make([]domain.WebhookEvent, len(doc.Events))
	for i, e := range doc.Events {
		events[i] = domain.WebhookEvent(e)
	}

	webhook := &domain.Webhook{
		ID:             domain.ID(doc.ID),
		UserID:         domain.ID(doc.UserID),
		Name:           doc.Name,
		URL:            doc.URL,
		Secret:         doc.Secret,
		Events:         events,
		Status:         domain.WebhookStatus(doc.Status),
		Headers:        doc.Headers,
		RetryCount:     doc.RetryCount,
		MaxRetries:     doc.MaxRetries,
		TimeoutSeconds: doc.TimeoutSeconds,
		LastError:      doc.LastError,
		SuccessCount:   doc.SuccessCount,
		FailureCount:   doc.FailureCount,
		CreatedAt:      domain.Timestamp{Time: doc.CreatedAt},
		UpdatedAt:      domain.Timestamp{Time: doc.UpdatedAt},
	}

	if doc.LastTriggeredAt != nil {
		ts := domain.Timestamp{Time: *doc.LastTriggeredAt}
		webhook.LastTriggeredAt = &ts
	}
	if doc.LastSuccessAt != nil {
		ts := domain.Timestamp{Time: *doc.LastSuccessAt}
		webhook.LastSuccessAt = &ts
	}
	if doc.LastFailureAt != nil {
		ts := domain.Timestamp{Time: *doc.LastFailureAt}
		webhook.LastFailureAt = &ts
	}

	return webhook
}

// deliveryToDocument converts a domain.WebhookDelivery to a MongoDB document.
func (w *WebhookRepository) deliveryToDocument(d *domain.WebhookDelivery) *webhookDeliveryDocument {
	return &webhookDeliveryDocument{
		ID:            string(d.ID),
		WebhookID:     string(d.WebhookID),
		Event:         string(d.Event),
		Payload:       d.Payload,
		StatusCode:    d.StatusCode,
		Response:      d.Response,
		Error:         d.Error,
		Success:       d.Success,
		Duration:      d.Duration,
		AttemptNumber: d.AttemptNumber,
		CreatedAt:     d.CreatedAt.Time,
	}
}

// deliveryToDomain converts a MongoDB document to a domain.WebhookDelivery.
func (w *WebhookRepository) deliveryToDomain(doc *webhookDeliveryDocument) *domain.WebhookDelivery {
	return &domain.WebhookDelivery{
		ID:            domain.ID(doc.ID),
		WebhookID:     domain.ID(doc.WebhookID),
		Event:         domain.WebhookEvent(doc.Event),
		Payload:       doc.Payload,
		StatusCode:    doc.StatusCode,
		Response:      doc.Response,
		Error:         doc.Error,
		Success:       doc.Success,
		Duration:      doc.Duration,
		AttemptNumber: doc.AttemptNumber,
		CreatedAt:     domain.Timestamp{Time: doc.CreatedAt},
	}
}

// GetByID retrieves a webhook by its unique identifier.
func (w *WebhookRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Webhook, error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}

	var doc webhookDocument
	if err := w.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("webhook", string(id))
		}
		return nil, fmt.Errorf("failed to get webhook by ID: %w", err)
	}

	return w.toDomain(&doc), nil
}

// List retrieves webhooks with optional filtering, sorting, and pagination.
func (w *WebhookRepository) List(ctx context.Context, filter *repository.WebhookFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	ctx = w.repo.getSessionContext(ctx)

	mongoFilter := w.buildFilter(filter)
	findOpts := w.buildFindOptions(opts)

	total, err := w.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count webhooks: %w", err)
	}

	cursor, err := w.collection().Find(ctx, mongoFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []webhookDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode webhooks: %w", err)
	}

	webhooks := make([]*domain.Webhook, len(docs))
	for i, doc := range docs {
		webhooks[i] = w.toDomain(&doc)
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

// ListByUser retrieves all webhooks owned by a specific user.
func (w *WebhookRepository) ListByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Webhook], error) {
	id := userID
	filter := &repository.WebhookFilter{UserID: &id}
	return w.List(ctx, filter, opts)
}

// ListByEvent retrieves all webhooks subscribed to a specific event.
func (w *WebhookRepository) ListByEvent(ctx context.Context, event domain.WebhookEvent) ([]*domain.Webhook, error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"events": string(event)}
	cursor, err := w.collection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks by event: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []webhookDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode webhooks: %w", err)
	}

	webhooks := make([]*domain.Webhook, len(docs))
	for i, doc := range docs {
		webhooks[i] = w.toDomain(&doc)
	}

	return webhooks, nil
}

// ListActiveByEvent retrieves all active webhooks subscribed to a specific event.
func (w *WebhookRepository) ListActiveByEvent(ctx context.Context, event domain.WebhookEvent) ([]*domain.Webhook, error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{
		"events": string(event),
		"status": string(domain.WebhookStatusActive),
	}
	cursor, err := w.collection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list active webhooks by event: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []webhookDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode webhooks: %w", err)
	}

	webhooks := make([]*domain.Webhook, len(docs))
	for i, doc := range docs {
		webhooks[i] = w.toDomain(&doc)
	}

	return webhooks, nil
}

// buildFilter builds the MongoDB filter from repository.WebhookFilter.
func (w *WebhookRepository) buildFilter(filter *repository.WebhookFilter) bson.M {
	f := bson.M{}

	if filter == nil {
		return f
	}

	if len(filter.IDs) > 0 {
		ids := make([]string, len(filter.IDs))
		for i, id := range filter.IDs {
			ids[i] = string(id)
		}
		f["_id"] = bson.M{"$in": ids}
	}

	if filter.UserID != nil {
		f["userId"] = string(*filter.UserID)
	}

	if len(filter.UserIDs) > 0 {
		userIDs := make([]string, len(filter.UserIDs))
		for i, id := range filter.UserIDs {
			userIDs[i] = string(id)
		}
		f["userId"] = bson.M{"$in": userIDs}
	}

	if filter.Status != nil {
		f["status"] = string(*filter.Status)
	}

	if len(filter.Statuses) > 0 {
		statuses := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			statuses[i] = string(s)
		}
		f["status"] = bson.M{"$in": statuses}
	}

	if filter.Event != nil {
		f["events"] = string(*filter.Event)
	}

	if len(filter.Events) > 0 {
		events := make([]string, len(filter.Events))
		for i, e := range filter.Events {
			events[i] = string(e)
		}
		f["events"] = bson.M{"$in": events}
	}

	if filter.URL != "" {
		f["url"] = filter.URL
	}

	if filter.URLContains != "" {
		f["url"] = bson.M{"$regex": regexp.QuoteMeta(filter.URLContains), "$options": "i"}
	}

	if filter.Name != "" {
		f["name"] = filter.Name
	}

	if filter.NameContains != "" {
		f["name"] = bson.M{"$regex": regexp.QuoteMeta(filter.NameContains), "$options": "i"}
	}

	if filter.Search != "" {
		f["$text"] = bson.M{"$search": filter.Search}
	}

	if filter.HasFailures != nil {
		if *filter.HasFailures {
			f["failureCount"] = bson.M{"$gt": 0}
		} else {
			f["failureCount"] = 0
		}
	}

	if filter.LastTriggeredAfter != nil {
		f["lastTriggeredAt"] = bson.M{"$gt": filter.LastTriggeredAfter.Time}
	}

	if filter.LastTriggeredBefore != nil {
		if _, exists := f["lastTriggeredAt"]; exists {
			f["lastTriggeredAt"].(bson.M)["$lt"] = filter.LastTriggeredBefore.Time
		} else {
			f["lastTriggeredAt"] = bson.M{"$lt": filter.LastTriggeredBefore.Time}
		}
	}

	if filter.NeverTriggered != nil && *filter.NeverTriggered {
		f["lastTriggeredAt"] = bson.M{"$exists": false}
	}

	if filter.CreatedAfter != nil {
		f["createdAt"] = bson.M{"$gt": filter.CreatedAfter.Time}
	}

	if filter.CreatedBefore != nil {
		if _, exists := f["createdAt"]; exists {
			f["createdAt"].(bson.M)["$lt"] = filter.CreatedBefore.Time
		} else {
			f["createdAt"] = bson.M{"$lt": filter.CreatedBefore.Time}
		}
	}

	return f
}

// buildFindOptions builds MongoDB find options from repository.ListOptions.
func (w *WebhookRepository) buildFindOptions(opts *repository.ListOptions) *options.FindOptions {
	findOpts := options.Find()

	if opts == nil {
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
		return findOpts
	}

	if opts.Sort != nil {
		sortOrder := 1
		if opts.Sort.Order == domain.SortDesc {
			sortOrder = -1
		}
		field := w.mapSortField(opts.Sort.Field)
		findOpts.SetSort(bson.D{{Key: field, Value: sortOrder}})
	} else {
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	}

	if opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	return findOpts
}

// mapSortField maps repository sort field to MongoDB field.
func (w *WebhookRepository) mapSortField(field string) string {
	switch field {
	case "name":
		return "name"
	case "url":
		return "url"
	case "status":
		return "status"
	case "successCount":
		return "successCount"
	case "failureCount":
		return "failureCount"
	case "lastTriggeredAt":
		return "lastTriggeredAt"
	case "createdAt":
		return "createdAt"
	case "updatedAt":
		return "updatedAt"
	default:
		return "createdAt"
	}
}

// Create creates a new webhook.
func (w *WebhookRepository) Create(ctx context.Context, webhook *domain.Webhook) error {
	ctx = w.repo.getSessionContext(ctx)

	// Check for duplicate URL for the same user
	exists, err := w.ExistsByURL(ctx, webhook.UserID, webhook.URL)
	if err != nil {
		return fmt.Errorf("failed to check URL existence: %w", err)
	}
	if exists {
		return domain.NewAlreadyExistsError("webhook", "url", webhook.URL)
	}

	doc := w.toDocument(webhook)
	_, err = w.collection().InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.NewAlreadyExistsError("webhook", "id", string(webhook.ID))
		}
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil
}

// Update updates an existing webhook.
func (w *WebhookRepository) Update(ctx context.Context, webhook *domain.Webhook) error {
	ctx = w.repo.getSessionContext(ctx)

	doc := w.toDocument(webhook)
	doc.UpdatedAt = time.Now().UTC()

	filter := bson.M{"_id": string(webhook.ID)}
	update := bson.M{"$set": doc}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("webhook", string(webhook.ID))
	}

	return nil
}

// Delete permanently removes a webhook by its ID.
func (w *WebhookRepository) Delete(ctx context.Context, id domain.ID) error {
	ctx = w.repo.getSessionContext(ctx)

	// Delete deliveries first
	w.deliveriesCollection().DeleteMany(ctx, bson.M{"webhookId": string(id)})

	// Delete the webhook
	filter := bson.M{"_id": string(id)}
	result, err := w.collection().DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	if result.DeletedCount == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// DeleteByUser removes all webhooks owned by a user.
func (w *WebhookRepository) DeleteByUser(ctx context.Context, userID domain.ID) (int64, error) {
	ctx = w.repo.getSessionContext(ctx)

	// Get webhook IDs first
	cursor, err := w.collection().Find(ctx, bson.M{"userId": string(userID)}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return 0, fmt.Errorf("failed to get webhook IDs: %w", err)
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		ids = append(ids, doc.ID)
	}

	// Delete deliveries
	if len(ids) > 0 {
		w.deliveriesCollection().DeleteMany(ctx, bson.M{"webhookId": bson.M{"$in": ids}})
	}

	// Delete webhooks
	filter := bson.M{"userId": string(userID)}
	result, err := w.collection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete user webhooks: %w", err)
	}

	return result.DeletedCount, nil
}

// Exists checks if a webhook with the given ID exists.
func (w *WebhookRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	count, err := w.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check webhook existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByURL checks if a webhook with the given URL exists for a user.
func (w *WebhookRepository) ExistsByURL(ctx context.Context, userID domain.ID, url string) (bool, error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{
		"userId": string(userID),
		"url":    url,
	}
	count, err := w.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check URL existence: %w", err)
	}

	return count > 0, nil
}

// Count returns the total number of webhooks matching the filter.
func (w *WebhookRepository) Count(ctx context.Context, filter *repository.WebhookFilter) (int64, error) {
	ctx = w.repo.getSessionContext(ctx)

	mongoFilter := w.buildFilter(filter)
	count, err := w.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count webhooks: %w", err)
	}

	return count, nil
}

// CountByUser returns the number of webhooks owned by a user.
func (w *WebhookRepository) CountByUser(ctx context.Context, userID domain.ID) (int64, error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"userId": string(userID)}
	count, err := w.collection().CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count user webhooks: %w", err)
	}

	return count, nil
}

// CountByStatus returns webhook counts grouped by status.
func (w *WebhookRepository) CountByStatus(ctx context.Context) (map[domain.WebhookStatus]int64, error) {
	ctx = w.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := w.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to count webhooks by status: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[domain.WebhookStatus]int64)
	for cursor.Next(ctx) {
		var doc struct {
			Status string `bson:"_id"`
			Count  int64  `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result[domain.WebhookStatus(doc.Status)] = doc.Count
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
	ctx = w.repo.getSessionContext(ctx)

	now := time.Now().UTC()
	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"status":        string(domain.WebhookStatusFailed),
			"lastError":     errorMsg,
			"lastFailureAt": now,
			"updatedAt":     now,
		},
	}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark webhook as failed: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// UpdateStatus updates a webhook's status.
func (w *WebhookRepository) UpdateStatus(ctx context.Context, id domain.ID, status domain.WebhookStatus) error {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"status":    string(status),
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update webhook status: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// UpdateSecret updates a webhook's secret.
func (w *WebhookRepository) UpdateSecret(ctx context.Context, id domain.ID, secret string) error {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"secret":    secret,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update webhook secret: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// AddEvent adds an event subscription to a webhook.
func (w *WebhookRepository) AddEvent(ctx context.Context, id domain.ID, event domain.WebhookEvent) (bool, error) {
	ctx = w.repo.getSessionContext(ctx)

	// Check if already subscribed
	webhook, err := w.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	for _, e := range webhook.Events {
		if e == event {
			return false, nil // Already subscribed
		}
	}

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$addToSet": bson.M{"events": string(event)},
		"$set":      bson.M{"updatedAt": time.Now().UTC()},
	}

	_, err = w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("failed to add event: %w", err)
	}

	return true, nil
}

// RemoveEvent removes an event subscription from a webhook.
func (w *WebhookRepository) RemoveEvent(ctx context.Context, id domain.ID, event domain.WebhookEvent) (bool, error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$pull": bson.M{"events": string(event)},
		"$set":  bson.M{"updatedAt": time.Now().UTC()},
	}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("failed to remove event: %w", err)
	}

	if result.MatchedCount == 0 {
		return false, domain.NewNotFoundError("webhook", string(id))
	}

	return result.ModifiedCount > 0, nil
}

// SetEvents replaces all event subscriptions for a webhook.
func (w *WebhookRepository) SetEvents(ctx context.Context, id domain.ID, events []domain.WebhookEvent) error {
	ctx = w.repo.getSessionContext(ctx)

	eventStrings := make([]string, len(events))
	for i, e := range events {
		eventStrings[i] = string(e)
	}

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"events":    eventStrings,
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to set events: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// RecordSuccess records a successful delivery for a webhook.
func (w *WebhookRepository) RecordSuccess(ctx context.Context, id domain.ID) error {
	ctx = w.repo.getSessionContext(ctx)

	now := time.Now().UTC()
	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"lastTriggeredAt": now,
			"lastSuccessAt":   now,
			"retryCount":      0,
			"lastError":       "",
			"updatedAt":       now,
		},
		"$inc": bson.M{"successCount": 1},
	}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to record success: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// RecordFailure records a failed delivery for a webhook.
func (w *WebhookRepository) RecordFailure(ctx context.Context, id domain.ID, errorMsg string) error {
	ctx = w.repo.getSessionContext(ctx)

	now := time.Now().UTC()
	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"lastTriggeredAt": now,
			"lastFailureAt":   now,
			"lastError":       errorMsg,
			"updatedAt":       now,
		},
		"$inc": bson.M{
			"failureCount": 1,
			"retryCount":   1,
		},
	}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to record failure: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("webhook", string(id))
	}

	return nil
}

// ResetRetryCount resets the retry counter for a webhook.
func (w *WebhookRepository) ResetRetryCount(ctx context.Context, id domain.ID) error {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	update := bson.M{
		"$set": bson.M{
			"retryCount": 0,
			"updatedAt":  time.Now().UTC(),
		},
	}

	result, err := w.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to reset retry count: %w", err)
	}

	if result.MatchedCount == 0 {
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
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{
		"status": bson.M{"$ne": string(domain.WebhookStatusFailed)},
		"$expr": bson.M{
			"$lte": bson.A{"$retryCount", "$maxRetries"},
		},
		"lastError": bson.M{"$ne": ""},
	}

	cursor, err := w.collection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks needing retry: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []webhookDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode webhooks: %w", err)
	}

	webhooks := make([]*domain.Webhook, len(docs))
	for i, doc := range docs {
		webhooks[i] = w.toDomain(&doc)
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
	ctx = w.repo.getSessionContext(ctx)

	doc := w.deliveryToDocument(delivery)
	_, err := w.deliveriesCollection().InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to create delivery: %w", err)
	}

	return nil
}

// GetDelivery retrieves a delivery record by its ID.
func (w *WebhookRepository) GetDelivery(ctx context.Context, id domain.ID) (*domain.WebhookDelivery, error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}

	var doc webhookDeliveryDocument
	if err := w.deliveriesCollection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("webhook delivery", string(id))
		}
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}

	return w.deliveryToDomain(&doc), nil
}

// ListDeliveries retrieves delivery records for a webhook.
func (w *WebhookRepository) ListDeliveries(ctx context.Context, webhookID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"webhookId": string(webhookID)}
	findOpts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	total, err := w.deliveriesCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count deliveries: %w", err)
	}

	cursor, err := w.deliveriesCollection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []webhookDeliveryDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode deliveries: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(docs))
	for i, doc := range docs {
		deliveries[i] = w.deliveryToDomain(&doc)
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
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{
		"webhookId": string(webhookID),
		"event":     string(event),
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	total, err := w.deliveriesCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count deliveries: %w", err)
	}

	cursor, err := w.deliveriesCollection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []webhookDeliveryDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode deliveries: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(docs))
	for i, doc := range docs {
		deliveries[i] = w.deliveryToDomain(&doc)
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
	ctx = w.repo.getSessionContext(ctx)

	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	filter := bson.M{
		"webhookId": string(webhookID),
		"createdAt": bson.M{"$gte": since},
	}

	cursor, err := w.deliveriesCollection().Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list recent deliveries: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []webhookDeliveryDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode deliveries: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(docs))
	for i, doc := range docs {
		deliveries[i] = w.deliveryToDomain(&doc)
	}

	return deliveries, nil
}

// ListFailedDeliveries retrieves failed delivery records for a webhook.
func (w *WebhookRepository) ListFailedDeliveries(ctx context.Context, webhookID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.WebhookDelivery], error) {
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{
		"webhookId": string(webhookID),
		"success":   false,
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	total, err := w.deliveriesCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count deliveries: %w", err)
	}

	cursor, err := w.deliveriesCollection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []webhookDeliveryDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode deliveries: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(docs))
	for i, doc := range docs {
		deliveries[i] = w.deliveryToDomain(&doc)
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
	ctx = w.repo.getSessionContext(ctx)

	filter := bson.M{"webhookId": string(webhookID)}
	result, err := w.deliveriesCollection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete deliveries: %w", err)
	}

	return result.DeletedCount, nil
}

// DeleteOldDeliveries removes delivery records older than the specified days.
func (w *WebhookRepository) DeleteOldDeliveries(ctx context.Context, olderThanDays int) (int64, error) {
	ctx = w.repo.getSessionContext(ctx)

	before := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	filter := bson.M{"createdAt": bson.M{"$lt": before}}

	result, err := w.deliveriesCollection().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old deliveries: %w", err)
	}

	return result.DeletedCount, nil
}

// GetDeliveryStats retrieves delivery statistics for a webhook.
func (w *WebhookRepository) GetDeliveryStats(ctx context.Context, webhookID domain.ID) (*repository.WebhookDeliveryStats, error) {
	ctx = w.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"webhookId": string(webhookID)}}},
		{{Key: "$group", Value: bson.M{
			"_id":          nil,
			"total":        bson.M{"$sum": 1},
			"successful":   bson.M{"$sum": bson.M{"$cond": bson.A{"$success", 1, 0}}},
			"failed":       bson.M{"$sum": bson.M{"$cond": bson.A{"$success", 0, 1}}},
			"avgDuration":  bson.M{"$avg": "$duration"},
			"maxDuration":  bson.M{"$max": "$duration"},
			"minDuration":  bson.M{"$min": "$duration"},
			"lastDelivery": bson.M{"$max": "$createdAt"},
			"lastSuccess":  bson.M{"$max": bson.M{"$cond": bson.A{"$success", "$createdAt", nil}}},
			"lastFailure":  bson.M{"$max": bson.M{"$cond": bson.A{"$success", nil, "$createdAt"}}},
		}}},
	}

	cursor, err := w.deliveriesCollection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := &repository.WebhookDeliveryStats{}
	if cursor.Next(ctx) {
		var doc struct {
			Total        int64      `bson:"total"`
			Successful   int64      `bson:"successful"`
			Failed       int64      `bson:"failed"`
			AvgDuration  float64    `bson:"avgDuration"`
			MaxDuration  int64      `bson:"maxDuration"`
			MinDuration  int64      `bson:"minDuration"`
			LastDelivery *time.Time `bson:"lastDelivery"`
			LastSuccess  *time.Time `bson:"lastSuccess"`
			LastFailure  *time.Time `bson:"lastFailure"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}

		stats.TotalDeliveries = doc.Total
		stats.SuccessfulDeliveries = doc.Successful
		stats.FailedDeliveries = doc.Failed
		stats.AverageDuration = doc.AvgDuration
		stats.MaxDuration = doc.MaxDuration
		stats.MinDuration = doc.MinDuration

		if doc.Total > 0 {
			stats.SuccessRate = float64(doc.Successful) / float64(doc.Total) * 100
		}

		if doc.LastDelivery != nil {
			ts := domain.Timestamp{Time: *doc.LastDelivery}
			stats.LastDeliveryAt = &ts
		}
		if doc.LastSuccess != nil {
			ts := domain.Timestamp{Time: *doc.LastSuccess}
			stats.LastSuccessAt = &ts
		}
		if doc.LastFailure != nil {
			ts := domain.Timestamp{Time: *doc.LastFailure}
			stats.LastFailureAt = &ts
		}
	}

	return stats, nil
}

// GetDeliveryStatsByDateRange retrieves delivery statistics within a date range.
func (w *WebhookRepository) GetDeliveryStatsByDateRange(ctx context.Context, webhookID domain.ID, dateRange *repository.DateRangeFilter) (*repository.WebhookDeliveryStats, error) {
	ctx = w.repo.getSessionContext(ctx)

	matchStage := bson.M{"webhookId": string(webhookID)}
	if dateRange != nil {
		if dateRange.From != nil {
			matchStage["createdAt"] = bson.M{"$gte": dateRange.From.Time}
		}
		if dateRange.To != nil {
			if _, exists := matchStage["createdAt"]; exists {
				matchStage["createdAt"].(bson.M)["$lte"] = dateRange.To.Time
			} else {
				matchStage["createdAt"] = bson.M{"$lte": dateRange.To.Time}
			}
		}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.M{
			"_id":         nil,
			"total":       bson.M{"$sum": 1},
			"successful":  bson.M{"$sum": bson.M{"$cond": bson.A{"$success", 1, 0}}},
			"failed":      bson.M{"$sum": bson.M{"$cond": bson.A{"$success", 0, 1}}},
			"avgDuration": bson.M{"$avg": "$duration"},
			"maxDuration": bson.M{"$max": "$duration"},
			"minDuration": bson.M{"$min": "$duration"},
		}}},
	}

	cursor, err := w.deliveriesCollection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := &repository.WebhookDeliveryStats{}
	if cursor.Next(ctx) {
		var doc struct {
			Total       int64   `bson:"total"`
			Successful  int64   `bson:"successful"`
			Failed      int64   `bson:"failed"`
			AvgDuration float64 `bson:"avgDuration"`
			MaxDuration int64   `bson:"maxDuration"`
			MinDuration int64   `bson:"minDuration"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}

		stats.TotalDeliveries = doc.Total
		stats.SuccessfulDeliveries = doc.Successful
		stats.FailedDeliveries = doc.Failed
		stats.AverageDuration = doc.AvgDuration
		stats.MaxDuration = doc.MaxDuration
		stats.MinDuration = doc.MinDuration

		if doc.Total > 0 {
			stats.SuccessRate = float64(doc.Successful) / float64(doc.Total) * 100
		}
	}

	return stats, nil
}

// GetDailyDeliveryCounts retrieves delivery counts grouped by day.
func (w *WebhookRepository) GetDailyDeliveryCounts(ctx context.Context, webhookID domain.ID, dateRange *repository.DateRangeFilter) ([]repository.DateCount, error) {
	ctx = w.repo.getSessionContext(ctx)

	matchStage := bson.M{"webhookId": string(webhookID)}
	if dateRange != nil {
		if dateRange.From != nil {
			matchStage["createdAt"] = bson.M{"$gte": dateRange.From.Time}
		}
		if dateRange.To != nil {
			if _, exists := matchStage["createdAt"]; exists {
				matchStage["createdAt"].(bson.M)["$lte"] = dateRange.To.Time
			} else {
				matchStage["createdAt"] = bson.M{"$lte": dateRange.To.Time}
			}
		}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt"},
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}

	cursor, err := w.deliveriesCollection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily counts: %w", err)
	}
	defer cursor.Close(ctx)

	var counts []repository.DateCount
	for cursor.Next(ctx) {
		var doc struct {
			Date  string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		counts = append(counts, repository.DateCount{
			Date:  doc.Date,
			Count: doc.Count,
		})
	}

	return counts, nil
}

// GetEventDeliveryCounts retrieves delivery counts grouped by event type.
func (w *WebhookRepository) GetEventDeliveryCounts(ctx context.Context, webhookID domain.ID) ([]repository.EventCount, error) {
	ctx = w.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"webhookId": string(webhookID)}}},
		{{Key: "$group", Value: bson.M{
			"_id":          "$event",
			"count":        bson.M{"$sum": 1},
			"successCount": bson.M{"$sum": bson.M{"$cond": bson.A{"$success", 1, 0}}},
			"failureCount": bson.M{"$sum": bson.M{"$cond": bson.A{"$success", 0, 1}}},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
	}

	cursor, err := w.deliveriesCollection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get event counts: %w", err)
	}
	defer cursor.Close(ctx)

	var counts []repository.EventCount
	for cursor.Next(ctx) {
		var doc struct {
			Event        string `bson:"_id"`
			Count        int64  `bson:"count"`
			SuccessCount int64  `bson:"successCount"`
			FailureCount int64  `bson:"failureCount"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		counts = append(counts, repository.EventCount{
			Event:        domain.WebhookEvent(doc.Event),
			Count:        doc.Count,
			SuccessCount: doc.SuccessCount,
			FailureCount: doc.FailureCount,
		})
	}

	return counts, nil
}

// Ensure WebhookRepository implements repository.WebhookRepository
var _ repository.WebhookRepository = (*WebhookRepository)(nil)

// Unused import prevention
var _ = json.Marshal
