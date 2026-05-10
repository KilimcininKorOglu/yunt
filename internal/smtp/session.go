package smtp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"yunt/internal/domain"
)

// Session implements the smtp.Session interface.
// It manages the state for a single SMTP transaction (from connection to QUIT).
type Session struct {
	backend    *Backend
	conn       *smtp.Conn
	remoteAddr string
	logger     zerolog.Logger

	// Session state
	from        string
	fromOpts    *smtp.MailOptions
	recipients  []recipientInfo
	messageSize int64

	// Authentication state
	authenticated bool
	authUser      *domain.User

	// TLS security state
	security *ConnectionSecurity

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// recipientInfo holds information about a recipient.
type recipientInfo struct {
	address string
	mailbox *domain.Mailbox
}

// NewSession creates a new SMTP session.
func NewSession(b *Backend, c *smtp.Conn, remoteAddr string) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		backend:    b,
		conn:       c,
		remoteAddr: remoteAddr,
		logger:     b.server.logger.With().Str("remoteAddr", remoteAddr).Logger(),
		recipients: make([]recipientInfo, 0),
		security:   NewConnectionSecurity(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// AuthMechanisms returns the list of supported authentication mechanisms.
// Returns the supported mechanisms if authentication is enabled (required or optional).
// Returns an empty slice if no authenticator is configured.
func (s *Session) AuthMechanisms() []string {
	// Only advertise auth mechanisms if we have an authenticator configured
	if s.backend.AuthEnabled() {
		return SupportedAuthMechanisms()
	}
	return []string{}
}

// Auth handles SMTP authentication.
// Returns sasl.Server for the requested mechanism or an error.
func (s *Session) Auth(mech string) (sasl.Server, error) {
	s.logger.Debug().
		Str("mechanism", mech).
		Msg("auth attempt")

	// Check if authentication is available
	if !s.backend.AuthEnabled() {
		s.logger.Warn().
			Str("mechanism", mech).
			Msg("auth attempted but no authenticator configured")
		return nil, smtp.ErrAuthUnsupported
	}

	authenticator := s.backend.Authenticator()

	switch AuthMechanism(mech) {
	case AuthMechanismPlain:
		return s.createPlainServer(authenticator), nil

	case AuthMechanismLogin:
		return s.createLoginServer(authenticator), nil

	default:
		s.logger.Warn().
			Str("mechanism", mech).
			Msg("unsupported auth mechanism")
		return nil, smtp.ErrAuthUnsupported
	}
}

// createPlainServer creates a PLAIN SASL server for authentication.
func (s *Session) createPlainServer(authenticator *Authenticator) sasl.Server {
	return NewPlainServer(func(identity, username, password string) error {
		// If identity is provided and different from username, use identity
		// This follows RFC 4616
		authUsername := username
		if identity != "" && identity != username {
			// Some clients send identity, we'll use username for auth
			s.logger.Debug().
				Str("identity", identity).
				Str("username", username).
				Msg("PLAIN auth with identity")
		}

		if s.backend.server.rateLimiter != nil && s.backend.server.rateLimiter.IsAuthBlocked(s.remoteAddr) {
			return &smtp.SMTPError{Code: 421, EnhancedCode: smtp.EnhancedCode{4, 7, 0}, Message: "Too many authentication failures, try again later"}
		}

		result, err := authenticator.Authenticate(s.ctx, authUsername, password)
		if err != nil {
			if s.backend.server.rateLimiter != nil {
				s.backend.server.rateLimiter.RecordAuthFailure(s.remoteAddr)
			}
			s.logAuthFailure(authUsername, "PLAIN", err)
			return err
		}

		s.authenticated = true
		s.authUser = result.User
		s.logAuthSuccess(authUsername, "PLAIN")
		return nil
	})
}

// createLoginServer creates a LOGIN SASL server for authentication.
func (s *Session) createLoginServer(authenticator *Authenticator) sasl.Server {
	return NewLoginServer(func(username, password string) error {
		if s.backend.server.rateLimiter != nil && s.backend.server.rateLimiter.IsAuthBlocked(s.remoteAddr) {
			return &smtp.SMTPError{Code: 421, EnhancedCode: smtp.EnhancedCode{4, 7, 0}, Message: "Too many authentication failures, try again later"}
		}

		result, err := authenticator.Authenticate(s.ctx, username, password)
		if err != nil {
			if s.backend.server.rateLimiter != nil {
				s.backend.server.rateLimiter.RecordAuthFailure(s.remoteAddr)
			}
			s.logAuthFailure(username, "LOGIN", err)
			return err
		}

		s.authenticated = true
		s.authUser = result.User
		s.logAuthSuccess(username, "LOGIN")
		return nil
	})
}

// logAuthSuccess logs a successful authentication attempt.
func (s *Session) logAuthSuccess(username, mechanism string) {
	s.logger.Info().
		Str("username", username).
		Str("mechanism", mechanism).
		Str("remoteAddr", s.remoteAddr).
		Msg("authentication successful")
}

// logAuthFailure logs a failed authentication attempt.
// Uses structured logging to track failed attempts without leaking sensitive info.
func (s *Session) logAuthFailure(username, mechanism string, err error) {
	var authErr *AuthenticationError
	reason := "unknown"
	if errors.As(err, &authErr) {
		reason = authErr.Reason
	}

	s.logger.Warn().
		Str("username", username).
		Str("mechanism", mechanism).
		Str("remoteAddr", s.remoteAddr).
		Str("reason", reason).
		Msg("authentication failed")
}

// Mail handles the MAIL FROM command.
// Validates the sender address and size restrictions.
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.logger.Debug().
		Str("from", from).
		Msg("MAIL FROM")

	// Reset session state for new message
	s.Reset()

	// Validate sender address format (basic validation)
	if from == "" {
		// Empty MAIL FROM is allowed for bounce messages (RFC 5321)
		s.from = ""
	} else {
		// Extract address if it's in angle bracket format
		from = extractAddress(from)
		if !isValidEmailFormat(from) {
			// RFC 5321: 553 - Requested action not taken: mailbox name not allowed
			return &smtp.SMTPError{
				Code:         553,
				EnhancedCode: smtp.EnhancedCode{5, 1, 3},
				Message:      "invalid sender address format",
			}
		}
		s.from = from
	}

	// Check SIZE parameter if provided
	if opts != nil && opts.Size > 0 {
		maxSize := s.backend.MaxMessageSize()
		if maxSize > 0 && opts.Size > maxSize {
			// RFC 1870: 552 - Message size exceeds fixed maximum message size
			return &smtp.SMTPError{
				Code:         552,
				EnhancedCode: smtp.EnhancedCode{5, 3, 4},
				Message:      fmt.Sprintf("message size %d exceeds maximum %d", opts.Size, maxSize),
			}
		}
		s.messageSize = opts.Size
	}

	s.fromOpts = opts
	return nil
}

// Rcpt handles the RCPT TO command.
// Validates the recipient address against the database.
func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// Extract address if it's in angle bracket format
	to = extractAddress(to)

	s.logger.Debug().
		Str("to", to).
		Int("recipientCount", len(s.recipients)+1).
		Msg("RCPT TO")

	// Check recipient limit
	maxRecipients := s.backend.MaxRecipients()
	if maxRecipients > 0 && len(s.recipients) >= maxRecipients {
		// RFC 5321: 452 - Too many recipients
		return &smtp.SMTPError{
			Code:         452,
			EnhancedCode: smtp.EnhancedCode{4, 5, 3},
			Message:      fmt.Sprintf("too many recipients, maximum is %d", maxRecipients),
		}
	}

	// Validate address format
	if !isValidEmailFormat(to) {
		// RFC 5321: 553 - Requested action not taken: mailbox name not allowed
		return &smtp.SMTPError{
			Code:         553,
			EnhancedCode: smtp.EnhancedCode{5, 1, 3},
			Message:      "invalid recipient address format",
		}
	}

	// Check for duplicate recipients
	for _, r := range s.recipients {
		if strings.EqualFold(r.address, to) {
			// Silently accept duplicates but don't add them again
			s.logger.Debug().
				Str("to", to).
				Msg("duplicate recipient ignored")
			return nil
		}
	}

	// Validate recipient against database
	if err := s.backend.validateRecipient(s.ctx, to); err != nil {
		return err
	}

	// Get the mailbox for this recipient
	mailbox, _ := s.backend.getMailbox(s.ctx, to)

	s.recipients = append(s.recipients, recipientInfo{
		address: to,
		mailbox: mailbox,
	})

	return nil
}

