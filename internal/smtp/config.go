// Package smtp provides the SMTP server implementation for Yunt mail server.
// It uses the emersion/go-smtp library for SMTP protocol handling.
package smtp

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"yunt/internal/config"
)

// Config holds the SMTP server configuration.
// It wraps the SMTPConfig from the config package with additional
// runtime settings and helper methods.
type Config struct {
	// Enabled determines if the SMTP server should start.
	Enabled bool

	// Host is the address to bind the SMTP server to.
	Host string

	// Port is the port number for the SMTP server.
	Port int

	// Domain is the server hostname used in SMTP HELO/EHLO.
	Domain string

	// LocalDomains lists all domains considered local (not relayed).
	LocalDomains []string

	// MaxMessageSize is the maximum message size in bytes.
	MaxMessageSize int64

	// MaxRecipients is the maximum number of recipients per message.
	MaxRecipients int

	// ReadTimeout is the read timeout for SMTP connections.
	ReadTimeout time.Duration

	// WriteTimeout is the write timeout for SMTP connections.
	WriteTimeout time.Duration

	// AuthRequired determines if authentication is required for SMTP.
	AuthRequired bool

	// AllowInsecureAuth allows PLAIN auth over non-TLS connections.
	AllowInsecureAuth bool

	// TLSConfig holds the TLS configuration for the server.
	TLSConfig *tls.Config

	// EnableStartTLS enables STARTTLS support.
	EnableStartTLS bool

	// GracefulTimeout is the duration to wait for graceful shutdown.
	GracefulTimeout time.Duration

	// RateLimitEnabled determines if rate limiting is enabled.
	RateLimitEnabled bool

	// RateLimitConfig holds the rate limiting configuration.
	RateLimitConfig *RateLimitConfig
}

// NewConfig creates a new SMTP Config from the application configuration.
func NewConfig(cfg *config.Config) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	smtpCfg := &Config{
		Enabled:           cfg.SMTP.Enabled,
		Host:              cfg.SMTP.Host,
		Port:              cfg.SMTP.Port,
		Domain:            cfg.Server.Domain,
		LocalDomains:      cfg.Server.LocalDomains,
		MaxMessageSize:    cfg.SMTP.MaxMessageSize,
		MaxRecipients:     cfg.SMTP.MaxRecipients,
		ReadTimeout:       cfg.SMTP.ReadTimeout,
		WriteTimeout:      cfg.SMTP.WriteTimeout,
		AuthRequired:      cfg.SMTP.AuthRequired,
		AllowInsecureAuth: !cfg.SMTP.AuthRequired, // Allow insecure auth if auth is not required
		EnableStartTLS:    cfg.SMTP.TLS.StartTLS,
		GracefulTimeout:   cfg.Server.GracefulTimeout,
	}

	// Set up TLS configuration if enabled
	if cfg.SMTP.TLS.Enabled || cfg.SMTP.TLS.StartTLS {
		tlsConfig, err := createTLSConfig(cfg.SMTP.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		smtpCfg.TLSConfig = tlsConfig
	}

	return smtpCfg, nil
}

// NewDefaultConfig creates a Config with default values.
func NewDefaultConfig() *Config {
	return &Config{
		Enabled:           config.DefaultSMTPEnabled,
		Host:              config.DefaultSMTPHost,
		Port:              config.DefaultSMTPPort,
		Domain:            config.DefaultServerDomain,
		MaxMessageSize:    config.DefaultSMTPMaxMessageSize,
		MaxRecipients:     config.DefaultSMTPMaxRecipients,
		ReadTimeout:       config.DefaultSMTPReadTimeout,
		WriteTimeout:      config.DefaultSMTPWriteTimeout,
		AuthRequired:      config.DefaultSMTPAuthRequired,
		AllowInsecureAuth: true,
		EnableStartTLS:    true,
		GracefulTimeout:   config.DefaultServerGracefulTimeout,
	}
}

// Addr returns the server address in host:port format.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsLocalDomain checks if the given domain is local.
func (c *Config) IsLocalDomain(domain string) bool {
	for _, d := range c.LocalDomains {
		if strings.EqualFold(d, domain) {
			return true
		}
	}
	return strings.EqualFold(c.Domain, domain)
}

// Validate validates the configuration and returns an error if invalid.
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", c.Port)
	}

	if c.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if c.MaxMessageSize < 0 {
		return fmt.Errorf("max message size cannot be negative")
	}

	if c.MaxRecipients < 1 {
		return fmt.Errorf("max recipients must be at least 1")
	}

	if c.ReadTimeout < 0 {
		return fmt.Errorf("read timeout cannot be negative")
	}

	if c.WriteTimeout < 0 {
		return fmt.Errorf("write timeout cannot be negative")
	}

	if c.GracefulTimeout < 0 {
		return fmt.Errorf("graceful timeout cannot be negative")
	}

	// Validate TLS configuration if auth is required
	if c.AuthRequired && c.TLSConfig == nil && !c.AllowInsecureAuth {
		return fmt.Errorf("TLS must be enabled when authentication is required")
	}

	return nil
}

// createTLSConfig creates a tls.Config from the TLS configuration.
func createTLSConfig(tlsCfg config.TLSConfig) (*tls.Config, error) {
	if tlsCfg.CertFile == "" && tlsCfg.KeyFile == "" {
		// No certificate configured, return nil (STARTTLS won't be available)
		return nil, nil
	}

	if tlsCfg.CertFile == "" {
		return nil, fmt.Errorf("TLS certificate file is required when key file is set")
	}

	if tlsCfg.KeyFile == "" {
		return nil, fmt.Errorf("TLS key file is required when certificate file is set")
	}

	cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
