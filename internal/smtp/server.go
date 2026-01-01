package smtp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/rs/zerolog"

	"yunt/internal/repository"
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

// Stats holds server statistics.
type Stats struct {
	mu               sync.RWMutex
	startTime        time.Time
	connectionsOpen  int64
	connectionsTotal int64
	messagesTotal    int64
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

	// Create the SMTP backend with repository options
	backendOpts := make([]BackendOption, 0)
	if s.mailboxRepo != nil {
		backendOpts = append(backendOpts, WithMailboxRepository(s.mailboxRepo))
	}
	if s.messageRepo != nil {
		backendOpts = append(backendOpts, WithMessageRepository(s.messageRepo))
	}
	backend := NewBackend(s, backendOpts...)
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
