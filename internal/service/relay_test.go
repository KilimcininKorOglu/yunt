package service

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestRelayConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RelayConfig
		wantErr bool
	}{
		{
			name: "disabled relay requires no config",
			config: &RelayConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "enabled relay requires host",
			config: &RelayConfig{
				Enabled: true,
				Host:    "",
				Port:    587,
			},
			wantErr: true,
		},
		{
			name: "enabled relay with valid config",
			config: &RelayConfig{
				Enabled: true,
				Host:    "smtp.example.com",
				Port:    587,
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			config: &RelayConfig{
				Enabled: true,
				Host:    "smtp.example.com",
				Port:    0,
			},
			wantErr: true,
		},
		{
			name: "port too high",
			config: &RelayConfig{
				Enabled: true,
				Host:    "smtp.example.com",
				Port:    70000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRelayConfig_Defaults(t *testing.T) {
	cfg := DefaultRelayConfig()

	if cfg.Enabled {
		t.Error("default should have relay disabled")
	}

	if cfg.Port != 587 {
		t.Errorf("default port should be 587, got %d", cfg.Port)
	}

	if !cfg.UseSTARTTLS {
		t.Error("default should use STARTTLS")
	}

	if cfg.UseTLS {
		t.Error("default should not use implicit TLS")
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("default timeout should be 30s, got %v", cfg.Timeout)
	}

	if cfg.RetryCount != 3 {
		t.Errorf("default retry count should be 3, got %d", cfg.RetryCount)
	}
}

func TestNewRelayService(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		config  *RelayConfig
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid disabled config",
			config: &RelayConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid enabled config",
			config: &RelayConfig{
				Enabled: true,
				Host:    "smtp.example.com",
				Port:    587,
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			config: &RelayConfig{
				Enabled: true,
				Host:    "",
				Port:    587,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewRelayService(tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRelayService() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && svc == nil {
				t.Error("NewRelayService() returned nil service without error")
			}
		})
	}
}

func TestRelayService_IsEnabled(t *testing.T) {
	logger := zerolog.Nop()

	// Disabled relay
	cfg := &RelayConfig{Enabled: false}
	svc, _ := NewRelayService(cfg, logger)
	if svc.IsEnabled() {
		t.Error("IsEnabled() should return false for disabled relay")
	}

	// Enabled relay
	cfg = &RelayConfig{
		Enabled: true,
		Host:    "smtp.example.com",
		Port:    587,
	}
	svc, _ = NewRelayService(cfg, logger)
	if !svc.IsEnabled() {
		t.Error("IsEnabled() should return true for enabled relay")
	}
}

func TestRelayService_IsDomainAllowed(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name           string
		allowedDomains []string
		testDomain     string
		want           bool
	}{
		{
			name:           "all domains allowed when list is empty",
			allowedDomains: nil,
			testDomain:     "example.com",
			want:           true,
		},
		{
			name:           "allowed domain",
			allowedDomains: []string{"example.com", "test.com"},
			testDomain:     "example.com",
			want:           true,
		},
		{
			name:           "disallowed domain",
			allowedDomains: []string{"example.com"},
			testDomain:     "other.com",
			want:           false,
		},
		{
			name:           "case insensitive match",
			allowedDomains: []string{"EXAMPLE.COM"},
			testDomain:     "example.com",
			want:           true,
		},
		{
			name:           "whitespace trimmed",
			allowedDomains: []string{" example.com "},
			testDomain:     "example.com",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RelayConfig{
				Enabled:        true,
				Host:           "smtp.relay.com",
				Port:           587,
				AllowedDomains: tt.allowedDomains,
			}
			svc, _ := NewRelayService(cfg, logger)

			got := svc.IsDomainAllowed(tt.testDomain)
			if got != tt.want {
				t.Errorf("IsDomainAllowed(%q) = %v, want %v", tt.testDomain, got, tt.want)
			}
		})
	}
}

func TestRelayService_IsRecipientAllowed(t *testing.T) {
	logger := zerolog.Nop()

	cfg := &RelayConfig{
		Enabled:        true,
		Host:           "smtp.relay.com",
		Port:           587,
		AllowedDomains: []string{"example.com"},
	}
	svc, _ := NewRelayService(cfg, logger)

	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"user@other.com", false},
		{"invalid-email", false},
		{"@example.com", false},
		{"user@", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got := svc.IsRecipientAllowed(tt.email)
			if got != tt.want {
				t.Errorf("IsRecipientAllowed(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestRelayService_FilterAllowedRecipients(t *testing.T) {
	logger := zerolog.Nop()

	cfg := &RelayConfig{
		Enabled:        true,
		Host:           "smtp.relay.com",
		Port:           587,
		AllowedDomains: []string{"example.com", "test.org"},
	}
	svc, _ := NewRelayService(cfg, logger)

	recipients := []string{
		"user1@example.com",
		"user2@other.com",
		"user3@test.org",
		"user4@invalid",
	}

	allowed := svc.FilterAllowedRecipients(recipients)

	if len(allowed) != 2 {
		t.Errorf("expected 2 allowed recipients, got %d", len(allowed))
	}

	expectedAllowed := map[string]bool{
		"user1@example.com": true,
		"user3@test.org":    true,
	}

	for _, r := range allowed {
		if !expectedAllowed[r] {
			t.Errorf("unexpected recipient in allowed list: %s", r)
		}
	}
}

func TestRelayService_FilterAllowedRecipients_DisabledRelay(t *testing.T) {
	logger := zerolog.Nop()

	cfg := &RelayConfig{
		Enabled: false,
	}
	svc, _ := NewRelayService(cfg, logger)

	recipients := []string{"user@example.com"}
	allowed := svc.FilterAllowedRecipients(recipients)

	if len(allowed) != 0 {
		t.Errorf("expected empty list when relay is disabled, got %d", len(allowed))
	}
}

func TestRelayService_Relay_NotEnabled(t *testing.T) {
	logger := zerolog.Nop()

	cfg := &RelayConfig{
		Enabled: false,
	}
	svc, _ := NewRelayService(cfg, logger)

	result := svc.Relay(context.Background(), "from@example.com", []string{"to@example.com"}, []byte("test"))

	if result.Success {
		t.Error("relay should not succeed when disabled")
	}
	if result.Error == nil {
		t.Error("expected error when relay is disabled")
	}
}

func TestRelayService_Relay_NoAllowedRecipients(t *testing.T) {
	logger := zerolog.Nop()

	cfg := &RelayConfig{
		Enabled:        true,
		Host:           "smtp.relay.com",
		Port:           587,
		AllowedDomains: []string{"allowed.com"},
	}
	svc, _ := NewRelayService(cfg, logger)

	result := svc.Relay(
		context.Background(),
		"from@example.com",
		[]string{"to@disallowed.com"},
		[]byte("test"),
	)

	if result.Success {
		t.Error("relay should not succeed with no allowed recipients")
	}
	if result.Error == nil {
		t.Error("expected error with no allowed recipients")
	}
}

func TestRelayService_GetStats(t *testing.T) {
	logger := zerolog.Nop()

	cfg := &RelayConfig{
		Enabled: false,
	}
	svc, _ := NewRelayService(cfg, logger)

	attempts, successes, failures := svc.GetStats()
	if attempts != 0 || successes != 0 || failures != 0 {
		t.Error("initial stats should be zero")
	}
}

func TestRelayService_UpdateConfig(t *testing.T) {
	logger := zerolog.Nop()

	cfg := &RelayConfig{
		Enabled: false,
	}
	svc, _ := NewRelayService(cfg, logger)

	if svc.IsEnabled() {
		t.Error("relay should be disabled initially")
	}

	// Update to enabled config
	newCfg := &RelayConfig{
		Enabled: true,
		Host:    "new-smtp.example.com",
		Port:    587,
	}
	err := svc.UpdateConfig(newCfg)
	if err != nil {
		t.Errorf("UpdateConfig failed: %v", err)
	}

	if !svc.IsEnabled() {
		t.Error("relay should be enabled after update")
	}

	// Try invalid config
	invalidCfg := &RelayConfig{
		Enabled: true,
		Host:    "",
		Port:    587,
	}
	err = svc.UpdateConfig(invalidCfg)
	if err == nil {
		t.Error("UpdateConfig should fail with invalid config")
	}

	// Verify original config unchanged after invalid update
	if !svc.IsEnabled() {
		t.Error("relay should still be enabled after failed update")
	}
}

func TestRelayResult(t *testing.T) {
	result := &RelayResult{
		Success:          true,
		Recipients:       []string{"user@example.com"},
		FailedRecipients: []string{},
		Attempts:         1,
		Duration:         100 * time.Millisecond,
	}

	if !result.Success {
		t.Error("result should be successful")
	}
	if len(result.Recipients) != 1 {
		t.Error("should have 1 recipient")
	}
	if result.Attempts != 1 {
		t.Error("should have 1 attempt")
	}
}

func TestRelayError(t *testing.T) {
	err := &RelayError{
		Op:        "connect",
		Message:   "connection failed",
		Retryable: true,
	}

	expected := "relay connect: connection failed"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	if !err.IsRetryable() {
		t.Error("error should be retryable")
	}
}
