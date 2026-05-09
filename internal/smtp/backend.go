package smtp

import (
	"context"
	"fmt"

	"github.com/emersion/go-smtp"

	"yunt/internal/domain"
	"yunt/internal/parser"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// Backend implements the smtp.Backend interface.
// It manages SMTP connections and creates sessions for handling mail transactions.
type Backend struct {
	server        *Server
	mailboxRepo   repository.MailboxRepository
	messageRepo   repository.MessageRepository
	repo          repository.Repository
	authenticator *Authenticator
	relayService  *service.RelayService
	mimeParser    *parser.Parser
}

// BackendOption is a functional option for configuring the Backend.
type BackendOption func(*Backend)

// WithMailboxRepository sets the mailbox repository for recipient validation.
func WithMailboxRepository(repo repository.MailboxRepository) BackendOption {
	return func(b *Backend) {
		b.mailboxRepo = repo
	}
}

// WithMessageRepository sets the message repository for storing messages.
func WithMessageRepository(repo repository.MessageRepository) BackendOption {
	return func(b *Backend) {
		b.messageRepo = repo
	}
}

// WithRepository sets the main repository for user authentication.
func WithRepository(repo repository.Repository) BackendOption {
	return func(b *Backend) {
		b.repo = repo
		b.authenticator = NewAuthenticator(repo)
	}
}

// WithRelayService sets the relay service for forwarding messages.
func WithRelayService(svc *service.RelayService) BackendOption {
	return func(b *Backend) {
		b.relayService = svc
	}
}

// NewBackend creates a new SMTP backend with the given options.
func NewBackend(s *Server, opts ...BackendOption) *Backend {
	b := &Backend{
		server:     s,
		mimeParser: parser.NewParser(),
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// NewSession creates a new SMTP session for handling a connection.
// This method is called by the go-smtp library when a new connection is established.
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	remoteAddr := c.Conn().RemoteAddr().String()

	// Check rate limits before accepting connection
	if b.server.rateLimiter != nil {
		if err := b.server.rateLimiter.CheckConnection(context.Background(), remoteAddr); err != nil {
			b.server.stats.RateLimitRejected()
			b.server.logger.Warn().
				Str("remoteAddr", remoteAddr).
				Err(err).
				Msg("connection rejected by rate limiter")
			return nil, err
		}
		// Track connection for rate limiting
		b.server.rateLimiter.OnConnectionOpened(remoteAddr)
	}

	b.server.stats.ConnectionOpened()

	b.server.logger.Info().
		Str("remoteAddr", remoteAddr).
		Str("hostname", c.Hostname()).
		Msg("new connection")

	return NewSession(b, c, remoteAddr), nil
}

// validateRecipient checks if the recipient address is valid and has a mailbox.
// Returns nil if the recipient is valid, or an SMTP error otherwise.
func (b *Backend) validateRecipient(ctx context.Context, address string) error {
	// If no mailbox repository is configured, accept all recipients
	// This is useful for development/testing without a database
	if b.mailboxRepo == nil {
		return nil
	}

	// Try to find a matching mailbox for the address
	mailbox, err := b.mailboxRepo.FindMatchingMailbox(ctx, address)
	if err != nil {
		if domain.IsNotFound(err) {
			// RFC 5321: 550 - Requested action not taken: mailbox unavailable
			return &smtp.SMTPError{
				Code:         550,
				EnhancedCode: smtp.EnhancedCode{5, 1, 1},
				Message:      fmt.Sprintf("mailbox unavailable: %s", address),
			}
		}
		// Internal error - log and return temporary failure
		b.server.logger.Error().
			Err(err).
			Str("address", address).
			Msg("failed to validate recipient")
		// RFC 5321: 451 - Requested action aborted: local error in processing
		return &smtp.SMTPError{
			Code:         451,
			EnhancedCode: smtp.EnhancedCode{4, 0, 0},
			Message:      "temporary failure, please try again later",
		}
	}

	b.server.logger.Debug().
		Str("address", address).
		Str("mailboxId", mailbox.ID.String()).
		Str("mailboxName", mailbox.Name).
		Msg("recipient validated")

	return nil
}

// storeMessage stores a received message in the repository.
// Returns nil if successful, or an error otherwise.
func (b *Backend) storeMessage(ctx context.Context, msg *domain.Message) error {
	if b.messageRepo == nil {
		// No message repository configured - message is effectively dropped
		b.server.logger.Warn().
			Str("from", msg.From.Address).
			Msg("message received but no repository configured to store it")
		return nil
	}

	if len(msg.RawBody) > 0 && b.mimeParser != nil {
		parsed, err := b.mimeParser.Parse(msg.RawBody)
		if err != nil {
			b.server.logger.Warn().
				Err(err).
				Str("from", msg.From.Address).
				Msg("failed to parse MIME content, storing with envelope data only")
		} else {
			envelopeFrom := msg.From.Address
			parsed.ApplyTo(msg)
			if msg.From.Address == "" {
				msg.From.Address = envelopeFrom
			}
		}
	}

	if err := b.messageRepo.Create(ctx, msg); err != nil {
		b.server.logger.Error().
			Err(err).
			Str("from", msg.From.Address).
			Msg("failed to store message")
		return err
	}

	return nil
}

// getMailbox retrieves a mailbox by address for message delivery.
func (b *Backend) getMailbox(ctx context.Context, address string) (*domain.Mailbox, error) {
	if b.mailboxRepo == nil {
		return nil, nil
	}
	return b.mailboxRepo.FindMatchingMailbox(ctx, address)
}

// MaxMessageSize returns the maximum message size in bytes.
func (b *Backend) MaxMessageSize() int64 {
	return b.server.config.MaxMessageSize
}

// MaxRecipients returns the maximum number of recipients per message.
func (b *Backend) MaxRecipients() int {
	return b.server.config.MaxRecipients
}

// AuthRequired returns true if authentication is required.
func (b *Backend) AuthRequired() bool {
	return b.server.config.AuthRequired
}

// Authenticator returns the authenticator for SMTP authentication.
// Returns nil if no repository is configured.
func (b *Backend) Authenticator() *Authenticator {
	return b.authenticator
}

// AuthEnabled returns true if authentication is available (repository is configured).
func (b *Backend) AuthEnabled() bool {
	return b.authenticator != nil
}

// RelayEnabled returns true if relay functionality is enabled.
func (b *Backend) RelayEnabled() bool {
	if b.relayService == nil {
		return false
	}
	return b.relayService.IsEnabled()
}

// RelayMessage forwards a message to the external relay server.
func (b *Backend) RelayMessage(ctx context.Context, from string, recipients []string, data []byte) *service.RelayResult {
	if b.relayService == nil {
		return &service.RelayResult{
			Success: false,
			Error:   fmt.Errorf("relay service not configured"),
		}
	}
	return b.relayService.Relay(ctx, from, recipients, data)
}
