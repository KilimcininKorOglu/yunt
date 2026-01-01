package smtp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"yunt/internal/config"
)

// createTestCertificate creates a self-signed certificate for testing.
func createTestCertificate(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()

	// Generate private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Yunt Test"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	// Write certificate file
	certFile = filepath.Join(dir, "cert.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("failed to encode certificate: %v", err)
	}
	certOut.Close()

	// Write key file
	keyFile = filepath.Join(dir, "key.pem")
	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		t.Fatalf("failed to encode private key: %v", err)
	}
	keyOut.Close()

	return certFile, keyFile
}

func TestTLSState(t *testing.T) {
	tests := []struct {
		name     string
		state    TLSState
		wantStr  string
		isSecure bool
	}{
		{
			name:     "none",
			state:    TLSStateNone,
			wantStr:  "none",
			isSecure: false,
		},
		{
			name:     "starttls",
			state:    TLSStateStartTLS,
			wantStr:  "starttls",
			isSecure: true,
		},
		{
			name:     "implicit",
			state:    TLSStateImplicit,
			wantStr:  "implicit",
			isSecure: true,
		},
		{
			name:     "unknown",
			state:    TLSState(99),
			wantStr:  "unknown",
			isSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.wantStr {
				t.Errorf("TLSState.String() = %v, want %v", got, tt.wantStr)
			}
			if got := tt.state.IsSecure(); got != tt.isSecure {
				t.Errorf("TLSState.IsSecure() = %v, want %v", got, tt.isSecure)
			}
		})
	}
}

func TestConnectionSecurity(t *testing.T) {
	cs := NewConnectionSecurity()

	// Test initial state
	if cs.TLSState() != TLSStateNone {
		t.Errorf("initial TLSState = %v, want TLSStateNone", cs.TLSState())
	}
	if cs.IsSecure() {
		t.Error("initial IsSecure should be false")
	}

	// Test SetTLSState
	cs.SetTLSState(TLSStateStartTLS)
	if cs.TLSState() != TLSStateStartTLS {
		t.Errorf("after SetTLSState: TLSState = %v, want TLSStateStartTLS", cs.TLSState())
	}
	if !cs.IsSecure() {
		t.Error("after SetTLSState: IsSecure should be true")
	}

	// Test UpdateFromTLSState
	cs2 := NewConnectionSecurity()
	tlsState := tls.ConnectionState{
		Version:           tls.VersionTLS13,
		CipherSuite:       tls.TLS_AES_128_GCM_SHA256,
		ServerName:        "test.example.com",
		HandshakeComplete: true,
	}
	cs2.UpdateFromTLSState(tlsState, TLSStateImplicit)

	if cs2.TLSState() != TLSStateImplicit {
		t.Errorf("after UpdateFromTLSState: TLSState = %v, want TLSStateImplicit", cs2.TLSState())
	}
	if !cs2.HandshakeComplete() {
		t.Error("HandshakeComplete should be true")
	}
	if cs2.TLSVersion() != "TLS 1.3" {
		t.Errorf("TLSVersion = %v, want TLS 1.3", cs2.TLSVersion())
	}
	if cs2.ServerName() != "test.example.com" {
		t.Errorf("ServerName = %v, want test.example.com", cs2.ServerName())
	}

	// Test LogFields
	fields := cs2.LogFields()
	if fields["tlsState"] != "implicit" {
		t.Errorf("LogFields[tlsState] = %v, want implicit", fields["tlsState"])
	}
	if fields["secure"] != true {
		t.Errorf("LogFields[secure] = %v, want true", fields["secure"])
	}
}

