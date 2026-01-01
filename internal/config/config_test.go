package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg == nil {
		t.Fatal("Default() returned nil")
	}

	// Test server defaults
	if cfg.Server.Name != DefaultServerName {
		t.Errorf("expected Server.Name=%s, got %s", DefaultServerName, cfg.Server.Name)
	}
	if cfg.Server.Domain != DefaultServerDomain {
		t.Errorf("expected Server.Domain=%s, got %s", DefaultServerDomain, cfg.Server.Domain)
	}
	if cfg.Server.GracefulTimeout != DefaultServerGracefulTimeout {
		t.Errorf("expected Server.GracefulTimeout=%v, got %v", DefaultServerGracefulTimeout, cfg.Server.GracefulTimeout)
	}

	// Test SMTP defaults
	if cfg.SMTP.Enabled != DefaultSMTPEnabled {
		t.Errorf("expected SMTP.Enabled=%v, got %v", DefaultSMTPEnabled, cfg.SMTP.Enabled)
	}
	if cfg.SMTP.Port != DefaultSMTPPort {
		t.Errorf("expected SMTP.Port=%d, got %d", DefaultSMTPPort, cfg.SMTP.Port)
	}
	if cfg.SMTP.MaxMessageSize != DefaultSMTPMaxMessageSize {
		t.Errorf("expected SMTP.MaxMessageSize=%d, got %d", DefaultSMTPMaxMessageSize, cfg.SMTP.MaxMessageSize)
	}

	// Test IMAP defaults
	if cfg.IMAP.Enabled != DefaultIMAPEnabled {
		t.Errorf("expected IMAP.Enabled=%v, got %v", DefaultIMAPEnabled, cfg.IMAP.Enabled)
	}
	if cfg.IMAP.Port != DefaultIMAPPort {
		t.Errorf("expected IMAP.Port=%d, got %d", DefaultIMAPPort, cfg.IMAP.Port)
	}

	// Test API defaults
	if cfg.API.Enabled != DefaultAPIEnabled {
		t.Errorf("expected API.Enabled=%v, got %v", DefaultAPIEnabled, cfg.API.Enabled)
	}
	if cfg.API.Port != DefaultAPIPort {
		t.Errorf("expected API.Port=%d, got %d", DefaultAPIPort, cfg.API.Port)
	}

	// Test Database defaults
	if cfg.Database.Driver != DefaultDatabaseDriver {
		t.Errorf("expected Database.Driver=%s, got %s", DefaultDatabaseDriver, cfg.Database.Driver)
	}
	if cfg.Database.DSN != DefaultDatabaseDSN {
		t.Errorf("expected Database.DSN=%s, got %s", DefaultDatabaseDSN, cfg.Database.DSN)
	}

	// Test Auth defaults
	if cfg.Auth.BCryptCost != DefaultAuthBCryptCost {
		t.Errorf("expected Auth.BCryptCost=%d, got %d", DefaultAuthBCryptCost, cfg.Auth.BCryptCost)
	}

	// Test Logging defaults
	if cfg.Logging.Level != DefaultLoggingLevel {
		t.Errorf("expected Logging.Level=%s, got %s", DefaultLoggingLevel, cfg.Logging.Level)
	}
	if cfg.Logging.Format != DefaultLoggingFormat {
		t.Errorf("expected Logging.Format=%s, got %s", DefaultLoggingFormat, cfg.Logging.Format)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  name: testserver
  domain: test.com
  gracefulTimeout: 60s

smtp:
  enabled: true
  port: 2525
  host: 127.0.0.1
  maxMessageSize: 5242880
  maxRecipients: 50

imap:
  enabled: false
  port: 2143

api:
  enabled: true
  port: 9025

database:
  driver: postgres
  host: localhost
  port: 5432
  name: testdb
  username: testuser
  password: testpass

logging:
  level: debug
  format: json
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	// Verify loaded values
	if cfg.Server.Name != "testserver" {
		t.Errorf("expected Server.Name=testserver, got %s", cfg.Server.Name)
	}
	if cfg.Server.Domain != "test.com" {
		t.Errorf("expected Server.Domain=test.com, got %s", cfg.Server.Domain)
	}
	if cfg.Server.GracefulTimeout != 60*time.Second {
		t.Errorf("expected Server.GracefulTimeout=60s, got %v", cfg.Server.GracefulTimeout)
	}

	if cfg.SMTP.Port != 2525 {
		t.Errorf("expected SMTP.Port=2525, got %d", cfg.SMTP.Port)
	}
	if cfg.SMTP.MaxMessageSize != 5242880 {
		t.Errorf("expected SMTP.MaxMessageSize=5242880, got %d", cfg.SMTP.MaxMessageSize)
	}

	if cfg.IMAP.Enabled != false {
		t.Errorf("expected IMAP.Enabled=false, got %v", cfg.IMAP.Enabled)
	}

	if cfg.Database.Driver != "postgres" {
		t.Errorf("expected Database.Driver=postgres, got %s", cfg.Database.Driver)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("expected Logging.Level=debug, got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("expected Logging.Format=json, got %s", cfg.Logging.Format)
	}
}

func TestLoadWithEnvOverrides(t *testing.T) {
	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
server:
  name: fileserver
  domain: file.com

smtp:
  port: 1025
  maxRecipients: 100

database:
  driver: sqlite
  dsn: test.db
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Set environment variables
	os.Setenv("YUNT_SERVER_NAME", "envserver")
	os.Setenv("YUNT_SMTP_PORT", "3025")
	os.Setenv("YUNT_SMTP_MAX_RECIPIENTS", "200")
	os.Setenv("YUNT_LOGGING_LEVEL", "warn")
	defer func() {
		os.Unsetenv("YUNT_SERVER_NAME")
		os.Unsetenv("YUNT_SMTP_PORT")
		os.Unsetenv("YUNT_SMTP_MAX_RECIPIENTS")
		os.Unsetenv("YUNT_LOGGING_LEVEL")
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Env should override file
	if cfg.Server.Name != "envserver" {
		t.Errorf("expected Server.Name=envserver (from env), got %s", cfg.Server.Name)
	}
	if cfg.SMTP.Port != 3025 {
		t.Errorf("expected SMTP.Port=3025 (from env), got %d", cfg.SMTP.Port)
	}
	if cfg.SMTP.MaxRecipients != 200 {
		t.Errorf("expected SMTP.MaxRecipients=200 (from env), got %d", cfg.SMTP.MaxRecipients)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("expected Logging.Level=warn (from env), got %s", cfg.Logging.Level)
	}

	// File value should remain when not overridden
	if cfg.Server.Domain != "file.com" {
		t.Errorf("expected Server.Domain=file.com (from file), got %s", cfg.Server.Domain)
	}
}

func TestLoadWithoutFile(t *testing.T) {
	// Clear any environment variables that might interfere
	envVars := []string{
		"YUNT_SERVER_NAME",
		"YUNT_SMTP_PORT",
		"YUNT_DATABASE_DRIVER",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() with empty path error: %v", err)
	}

	// Should use defaults
	if cfg.Server.Name != DefaultServerName {
		t.Errorf("expected Server.Name=%s (default), got %s", DefaultServerName, cfg.Server.Name)
	}
	if cfg.SMTP.Port != DefaultSMTPPort {
		t.Errorf("expected SMTP.Port=%d (default), got %d", DefaultSMTPPort, cfg.SMTP.Port)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `
server:
  name: test
  domain: [invalid yaml structure
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestEnvDurationParsing(t *testing.T) {
	os.Setenv("YUNT_SERVER_GRACEFUL_TIMEOUT", "2m30s")
	defer os.Unsetenv("YUNT_SERVER_GRACEFUL_TIMEOUT")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	expected := 2*time.Minute + 30*time.Second
	if cfg.Server.GracefulTimeout != expected {
		t.Errorf("expected Server.GracefulTimeout=%v, got %v", expected, cfg.Server.GracefulTimeout)
	}
}

func TestEnvBoolParsing(t *testing.T) {
	os.Setenv("YUNT_SMTP_ENABLED", "false")
	os.Setenv("YUNT_API_ENABLE_SWAGGER", "true")
	defer func() {
		os.Unsetenv("YUNT_SMTP_ENABLED")
		os.Unsetenv("YUNT_API_ENABLE_SWAGGER")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.SMTP.Enabled != false {
		t.Errorf("expected SMTP.Enabled=false, got %v", cfg.SMTP.Enabled)
	}
	if cfg.API.EnableSwagger != true {
		t.Errorf("expected API.EnableSwagger=true, got %v", cfg.API.EnableSwagger)
	}
}

func TestEnvSliceParsing(t *testing.T) {
	os.Setenv("YUNT_API_CORS_ALLOWED_ORIGINS", "http://localhost:3000, http://example.com")
	defer os.Unsetenv("YUNT_API_CORS_ALLOWED_ORIGINS")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	expected := []string{"http://localhost:3000", "http://example.com"}
	if len(cfg.API.CORSAllowedOrigins) != len(expected) {
		t.Fatalf("expected %d CORS origins, got %d", len(expected), len(cfg.API.CORSAllowedOrigins))
	}
	for i, v := range expected {
		if cfg.API.CORSAllowedOrigins[i] != v {
			t.Errorf("expected CORS origin[%d]=%s, got %s", i, v, cfg.API.CORSAllowedOrigins[i])
		}
	}
}

func TestMustLoadPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustLoad to panic on error, but it didn't")
		}
	}()

	MustLoad("/nonexistent/path/config.yaml")
}

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "saved.yaml")

	cfg := Default()
	cfg.Server.Name = "saved-server"
	cfg.SMTP.Port = 5025

	err := SaveToFile(cfg, savePath)
	if err != nil {
		t.Fatalf("SaveToFile() error: %v", err)
	}

	// Load it back
	loaded, err := LoadFromFile(savePath)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	if loaded.Server.Name != "saved-server" {
		t.Errorf("expected Server.Name=saved-server, got %s", loaded.Server.Name)
	}
	if loaded.SMTP.Port != 5025 {
		t.Errorf("expected SMTP.Port=5025, got %d", loaded.SMTP.Port)
	}
}
