// Package service provides business logic and service layer implementations
// for the Yunt mail server.
package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// RelayConfig holds the configuration for the SMTP relay service.
type RelayConfig struct {
	// Enabled determines if relay functionality is active.
	Enabled bool

	// Host is the external SMTP server address.
	Host string

	// Port is the external SMTP server port.
	Port int

	// Username for authentication with the relay server.
	Username string

	// Password for authentication with the relay server.
	Password string

	// UseTLS enables TLS for the relay connection.
	UseTLS bool

	// UseSTARTTLS enables STARTTLS upgrade for the relay connection.
	UseSTARTTLS bool

	// AllowedDomains is a list of domains allowed for relay.
	// If empty, all domains are allowed.
	AllowedDomains []string

	// Timeout for relay operations.
	Timeout time.Duration

	// RetryCount is the number of retry attempts for failed relay.
	RetryCount int

	// RetryDelay is the delay between retry attempts.
	RetryDelay time.Duration

	// InsecureSkipVerify skips TLS certificate verification (not recommended for production).
	InsecureSkipVerify bool
}

// Validate validates the relay configuration.
func (c *RelayConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Host == "" {
		return errors.New("relay host is required when relay is enabled")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid relay port: %d", c.Port)
	}

	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	if c.RetryCount < 0 {
		c.RetryCount = 0
	}

	if c.RetryDelay <= 0 {
		c.RetryDelay = 5 * time.Second
	}

	return nil
}

// DefaultRelayConfig returns a RelayConfig with sensible defaults.
func DefaultRelayConfig() *RelayConfig {
	return &RelayConfig{
		Enabled:            false,
		Port:               587,
		UseTLS:             false,
		UseSTARTTLS:        true,
		AllowedDomains:     nil,
		Timeout:            30 * time.Second,
		RetryCount:         3,
		RetryDelay:         5 * time.Second,
		InsecureSkipVerify: false,
	}
}

// RelayResult holds the result of a relay attempt.
type RelayResult struct {
	// Success indicates if the relay was successful.
	Success bool

	// Error contains the error if relay failed.
	Error error

	// Attempts is the number of attempts made.
	Attempts int

	// Duration is the total time taken for the relay operation.
	Duration time.Duration

	// Recipients contains the successfully relayed recipients.
	Recipients []string

	// FailedRecipients contains recipients that failed to relay.
	FailedRecipients []string
}

// RelayService handles forwarding emails to an external SMTP server.
type RelayService struct {
	config *RelayConfig
	logger zerolog.Logger
	mu     sync.RWMutex

	// Statistics
	totalAttempts  int64
	totalSuccesses int64
	totalFailures  int64
}

// NewRelayService creates a new RelayService with the given configuration.
func NewRelayService(config *RelayConfig, logger zerolog.Logger) (*RelayService, error) {
	if config == nil {
		config = DefaultRelayConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid relay config: %w", err)
	}

	return &RelayService{
		config: config,
		logger: logger.With().Str("component", "relay").Logger(),
	}, nil
}

// IsEnabled returns true if relay is enabled.
func (s *RelayService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Enabled
}

// IsDomainAllowed checks if a domain is allowed for relay.
func (s *RelayService) IsDomainAllowed(domain string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.config.Enabled {
		return false
	}

	// If no allowed domains configured, all domains are allowed
	if len(s.config.AllowedDomains) == 0 {
		return true
	}

	domain = strings.ToLower(strings.TrimSpace(domain))
	for _, allowedDomain := range s.config.AllowedDomains {
		if strings.ToLower(strings.TrimSpace(allowedDomain)) == domain {
			return true
		}
	}

	return false
}

// IsRecipientAllowed checks if a recipient email address is allowed for relay.
func (s *RelayService) IsRecipientAllowed(email string) bool {
	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	// Both local part and domain must be non-empty
	localPart := parts[0]
	domain := parts[1]
	if localPart == "" || domain == "" {
		return false
	}

	return s.IsDomainAllowed(domain)
}

// FilterAllowedRecipients filters a list of recipients to only those allowed for relay.
func (s *RelayService) FilterAllowedRecipients(recipients []string) []string {
	if !s.IsEnabled() {
		return nil
	}

	allowed := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if s.IsRecipientAllowed(r) {
			allowed = append(allowed, r)
		}
	}

	return allowed
}