func TestTLSVersionString(t *testing.T) {
	tests := []struct {
		version uint16
		want    string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
		{0x0000, "unknown (0x0000)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tlsVersionString(tt.version); got != tt.want {
				t.Errorf("tlsVersionString(%d) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestTLSManager(t *testing.T) {
	logger := zerolog.New(io.Discard)

	t.Run("empty config", func(t *testing.T) {
		tm := NewTLSManager(logger)

		err := tm.LoadFromConfig(config.TLSConfig{
			Enabled:  false,
			StartTLS: false,
		})
		if err != nil {
			t.Errorf("LoadFromConfig with empty config should not error: %v", err)
		}
		if tm.IsEnabled() {
			t.Error("IsEnabled should be false with empty config")
		}
		if tm.IsStartTLSEnabled() {
			t.Error("IsStartTLSEnabled should be false with empty config")
		}
		if tm.TLSConfig() != nil {
			t.Error("TLSConfig should be nil with empty config")
		}
	})

	t.Run("no cert files configured", func(t *testing.T) {
		tm := NewTLSManager(logger)

		err := tm.LoadFromConfig(config.TLSConfig{
			Enabled:  true,
			StartTLS: true,
			CertFile: "",
			KeyFile:  "",
		})
		if err != nil {
			t.Errorf("LoadFromConfig without cert files should not error: %v", err)
		}
		// Without cert files, TLS should be disabled
		if tm.IsEnabled() {
			t.Error("IsEnabled should be false without cert files")
		}
		if tm.IsStartTLSEnabled() {
			t.Error("IsStartTLSEnabled should be false without cert files")
		}
	})

	t.Run("missing cert file", func(t *testing.T) {
		tm := NewTLSManager(logger)

		err := tm.LoadFromConfig(config.TLSConfig{
			Enabled:  true,
			CertFile: "/nonexistent/cert.pem",
			KeyFile:  "/nonexistent/key.pem",
		})
		if err == nil {
			t.Error("LoadFromConfig should error with missing cert files")
		}
	})

	t.Run("valid certificates", func(t *testing.T) {
		// Create temp directory for test certificates
		tmpDir := t.TempDir()
		certFile, keyFile := createTestCertificate(t, tmpDir)

		tm := NewTLSManager(logger)

		err := tm.LoadFromConfig(config.TLSConfig{
			Enabled:  true,
			StartTLS: true,
			CertFile: certFile,
			KeyFile:  keyFile,
		})
		if err != nil {
			t.Fatalf("LoadFromConfig should not error with valid certs: %v", err)
		}

		if !tm.IsEnabled() {
			t.Error("IsEnabled should be true")
		}
		if !tm.IsStartTLSEnabled() {
			t.Error("IsStartTLSEnabled should be true")
		}
		if tm.TLSConfig() == nil {
			t.Error("TLSConfig should not be nil")
		}
		if tm.CertFile() != certFile {
			t.Errorf("CertFile = %v, want %v", tm.CertFile(), certFile)
		}
		if tm.KeyFile() != keyFile {
			t.Errorf("KeyFile = %v, want %v", tm.KeyFile(), keyFile)
		}
	})

	t.Run("reload certificates", func(t *testing.T) {
		tmpDir := t.TempDir()
		certFile, keyFile := createTestCertificate(t, tmpDir)

		tm := NewTLSManager(logger)

		err := tm.LoadFromConfig(config.TLSConfig{
			Enabled:  true,
			CertFile: certFile,
			KeyFile:  keyFile,
		})
		if err != nil {
			t.Fatalf("initial load failed: %v", err)
		}

		// Reload should succeed
		err = tm.ReloadCertificates()
		if err != nil {
			t.Errorf("ReloadCertificates should succeed: %v", err)
		}
	})

	t.Run("reload without config", func(t *testing.T) {
		tm := NewTLSManager(logger)

		err := tm.ReloadCertificates()
		if err == nil {
			t.Error("ReloadCertificates should fail without prior config")
		}
	})
}

func TestTLSManagerCertFileValidation(t *testing.T) {
	logger := zerolog.New(io.Discard)

	t.Run("cert file only", func(t *testing.T) {
		tm := NewTLSManager(logger)

		// Create a temp file for cert
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "cert.pem")
		if err := os.WriteFile(certFile, []byte("dummy"), 0600); err != nil {
			t.Fatal(err)
		}

		err := tm.LoadFromConfig(config.TLSConfig{
			Enabled:  true,
			CertFile: certFile,
			KeyFile:  "",
		})
		if err == nil {
			t.Error("should error when key file is missing")
		}
	})

	t.Run("key file only", func(t *testing.T) {
		tm := NewTLSManager(logger)

		// Create a temp file for key
		tmpDir := t.TempDir()
		keyFile := filepath.Join(tmpDir, "key.pem")
		if err := os.WriteFile(keyFile, []byte("dummy"), 0600); err != nil {
			t.Fatal(err)
		}

		err := tm.LoadFromConfig(config.TLSConfig{
			Enabled:  true,
			CertFile: "",
			KeyFile:  keyFile,
		})
		// With only keyFile, certFile is empty so both are empty from validation perspective
		// Actually if only KeyFile is provided, CertFile check comes first
		// When CertFile is empty and KeyFile is set, it errors
		// But the code path checks: if tlsCfg.CertFile == "" && tlsCfg.KeyFile == "" - no error
		// Then calls validateCertConfig which checks if either is missing
		// Actually: cert="" key="something" -> validateCertConfig is called because at least one is set
		// Then validateCertConfig returns error "TLS certificate file is required when key file is set"
		if err == nil {
			t.Error("should error when cert file is missing but key file is provided")
		}
	})
}

func TestStatsWithTLS(t *testing.T) {
	stats := NewStats()

	// Initial TLS stats
	tlsConns, startTLSUpgrades := stats.GetTLSStats()
	if tlsConns != 0 {
		t.Errorf("initial tlsConnectionsTotal = %d, want 0", tlsConns)
	}
	if startTLSUpgrades != 0 {
		t.Errorf("initial startTLSUpgradesTotal = %d, want 0", startTLSUpgrades)
	}

	// Increment TLS connections
	stats.TLSConnectionOpened()
	stats.TLSConnectionOpened()
	tlsConns, _ = stats.GetTLSStats()
	if tlsConns != 2 {
		t.Errorf("after TLSConnectionOpened: tlsConnectionsTotal = %d, want 2", tlsConns)
	}

	// Increment STARTTLS upgrades
	stats.StartTLSUpgraded()
	_, startTLSUpgrades = stats.GetTLSStats()
	if startTLSUpgrades != 1 {
		t.Errorf("after StartTLSUpgraded: startTLSUpgradesTotal = %d, want 1", startTLSUpgrades)
	}
}

func TestServerWithSTARTTLS(t *testing.T) {
	// Create temp directory for test certificates
	tmpDir := t.TempDir()
	certFile, keyFile := createTestCertificate(t, tmpDir)

	logger := zerolog.New(io.Discard)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Load TLS config
	tlsConfig, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("failed to load TLS key pair: %v", err)
	}

	cfg := &Config{
		Host:            "127.0.0.1",
		Port:            port,
		Domain:          "starttls.example.com",
		MaxMessageSize:  10 * 1024 * 1024,
		MaxRecipients:   100,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		GracefulTimeout: 5 * time.Second,
		EnableStartTLS:  true,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsConfig},
			MinVersion:   tls.VersionTLS12,
		},
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	// Connect and send EHLO to verify STARTTLS is advertised
	conn, err := net.DialTimeout("tcp", server.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Read greeting
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read greeting: %v", err)
	}
	greeting := string(buf[:n])
	if !strings.Contains(greeting, "220") {
		t.Fatalf("expected 220 greeting, got: %s", greeting)
	}

	// Send EHLO
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte("EHLO test.client.com\r\n"))
	if err != nil {
		t.Fatalf("failed to send EHLO: %v", err)
	}

	// Read EHLO response (multi-line)
	response := readEHLOResponse(t, conn)

	// Verify STARTTLS is advertised
	if !strings.Contains(response, "STARTTLS") {
		t.Errorf("STARTTLS not advertised in EHLO response: %s", response)
	}

	// Verify 250 response
	if !strings.Contains(response, "250") {
		t.Errorf("expected 250 response, got: %s", response)
	}

	// Send QUIT
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	conn.Write([]byte("QUIT\r\n"))
}

