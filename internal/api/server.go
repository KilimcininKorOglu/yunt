package api

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/config"
)

// Server represents the API server.
type Server struct {
	router   *Router
	config   config.APIConfig
	logger   *config.Logger
	server   *http.Server
	shutdown chan struct{}
}

// ServerOption is a function that configures the Server.
type ServerOption func(*Server)

// WithLogger sets the logger for the server.
func WithLogger(logger *config.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// New creates a new API server instance.
func New(cfg config.APIConfig, opts ...ServerOption) *Server {
	s := &Server{
		config:   cfg,
		shutdown: make(chan struct{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create default logger if not provided
	if s.logger == nil {
		s.logger = config.NewDefaultLogger()
	}

	// Create router
	routerCfg := RouterConfig{
		Logger:        s.logger,
		CORSOrigins:   cfg.CORSAllowedOrigins,
		EnableSwagger: cfg.EnableSwagger,
		EnableMetrics: cfg.EnableMetrics,
	}
	s.router = NewRouter(routerCfg)

	return s
}

// Echo returns the underlying Echo instance for additional configuration.
func (s *Server) Echo() *echo.Echo {
	return s.router.Echo
}

// Router returns the router for registering handlers.
func (s *Server) Router() *Router {
	return s.router
}

// Start starts the API server and blocks until it is stopped.
// It listens for shutdown signals (SIGINT, SIGTERM) and performs graceful shutdown.
func (s *Server) Start() error {
	return s.StartWithContext(context.Background())
}

// StartWithContext starts the API server with the given context.
// The server will shut down when the context is canceled.
func (s *Server) StartWithContext(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Configure the HTTP server
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Configure TLS if enabled
	if s.config.TLS.Enabled {
		if s.config.TLS.CertFile == "" || s.config.TLS.KeyFile == "" {
			return fmt.Errorf("TLS certificate and key files are required when TLS is enabled")
		}

		cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}

		s.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	// Channel to receive server errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		s.logger.Info().
			Str("address", addr).
			Bool("tls", s.config.TLS.Enabled).
			Msg("Starting API server")

		var err error
		if s.config.TLS.Enabled {
			err = s.server.ListenAndServeTLS("", "")
		} else {
			err = s.server.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		s.logger.Info().Msg("Context canceled, initiating shutdown")
		return s.gracefulShutdown(context.Background())
	case <-s.shutdown:
		s.logger.Info().Msg("Shutdown signal received")
		return s.gracefulShutdown(context.Background())
	}
}

// StartWithSignals starts the server and listens for OS signals for graceful shutdown.
func (s *Server) StartWithSignals(gracefulTimeout time.Duration) error {
	// Create a context that cancels on OS signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start signal handler
	go func() {
		sig := <-sigChan
		s.logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Configure the HTTP server
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Configure TLS if enabled
	if s.config.TLS.Enabled {
		if s.config.TLS.CertFile == "" || s.config.TLS.KeyFile == "" {
			return fmt.Errorf("TLS certificate and key files are required when TLS is enabled")
		}

		cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}

		s.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	// Channel to receive server errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		s.logger.Info().
			Str("address", addr).
			Bool("tls", s.config.TLS.Enabled).
			Msg("Starting API server")

		var err error
		if s.config.TLS.Enabled {
			err = s.server.ListenAndServeTLS("", "")
		} else {
			err = s.server.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		s.logger.Info().Msg("Initiating graceful shutdown")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulTimeout)
		defer shutdownCancel()
		return s.gracefulShutdown(shutdownCtx)
	}
}

// Shutdown initiates a graceful shutdown of the server.
func (s *Server) Shutdown(ctx context.Context) error {
	close(s.shutdown)
	return s.gracefulShutdown(ctx)
}

// gracefulShutdown performs a graceful shutdown of the HTTP server.
func (s *Server) gracefulShutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.logger.Info().Msg("Shutting down API server")

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error().Err(err).Msg("Error during server shutdown")
		return fmt.Errorf("server shutdown error: %w", err)
	}

	s.logger.Info().Msg("API server stopped gracefully")
	return nil
}

// Address returns the address the server is configured to listen on.
func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// IsRunning returns true if the server is currently running.
func (s *Server) IsRunning() bool {
	return s.server != nil
}
