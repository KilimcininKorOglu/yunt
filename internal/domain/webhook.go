package domain

import (
	"net/url"
	"strings"
)

// Webhook represents a webhook configuration for event notifications.
// Webhooks allow external systems to receive real-time notifications
// when events occur in the mail server (e.g., new message received).
type Webhook struct {
	// ID is the unique identifier for the webhook.
	ID ID `json:"id"`

	// UserID is the ID of the user who owns this webhook.
	UserID ID `json:"userId"`

	// Name is a human-readable name for the webhook.
	Name string `json:"name"`

	// URL is the endpoint that will receive webhook notifications.
	URL string `json:"url"`

	// Secret is the shared secret used to sign webhook payloads.
	// This allows the receiver to verify the authenticity of requests.
	// This field is not serialized to JSON for security.
	Secret string `json:"-"`

	// Events is the list of event types this webhook subscribes to.
	Events []WebhookEvent `json:"events"`

	// Status indicates whether the webhook is active.
	Status WebhookStatus `json:"status"`

	// Headers contains custom HTTP headers to include in webhook requests.
	Headers map[string]string `json:"headers,omitempty"`

	// RetryCount is the number of retry attempts for failed deliveries.
	RetryCount int `json:"retryCount"`

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int `json:"maxRetries"`

	// TimeoutSeconds is the request timeout in seconds.
	TimeoutSeconds int `json:"timeoutSeconds"`

	// LastTriggeredAt is the timestamp of the last webhook trigger.
	LastTriggeredAt *Timestamp `json:"lastTriggeredAt,omitempty"`

	// LastSuccessAt is the timestamp of the last successful delivery.
	LastSuccessAt *Timestamp `json:"lastSuccessAt,omitempty"`

	// LastFailureAt is the timestamp of the last failed delivery.
	LastFailureAt *Timestamp `json:"lastFailureAt,omitempty"`

	// LastError is the error message from the last failed delivery.
	LastError string `json:"lastError,omitempty"`

	// SuccessCount is the total number of successful deliveries.
	SuccessCount int64 `json:"successCount"`

	// FailureCount is the total number of failed deliveries.
	FailureCount int64 `json:"failureCount"`

	// CreatedAt is the timestamp when the webhook was created.
	CreatedAt Timestamp `json:"createdAt"`

	// UpdatedAt is the timestamp when the webhook was last updated.
	UpdatedAt Timestamp `json:"updatedAt"`
}

