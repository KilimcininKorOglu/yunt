package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNewWebhook(t *testing.T) {
	events := []WebhookEvent{WebhookEventMessageReceived, WebhookEventMailboxCreated}
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test Webhook", "https://example.com/webhook", events)

	if webhook.ID != ID("wh1") {
		t.Errorf("NewWebhook().ID = %v, want %v", webhook.ID, "wh1")
	}
	if webhook.UserID != ID("u1") {
		t.Errorf("NewWebhook().UserID = %v, want %v", webhook.UserID, "u1")
	}
	if webhook.Name != "Test Webhook" {
		t.Errorf("NewWebhook().Name = %v, want %v", webhook.Name, "Test Webhook")
	}
	if webhook.URL != "https://example.com/webhook" {
		t.Errorf("NewWebhook().URL = %v, want %v", webhook.URL, "https://example.com/webhook")
	}
	if len(webhook.Events) != 2 {
		t.Errorf("NewWebhook().Events length = %v, want 2", len(webhook.Events))
	}
	if webhook.Status != WebhookStatusActive {
		t.Errorf("NewWebhook().Status = %v, want %v", webhook.Status, WebhookStatusActive)
	}
	if webhook.MaxRetries != 3 {
		t.Errorf("NewWebhook().MaxRetries = %v, want 3", webhook.MaxRetries)
	}
	if webhook.TimeoutSeconds != 30 {
		t.Errorf("NewWebhook().TimeoutSeconds = %v, want 30", webhook.TimeoutSeconds)
	}
}

func TestWebhook_Validate(t *testing.T) {
	tests := []struct {
		name    string
		webhook *Webhook
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid webhook",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: false,
		},
		{
			name: "missing id",
			webhook: &Webhook{
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"id"},
		},
		{
			name: "missing user id",
			webhook: &Webhook{
				ID:             ID("wh1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"userId"},
		},
		{
			name: "missing name",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"name"},
		},
		{
			name: "name too long",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           strings.Repeat("a", 101),
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"name"},
		},
		{
			name: "missing url",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"url"},
		},
		{
			name: "invalid url (http non-localhost)",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "http://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"url"},
		},
		{
			name: "valid url (http localhost)",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "http://localhost:8080/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: false,
		},
		{
			name: "no events",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"events"},
		},
		{
			name: "invalid event",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEvent("invalid")},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"events"},
		},
		{
			name: "invalid status",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatus("invalid"),
				MaxRetries:     3,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"status"},
		},
		{
			name: "negative max retries",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     -1,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"maxRetries"},
		},
		{
			name: "max retries too high",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     11,
				TimeoutSeconds: 30,
			},
			wantErr: true,
			errMsgs: []string{"maxRetries"},
		},
		{
			name: "timeout too low",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 0,
			},
			wantErr: true,
			errMsgs: []string{"timeoutSeconds"},
		},
		{
			name: "timeout too high",
			webhook: &Webhook{
				ID:             ID("wh1"),
				UserID:         ID("u1"),
				Name:           "Test",
				URL:            "https://example.com/webhook",
				Events:         []WebhookEvent{WebhookEventMessageReceived},
				Status:         WebhookStatusActive,
				MaxRetries:     3,
				TimeoutSeconds: 61,
			},
			wantErr: true,
			errMsgs: []string{"timeoutSeconds"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.webhook.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Webhook.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				errStr := err.Error()
				for _, msg := range tt.errMsgs {
					if !strings.Contains(errStr, msg) {
						t.Errorf("Webhook.Validate() error should contain '%s', got %v", msg, errStr)
					}
				}
			}
		})
	}
}

func TestWebhook_ActivateDeactivate(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})

	webhook.Deactivate()
	if webhook.Status != WebhookStatusInactive {
		t.Errorf("Deactivate() Status = %v, want %v", webhook.Status, WebhookStatusInactive)
	}
	if webhook.IsActive() {
		t.Error("IsActive() should return false after Deactivate()")
	}

	webhook.Activate()
	if webhook.Status != WebhookStatusActive {
		t.Errorf("Activate() Status = %v, want %v", webhook.Status, WebhookStatusActive)
	}
	if !webhook.IsActive() {
		t.Error("IsActive() should return true after Activate()")
	}
}

func TestWebhook_MarkAsFailed(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})

	webhook.MarkAsFailed("Connection refused")

	if webhook.Status != WebhookStatusFailed {
		t.Errorf("MarkAsFailed() Status = %v, want %v", webhook.Status, WebhookStatusFailed)
	}
	if webhook.LastError != "Connection refused" {
		t.Errorf("MarkAsFailed() LastError = %v, want %v", webhook.LastError, "Connection refused")
	}
	if webhook.LastFailureAt == nil {
		t.Error("MarkAsFailed() should set LastFailureAt")
	}
}

