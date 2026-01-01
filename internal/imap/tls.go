// Package imap provides an IMAP server implementation for the Yunt mail server.
package imap

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// TLSLoader provides TLS configuration loading with enhanced logging.
type TLSLoader struct {
	logger zerolog.Logger
}

// NewTLSLoader creates a new TLS loader with the given logger.
func NewTLSLoader(logger zerolog.Logger) *TLSLoader {
	return &TLSLoader{
		logger: logger.With().Str("component", "imap-tls").Logger(),
	}
}

// LoadTLSConfig loads the TLS configuration from the given config.
// It provides detailed error logging for certificate issues.
// Returns nil if TLS is not configured.
func (l *TLSLoader) LoadTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}

	if !cfg.Enabled && !cfg.StartTLS {
		l.logger.Debug().Msg("TLS is disabled, no certificate loading required")
		return nil, nil
	}

	// Log TLS mode
	if cfg.Enabled {
		l.logger.Info().Msg("Loading TLS configuration for implicit TLS mode")
	} else if cfg.StartTLS {
		l.logger.Info().Msg("Loading TLS configuration for STARTTLS mode")
	}

	// Validate certificate paths
	if cfg.CertFile == "" {
		l.logger.Error().Msg("TLS certificate file path is empty")
		return nil, fmt.Errorf("imap: TLS certificate file path is required")
	}

	if cfg.KeyFile == "" {
		l.logger.Error().Msg("TLS private key file path is empty")
		return nil, fmt.Errorf("imap: TLS private key file path is required")
	}

	// Resolve absolute paths for logging
	certPath, _ := filepath.Abs(cfg.CertFile)
	keyPath, _ := filepath.Abs(cfg.KeyFile)

	// Check if certificate file exists
	if _, err := os.Stat(cfg.CertFile); os.IsNotExist(err) {
		l.logger.Error().
			Str("path", certPath).
			Msg("TLS certificate file does not exist")
		return nil, fmt.Errorf("imap: TLS certificate file not found: %s", certPath)
	}

	// Check if key file exists
	if _, err := os.Stat(cfg.KeyFile); os.IsNotExist(err) {
		l.logger.Error().
			Str("path", keyPath).
			Msg("TLS private key file does not exist")
		return nil, fmt.Errorf("imap: TLS private key file not found: %s", keyPath)
	}

	// Load the certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		l.logger.Error().
			Err(err).
			Str("cert_file", certPath).
			Str("key_file", keyPath).
			Msg("Failed to load TLS certificate/key pair")
		return nil, fmt.Errorf("imap: failed to load TLS certificate: %w", err)
	}

	// Parse the certificate for additional validation and logging
	if len(cert.Certificate) > 0 {
		x509Cert, parseErr := x509.ParseCertificate(cert.Certificate[0])
		if parseErr != nil {
			l.logger.Warn().
				Err(parseErr).
				Msg("Could not parse certificate for validation")
		} else {
			l.logCertificateInfo(x509Cert)
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	l.logger.Info().
		Str("cert_file", certPath).
		Str("min_tls_version", "TLS 1.2").
		Msg("TLS configuration loaded successfully")

	return tlsConfig, nil
}

// logCertificateInfo logs detailed information about the certificate.
func (l *TLSLoader) logCertificateInfo(cert *x509.Certificate) {
	l.logger.Info().
		Str("subject", cert.Subject.CommonName).
		Time("not_before", cert.NotBefore).
		Time("not_after", cert.NotAfter).
		Strs("dns_names", cert.DNSNames).
		Msg("Certificate details")

	// Check for expiration warning
	if cert.NotAfter.Before(cert.NotBefore) {
		l.logger.Error().
			Time("not_before", cert.NotBefore).
			Time("not_after", cert.NotAfter).
			Msg("Certificate has invalid validity period")
	}
}

// ValidateCertificate performs comprehensive validation of a certificate file.
// This can be used for pre-flight checks before starting the server.
func (l *TLSLoader) ValidateCertificate(certFile, keyFile string) error {
	// Check file existence
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return fmt.Errorf("certificate file not found: %s", certFile)
	}

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return fmt.Errorf("private key file not found: %s", keyFile)
	}

	// Try to load the certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	// Parse and validate
	if len(cert.Certificate) == 0 {
		return fmt.Errorf("certificate file contains no certificates")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check validity period
	if x509Cert.NotAfter.Before(x509Cert.NotBefore) {
		return fmt.Errorf("certificate has invalid validity period")
	}

	return nil
}

// IsTLSRequired returns true if the connection should require TLS for sensitive operations.
// This is used to determine if LOGIN should be disabled before STARTTLS.
func IsTLSRequired(tlsCfg *TLSConfig, insecureAuth bool) bool {
	// If insecure auth is explicitly allowed, TLS is not required
	if insecureAuth {
		return false
	}

	// TLS is required if STARTTLS is enabled but implicit TLS is not
	// (meaning we start plaintext and expect upgrade)
	if tlsCfg.StartTLS && !tlsCfg.Enabled {
		return true
	}

	return false
}