// NewWebhook creates a new Webhook with default values.
func NewWebhook(id, userID ID, name, webhookURL string, events []WebhookEvent) *Webhook {
	now := Now()
	return &Webhook{
		ID:             id,
		UserID:         userID,
		Name:           name,
		URL:            webhookURL,
		Events:         events,
		Status:         WebhookStatusActive,
		Headers:        make(map[string]string),
		RetryCount:     0,
		MaxRetries:     3,
		TimeoutSeconds: 30,
		SuccessCount:   0,
		FailureCount:   0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// Validate checks if the webhook has valid field values.
func (w *Webhook) Validate() error {
	errs := NewValidationErrors()

	// Validate ID
	if w.ID.IsEmpty() {
		errs.Add("id", "id is required")
	}

	// Validate UserID
	if w.UserID.IsEmpty() {
		errs.Add("userId", "user id is required")
	}

	// Validate Name
	if w.Name == "" {
		errs.Add("name", "name is required")
	} else if len(w.Name) > 100 {
		errs.Add("name", "name must be at most 100 characters")
	}

	// Validate URL
	if w.URL == "" {
		errs.Add("url", "url is required")
	} else if !isValidWebhookURL(w.URL) {
		errs.Add("url", "url must be a valid HTTPS URL")
	}

	// Validate Events
	if len(w.Events) == 0 {
		errs.Add("events", "at least one event is required")
	} else {
		for i, event := range w.Events {
			if !event.IsValid() {
				errs.Add("events", "invalid event at index "+intToString(int64(i)))
			}
		}
	}

	// Validate Status
	if !w.Status.IsValid() {
		errs.Add("status", "invalid status")
	}

	// Validate MaxRetries
	if w.MaxRetries < 0 {
		errs.Add("maxRetries", "max retries cannot be negative")
	} else if w.MaxRetries > 10 {
		errs.Add("maxRetries", "max retries cannot exceed 10")
	}

	// Validate TimeoutSeconds
	if w.TimeoutSeconds < 1 {
		errs.Add("timeoutSeconds", "timeout must be at least 1 second")
	} else if w.TimeoutSeconds > 60 {
		errs.Add("timeoutSeconds", "timeout cannot exceed 60 seconds")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// isValidWebhookURL validates that the URL is a valid HTTPS URL.
func isValidWebhookURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Must be HTTPS for security (allow HTTP only for localhost in development)
	if parsed.Scheme != "https" {
		// Allow HTTP for localhost/127.0.0.1 for development
		if parsed.Scheme == "http" {
			host := strings.Split(parsed.Host, ":")[0]
			if host != "localhost" && host != "127.0.0.1" {
				return false
			}
		} else {
			return false
		}
	}

	// Must have a host
	if parsed.Host == "" {
		return false
	}

	return true
}

// Activate sets the webhook status to active.
func (w *Webhook) Activate() {
	w.Status = WebhookStatusActive
	w.UpdatedAt = Now()
}

// Deactivate sets the webhook status to inactive.
func (w *Webhook) Deactivate() {
	w.Status = WebhookStatusInactive
	w.UpdatedAt = Now()
}

// MarkAsFailed sets the webhook status to failed.
func (w *Webhook) MarkAsFailed(errorMsg string) {
	w.Status = WebhookStatusFailed
	w.LastError = errorMsg
	now := Now()
	w.LastFailureAt = &now
	w.UpdatedAt = now
}

// IsActive returns true if the webhook is active.
func (w *Webhook) IsActive() bool {
	return w.Status == WebhookStatusActive
}

// SubscribesToEvent returns true if the webhook is subscribed to the event.
func (w *Webhook) SubscribesToEvent(event WebhookEvent) bool {
	for _, e := range w.Events {
		if e == event {
			return true
		}
	}
	return false
}

// AddEvent adds an event subscription if not already subscribed.
func (w *Webhook) AddEvent(event WebhookEvent) bool {
	if w.SubscribesToEvent(event) {
		return false
	}
	w.Events = append(w.Events, event)
	w.UpdatedAt = Now()
	return true
}

// RemoveEvent removes an event subscription.
func (w *Webhook) RemoveEvent(event WebhookEvent) bool {
	for i, e := range w.Events {
		if e == event {
			w.Events = append(w.Events[:i], w.Events[i+1:]...)
			w.UpdatedAt = Now()
			return true
		}
	}
	return false
}

// RecordSuccess records a successful webhook delivery.
func (w *Webhook) RecordSuccess() {
	now := Now()
	w.LastTriggeredAt = &now
	w.LastSuccessAt = &now
	w.SuccessCount++
	w.RetryCount = 0
	w.LastError = ""
	w.UpdatedAt = now
}

// RecordFailure records a failed webhook delivery.
func (w *Webhook) RecordFailure(errorMsg string) {
	now := Now()
	w.LastTriggeredAt = &now
	w.LastFailureAt = &now
	w.LastError = errorMsg
	w.FailureCount++
	w.RetryCount++
	w.UpdatedAt = now

	// Mark as failed if max retries exceeded
	if w.RetryCount > w.MaxRetries {
		w.Status = WebhookStatusFailed
	}
}

// ShouldRetry returns true if the webhook should be retried.
func (w *Webhook) ShouldRetry() bool {
	return w.RetryCount <= w.MaxRetries && w.Status != WebhookStatusFailed
}

// ResetRetryCount resets the retry counter.
func (w *Webhook) ResetRetryCount() {
	w.RetryCount = 0
	w.UpdatedAt = Now()
}

// SetHeader sets a custom header for webhook requests.
func (w *Webhook) SetHeader(name, value string) {
	if w.Headers == nil {
		w.Headers = make(map[string]string)
	}
	w.Headers[name] = value
	w.UpdatedAt = Now()
}

// RemoveHeader removes a custom header.
func (w *Webhook) RemoveHeader(name string) {
	delete(w.Headers, name)
	w.UpdatedAt = Now()
}

// WebhookCreateInput represents the input for creating a new webhook.
type WebhookCreateInput struct {
	// Name is the human-readable name for the webhook.
	Name string `json:"name"`

	// URL is the endpoint that will receive webhook notifications.
	URL string `json:"url"`

	// Secret is the shared secret for payload signing.
	Secret string `json:"secret,omitempty"`

	// Events is the list of event types to subscribe to.
	Events []WebhookEvent `json:"events"`

	// Headers contains custom HTTP headers to include.
	Headers map[string]string `json:"headers,omitempty"`

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries *int `json:"maxRetries,omitempty"`

	// TimeoutSeconds is the request timeout in seconds.
	TimeoutSeconds *int `json:"timeoutSeconds,omitempty"`
}

// Validate checks if the create input is valid.
func (i *WebhookCreateInput) Validate() error {
	errs := NewValidationErrors()

	// Validate Name
	name := strings.TrimSpace(i.Name)
	if name == "" {
		errs.Add("name", "name is required")
	} else if len(name) > 100 {
		errs.Add("name", "name must be at most 100 characters")
	}

	// Validate URL
	webhookURL := strings.TrimSpace(i.URL)
	if webhookURL == "" {
		errs.Add("url", "url is required")
	} else if !isValidWebhookURL(webhookURL) {
		errs.Add("url", "url must be a valid HTTPS URL")
	}

	// Validate Events
	if len(i.Events) == 0 {
		errs.Add("events", "at least one event is required")
	} else {
		for idx, event := range i.Events {
			if !event.IsValid() {
				errs.Add("events", "invalid event at index "+intToString(int64(idx)))
			}
		}
	}

	// Validate MaxRetries if provided
	if i.MaxRetries != nil {
		if *i.MaxRetries < 0 {
			errs.Add("maxRetries", "max retries cannot be negative")
		} else if *i.MaxRetries > 10 {
			errs.Add("maxRetries", "max retries cannot exceed 10")
		}
	}

	// Validate TimeoutSeconds if provided
	if i.TimeoutSeconds != nil {
		if *i.TimeoutSeconds < 1 {
			errs.Add("timeoutSeconds", "timeout must be at least 1 second")
		} else if *i.TimeoutSeconds > 60 {
			errs.Add("timeoutSeconds", "timeout cannot exceed 60 seconds")
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Normalize trims and normalizes the input fields.
func (i *WebhookCreateInput) Normalize() {
	i.Name = strings.TrimSpace(i.Name)
	i.URL = strings.TrimSpace(i.URL)
}

// WebhookUpdateInput represents the input for updating a webhook.
type WebhookUpdateInput struct {
	// Name is the new name (optional).
	Name *string `json:"name,omitempty"`

	// URL is the new URL (optional).
	URL *string `json:"url,omitempty"`

	// Secret is the new secret (optional).
	Secret *string `json:"secret,omitempty"`

	// Events is the new list of events (optional).
	Events []WebhookEvent `json:"events,omitempty"`

	// Status is the new status (optional).
	Status *WebhookStatus `json:"status,omitempty"`

	// Headers is the new headers map (optional).
	Headers map[string]string `json:"headers,omitempty"`

	// MaxRetries is the new max retries (optional).
	MaxRetries *int `json:"maxRetries,omitempty"`

	// TimeoutSeconds is the new timeout (optional).
	TimeoutSeconds *int `json:"timeoutSeconds,omitempty"`
}

// Validate checks if the update input is valid.
func (i *WebhookUpdateInput) Validate() error {
	errs := NewValidationErrors()

	// Validate Name if provided
	if i.Name != nil {
		name := strings.TrimSpace(*i.Name)
		if name == "" {
			errs.Add("name", "name cannot be empty")
		} else if len(name) > 100 {
			errs.Add("name", "name must be at most 100 characters")
		}
	}

	// Validate URL if provided
	if i.URL != nil {
		webhookURL := strings.TrimSpace(*i.URL)
		if webhookURL == "" {
			errs.Add("url", "url cannot be empty")
		} else if !isValidWebhookURL(webhookURL) {
			errs.Add("url", "url must be a valid HTTPS URL")
		}
	}

	// Validate Events if provided
	if i.Events != nil {
		if len(i.Events) == 0 {
			errs.Add("events", "at least one event is required")
		} else {
			for idx, event := range i.Events {
				if !event.IsValid() {
					errs.Add("events", "invalid event at index "+intToString(int64(idx)))
				}
			}
		}
	}

	// Validate Status if provided
	if i.Status != nil && !i.Status.IsValid() {
		errs.Add("status", "invalid status")
	}

	// Validate MaxRetries if provided
	if i.MaxRetries != nil {
		if *i.MaxRetries < 0 {
			errs.Add("maxRetries", "max retries cannot be negative")
		} else if *i.MaxRetries > 10 {
			errs.Add("maxRetries", "max retries cannot exceed 10")
		}
	}

	// Validate TimeoutSeconds if provided
	if i.TimeoutSeconds != nil {
		if *i.TimeoutSeconds < 1 {
			errs.Add("timeoutSeconds", "timeout must be at least 1 second")
		} else if *i.TimeoutSeconds > 60 {
			errs.Add("timeoutSeconds", "timeout cannot exceed 60 seconds")
		}
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Apply applies the update to the given webhook.
func (i *WebhookUpdateInput) Apply(webhook *Webhook) {
	if i.Name != nil {
		webhook.Name = strings.TrimSpace(*i.Name)
	}
	if i.URL != nil {
		webhook.URL = strings.TrimSpace(*i.URL)
	}
	if i.Secret != nil {
		webhook.Secret = *i.Secret
	}
	if i.Events != nil {
		webhook.Events = i.Events
	}
	if i.Status != nil {
		webhook.Status = *i.Status
	}
	if i.Headers != nil {
		webhook.Headers = i.Headers
	}
	if i.MaxRetries != nil {
		webhook.MaxRetries = *i.MaxRetries
	}
	if i.TimeoutSeconds != nil {
		webhook.TimeoutSeconds = *i.TimeoutSeconds
	}
	webhook.UpdatedAt = Now()
}

// WebhookFilter represents filtering options for listing webhooks.
type WebhookFilter struct {
	// UserID filters by owner user ID.
	UserID *ID `json:"userId,omitempty"`

	// Status filters by webhook status.
	Status *WebhookStatus `json:"status,omitempty"`

	// Event filters by subscribed event.
	Event *WebhookEvent `json:"event,omitempty"`

	// Search is a text search on name and URL.
	Search string `json:"search,omitempty"`
}

// WebhookDelivery represents a record of a webhook delivery attempt.
type WebhookDelivery struct {
	// ID is the unique identifier for the delivery.
	ID ID `json:"id"`

	// WebhookID is the ID of the webhook.
	WebhookID ID `json:"webhookId"`

	// Event is the event type that triggered the delivery.
	Event WebhookEvent `json:"event"`

	// Payload is the JSON payload sent to the webhook.
	Payload string `json:"payload"`

	// StatusCode is the HTTP status code returned by the endpoint.
	StatusCode int `json:"statusCode"`

	// Response is the response body from the endpoint.
	Response string `json:"response,omitempty"`

	// Error is the error message if delivery failed.
	Error string `json:"error,omitempty"`

	// Success indicates if the delivery was successful.
	Success bool `json:"success"`

	// Duration is the request duration in milliseconds.
	Duration int64 `json:"duration"`

	// AttemptNumber is the attempt number (1 = first attempt).
	AttemptNumber int `json:"attemptNumber"`

	// CreatedAt is the timestamp when the delivery was attempted.
	CreatedAt Timestamp `json:"createdAt"`
}

// NewWebhookDelivery creates a new WebhookDelivery record.
func NewWebhookDelivery(id, webhookID ID, event WebhookEvent, payload string, attemptNumber int) *WebhookDelivery {
	return &WebhookDelivery{
		ID:            id,
		WebhookID:     webhookID,
		Event:         event,
		Payload:       payload,
		AttemptNumber: attemptNumber,
		Success:       false,
		CreatedAt:     Now(),
	}
}

// RecordResult records the result of a delivery attempt.
func (d *WebhookDelivery) RecordResult(statusCode int, response string, err error, durationMs int64) {
	d.StatusCode = statusCode
	d.Response = response
	d.Duration = durationMs

	if err != nil {
		d.Error = err.Error()
		d.Success = false
	} else {
		d.Success = statusCode >= 200 && statusCode < 300
	}
}
