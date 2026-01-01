// Package imap provides an IMAP server implementation for the Yunt mail server.
// It uses the emersion/go-imap/v2 library for IMAP protocol handling.
package imap

import (
	"crypto/tls"
	"fmt"
	"time"

	"yunt/internal/config"
)

// Config holds the IMAP server configuration.
type Config struct {
	// Enabled determines if the IMAP server should start.
	Enabled bool

	// Host is the address to bind the IMAP server to.
	Host string

	// Port is the port number for the IMAP server.
	Port int

	// TLS contains TLS configuration for IMAP.
	TLS TLSConfig

	// ReadTimeout is the read timeout for IMAP connections.
	ReadTimeout time.Duration

	// WriteTimeout is the write timeout for IMAP connections.
	WriteTimeout time.Duration

	// IdleTimeout is the timeout for IMAP IDLE connections.
	IdleTimeout time.Duration

	// ServerName is the server hostname for IMAP greeting.
	ServerName string

	// InsecureAuth allows clients to authenticate without TLS.
	// WARNING: This makes the server susceptible to man-in-the-middle attacks.
	InsecureAuth bool
}

// TLSConfig contains TLS configuration settings.
type TLSConfig struct {
	// Enabled determines if TLS is enabled (implicit TLS).
	Enabled bool

	// CertFile is the path to the TLS certificate file.
	CertFile string

	// KeyFile is the path to the TLS private key file.
	KeyFile string

	// StartTLS determines if STARTTLS is supported.
	StartTLS bool
}

// Address returns the full address string (host:port) for the IMAP server.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Host == "" {
		return fmt.Errorf("imap: host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("imap: port must be between 1 and 65535")
	}

	if c.ReadTimeout <= 0 {
		return fmt.Errorf("imap: read timeout must be positive")
	}

	if c.WriteTimeout <= 0 {
		return fmt.Errorf("imap: write timeout must be positive")
	}

	if c.IdleTimeout <= 0 {
		return fmt.Errorf("imap: idle timeout must be positive")
	}

	// Validate TLS configuration if enabled
	if c.TLS.Enabled || c.TLS.StartTLS {
		if c.TLS.CertFile == "" {
			return fmt.Errorf("imap: TLS certificate file is required when TLS is enabled")
		}
		if c.TLS.KeyFile == "" {
			return fmt.Errorf("imap: TLS key file is required when TLS is enabled")
		}
	}

	return nil
}

// LoadTLSConfig loads and returns the TLS configuration.
// Returns nil if TLS is not enabled.
func (c *Config) LoadTLSConfig() (*tls.Config, error) {
	if !c.TLS.Enabled && !c.TLS.StartTLS {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(c.TLS.CertFile, c.TLS.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("imap: failed to load TLS certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// NewConfigFromApp creates an IMAP Config from the application configuration.
func NewConfigFromApp(appCfg *config.Config) *Config {
	return &Config{
		Enabled:      appCfg.IMAP.Enabled,
		Host:         appCfg.IMAP.Host,
		Port:         appCfg.IMAP.Port,
		ReadTimeout:  appCfg.IMAP.ReadTimeout,
		WriteTimeout: appCfg.IMAP.WriteTimeout,
		IdleTimeout:  appCfg.IMAP.IdleTimeout,
		ServerName:   appCfg.Server.Name,
		InsecureAuth: false, // Default to secure by default
		TLS: TLSConfig{
			Enabled:  appCfg.IMAP.TLS.Enabled,
			CertFile: appCfg.IMAP.TLS.CertFile,
			KeyFile:  appCfg.IMAP.TLS.KeyFile,
			StartTLS: appCfg.IMAP.TLS.StartTLS,
		},
	}
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Enabled:      true,
		Host:         "0.0.0.0",
		Port:         1143,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  30 * time.Minute,
		ServerName:   "localhost",
		InsecureAuth: true, // Allow insecure auth for development
		TLS: TLSConfig{
			Enabled:  false,
			StartTLS: true,
		},
	}
}
