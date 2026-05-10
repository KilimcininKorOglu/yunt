// Package service provides business logic services for the Yunt mail server.
package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// WebhookService provides business logic for webhook management and dispatch.
type WebhookService struct {
	repo        repository.Repository
	idGenerator IDGenerator
	httpClient  *http.Client
	logger      WebhookLogger
}

// WebhookLogger defines the interface for logging webhook operations.
type WebhookLogger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

// defaultLogger is a simple logger that logs to stdout.
type defaultLogger struct{}

func (d *defaultLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Printf("INFO: %s %v", msg, keysAndValues)
}

func (d *defaultLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Printf("ERROR: %s %v", msg, keysAndValues)
}

func (d *defaultLogger) Debug(msg string, keysAndValues ...interface{}) {
	log.Printf("DEBUG: %s %v", msg, keysAndValues)
}

// NewWebhookService creates a new WebhookService with the given dependencies.
func NewWebhookService(repo repository.Repository, idGenerator IDGenerator) *WebhookService {
	return &WebhookService{
		repo:        repo,
		idGenerator: idGenerator,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: &defaultLogger{},
	}
}

// WithLogger sets a custom logger for the service.
func (s *WebhookService) WithLogger(logger WebhookLogger) *WebhookService {
	s.logger = logger
	return s
}

// WithHTTPClient sets a custom HTTP client for webhook dispatch.
func (s *WebhookService) WithHTTPClient(client *http.Client) *WebhookService {
	s.httpClient = client
	return s
}

// CreateWebhook creates a new webhook for a user.
func (s *WebhookService) CreateWebhook(ctx context.Context, userID domain.ID, input *domain.WebhookCreateInput) (*domain.Webhook, error) {
	if userID.IsEmpty() {
		return nil, &WebhookServiceError{
			Op:      "create",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Validate and normalize input
	input.Normalize()
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check for duplicate URL
	exists, err := s.repo.Webhooks().ExistsByURL(ctx, userID, input.URL)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "create",
			Message: "failed to check for duplicate URL",
			Err:     err,
		}
	}
	if exists {
		return nil, &WebhookServiceError{
			Op:      "create",
			Message: "webhook with this URL already exists",
			Err:     domain.ErrAlreadyExists,
		}
	}

	// Generate ID and create webhook
	webhookID := s.idGenerator.Generate()
	webhook := domain.NewWebhook(webhookID, userID, input.Name, input.URL, input.Events)

	// Set optional fields
	if input.Secret != "" {
		webhook.Secret = input.Secret
	}
	if input.Headers != nil {
		webhook.Headers = input.Headers
	}
	if input.MaxRetries != nil {
		webhook.MaxRetries = *input.MaxRetries
	}
	if input.TimeoutSeconds != nil {
		webhook.TimeoutSeconds = *input.TimeoutSeconds
	}

	// Save to repository
	if err := s.repo.Webhooks().Create(ctx, webhook); err != nil {
		return nil, &WebhookServiceError{
			Op:      "create",
			Message: "failed to create webhook",
			Err:     err,
		}
	}

	s.logger.Info("webhook created",
		"webhookId", webhook.ID,
		"userId", userID,
		"url", webhook.URL,
	)

	return webhook, nil
}

// GetWebhook retrieves a webhook by ID.
func (s *WebhookService) GetWebhook(ctx context.Context, id domain.ID) (*domain.Webhook, error) {
	if id.IsEmpty() {
		return nil, &WebhookServiceError{
			Op:      "get",
			Message: "webhook ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	webhook, err := s.repo.Webhooks().GetByID(ctx, id)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "get",
			Message: "failed to get webhook",
			Err:     err,
		}
	}

	return webhook, nil
}

// GetWebhookForUser retrieves a webhook by ID and verifies ownership.
func (s *WebhookService) GetWebhookForUser(ctx context.Context, id, userID domain.ID) (*domain.Webhook, error) {
	webhook, err := s.GetWebhook(ctx, id)
	if err != nil {
		return nil, err
	}

	if webhook.UserID != userID {
		return nil, &WebhookServiceError{
			Op:      "get",
			Message: "webhook not found",
			Err:     domain.ErrNotFound,
		}
	}

	return webhook, nil
}

