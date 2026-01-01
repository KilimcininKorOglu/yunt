// Package smtp provides TLS support for the SMTP server.
// This file contains TLS configuration management, certificate loading,
// and security state tracking for STARTTLS support.
package smtp

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog"

	"yunt/internal/config"
)

// TLSState represents the current TLS state of a connection.
type TLSState int

const (
	// TLSStateNone indicates no TLS encryption (plain connection).
	TLSStateNone TLSState = iota
	// TLSStateStartTLS indicates TLS established via STARTTLS upgrade.
	TLSStateStartTLS
	// TLSStateImplicit indicates TLS established from connection start (implicit TLS).
	TLSStateImplicit
)

// String returns a string representation of the TLS state.
func (s TLSState) String() string {
	switch s {
	case TLSStateNone:
		return "none"
	case TLSStateStartTLS:
		return "starttls"
	case TLSStateImplicit:
		return "implicit"
	default:
		return "unknown"
	}
}

// IsSecure returns true if the connection has TLS encryption.
func (s TLSState) IsSecure() bool {
	return s == TLSStateStartTLS || s == TLSStateImplicit
}

// ConnectionSecurity tracks the security state of an SMTP connection.
type ConnectionSecurity struct {
	mu            sync.RWMutex
	tlsState      TLSState
	tlsVersion    uint16
	cipherSuite   uint16
	serverName    string
	peerCertDN    string
	handshakeDone bool
}

// NewConnectionSecurity creates a new ConnectionSecurity instance.
func NewConnectionSecurity() *ConnectionSecurity {
	return &ConnectionSecurity{
		tlsState: TLSStateNone,
	}
}

// SetTLSState sets the TLS state of the connection.
func (cs *ConnectionSecurity) SetTLSState(state TLSState) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.tlsState = state
}

// TLSState returns the current TLS state.
func (cs *ConnectionSecurity) TLSState() TLSState {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.tlsState
}

// IsSecure returns true if the connection is encrypted.
func (cs *ConnectionSecurity) IsSecure() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.tlsState.IsSecure()
}

// UpdateFromTLSState updates the connection security from a TLS connection state.
func (cs *ConnectionSecurity) UpdateFromTLSState(state tls.ConnectionState, tlsType TLSState) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.tlsState = tlsType
	cs.tlsVersion = state.Version
	cs.cipherSuite = state.CipherSuite
	cs.serverName = state.ServerName
	cs.handshakeDone = state.HandshakeComplete

	// Extract peer certificate DN if available
	if len(state.PeerCertificates) > 0 {
		cs.peerCertDN = state.PeerCertificates[0].Subject.String()
	}
}

// TLSVersion returns the negotiated TLS version as a string.
func (cs *ConnectionSecurity) TLSVersion() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return tlsVersionString(cs.tlsVersion)
}

// CipherSuite returns the negotiated cipher suite name.
func (cs *ConnectionSecurity) CipherSuite() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return tls.CipherSuiteName(cs.cipherSuite)
}

// ServerName returns the server name from TLS handshake (SNI).
func (cs *ConnectionSecurity) ServerName() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.serverName
}

// PeerCertDN returns the peer certificate distinguished name if available.
func (cs *ConnectionSecurity) PeerCertDN() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.peerCertDN
}

// HandshakeComplete returns true if TLS handshake completed successfully.
func (cs *ConnectionSecurity) HandshakeComplete() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.handshakeDone
}

// LogFields returns a map of fields for structured logging.
func (cs *ConnectionSecurity) LogFields() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	fields := map[string]interface{}{
		"tlsState": cs.tlsState.String(),
		"secure":   cs.tlsState.IsSecure(),
	}

	if cs.tlsState.IsSecure() {
		fields["tlsVersion"] = tlsVersionString(cs.tlsVersion)
		fields["cipherSuite"] = tls.CipherSuiteName(cs.cipherSuite)
		if cs.serverName != "" {
			fields["serverName"] = cs.serverName
		}
		if cs.peerCertDN != "" {
			fields["peerCertDN"] = cs.peerCertDN
		}
	}

	return fields
}

// tlsVersionString converts a TLS version constant to a human-readable string.
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", version)
	}
}

// TLSManager manages TLS configuration and certificate loading for the SMTP server.
type TLSManager struct {
	mu        sync.RWMutex
	tlsConfig *tls.Config
	certFile  string
	keyFile   string
	enabled   bool
	startTLS  bool
	logger    zerolog.Logger
}

// NewTLSManager creates a new TLSManager with the given configuration.
func NewTLSManager(logger zerolog.Logger) *TLSManager {
	return &TLSManager{
		logger: logger.With().Str("component", "tls").Logger(),
	}
}

