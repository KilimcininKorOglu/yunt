package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/rs/zerolog"

	"yunt/internal/repository"
	"yunt/internal/service"
)

// Server represents an SMTP server instance.
type Server struct {
	config   *Config
	server   *smtp.Server
	logger   zerolog.Logger
	listener net.Listener
	backend  *Backend

	// Repositories for message handling
	mailboxRepo repository.MailboxRepository
	messageRepo repository.MessageRepository
	repo        repository.Repository

	// Relay service for forwarding messages
	relayService *service.RelayService

	// Notification service for real-time events
	notifyService *service.NotifyService

	// Rate limiter for connection and message throttling
	rateLimiter *RateLimiter

	// State management
	running  atomic.Bool
	mu       sync.RWMutex
	doneChan chan struct{}

	// Statistics
	stats *Stats
}

// ServerOption is a functional option for configuring the Server.
type ServerOption func(*Server)

// WithMailboxRepo sets the mailbox repository for recipient validation.
func WithMailboxRepo(repo repository.MailboxRepository) ServerOption {
	return func(s *Server) {
		s.mailboxRepo = repo
	}
}

// WithMessageRepo sets the message repository for message storage.
func WithMessageRepo(repo repository.MessageRepository) ServerOption {
	return func(s *Server) {
		s.messageRepo = repo
	}
}

// WithRepo sets the main repository for user authentication.
func WithRepo(repo repository.Repository) ServerOption {
	return func(s *Server) {
		s.repo = repo
	}
}

// WithNotifyService sets the notification service for real-time events.
func WithNotifyService(ns *service.NotifyService) ServerOption {
	return func(s *Server) {
		s.notifyService = ns
	}
}

// Stats holds server statistics.
type Stats struct {
	mu                    sync.RWMutex
	startTime             time.Time
	connectionsOpen       int64
	connectionsTotal      int64
	messagesTotal         int64
	tlsConnectionsTotal   int64
	startTLSUpgradesTotal int64
	rateLimitRejected     int64
	relayAttemptsTotal    int64
	relaySuccessesTotal   int64
	relayFailuresTotal    int64
}

// NewStats creates a new Stats instance.
func NewStats() *Stats {
	return &Stats{
		startTime: time.Now(),
	}
}

// ConnectionOpened increments the open connections counter.
func (s *Stats) ConnectionOpened() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connectionsOpen++
	s.connectionsTotal++
}

// ConnectionClosed decrements the open connections counter.
func (s *Stats) ConnectionClosed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connectionsOpen > 0 {
		s.connectionsOpen--
	}
}

// MessageReceived increments the messages counter.
func (s *Stats) MessageReceived() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messagesTotal++
}

// GetStats returns the current statistics.
func (s *Stats) GetStats() (uptime time.Duration, connectionsOpen, connectionsTotal, messagesTotal int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.startTime), s.connectionsOpen, s.connectionsTotal, s.messagesTotal
}

// TLSConnectionOpened increments the TLS connections counter.
func (s *Stats) TLSConnectionOpened() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tlsConnectionsTotal++
}

// StartTLSUpgraded increments the STARTTLS upgrades counter.
func (s *Stats) StartTLSUpgraded() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startTLSUpgradesTotal++
}

// GetTLSStats returns TLS-related statistics.
func (s *Stats) GetTLSStats() (tlsConnectionsTotal, startTLSUpgradesTotal int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tlsConnectionsTotal, s.startTLSUpgradesTotal
}

// RateLimitRejected increments the rate limit rejected counter.
func (s *Stats) RateLimitRejected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateLimitRejected++
}

// RelayAttempted increments the relay attempts counter.
func (s *Stats) RelayAttempted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.relayAttemptsTotal++
}

// RelaySucceeded increments the relay success counter.
func (s *Stats) RelaySucceeded() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.relaySuccessesTotal++
}

// RelayFailed increments the relay failure counter.
func (s *Stats) RelayFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.relayFailuresTotal++
}

// GetRelayStats returns relay-related statistics.
func (s *Stats) GetRelayStats() (attempts, successes, failures int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.relayAttemptsTotal, s.relaySuccessesTotal, s.relayFailuresTotal
}