// ListWebhooks lists webhooks with optional filtering and pagination.
func (s *WebhookService) ListWebhooks(
	ctx context.Context,
	filter *repository.WebhookFilter,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.Webhook], error) {
	result, err := s.repo.Webhooks().List(ctx, filter, opts)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "list",
			Message: "failed to list webhooks",
			Err:     err,
		}
	}

	return result, nil
}

// ListWebhooksByUser lists all webhooks owned by a user.
func (s *WebhookService) ListWebhooksByUser(
	ctx context.Context,
	userID domain.ID,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.Webhook], error) {
	if userID.IsEmpty() {
		return nil, &WebhookServiceError{
			Op:      "list_by_user",
			Message: "user ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result, err := s.repo.Webhooks().ListByUser(ctx, userID, opts)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "list_by_user",
			Message: "failed to list webhooks by user",
			Err:     err,
		}
	}

	return result, nil
}

// UpdateWebhook updates an existing webhook.
func (s *WebhookService) UpdateWebhook(ctx context.Context, id domain.ID, input *domain.WebhookUpdateInput) (*domain.Webhook, error) {
	if id.IsEmpty() {
		return nil, &WebhookServiceError{
			Op:      "update",
			Message: "webhook ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	// Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Get existing webhook
	webhook, err := s.repo.Webhooks().GetByID(ctx, id)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "update",
			Message: "failed to get webhook",
			Err:     err,
		}
	}

	// Check for duplicate URL if URL is being changed
	if input.URL != nil && *input.URL != webhook.URL {
		exists, err := s.repo.Webhooks().ExistsByURL(ctx, webhook.UserID, *input.URL)
		if err != nil {
			return nil, &WebhookServiceError{
				Op:      "update",
				Message: "failed to check for duplicate URL",
				Err:     err,
			}
		}
		if exists {
			return nil, &WebhookServiceError{
				Op:      "update",
				Message: "webhook with this URL already exists",
				Err:     domain.ErrAlreadyExists,
			}
		}
	}

	// Apply updates
	input.Apply(webhook)

	// Save changes
	if err := s.repo.Webhooks().Update(ctx, webhook); err != nil {
		return nil, &WebhookServiceError{
			Op:      "update",
			Message: "failed to update webhook",
			Err:     err,
		}
	}

	s.logger.Info("webhook updated",
		"webhookId", webhook.ID,
	)

	return webhook, nil
}

// UpdateWebhookForUser updates a webhook and verifies ownership.
func (s *WebhookService) UpdateWebhookForUser(ctx context.Context, id, userID domain.ID, input *domain.WebhookUpdateInput) (*domain.Webhook, error) {
	// Get webhook and verify ownership
	webhook, err := s.GetWebhookForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check for duplicate URL if URL is being changed
	if input.URL != nil && *input.URL != webhook.URL {
		exists, err := s.repo.Webhooks().ExistsByURL(ctx, userID, *input.URL)
		if err != nil {
			return nil, &WebhookServiceError{
				Op:      "update",
				Message: "failed to check for duplicate URL",
				Err:     err,
			}
		}
		if exists {
			return nil, &WebhookServiceError{
				Op:      "update",
				Message: "webhook with this URL already exists",
				Err:     domain.ErrAlreadyExists,
			}
		}
	}

	// Apply updates
	input.Apply(webhook)

	// Save changes
	if err := s.repo.Webhooks().Update(ctx, webhook); err != nil {
		return nil, &WebhookServiceError{
			Op:      "update",
			Message: "failed to update webhook",
			Err:     err,
		}
	}

	s.logger.Info("webhook updated",
		"webhookId", webhook.ID,
		"userId", userID,
	)

	return webhook, nil
}

