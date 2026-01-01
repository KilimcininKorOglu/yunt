package imap

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// generateTestCertificate generates a self-signed certificate for testing.
func generateTestCertificate(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()

	// Generate private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	// Self-sign the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Write certificate
	certFile = filepath.Join(dir, "test.crt")
	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("Failed to write cert: %v", err)
	}
	certOut.Close()

	// Write private key
	keyFile = filepath.Join(dir, "test.key")
	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		t.Fatalf("Failed to write key: %v", err)
	}
	keyOut.Close()

	return certFile, keyFile
}

func TestNewTLSLoader(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	if loader == nil {
		t.Fatal("NewTLSLoader returned nil")
	}
}

func TestTLSLoader_LoadTLSConfig_Disabled(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	// Test with nil config
	tlsCfg, err := loader.LoadTLSConfig(nil)
	if err != nil {
		t.Errorf("LoadTLSConfig(nil) error = %v, want nil", err)
	}
	if tlsCfg != nil {
		t.Error("LoadTLSConfig(nil) should return nil TLS config")
	}

	// Test with disabled TLS
	cfg := &TLSConfig{
		Enabled:  false,
		StartTLS: false,
	}
	tlsCfg, err = loader.LoadTLSConfig(cfg)
	if err != nil {
		t.Errorf("LoadTLSConfig(disabled) error = %v, want nil", err)
	}
	if tlsCfg != nil {
		t.Error("LoadTLSConfig(disabled) should return nil TLS config")
	}
}

func TestTLSLoader_LoadTLSConfig_Success(t *testing.T) {
	// Create temp directory for test certificates
	dir := t.TempDir()
	certFile, keyFile := generateTestCertificate(t, dir)

	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	tests := []struct {
		name string
		cfg  *TLSConfig
	}{
		{
			name: "implicit TLS enabled",
			cfg: &TLSConfig{
				Enabled:  true,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
		{
			name: "STARTTLS enabled",
			cfg: &TLSConfig{
				StartTLS: true,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
		{
			name: "both TLS modes enabled",
			cfg: &TLSConfig{
				Enabled:  true,
				StartTLS: true,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsCfg, err := loader.LoadTLSConfig(tt.cfg)
			if err != nil {
				t.Errorf("LoadTLSConfig() error = %v", err)
				return
			}
			if tlsCfg == nil {
				t.Error("LoadTLSConfig() returned nil, want *tls.Config")
				return
			}
			if len(tlsCfg.Certificates) == 0 {
				t.Error("LoadTLSConfig() returned config with no certificates")
			}
		})
	}
}

func TestTLSLoader_LoadTLSConfig_MissingCertFile(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "nonexistent.key")

	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	cfg := &TLSConfig{
		Enabled:  true,
		CertFile: "",
		KeyFile:  keyFile,
	}

	_, err := loader.LoadTLSConfig(cfg)
	if err == nil {
		t.Error("LoadTLSConfig() should return error for empty cert file")
	}
}

func TestTLSLoader_LoadTLSConfig_MissingKeyFile(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "nonexistent.crt")

	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	cfg := &TLSConfig{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  "",
	}

	_, err := loader.LoadTLSConfig(cfg)
	if err == nil {
		t.Error("LoadTLSConfig() should return error for empty key file")
	}
}

func TestTLSLoader_LoadTLSConfig_CertFileNotFound(t *testing.T) {
	dir := t.TempDir()
	_, keyFile := generateTestCertificate(t, dir)

	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	cfg := &TLSConfig{
		Enabled:  true,
		CertFile: filepath.Join(dir, "nonexistent.crt"),
		KeyFile:  keyFile,
	}

	_, err := loader.LoadTLSConfig(cfg)
	if err == nil {
		t.Error("LoadTLSConfig() should return error for nonexistent cert file")
	}
}

func TestTLSLoader_LoadTLSConfig_KeyFileNotFound(t *testing.T) {
	dir := t.TempDir()
	certFile, _ := generateTestCertificate(t, dir)

	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	cfg := &TLSConfig{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  filepath.Join(dir, "nonexistent.key"),
	}

	_, err := loader.LoadTLSConfig(cfg)
	if err == nil {
		t.Error("LoadTLSConfig() should return error for nonexistent key file")
	}
}

func TestTLSLoader_LoadTLSConfig_InvalidCertificate(t *testing.T) {
	dir := t.TempDir()

	// Create invalid cert file
	certFile := filepath.Join(dir, "invalid.crt")
	if err := os.WriteFile(certFile, []byte("invalid certificate"), 0644); err != nil {
		t.Fatalf("Failed to create invalid cert file: %v", err)
	}

	keyFile := filepath.Join(dir, "invalid.key")
	if err := os.WriteFile(keyFile, []byte("invalid key"), 0644); err != nil {
		t.Fatalf("Failed to create invalid key file: %v", err)
	}

	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	cfg := &TLSConfig{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  keyFile,
	}

	_, err := loader.LoadTLSConfig(cfg)
	if err == nil {
		t.Error("LoadTLSConfig() should return error for invalid certificate")
	}
}

func TestTLSLoader_ValidateCertificate(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateTestCertificate(t, dir)

	logger := zerolog.Nop()
	loader := NewTLSLoader(logger)

	// Valid certificate
	err := loader.ValidateCertificate(certFile, keyFile)
	if err != nil {
		t.Errorf("ValidateCertificate() error = %v, want nil", err)
	}

	// Nonexistent cert file
	err = loader.ValidateCertificate(filepath.Join(dir, "nonexistent.crt"), keyFile)
	if err == nil {
		t.Error("ValidateCertificate() should return error for nonexistent cert file")
	}

	// Nonexistent key file
	err = loader.ValidateCertificate(certFile, filepath.Join(dir, "nonexistent.key"))
	if err == nil {
		t.Error("ValidateCertificate() should return error for nonexistent key file")
	}
}

func TestIsTLSRequired(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *TLSConfig
		insecureAuth bool
		want         bool
	}{
		{
			name: "insecure auth allowed",
			cfg: &TLSConfig{
				StartTLS: true,
			},
			insecureAuth: true,
			want:         false,
		},
		{
			name: "STARTTLS without implicit TLS",
			cfg: &TLSConfig{
				Enabled:  false,
				StartTLS: true,
			},
			insecureAuth: false,
			want:         true,
		},
		{
			name: "implicit TLS enabled",
			cfg: &TLSConfig{
				Enabled:  true,
				StartTLS: false,
			},
			insecureAuth: false,
			want:         false,
		},
		{
			name: "both TLS modes enabled",
			cfg: &TLSConfig{
				Enabled:  true,
				StartTLS: true,
			},
			insecureAuth: false,
			want:         false,
		},
		{
			name: "no TLS configured",
			cfg: &TLSConfig{
				Enabled:  false,
				StartTLS: false,
			},
			insecureAuth: false,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTLSRequired(tt.cfg, tt.insecureAuth)
			if got != tt.want {
				t.Errorf("IsTLSRequired() = %v, want %v", got, tt.want)
			}
		})
	}
}
