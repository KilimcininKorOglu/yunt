// Package imap provides an IMAP server implementation for the Yunt mail server.
package imap

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"yunt/internal/metrics"
)

// Server represents the IMAP server instance.
type Server struct {
	config     *Config
	server     *imapserver.Server
	listener   net.Listener
	logger     zerolog.Logger
	running    atomic.Bool
	connCount  atomic.Int64
	mu         sync.RWMutex
	shutdownCh chan struct{}
	wg         sync.WaitGroup

	// backend provides user authentication and session management.
	backend *Backend

	// idleManager manages IDLE sessions for real-time notifications.
	idleManager *IdleManager
}

// NewServer creates a new IMAP server with the given configuration.
func NewServer(cfg *Config, logger zerolog.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	s := &Server{
		config:     cfg,
		logger:     logger.With().Str("component", "imap").Logger(),
		shutdownCh: make(chan struct{}),
	}

	return s, nil
}

// SetBackend sets the IMAP backend for authentication and data access.
// This should be called before Start() to enable user authentication.
func (s *Server) SetBackend(backend *Backend) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backend = backend
	s.logger.Info().Msg("IMAP backend configured")
}

// SetIdleManager sets the IDLE manager for real-time notifications.
// This should be called before Start() to enable IDLE notifications.
func (s *Server) SetIdleManager(manager *IdleManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idleManager = manager
	s.logger.Info().Msg("IMAP IDLE manager configured")
}

// IdleManager returns the current idle manager, or nil if not set.
func (s *Server) IdleManager() *IdleManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.idleManager
}

// Backend returns the current backend, or nil if not set.
func (s *Server) Backend() *Backend {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.backend
}

// Start starts the IMAP server and begins accepting connections.
func (s *Server) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info().Msg("IMAP server is disabled")
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return fmt.Errorf("imap: server is already running")
	}

	// Load TLS configuration if needed using the enhanced TLS loader
	tlsLoader := NewTLSLoader(s.logger)
	tlsConfig, err := tlsLoader.LoadTLSConfig(&s.config.TLS)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to load TLS configuration")
		return err
	}

	// Create IMAP server options
	opts := &imapserver.Options{
		NewSession:   s.newSession,
		Caps:         s.buildCapabilities(),
		Logger:       &serverLogger{logger: s.logger},
		TLSConfig:    tlsConfig,
		InsecureAuth: s.config.InsecureAuth,
	}

	// Create the IMAP server
	s.server = imapserver.New(opts)

	// Create listener
	addr := s.config.Address()
	var listener net.Listener

	if s.config.TLS.Enabled {
		// Implicit TLS
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("imap: failed to create listener: %w", err)
		}
		s.logger.Info().
			Str("address", addr).
			Bool("tls", true).
			Msg("IMAP server listening with implicit TLS")
	} else {
		// Plain text (with optional STARTTLS)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("imap: failed to create listener: %w", err)
		}
		s.logger.Info().
			Str("address", addr).
			Bool("starttls", s.config.TLS.StartTLS).
			Msg("IMAP server listening")
	}

	s.listener = listener
	s.running.Store(true)

	// Start serving in a goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.server.Serve(s.listener); err != nil {
			select {
			case <-s.shutdownCh:
				// Expected during shutdown
				s.logger.Debug().Msg("IMAP server stopped serving")
			default:
				s.logger.Error().Err(err).Msg("IMAP server error")
			}
		}
	}()

	s.logger.Info().
		Str("address", addr).
		Str("server_name", s.config.ServerName).
		Dur("read_timeout", s.config.ReadTimeout).
		Dur("write_timeout", s.config.WriteTimeout).
		Dur("idle_timeout", s.config.IdleTimeout).
		Msg("IMAP server started successfully")

	return nil
}