// DeleteWebhook deletes a webhook.
func (s *WebhookService) DeleteWebhook(ctx context.Context, id domain.ID) error {
	if id.IsEmpty() {
		return &WebhookServiceError{
			Op:      "delete",
			Message: "webhook ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if err := s.repo.Webhooks().Delete(ctx, id); err != nil {
		return &WebhookServiceError{
			Op:      "delete",
			Message: "failed to delete webhook",
			Err:     err,
		}
	}

	s.logger.Info("webhook deleted",
		"webhookId", id,
	)

	return nil
}

// DeleteWebhookForUser deletes a webhook and verifies ownership.
func (s *WebhookService) DeleteWebhookForUser(ctx context.Context, id, userID domain.ID) error {
	// Verify ownership
	_, err := s.GetWebhookForUser(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.DeleteWebhook(ctx, id)
}

// ActivateWebhook activates a webhook.
func (s *WebhookService) ActivateWebhook(ctx context.Context, id domain.ID) error {
	if id.IsEmpty() {
		return &WebhookServiceError{
			Op:      "activate",
			Message: "webhook ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if err := s.repo.Webhooks().Activate(ctx, id); err != nil {
		return &WebhookServiceError{
			Op:      "activate",
			Message: "failed to activate webhook",
			Err:     err,
		}
	}

	s.logger.Info("webhook activated",
		"webhookId", id,
	)

	return nil
}

// DeactivateWebhook deactivates a webhook.
func (s *WebhookService) DeactivateWebhook(ctx context.Context, id domain.ID) error {
	if id.IsEmpty() {
		return &WebhookServiceError{
			Op:      "deactivate",
			Message: "webhook ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	if err := s.repo.Webhooks().Deactivate(ctx, id); err != nil {
		return &WebhookServiceError{
			Op:      "deactivate",
			Message: "failed to deactivate webhook",
			Err:     err,
		}
	}

	s.logger.Info("webhook deactivated",
		"webhookId", id,
	)

	return nil
}

// WebhookPayload represents the payload sent to webhook endpoints.
type WebhookPayload struct {
	// ID is the unique identifier for this webhook delivery.
	ID string `json:"id"`

	// Event is the event type that triggered the webhook.
	Event domain.WebhookEvent `json:"event"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Data contains the event-specific payload.
	Data interface{} `json:"data"`
}

// MessageEventData represents the data payload for message events.
type MessageEventData struct {
	// MessageID is the unique identifier of the message.
	MessageID string `json:"messageId"`

	// MailboxID is the ID of the mailbox containing the message.
	MailboxID string `json:"mailboxId"`

	// Subject is the message subject.
	Subject string `json:"subject"`

	// From is the sender address.
	From string `json:"from"`

	// To contains the recipient addresses.
	To []string `json:"to"`

	// ReceivedAt is when the message was received.
	ReceivedAt time.Time `json:"receivedAt"`
}

// MailboxEventData represents the data payload for mailbox events.
type MailboxEventData struct {
	// MailboxID is the unique identifier of the mailbox.
	MailboxID string `json:"mailboxId"`

	// Name is the mailbox name.
	Name string `json:"name"`

	// Email is the mailbox email address.
	Email string `json:"email"`
}

// TriggerEvent dispatches webhooks for a given event.
func (s *WebhookService) TriggerEvent(ctx context.Context, event domain.WebhookEvent, data interface{}) error {
	// Get all active webhooks subscribed to this event
	webhooks, err := s.repo.Webhooks().ListActiveByEvent(ctx, event)
	if err != nil {
		s.logger.Error("failed to list active webhooks for event",
			"event", event,
			"error", err,
		)
		return &WebhookServiceError{
			Op:      "trigger_event",
			Message: "failed to list webhooks for event",
			Err:     err,
		}
	}

	if len(webhooks) == 0 {
		s.logger.Debug("no webhooks subscribed to event",
			"event", event,
		)
		return nil
	}

	// Create the payload
	deliveryID := s.idGenerator.Generate()
	payload := &WebhookPayload{
		ID:        deliveryID.String(),
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	// Dispatch to all webhooks concurrently
	var wg sync.WaitGroup
	for _, webhook := range webhooks {
		wg.Add(1)
		go func(wh *domain.Webhook) {
			defer wg.Done()
			s.dispatchWebhook(ctx, wh, payload)
		}(webhook)
	}
	wg.Wait()

	return nil
}

// TriggerMessageReceived triggers webhooks for a message.received event.
func (s *WebhookService) TriggerMessageReceived(ctx context.Context, msg *domain.Message) error {
	toAddresses := make([]string, len(msg.To))
	for i, addr := range msg.To {
		toAddresses[i] = addr.Address
	}

	data := &MessageEventData{
		MessageID:  msg.ID.String(),
		MailboxID:  msg.MailboxID.String(),
		Subject:    msg.Subject,
		From:       msg.From.Address,
		To:         toAddresses,
		ReceivedAt: msg.ReceivedAt.Time,
	}

	return s.TriggerEvent(ctx, domain.WebhookEventMessageReceived, data)
}

// TriggerMessageDeleted triggers webhooks for a message.deleted event.
func (s *WebhookService) TriggerMessageDeleted(ctx context.Context, messageID, mailboxID domain.ID) error {
	data := map[string]string{
		"messageId": messageID.String(),
		"mailboxId": mailboxID.String(),
	}

	return s.TriggerEvent(ctx, domain.WebhookEventMessageDeleted, data)
}

// TriggerMailboxCreated triggers webhooks for a mailbox.created event.
func (s *WebhookService) TriggerMailboxCreated(ctx context.Context, mailbox *domain.Mailbox) error {
	data := &MailboxEventData{
		MailboxID: mailbox.ID.String(),
		Name:      mailbox.Name,
		Email:     mailbox.Address,
	}

	return s.TriggerEvent(ctx, domain.WebhookEventMailboxCreated, data)
}

// TriggerMailboxDeleted triggers webhooks for a mailbox.deleted event.
func (s *WebhookService) TriggerMailboxDeleted(ctx context.Context, mailboxID domain.ID, name, email string) error {
	data := &MailboxEventData{
		MailboxID: mailboxID.String(),
		Name:      name,
		Email:     email,
	}

	return s.TriggerEvent(ctx, domain.WebhookEventMailboxDeleted, data)
}

// TriggerUserCreated triggers webhooks for a user.created event.
func (s *WebhookService) TriggerUserCreated(ctx context.Context, user *domain.User) error {
	data := map[string]string{
		"userId":   user.ID.String(),
		"username": user.Username,
		"email":    user.Email,
		"role":     string(user.Role),
	}

	return s.TriggerEvent(ctx, domain.WebhookEventUserCreated, data)
}

// dispatchWebhook sends a webhook request and records the delivery.
func (s *WebhookService) dispatchWebhook(ctx context.Context, webhook *domain.Webhook, payload *WebhookPayload) {
	deliveryID := s.idGenerator.Generate()
	attemptNumber := webhook.RetryCount + 1

	delivery := domain.NewWebhookDelivery(
		deliveryID,
		webhook.ID,
		payload.Event,
		"",
		attemptNumber,
	)

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.recordDeliveryFailure(ctx, webhook, delivery, 0, "", err, 0)
		return
	}
	delivery.Payload = string(payloadBytes)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		s.recordDeliveryFailure(ctx, webhook, delivery, 0, "", err, 0)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Yunt-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", webhook.ID.String())
	req.Header.Set("X-Webhook-Event", string(payload.Event))
	req.Header.Set("X-Webhook-Delivery", deliveryID.String())

	// Add signature if secret is configured
	if webhook.Secret != "" {
		signature := s.signPayload(payloadBytes, webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
		req.Header.Set("X-Webhook-Signature-256", "sha256="+signature)
	}

	// Add custom headers
	for name, value := range webhook.Headers {
		req.Header.Set(name, value)
	}

	// Create client with custom timeout
	client := s.httpClient
	if webhook.TimeoutSeconds > 0 {
		client = &http.Client{
			Timeout: time.Duration(webhook.TimeoutSeconds) * time.Second,
		}
	}

	// Execute request
	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		s.recordDeliveryFailure(ctx, webhook, delivery, 0, "", err, duration)
		return
	}
	defer resp.Body.Close()

	// Read response body (limit to 64KB)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	responseBody := string(body)

	// Record result
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.recordDeliverySuccess(ctx, webhook, delivery, resp.StatusCode, responseBody, duration)
	} else {
		err := fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		s.recordDeliveryFailure(ctx, webhook, delivery, resp.StatusCode, responseBody, err, duration)
	}
}

// signPayload creates an HMAC-SHA256 signature for the payload.
func (s *WebhookService) signPayload(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// recordDeliverySuccess records a successful webhook delivery.
func (s *WebhookService) recordDeliverySuccess(
	ctx context.Context,
	webhook *domain.Webhook,
	delivery *domain.WebhookDelivery,
	statusCode int,
	response string,
	durationMs int64,
) {
	delivery.RecordResult(statusCode, response, nil, durationMs)

	// Save delivery record
	if err := s.repo.Webhooks().CreateDelivery(ctx, delivery); err != nil {
		s.logger.Error("failed to save delivery record",
			"webhookId", webhook.ID,
			"error", err,
		)
	}

	// Update webhook success stats
	if err := s.repo.Webhooks().RecordSuccess(ctx, webhook.ID); err != nil {
		s.logger.Error("failed to record webhook success",
			"webhookId", webhook.ID,
			"error", err,
		)
	}

	s.logger.Info("webhook delivery succeeded",
		"webhookId", webhook.ID,
		"deliveryId", delivery.ID,
		"statusCode", statusCode,
		"duration", durationMs,
	)
}

// recordDeliveryFailure records a failed webhook delivery.
func (s *WebhookService) recordDeliveryFailure(
	ctx context.Context,
	webhook *domain.Webhook,
	delivery *domain.WebhookDelivery,
	statusCode int,
	response string,
	err error,
	durationMs int64,
) {
	delivery.RecordResult(statusCode, response, err, durationMs)

	// Save delivery record
	if saveErr := s.repo.Webhooks().CreateDelivery(ctx, delivery); saveErr != nil {
		s.logger.Error("failed to save delivery record",
			"webhookId", webhook.ID,
			"error", saveErr,
		)
	}

	// Update webhook failure stats
	if recordErr := s.repo.Webhooks().RecordFailure(ctx, webhook.ID, err.Error()); recordErr != nil {
		s.logger.Error("failed to record webhook failure",
			"webhookId", webhook.ID,
			"error", recordErr,
		)
	}

	s.logger.Error("webhook delivery failed",
		"webhookId", webhook.ID,
		"deliveryId", delivery.ID,
		"statusCode", statusCode,
		"error", err,
		"duration", durationMs,
	)
}

// TestWebhook sends a test webhook to verify the endpoint configuration.
func (s *WebhookService) TestWebhook(ctx context.Context, id domain.ID) (*domain.WebhookDelivery, error) {
	if id.IsEmpty() {
		return nil, &WebhookServiceError{
			Op:      "test",
			Message: "webhook ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	webhook, err := s.repo.Webhooks().GetByID(ctx, id)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "test",
			Message: "failed to get webhook",
			Err:     err,
		}
	}

	// Create test payload
	deliveryID := s.idGenerator.Generate()
	payload := &WebhookPayload{
		ID:        deliveryID.String(),
		Event:     "test",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"message":   "This is a test webhook delivery from Yunt",
			"webhookId": webhook.ID.String(),
		},
	}

	// Create delivery record
	delivery := domain.NewWebhookDelivery(
		deliveryID,
		webhook.ID,
		"test",
		"",
		1,
	)

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "test",
			Message: "failed to marshal payload",
			Err:     err,
		}
	}
	delivery.Payload = string(payloadBytes)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "test",
			Message: "failed to create request",
			Err:     err,
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Yunt-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", webhook.ID.String())
	req.Header.Set("X-Webhook-Event", "test")
	req.Header.Set("X-Webhook-Delivery", deliveryID.String())

	// Add signature if secret is configured
	if webhook.Secret != "" {
		signature := s.signPayload(payloadBytes, webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
		req.Header.Set("X-Webhook-Signature-256", "sha256="+signature)
	}

	// Add custom headers
	for name, value := range webhook.Headers {
		req.Header.Set(name, value)
	}

	// Create client with custom timeout
	client := s.httpClient
	if webhook.TimeoutSeconds > 0 {
		client = &http.Client{
			Timeout: time.Duration(webhook.TimeoutSeconds) * time.Second,
		}
	}

	// Execute request
	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		delivery.RecordResult(0, "", err, duration)
		return delivery, nil
	}
	defer resp.Body.Close()

	// Read response body (limit to 64KB)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	responseBody := string(body)

	// Record result
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.RecordResult(resp.StatusCode, responseBody, nil, duration)
	} else {
		err := fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		delivery.RecordResult(resp.StatusCode, responseBody, err, duration)
	}

	s.logger.Info("test webhook delivery",
		"webhookId", webhook.ID,
		"deliveryId", delivery.ID,
		"success", delivery.Success,
		"statusCode", delivery.StatusCode,
		"duration", duration,
	)

	return delivery, nil
}

// TestWebhookForUser sends a test webhook and verifies ownership.
func (s *WebhookService) TestWebhookForUser(ctx context.Context, id, userID domain.ID) (*domain.WebhookDelivery, error) {
	// Verify ownership
	_, err := s.GetWebhookForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	return s.TestWebhook(ctx, id)
}

// ListDeliveries lists delivery records for a webhook.
func (s *WebhookService) ListDeliveries(
	ctx context.Context,
	webhookID domain.ID,
	opts *repository.ListOptions,
) (*repository.ListResult[*domain.WebhookDelivery], error) {
	if webhookID.IsEmpty() {
		return nil, &WebhookServiceError{
			Op:      "list_deliveries",
			Message: "webhook ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	result, err := s.repo.Webhooks().ListDeliveries(ctx, webhookID, opts)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "list_deliveries",
			Message: "failed to list deliveries",
			Err:     err,
		}
	}

	return result, nil
}

// GetDeliveryStats retrieves delivery statistics for a webhook.
func (s *WebhookService) GetDeliveryStats(ctx context.Context, webhookID domain.ID) (*repository.WebhookDeliveryStats, error) {
	if webhookID.IsEmpty() {
		return nil, &WebhookServiceError{
			Op:      "get_delivery_stats",
			Message: "webhook ID is required",
			Err:     domain.ErrInvalidInput,
		}
	}

	stats, err := s.repo.Webhooks().GetDeliveryStats(ctx, webhookID)
	if err != nil {
		return nil, &WebhookServiceError{
			Op:      "get_delivery_stats",
			Message: "failed to get delivery stats",
			Err:     err,
		}
	}

	return stats, nil
}

// WebhookServiceError represents an error that occurred in the webhook service.
type WebhookServiceError struct {
	// Op is the operation that failed.
	Op string
	// Message is a human-readable error description.
	Message string
	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *WebhookServiceError) Error() string {
	var sb strings.Builder
	sb.WriteString("webhook service ")
	sb.WriteString(e.Op)
	sb.WriteString(": ")
	sb.WriteString(e.Message)
	if e.Err != nil {
		sb.WriteString(": ")
		sb.WriteString(e.Err.Error())
	}
	return sb.String()
}

// Unwrap returns the underlying error.
func (e *WebhookServiceError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is for error comparison.
func (e *WebhookServiceError) Is(target error) bool {
	if e.Err == nil {
		return false
	}
	return errors.Is(e.Err, target)
}