// Relay forwards an email to the external SMTP server.
// It stores the message locally regardless of relay success.
func (s *RelayService) Relay(ctx context.Context, from string, recipients []string, data []byte) *RelayResult {
	start := time.Now()
	result := &RelayResult{
		Recipients: make([]string, 0),
	}

	s.mu.Lock()
	s.totalAttempts++
	s.mu.Unlock()

	if !s.IsEnabled() {
		result.Error = errors.New("relay is not enabled")
		s.logger.Debug().Msg("relay skipped: not enabled")
		return result
	}

	// Filter recipients by allowed domains
	allowedRecipients := s.FilterAllowedRecipients(recipients)
	if len(allowedRecipients) == 0 {
		result.Error = errors.New("no recipients allowed for relay")
		s.logger.Debug().
			Strs("recipients", recipients).
			Msg("relay skipped: no allowed recipients")
		return result
	}

	s.logger.Info().
		Str("from", from).
		Strs("recipients", allowedRecipients).
		Int("dataSize", len(data)).
		Msg("starting relay")

	// Attempt relay with retries
	var lastErr error
	for attempt := 1; attempt <= s.config.RetryCount+1; attempt++ {
		result.Attempts = attempt

		err := s.doRelay(ctx, from, allowedRecipients, data)
		if err == nil {
			result.Success = true
			result.Recipients = allowedRecipients
			result.Duration = time.Since(start)

			s.mu.Lock()
			s.totalSuccesses++
			s.mu.Unlock()

			s.logger.Info().
				Str("from", from).
				Strs("recipients", allowedRecipients).
				Int("attempts", attempt).
				Dur("duration", result.Duration).
				Msg("relay successful")

			return result
		}

		lastErr = err
		s.logger.Warn().
			Err(err).
			Int("attempt", attempt).
			Int("maxAttempts", s.config.RetryCount+1).
			Msg("relay attempt failed")

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			break
		}

		// Wait before retry (except on last attempt)
		if attempt <= s.config.RetryCount {
			select {
			case <-ctx.Done():
				break
			case <-time.After(s.config.RetryDelay):
				// Continue to next attempt
			}
		}
	}

	result.Error = lastErr
	result.FailedRecipients = allowedRecipients
	result.Duration = time.Since(start)

	s.mu.Lock()
	s.totalFailures++
	s.mu.Unlock()

	s.logger.Error().
		Err(lastErr).
		Str("from", from).
		Strs("recipients", allowedRecipients).
		Int("attempts", result.Attempts).
		Dur("duration", result.Duration).
		Msg("relay failed after all attempts")

	return result
}

// doRelay performs a single relay attempt.
func (s *RelayService) doRelay(ctx context.Context, from string, recipients []string, data []byte) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Create connection with timeout
	dialer := &net.Dialer{
		Timeout: s.config.Timeout,
	}

	var conn net.Conn
	var err error

	if s.config.UseTLS {
		// Direct TLS connection (port 465 typically)
		tlsConfig := s.getTLSConfig()
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to relay server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// STARTTLS upgrade if configured and not already using TLS
	if s.config.UseSTARTTLS && !s.config.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := s.getTLSConfig()
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
			s.logger.Debug().Msg("upgraded to TLS via STARTTLS")
		} else {
			s.logger.Warn().Msg("STARTTLS not supported by relay server")
		}
	}

	// Authenticate if credentials provided
	if s.config.Username != "" && s.config.Password != "" {
		auth := s.createAuth()
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		s.logger.Debug().Str("username", s.config.Username).Msg("authenticated with relay server")
	}

	// Send MAIL FROM
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Send RCPT TO for each recipient
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("RCPT TO failed for %s: %w", rcpt, err)
		}
	}

	// Send DATA
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write message data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Quit
	if err := client.Quit(); err != nil {
		// Log but don't fail on QUIT errors
		s.logger.Debug().Err(err).Msg("QUIT command failed (non-fatal)")
	}

	return nil
}

// getTLSConfig returns the TLS configuration for relay connections.
func (s *RelayService) getTLSConfig() *tls.Config {
	return &tls.Config{
		ServerName:         s.config.Host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: s.config.InsecureSkipVerify,
	}
}

// createAuth creates the SMTP authentication mechanism.
func (s *RelayService) createAuth() smtp.Auth {
	return smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
}

// GetStats returns relay statistics.
func (s *RelayService) GetStats() (attempts, successes, failures int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.totalAttempts, s.totalSuccesses, s.totalFailures
}

// UpdateConfig updates the relay configuration.
func (s *RelayService) UpdateConfig(config *RelayConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid relay config: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config

	s.logger.Info().
		Bool("enabled", config.Enabled).
		Str("host", config.Host).
		Int("port", config.Port).
		Bool("useTLS", config.UseTLS).
		Bool("useSTARTTLS", config.UseSTARTTLS).
		Strs("allowedDomains", config.AllowedDomains).
		Msg("relay configuration updated")

	return nil
}

// RelayError represents an error that occurred during relay.
type RelayError struct {
	// Op is the operation that failed.
	Op string
	// Message is a human-readable error description.
	Message string
	// Err is the underlying error.
	Err error
	// Retryable indicates if the error is retryable.
	Retryable bool
}

// Error implements the error interface.
func (e *RelayError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("relay %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("relay %s: %s", e.Op, e.Message)
}

// Unwrap returns the underlying error.
func (e *RelayError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is retryable.
func (e *RelayError) IsRetryable() bool {
	return e.Retryable
}