// Data handles the DATA command.
// Receives the message data and stores it for each recipient.
func (s *Session) Data(r io.Reader) error {
	// Verify we have at least one recipient
	if len(s.recipients) == 0 {
		// RFC 5321: 503 - Bad sequence of commands
		return &smtp.SMTPError{
			Code:         503,
			EnhancedCode: smtp.EnhancedCode{5, 5, 1},
			Message:      "no valid recipients",
		}
	}

	// Check rate limits before accepting message
	if s.backend.server.rateLimiter != nil {
		if err := s.backend.server.rateLimiter.CheckMessage(s.ctx, s.remoteAddr); err != nil {
			s.backend.server.stats.RateLimitRejected()
			s.logger.Warn().
				Err(err).
				Str("from", s.from).
				Msg("message rejected by rate limiter")
			return err
		}
	}

	s.logger.Info().
		Str("from", s.from).
		Int("recipientCount", len(s.recipients)).
		Msg("receiving message data")

	// Read the message data with size limit enforcement
	maxSize := s.backend.MaxMessageSize()
	var data []byte
	var err error

	if maxSize > 0 {
		// Use a limited reader to enforce size limits
		limitedReader := &limitedReader{
			r:        r,
			maxSize:  maxSize,
			readSize: 0,
		}
		data, err = io.ReadAll(limitedReader)
		if limitedReader.exceeded {
			// RFC 5321: 552 - Message exceeds fixed maximum message size
			return &smtp.SMTPError{
				Code:         552,
				EnhancedCode: smtp.EnhancedCode{5, 3, 4},
				Message:      fmt.Sprintf("message size exceeds maximum %d bytes", maxSize),
			}
		}
	} else {
		data, err = io.ReadAll(r)
	}

	if err != nil {
		s.logger.Error().Err(err).Msg("failed to read message data")
		// RFC 5321: 451 - Requested action aborted: local error in processing
		return &smtp.SMTPError{
			Code:         451,
			EnhancedCode: smtp.EnhancedCode{4, 0, 0},
			Message:      "error reading message data",
		}
	}

	messageSize := int64(len(data))

	// Store message for each recipient
	recipientAddresses := make([]string, 0, len(s.recipients))
	for _, r := range s.recipients {
		recipientAddresses = append(recipientAddresses, r.address)
	}

	s.logger.Info().
		Str("from", s.from).
		Strs("to", recipientAddresses).
		Int64("size", messageSize).
		Msg("message received")

	// Create and store messages for each recipient with a unique mailbox
	mailboxesSeen := make(map[string]bool)
	for _, recipient := range s.recipients {
		if recipient.mailbox == nil {
			// No mailbox, message will be logged but not stored
			continue
		}

		// Avoid storing duplicate messages for the same mailbox
		mailboxID := recipient.mailbox.ID.String()
		if mailboxesSeen[mailboxID] {
			continue
		}
		mailboxesSeen[mailboxID] = true

		// Create the message
		msg := s.createMessage(recipient.mailbox.ID, data, messageSize, recipientAddresses)

		// Store the message
		if err := s.backend.storeMessage(s.ctx, msg); err != nil {
			s.logger.Error().
				Err(err).
				Str("mailboxId", mailboxID).
				Msg("failed to store message")
			// Continue with other recipients even if one fails
		} else {
			s.logger.Debug().
				Str("messageId", msg.ID.String()).
				Str("mailboxId", mailboxID).
				Msg("message stored")
		}
	}

	s.backend.server.stats.MessageReceived()

	if s.backend.server.rateLimiter != nil {
		s.backend.server.rateLimiter.OnMessageSent(s.remoteAddr)
	}

	// Relay the message to external SMTP server if enabled
	// This happens after local storage to ensure message is preserved
	s.relayMessage(data, recipientAddresses)

	return nil
}