// LoadFromConfig loads TLS configuration from the application config.
func (tm *TLSManager) LoadFromConfig(tlsCfg config.TLSConfig) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.enabled = tlsCfg.Enabled
	tm.startTLS = tlsCfg.StartTLS
	tm.certFile = tlsCfg.CertFile
	tm.keyFile = tlsCfg.KeyFile

	// If TLS is not enabled and STARTTLS is not enabled, no certificates needed
	if !tlsCfg.Enabled && !tlsCfg.StartTLS {
		tm.logger.Debug().Msg("TLS disabled")
		return nil
	}

	// If no certificate files configured, TLS won't be available
	if tlsCfg.CertFile == "" && tlsCfg.KeyFile == "" {
		tm.logger.Info().Msg("TLS certificates not configured, TLS will be unavailable")
		tm.enabled = false
		tm.startTLS = false
		return nil
	}

	// Validate certificate configuration
	if err := tm.validateCertConfig(tlsCfg); err != nil {
		tm.logger.Error().Err(err).Msg("TLS certificate configuration error")
		return err
	}

	// Load certificates
	tlsConfig, err := tm.loadCertificates(tlsCfg)
	if err != nil {
		tm.logger.Error().Err(err).
			Str("certFile", tlsCfg.CertFile).
			Str("keyFile", tlsCfg.KeyFile).
			Msg("failed to load TLS certificates")
		return err
	}

	tm.tlsConfig = tlsConfig

	tm.logger.Info().
		Bool("implicitTLS", tlsCfg.Enabled).
		Bool("startTLS", tlsCfg.StartTLS).
		Str("certFile", tlsCfg.CertFile).
		Str("keyFile", tlsCfg.KeyFile).
		Msg("TLS certificates loaded successfully")

	return nil
}

// validateCertConfig validates the TLS certificate configuration.
func (tm *TLSManager) validateCertConfig(tlsCfg config.TLSConfig) error {
	if tlsCfg.CertFile == "" {
		return fmt.Errorf("TLS certificate file is required when key file is set")
	}

	if tlsCfg.KeyFile == "" {
		return fmt.Errorf("TLS key file is required when certificate file is set")
	}

	// Check if certificate file exists
	if _, err := os.Stat(tlsCfg.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("TLS certificate file not found: %s", tlsCfg.CertFile)
	}

	// Check if key file exists
	if _, err := os.Stat(tlsCfg.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("TLS key file not found: %s", tlsCfg.KeyFile)
	}

	return nil
}

// loadCertificates loads TLS certificates from files.
func (tm *TLSManager) loadCertificates(tlsCfg config.TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	// Validate the certificate
	if err := tm.validateCertificate(cert); err != nil {
		tm.logger.Warn().Err(err).Msg("certificate validation warning")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		// Prefer server cipher suites for better security control
		PreferServerCipherSuites: true,
	}, nil
}

// validateCertificate performs basic validation on the loaded certificate.
func (tm *TLSManager) validateCertificate(cert tls.Certificate) error {
	if len(cert.Certificate) == 0 {
		return fmt.Errorf("certificate chain is empty")
	}

	// Parse the leaf certificate
	leafCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Log certificate details
	tm.logger.Info().
		Str("subject", leafCert.Subject.String()).
		Str("issuer", leafCert.Issuer.String()).
		Time("notBefore", leafCert.NotBefore).
		Time("notAfter", leafCert.NotAfter).
		Strs("dnsNames", leafCert.DNSNames).
		Msg("certificate details")

	return nil
}

// TLSConfig returns the current TLS configuration.
// Returns nil if TLS is not configured.
func (tm *TLSManager) TLSConfig() *tls.Config {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tlsConfig
}

// IsEnabled returns true if implicit TLS is enabled.
func (tm *TLSManager) IsEnabled() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.enabled && tm.tlsConfig != nil
}

// IsStartTLSEnabled returns true if STARTTLS is enabled and available.
func (tm *TLSManager) IsStartTLSEnabled() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.startTLS && tm.tlsConfig != nil
}

// CertFile returns the configured certificate file path.
func (tm *TLSManager) CertFile() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.certFile
}

// KeyFile returns the configured key file path.
func (tm *TLSManager) KeyFile() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.keyFile
}

// ReloadCertificates reloads TLS certificates from files.
// This can be used for certificate rotation without server restart.
func (tm *TLSManager) ReloadCertificates() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.certFile == "" || tm.keyFile == "" {
		return fmt.Errorf("certificate files not configured")
	}

	tlsCfg := config.TLSConfig{
		Enabled:  tm.enabled,
		StartTLS: tm.startTLS,
		CertFile: tm.certFile,
		KeyFile:  tm.keyFile,
	}

	tlsConfig, err := tm.loadCertificates(tlsCfg)
	if err != nil {
		tm.logger.Error().Err(err).Msg("failed to reload TLS certificates")
		return err
	}

	tm.tlsConfig = tlsConfig
	tm.logger.Info().Msg("TLS certificates reloaded successfully")

	return nil
}