// readEHLOResponse reads a full SMTP EHLO response from the connection
func readEHLOResponse(t *testing.T, conn net.Conn) string {
	t.Helper()
	var response strings.Builder
	buf := make([]byte, 4096)

	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if response.Len() > 0 {
				return response.String()
			}
			t.Fatalf("failed to read response: %v", err)
		}
		response.Write(buf[:n])

		// Check if this is the end of the response
		resp := response.String()
		lines := strings.Split(resp, "\r\n")
		if len(lines) > 0 {
			// Find the last non-empty line
			lastLine := ""
			for i := len(lines) - 1; i >= 0; i-- {
				if lines[i] != "" {
					lastLine = lines[i]
					break
				}
			}
			// If the last line has a space after the code, it's the final line
			if len(lastLine) >= 4 && lastLine[3] == ' ' {
				return resp
			}
		}
	}
}

func TestServerWithoutTLS(t *testing.T) {
	logger := zerolog.New(io.Discard)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Config without TLS
	cfg := &Config{
		Host:            "127.0.0.1",
		Port:            port,
		Domain:          "notls.example.com",
		MaxMessageSize:  10 * 1024 * 1024,
		MaxRecipients:   100,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		GracefulTimeout: 5 * time.Second,
		EnableStartTLS:  false,
		TLSConfig:       nil,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	// Connect and send EHLO
	conn, err := net.DialTimeout("tcp", server.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Read greeting
	buf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read greeting: %v", err)
	}
	greeting := string(buf[:n])
	if !strings.Contains(greeting, "220") {
		t.Fatalf("expected 220 greeting, got: %s", greeting)
	}

	// Send EHLO
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte("EHLO test.client.com\r\n"))
	if err != nil {
		t.Fatalf("failed to send EHLO: %v", err)
	}

	// Read EHLO response
	response := readEHLOResponse(t, conn)

	// Verify STARTTLS is NOT advertised when TLS is disabled
	if strings.Contains(response, "STARTTLS") {
		t.Errorf("STARTTLS should not be advertised when TLS is disabled: %s", response)
	}

	// Send QUIT
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	conn.Write([]byte("QUIT\r\n"))
}