// relayMessage forwards the message to an external SMTP relay if enabled.
// Relay failures are logged but don't affect the return value of Data().
func (s *Session) relayMessage(data []byte, recipients []string) {
	if !s.backend.RelayEnabled() {
		return
	}

	s.backend.server.stats.RelayAttempted()
	result := s.backend.RelayMessage(s.ctx, s.from, recipients, data)

	if result.Success {
		s.backend.server.stats.RelaySucceeded()
		s.logger.Info().
			Str("from", s.from).
			Strs("recipients", result.Recipients).
			Int("attempts", result.Attempts).
			Dur("duration", result.Duration).
			Msg("message relayed successfully")
	} else {
		s.backend.server.stats.RelayFailed()
		s.logger.Warn().
			Err(result.Error).
			Str("from", s.from).
			Strs("failedRecipients", result.FailedRecipients).
			Int("attempts", result.Attempts).
			Dur("duration", result.Duration).
			Msg("relay failed, message stored locally only")
	}
}

// createMessage creates a new domain.Message from the received data.
func (s *Session) createMessage(mailboxID domain.ID, data []byte, size int64, recipients []string) *domain.Message {
	msgID := domain.ID(uuid.New().String())
	msg := domain.NewMessage(msgID, mailboxID)

	// Set basic fields
	msg.From = domain.EmailAddress{Address: s.from}
	msg.Size = size
	msg.RawBody = data
	msg.ReceivedAt = domain.Now()

	// Add recipients
	for _, addr := range recipients {
		msg.AddRecipient("", addr)
	}

	// Generate a Message-ID header if not present in raw data
	if msg.MessageID == "" {
		msg.MessageID = fmt.Sprintf("<%s@%s>", uuid.New().String(), s.backend.server.config.Domain)
	}

	return msg
}