// New creates a new SMTP server with the given configuration and logger.
// Optional ServerOptions can be provided to inject repositories for
// recipient validation and message storage.
func New(cfg *Config, logger zerolog.Logger, opts ...ServerOption) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	s := &Server{
		config:   cfg,
		logger:   logger.With().Str("component", "smtp").Logger(),
		doneChan: make(chan struct{}),
		stats:    NewStats(),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create rate limiter if not provided via options
	if s.rateLimiter == nil && cfg.RateLimitConfig != nil {
		s.rateLimiter = NewRateLimiter(cfg.RateLimitConfig, s.logger)
	} else if s.rateLimiter == nil && cfg.RateLimitEnabled {
		// Use default rate limit config if enabled but no config provided
		s.rateLimiter = NewRateLimiter(DefaultRateLimitConfig(), s.logger)
	}

	// Create the SMTP backend with repository options
	backendOpts := make([]BackendOption, 0)
	if s.mailboxRepo != nil {
		backendOpts = append(backendOpts, WithMailboxRepository(s.mailboxRepo))
	}
	if s.messageRepo != nil {
		backendOpts = append(backendOpts, WithMessageRepository(s.messageRepo))
	}
	if s.repo != nil {
		backendOpts = append(backendOpts, WithRepository(s.repo))
	}
	backend := NewBackend(s, backendOpts...)
	backend.notifyService = s.notifyService
	s.backend = backend

	// Create the go-smtp server
	smtpServer := smtp.NewServer(backend)
	smtpServer.Addr = cfg.Addr()
	smtpServer.Domain = cfg.Domain
	smtpServer.ReadTimeout = cfg.ReadTimeout
	smtpServer.WriteTimeout = cfg.WriteTimeout
	smtpServer.MaxMessageBytes = cfg.MaxMessageSize
	smtpServer.MaxRecipients = cfg.MaxRecipients
	smtpServer.AllowInsecureAuth = cfg.AllowInsecureAuth

	// Set TLS configuration if available
	if cfg.TLSConfig != nil {
		smtpServer.TLSConfig = cfg.TLSConfig
	}

	// Set up debug logging if debug level is enabled
	if logger.GetLevel() <= zerolog.DebugLevel {
		smtpServer.Debug = &logWriter{logger: logger}
	}

	// Set error logger
	smtpServer.ErrorLog = &smtpErrorLogger{logger: logger}

	s.server = smtpServer

	return s, nil
}

// Start starts the SMTP server.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return fmt.Errorf("server already running")
	}

	addr := s.config.Addr()
	s.logger.Info().
		Str("addr", addr).
		Str("domain", s.config.Domain).
		Int64("maxMessageSize", s.config.MaxMessageSize).
		Int("maxRecipients", s.config.MaxRecipients).
		Dur("readTimeout", s.config.ReadTimeout).
		Dur("writeTimeout", s.config.WriteTimeout).
		Bool("authRequired", s.config.AuthRequired).
		Bool("tlsEnabled", s.config.TLSConfig != nil).
		Bool("startTLS", s.config.EnableStartTLS).
		Msg("starting SMTP server")

	// Create listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	s.running.Store(true)
	s.stats = NewStats()

	// Start serving in a goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && s.running.Load() {
			s.logger.Error().Err(err).Msg("SMTP server error")
		}
		close(s.doneChan)
	}()

	s.logger.Info().Str("addr", addr).Msg("SMTP server started")
	return nil
}

// Stop gracefully stops the SMTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running.Load() {
		return nil
	}

	s.logger.Info().Msg("stopping SMTP server")
	s.running.Store(false)

	// Stop the rate limiter
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}

	// Use graceful timeout from config if context has no deadline
	shutdownCtx := ctx
	if _, ok := ctx.Deadline(); !ok && s.config.GracefulTimeout > 0 {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(ctx, s.config.GracefulTimeout)
		defer cancel()
	}

	// Attempt graceful shutdown
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Warn().Err(err).Msg("graceful shutdown failed, forcing close")
		if closeErr := s.server.Close(); closeErr != nil {
			return fmt.Errorf("failed to close server: %w", closeErr)
		}
	}

	// Wait for server to finish
	select {
	case <-s.doneChan:
		s.logger.Info().Msg("SMTP server stopped")
	case <-shutdownCtx.Done():
		s.logger.Warn().Msg("shutdown context expired")
	}

	return nil
}

// IsRunning returns true if the server is running.
func (s *Server) IsRunning() bool {
	return s.running.Load()
}

// Config returns the server configuration.
func (s *Server) Config() *Config {
	return s.config
}

// Stats returns the server statistics.
func (s *Server) Stats() *Stats {
	return s.stats
}

// RateLimiter returns the rate limiter instance.
func (s *Server) RateLimiter() *RateLimiter {
	return s.rateLimiter
}

// Addr returns the server address.
// Returns empty string if server is not running.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// logWriter implements io.Writer for debug logging.
type logWriter struct {
	logger zerolog.Logger
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.logger.Debug().Str("debug", string(p)).Msg("smtp protocol")
	return len(p), nil
}

// smtpErrorLogger implements smtp.Logger interface for error logging.
type smtpErrorLogger struct {
	logger zerolog.Logger
}

func (l *smtpErrorLogger) Printf(format string, v ...interface{}) {
	l.logger.Error().Msgf(format, v...)
}

func (l *smtpErrorLogger) Println(v ...interface{}) {
	l.logger.Error().Msg(fmt.Sprint(v...))
}

// Backend returns the SMTP backend.
func (s *Server) Backend() *Backend {
	return s.backend
}

// Security returns the connection security state.
func (s *Session) Security() *ConnectionSecurity {
	return s.security
}

// updateTLSState updates the security state after STARTTLS handshake.
// This is called internally when the connection is upgraded to TLS.
func (s *Session) updateTLSState() {
	if tlsConn, ok := s.conn.Conn().(*tls.Conn); ok {
		tlsState := tlsConn.ConnectionState()
		s.security.UpdateFromTLSState(tlsState, TLSStateStartTLS)
		s.backend.server.stats.StartTLSUpgraded()
		s.logger.Info().
			Str("tlsState", s.security.TLSState().String()).
			Str("tlsVersion", s.security.TLSVersion()).
			Str("cipherSuite", s.security.CipherSuite()).
			Msg("connection upgraded to TLS via STARTTLS")
	}
}

// IsTLS returns true if the connection is secured with TLS.
func (s *Session) IsTLS() bool {
	return s.security.IsSecure()
}