func TestWebhook_SubscribesToEvent(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})

	if !webhook.SubscribesToEvent(WebhookEventMessageReceived) {
		t.Error("SubscribesToEvent() should return true for subscribed event")
	}
	if webhook.SubscribesToEvent(WebhookEventMailboxCreated) {
		t.Error("SubscribesToEvent() should return false for unsubscribed event")
	}
}

func TestWebhook_AddRemoveEvent(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})

	// Add new event
	added := webhook.AddEvent(WebhookEventMailboxCreated)
	if !added {
		t.Error("AddEvent() should return true for new event")
	}
	if !webhook.SubscribesToEvent(WebhookEventMailboxCreated) {
		t.Error("AddEvent() should add the event")
	}

	// Try to add duplicate
	added = webhook.AddEvent(WebhookEventMailboxCreated)
	if added {
		t.Error("AddEvent() should return false for duplicate event")
	}

	// Remove event
	removed := webhook.RemoveEvent(WebhookEventMailboxCreated)
	if !removed {
		t.Error("RemoveEvent() should return true for existing event")
	}
	if webhook.SubscribesToEvent(WebhookEventMailboxCreated) {
		t.Error("RemoveEvent() should remove the event")
	}

	// Try to remove non-existent event
	removed = webhook.RemoveEvent(WebhookEventMailboxDeleted)
	if removed {
		t.Error("RemoveEvent() should return false for non-existent event")
	}
}

func TestWebhook_RecordSuccess(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})
	webhook.RetryCount = 2
	webhook.LastError = "previous error"

	webhook.RecordSuccess()

	if webhook.SuccessCount != 1 {
		t.Errorf("RecordSuccess() SuccessCount = %v, want 1", webhook.SuccessCount)
	}
	if webhook.RetryCount != 0 {
		t.Errorf("RecordSuccess() should reset RetryCount to 0, got %v", webhook.RetryCount)
	}
	if webhook.LastError != "" {
		t.Errorf("RecordSuccess() should clear LastError, got %v", webhook.LastError)
	}
	if webhook.LastSuccessAt == nil {
		t.Error("RecordSuccess() should set LastSuccessAt")
	}
	if webhook.LastTriggeredAt == nil {
		t.Error("RecordSuccess() should set LastTriggeredAt")
	}
}

func TestWebhook_RecordFailure(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})
	webhook.MaxRetries = 2

	webhook.RecordFailure("Connection timeout")

	if webhook.FailureCount != 1 {
		t.Errorf("RecordFailure() FailureCount = %v, want 1", webhook.FailureCount)
	}
	if webhook.RetryCount != 1 {
		t.Errorf("RecordFailure() RetryCount = %v, want 1", webhook.RetryCount)
	}
	if webhook.LastError != "Connection timeout" {
		t.Errorf("RecordFailure() LastError = %v, want %v", webhook.LastError, "Connection timeout")
	}
	if webhook.Status != WebhookStatusActive {
		t.Errorf("RecordFailure() should keep status active when retries available")
	}

	// Exceed max retries
	webhook.RecordFailure("Error 2")
	webhook.RecordFailure("Error 3")

	if webhook.Status != WebhookStatusFailed {
		t.Errorf("RecordFailure() should mark as failed when max retries exceeded")
	}
}

func TestWebhook_ShouldRetry(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})
	webhook.MaxRetries = 3

	if !webhook.ShouldRetry() {
		t.Error("ShouldRetry() should return true initially")
	}

	webhook.RetryCount = 3
	if !webhook.ShouldRetry() {
		t.Error("ShouldRetry() should return true when at max retries")
	}

	webhook.RetryCount = 4
	if webhook.ShouldRetry() {
		t.Error("ShouldRetry() should return false when past max retries")
	}

	webhook.RetryCount = 0
	webhook.Status = WebhookStatusFailed
	if webhook.ShouldRetry() {
		t.Error("ShouldRetry() should return false when status is failed")
	}
}

func TestWebhook_SetRemoveHeader(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})

	webhook.SetHeader("X-Custom", "value")

	if webhook.Headers["X-Custom"] != "value" {
		t.Errorf("SetHeader() should set header, got %v", webhook.Headers["X-Custom"])
	}

	webhook.RemoveHeader("X-Custom")

	if _, exists := webhook.Headers["X-Custom"]; exists {
		t.Error("RemoveHeader() should remove header")
	}
}

func TestWebhookCreateInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *WebhookCreateInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: &WebhookCreateInput{
				Name:   "Test",
				URL:    "https://example.com/webhook",
				Events: []WebhookEvent{WebhookEventMessageReceived},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			input: &WebhookCreateInput{
				URL:    "https://example.com/webhook",
				Events: []WebhookEvent{WebhookEventMessageReceived},
			},
			wantErr: true,
		},
		{
			name: "missing url",
			input: &WebhookCreateInput{
				Name:   "Test",
				Events: []WebhookEvent{WebhookEventMessageReceived},
			},
			wantErr: true,
		},
		{
			name: "no events",
			input: &WebhookCreateInput{
				Name:   "Test",
				URL:    "https://example.com/webhook",
				Events: []WebhookEvent{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WebhookCreateInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebhookUpdateInput_Apply(t *testing.T) {
	webhook := NewWebhook(ID("wh1"), ID("u1"), "Test", "https://example.com/webhook", []WebhookEvent{WebhookEventMessageReceived})

	newName := "Updated Name"
	newURL := "https://new.example.com/webhook"
	newSecret := "newsecret"
	newStatus := WebhookStatusInactive
	newMaxRetries := 5
	newTimeout := 45

	input := &WebhookUpdateInput{
		Name:           &newName,
		URL:            &newURL,
		Secret:         &newSecret,
		Events:         []WebhookEvent{WebhookEventMailboxCreated},
		Status:         &newStatus,
		MaxRetries:     &newMaxRetries,
		TimeoutSeconds: &newTimeout,
	}

	input.Apply(webhook)

	if webhook.Name != newName {
		t.Errorf("Apply() Name = %v, want %v", webhook.Name, newName)
	}
	if webhook.URL != newURL {
		t.Errorf("Apply() URL = %v, want %v", webhook.URL, newURL)
	}
	if webhook.Secret != newSecret {
		t.Errorf("Apply() Secret = %v, want %v", webhook.Secret, newSecret)
	}
	if len(webhook.Events) != 1 || webhook.Events[0] != WebhookEventMailboxCreated {
		t.Errorf("Apply() Events = %v", webhook.Events)
	}
	if webhook.Status != newStatus {
		t.Errorf("Apply() Status = %v, want %v", webhook.Status, newStatus)
	}
	if webhook.MaxRetries != newMaxRetries {
		t.Errorf("Apply() MaxRetries = %v, want %v", webhook.MaxRetries, newMaxRetries)
	}
	if webhook.TimeoutSeconds != newTimeout {
		t.Errorf("Apply() TimeoutSeconds = %v, want %v", webhook.TimeoutSeconds, newTimeout)
	}
}

func TestWebhookDelivery(t *testing.T) {
	delivery := NewWebhookDelivery(ID("del1"), ID("wh1"), WebhookEventMessageReceived, `{"test": true}`, 1)

	if delivery.ID != ID("del1") {
		t.Errorf("NewWebhookDelivery().ID = %v, want %v", delivery.ID, "del1")
	}
	if delivery.WebhookID != ID("wh1") {
		t.Errorf("NewWebhookDelivery().WebhookID = %v, want %v", delivery.WebhookID, "wh1")
	}
	if delivery.Event != WebhookEventMessageReceived {
		t.Errorf("NewWebhookDelivery().Event = %v, want %v", delivery.Event, WebhookEventMessageReceived)
	}
	if delivery.AttemptNumber != 1 {
		t.Errorf("NewWebhookDelivery().AttemptNumber = %v, want 1", delivery.AttemptNumber)
	}
	if delivery.Success {
		t.Error("NewWebhookDelivery().Success should be false")
	}
}

func TestWebhookDelivery_RecordResult(t *testing.T) {
	delivery := NewWebhookDelivery(ID("del1"), ID("wh1"), WebhookEventMessageReceived, `{}`, 1)

	// Success case
	delivery.RecordResult(200, `{"ok": true}`, nil, 150)

	if !delivery.Success {
		t.Error("RecordResult() with 200 should set Success to true")
	}
	if delivery.StatusCode != 200 {
		t.Errorf("RecordResult() StatusCode = %v, want 200", delivery.StatusCode)
	}
	if delivery.Response != `{"ok": true}` {
		t.Errorf("RecordResult() Response = %v", delivery.Response)
	}
	if delivery.Duration != 150 {
		t.Errorf("RecordResult() Duration = %v, want 150", delivery.Duration)
	}
	if delivery.Error != "" {
		t.Errorf("RecordResult() Error should be empty, got %v", delivery.Error)
	}

	// Error case
	delivery2 := NewWebhookDelivery(ID("del2"), ID("wh1"), WebhookEventMessageReceived, `{}`, 1)
	delivery2.RecordResult(500, "Internal Server Error", errors.New("server error"), 50)

	if delivery2.Success {
		t.Error("RecordResult() with error should set Success to false")
	}
	if delivery2.Error != "server error" {
		t.Errorf("RecordResult() Error = %v, want 'server error'", delivery2.Error)
	}
}