// Reset resets the session state for a new transaction.
// Called after RSET command or after successful DATA.
func (s *Session) Reset() {
	s.logger.Debug().Msg("session reset")
	s.from = ""
	s.fromOpts = nil
	s.recipients = make([]recipientInfo, 0)
	s.messageSize = 0
}

// Logout is called when the client disconnects.
// Cleans up session resources.
func (s *Session) Logout() error {
	s.logger.Info().Msg("connection closed")
	s.cancel() // Cancel any pending operations
	s.backend.server.stats.ConnectionClosed()

	// Track connection close for rate limiting
	if s.backend.server.rateLimiter != nil {
		s.backend.server.rateLimiter.OnConnectionClosed(s.remoteAddr)
	}

	return nil
}

// limitedReader wraps an io.Reader with a size limit.
type limitedReader struct {
	r        io.Reader
	maxSize  int64
	readSize int64
	exceeded bool
}

// Read implements io.Reader with size limit checking.
func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.exceeded {
		return 0, io.EOF
	}

	n, err = lr.r.Read(p)
	lr.readSize += int64(n)

	if lr.readSize > lr.maxSize {
		lr.exceeded = true
		return n, io.EOF
	}

	return n, err
}

// extractAddress extracts the email address from angle bracket format.
// e.g., "<user@example.com>" -> "user@example.com"
func extractAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "<") && strings.HasSuffix(addr, ">") {
		return addr[1 : len(addr)-1]
	}
	return addr
}

// isValidEmailFormat performs basic email format validation.
// This is a simplified check; full RFC 5322 validation is complex.
func isValidEmailFormat(email string) bool {
	if email == "" {
		return false
	}

	// Must contain exactly one @ symbol
	atIndex := strings.Index(email, "@")
	if atIndex == -1 || atIndex == 0 || atIndex == len(email)-1 {
		return false
	}

	// Check for multiple @ symbols
	if strings.Count(email, "@") != 1 {
		return false
	}

	// Local part and domain must not be empty
	localPart := email[:atIndex]
	domainPart := email[atIndex+1:]

	if localPart == "" || domainPart == "" {
		return false
	}

	// Domain must contain at least one dot (simplified check)
	// Note: This allows for local domains like "localhost" in dev mode
	// A stricter check would require a dot in the domain

	// Check for invalid characters (simplified)
	invalidChars := " \t\r\n"
	if strings.ContainsAny(email, invalidChars) {
		return false
	}

	return true
}

// SessionContext returns the session's context.
func (s *Session) SessionContext() context.Context {
	return s.ctx
}

// RemoteAddr returns the remote address of the client.
func (s *Session) RemoteAddr() string {
	return s.remoteAddr
}

// From returns the envelope sender.
func (s *Session) From() string {
	return s.from
}

// Recipients returns the list of recipient addresses.
func (s *Session) Recipients() []string {
	addrs := make([]string, len(s.recipients))
	for i, r := range s.recipients {
		addrs[i] = r.address
	}
	return addrs
}

// RecipientCount returns the number of recipients.
func (s *Session) RecipientCount() int {
	return len(s.recipients)
}

// Deadline returns the session deadline if set.
func (s *Session) Deadline() (time.Time, bool) {
	return s.ctx.Deadline()
}

// IsAuthenticated returns true if the session has been authenticated.
func (s *Session) IsAuthenticated() bool {
	return s.authenticated
}

// AuthenticatedUser returns the authenticated user, or nil if not authenticated.
func (s *Session) AuthenticatedUser() *domain.User {
	return s.authUser
}