// Stop gracefully stops the IMAP server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running.Load() {
		return nil
	}

	s.logger.Info().Msg("Stopping IMAP server...")
	close(s.shutdownCh)

	// Close the server
	if s.server != nil {
		if err := s.server.Close(); err != nil {
			s.logger.Error().Err(err).Msg("Error closing IMAP server")
		}
	}

	// Wait for all connections to close with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info().Msg("IMAP server stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn().Msg("IMAP server shutdown timed out")
	}

	s.running.Store(false)
	return nil
}

// IsRunning returns true if the server is currently running.
func (s *Server) IsRunning() bool {
	return s.running.Load()
}

// ConnectionCount returns the current number of active connections.
func (s *Server) ConnectionCount() int64 {
	return s.connCount.Load()
}

// Address returns the address the server is listening on.
func (s *Server) Address() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.config.Address()
}

// buildCapabilities returns the capabilities to advertise.
func (s *Server) buildCapabilities() imap.CapSet {
	caps := make(imap.CapSet)
	caps[imap.CapIMAP4rev1] = struct{}{}
	caps[imap.CapIMAP4rev2] = struct{}{}
	caps[imap.CapLiteralPlus] = struct{}{}
	caps[imap.CapIdle] = struct{}{}
	caps[imap.CapNamespace] = struct{}{}
	caps[imap.CapUIDPlus] = struct{}{}
	caps[imap.CapMove] = struct{}{}

	// Add STARTTLS capability if TLS config is available but not using implicit TLS
	if s.config.TLS.StartTLS && !s.config.TLS.Enabled {
		caps[imap.CapStartTLS] = struct{}{}

		// RFC 3501: Advertise LOGINDISABLED when STARTTLS is available
		// but connection is not yet encrypted (unless InsecureAuth is enabled)
		if !s.config.InsecureAuth {
			caps[imap.CapLoginDisabled] = struct{}{}
		}
	}

	// Add authentication capabilities
	// Note: AUTH=PLAIN and AUTH=LOGIN are advertised to indicate supported SASL mechanisms
	// The go-imap/v2 library handles the actual SASL negotiation
	caps[imap.Cap("AUTH=PLAIN")] = struct{}{}
	caps[imap.Cap("AUTH=LOGIN")] = struct{}{}

	return caps
}

// SupportedAuthMechanisms returns the list of supported authentication mechanisms.
func (s *Server) SupportedAuthMechanisms() []string {
	return SupportedAuthMechanisms()
}

// newSession creates a new IMAP session for an incoming connection.
func (s *Server) newSession(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
	s.connCount.Add(1)
	metrics.IMAPActiveConnections.Inc()

	remoteAddr := conn.NetConn().RemoteAddr().String()
	s.logger.Info().
		Str("remote_addr", remoteAddr).
		Int64("total_connections", s.connCount.Load()).
		Bool("backend_available", s.backend != nil).
		Msg("New IMAP connection")

	sessionID := uuid.New().String()
	session := &Session{
		server:     s,
		conn:       conn,
		logger:     s.logger.With().Str("remote_addr", remoteAddr).Str("session_id", sessionID).Logger(),
		remoteAddr: remoteAddr,
		createdAt:  time.Now(),
		sessionID:  sessionID,
		state:      SessionStateNotAuthenticated,
	}

	greeting := &imapserver.GreetingData{
		PreAuth: false,
	}

	return session, greeting, nil
}

// onSessionClose is called when a session is closed.
func (s *Server) onSessionClose(remoteAddr string) {
	s.connCount.Add(-1)
	metrics.IMAPActiveConnections.Dec()
	s.logger.Info().
		Str("remote_addr", remoteAddr).
		Int64("remaining_connections", s.connCount.Load()).
		Msg("IMAP connection closed")
}

// serverLogger adapts zerolog to the imapserver.Logger interface.
type serverLogger struct {
	logger zerolog.Logger
}

// Printf implements the imapserver.Logger interface.
func (l *serverLogger) Printf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}
