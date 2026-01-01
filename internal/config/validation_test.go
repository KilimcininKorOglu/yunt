package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateDefaultConfig(t *testing.T) {
	cfg := Default()

	err := Validate(cfg)
	if err != nil {
		t.Errorf("expected default config to be valid, got error: %v", err)
	}
}

func TestValidateServerConfig(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Config)
		wantErr   bool
		errField  string
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name:     "empty server name",
			modify:   func(c *Config) { c.Server.Name = "" },
			wantErr:  true,
			errField: "server.name",
		},
		{
			name:     "empty server domain",
			modify:   func(c *Config) { c.Server.Domain = "" },
			wantErr:  true,
			errField: "server.domain",
		},
		{
			name:     "negative graceful timeout",
			modify:   func(c *Config) { c.Server.GracefulTimeout = -1 * time.Second },
			wantErr:  true,
			errField: "server.gracefulTimeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSMTPConfig(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name:    "disabled SMTP skips validation",
			modify:  func(c *Config) { c.SMTP.Enabled = false; c.SMTP.Port = -1 },
			wantErr: false,
		},
		{
			name:     "invalid port too low",
			modify:   func(c *Config) { c.SMTP.Port = 0 },
			wantErr:  true,
			errField: "smtp.port",
		},
		{
			name:     "invalid port too high",
			modify:   func(c *Config) { c.SMTP.Port = 70000 },
			wantErr:  true,
			errField: "smtp.port",
		},
		{
			name:     "negative max message size",
			modify:   func(c *Config) { c.SMTP.MaxMessageSize = -1 },
			wantErr:  true,
			errField: "smtp.maxMessageSize",
		},
		{
			name:     "zero max recipients",
			modify:   func(c *Config) { c.SMTP.MaxRecipients = 0 },
			wantErr:  true,
			errField: "smtp.maxRecipients",
		},
		{
			name:     "relay enabled without host",
			modify:   func(c *Config) { c.SMTP.AllowRelay = true; c.SMTP.RelayHost = "" },
			wantErr:  true,
			errField: "smtp.relayHost",
		},
		{
			name:    "relay enabled with host",
			modify:  func(c *Config) { c.SMTP.AllowRelay = true; c.SMTP.RelayHost = "smtp.example.com"; c.SMTP.RelayPort = 587 },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateIMAPConfig(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name:    "disabled IMAP skips validation",
			modify:  func(c *Config) { c.IMAP.Enabled = false; c.IMAP.Port = -1 },
			wantErr: false,
		},
		{
			name:     "invalid port",
			modify:   func(c *Config) { c.IMAP.Port = 0 },
			wantErr:  true,
			errField: "imap.port",
		},
		{
			name:     "negative idle timeout",
			modify:   func(c *Config) { c.IMAP.IdleTimeout = -1 * time.Second },
			wantErr:  true,
			errField: "imap.idleTimeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateAPIConfig(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name:    "disabled API skips validation",
			modify:  func(c *Config) { c.API.Enabled = false; c.API.Port = -1 },
			wantErr: false,
		},
		{
			name:     "invalid port",
			modify:   func(c *Config) { c.API.Port = 0 },
			wantErr:  true,
			errField: "api.port",
		},
		{
			name:     "negative rate limit",
			modify:   func(c *Config) { c.API.RateLimit = -1 },
			wantErr:  true,
			errField: "api.rateLimit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateDatabaseConfig(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name:     "invalid driver",
			modify:   func(c *Config) { c.Database.Driver = "invalid" },
			wantErr:  true,
			errField: "database.driver",
		},
		{
			name:    "valid postgres driver",
			modify:  func(c *Config) { c.Database.Driver = "postgres"; c.Database.Host = "localhost"; c.Database.Name = "test" },
			wantErr: false,
		},
		{
			name:    "valid mysql driver",
			modify:  func(c *Config) { c.Database.Driver = "mysql"; c.Database.Host = "localhost"; c.Database.Name = "test" },
			wantErr: false,
		},
		{
			name:    "valid mongodb driver",
			modify:  func(c *Config) { c.Database.Driver = "mongodb"; c.Database.Host = "localhost"; c.Database.Name = "test" },
			wantErr: false,
		},
		{
			name:     "postgres without host",
			modify:   func(c *Config) { c.Database.Driver = "postgres"; c.Database.DSN = ""; c.Database.Host = "" },
			wantErr:  true,
			errField: "database.host",
		},
		{
			name:     "negative max open conns",
			modify:   func(c *Config) { c.Database.MaxOpenConns = -1 },
			wantErr:  true,
			errField: "database.maxOpenConns",
		},
		{
			name:     "max idle exceeds max open",
			modify:   func(c *Config) { c.Database.MaxOpenConns = 5; c.Database.MaxIdleConns = 10 },
			wantErr:  true,
			errField: "database.maxIdleConns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name:     "zero jwt expiration",
			modify:   func(c *Config) { c.Auth.JWTExpiration = 0 },
			wantErr:  true,
			errField: "auth.jwtExpiration",
		},
		{
			name:     "refresh less than jwt",
			modify:   func(c *Config) { c.Auth.JWTExpiration = 24 * time.Hour; c.Auth.RefreshExpiration = 1 * time.Hour },
			wantErr:  true,
			errField: "auth.refreshExpiration",
		},
		{
			name:     "bcrypt cost too low",
			modify:   func(c *Config) { c.Auth.BCryptCost = 3 },
			wantErr:  true,
			errField: "auth.bcryptCost",
		},
		{
			name:     "bcrypt cost too high",
			modify:   func(c *Config) { c.Auth.BCryptCost = 32 },
			wantErr:  true,
			errField: "auth.bcryptCost",
		},
		{
			name:     "negative lockout duration",
			modify:   func(c *Config) { c.Auth.LockoutDuration = -1 * time.Minute },
			wantErr:  true,
			errField: "auth.lockoutDuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateLoggingConfig(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name:     "invalid log level",
			modify:   func(c *Config) { c.Logging.Level = "invalid" },
			wantErr:  true,
			errField: "logging.level",
		},
		{
			name:    "valid log levels",
			modify:  func(c *Config) { c.Logging.Level = "debug" },
			wantErr: false,
		},
		{
			name:     "invalid format",
			modify:   func(c *Config) { c.Logging.Format = "xml" },
			wantErr:  true,
			errField: "logging.format",
		},
		{
			name:     "file output without path",
			modify:   func(c *Config) { c.Logging.Output = "file"; c.Logging.FilePath = "" },
			wantErr:  true,
			errField: "logging.filePath",
		},
		{
			name:    "file output with path",
			modify:  func(c *Config) { c.Logging.Output = "file"; c.Logging.FilePath = "/var/log/yunt.log" },
			wantErr: false,
		},
		{
			name:     "negative max size",
			modify:   func(c *Config) { c.Logging.MaxSize = -1 },
			wantErr:  true,
			errField: "logging.maxSize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateAdminConfig(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name:    "disabled create on startup skips validation",
			modify:  func(c *Config) { c.Admin.CreateOnStartup = false; c.Admin.Username = "" },
			wantErr: false,
		},
		{
			name:     "empty username",
			modify:   func(c *Config) { c.Admin.Username = "" },
			wantErr:  true,
			errField: "admin.username",
		},
		{
			name:     "short username",
			modify:   func(c *Config) { c.Admin.Username = "ab" },
			wantErr:  true,
			errField: "admin.username",
		},
		{
			name:     "empty email",
			modify:   func(c *Config) { c.Admin.Email = "" },
			wantErr:  true,
			errField: "admin.email",
		},
		{
			name:     "invalid email",
			modify:   func(c *Config) { c.Admin.Email = "notanemail" },
			wantErr:  true,
			errField: "admin.email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateStorageConfig(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name:     "invalid storage type",
			modify:   func(c *Config) { c.Storage.Type = "s3" },
			wantErr:  true,
			errField: "storage.type",
		},
		{
			name:    "valid database type",
			modify:  func(c *Config) { c.Storage.Type = "database" },
			wantErr: false,
		},
		{
			name:     "filesystem without path",
			modify:   func(c *Config) { c.Storage.Type = "filesystem"; c.Storage.Path = "" },
			wantErr:  true,
			errField: "storage.path",
		},
		{
			name:    "filesystem with path",
			modify:  func(c *Config) { c.Storage.Type = "filesystem"; c.Storage.Path = "/data/mail" },
			wantErr: false,
		},
		{
			name:     "negative max mailbox size",
			modify:   func(c *Config) { c.Storage.MaxMailboxSize = -1 },
			wantErr:  true,
			errField: "storage.maxMailboxSize",
		},
		{
			name:     "negative retention days",
			modify:   func(c *Config) { c.Storage.RetentionDays = -1 },
			wantErr:  true,
			errField: "storage.retentionDays",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateTLSConfig(t *testing.T) {
	// Create temporary cert and key files for testing
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	if err := os.WriteFile(certFile, []byte("fake cert"), 0644); err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, []byte("fake key"), 0644); err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}

	tests := []struct {
		name     string
		modify   func(*Config)
		wantErr  bool
		errField string
	}{
		{
			name: "TLS enabled with missing cert",
			modify: func(c *Config) {
				c.SMTP.TLS.Enabled = true
				c.SMTP.TLS.CertFile = "/nonexistent/cert.pem"
				c.SMTP.TLS.KeyFile = keyFile
			},
			wantErr:  true,
			errField: "smtp.tls.certFile",
		},
		{
			name: "TLS enabled with missing key",
			modify: func(c *Config) {
				c.SMTP.TLS.Enabled = true
				c.SMTP.TLS.CertFile = certFile
				c.SMTP.TLS.KeyFile = "/nonexistent/key.pem"
			},
			wantErr:  true,
			errField: "smtp.tls.keyFile",
		},
		{
			name: "TLS enabled with valid files",
			modify: func(c *Config) {
				c.SMTP.TLS.Enabled = true
				c.SMTP.TLS.CertFile = certFile
				c.SMTP.TLS.KeyFile = keyFile
			},
			wantErr: false,
		},
		{
			name: "TLS disabled skips validation",
			modify: func(c *Config) {
				c.SMTP.TLS.Enabled = false
				c.SMTP.TLS.StartTLS = false
				c.SMTP.TLS.CertFile = "/nonexistent/cert.pem"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := Validate(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("expected error for field %s, got: %v", tt.errField, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	cfg := Default()
	cfg.Server.Name = ""
	cfg.Server.Domain = ""
	cfg.SMTP.Port = 0
	cfg.Logging.Level = "invalid"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should contain multiple errors
	errStr := err.Error()
	if !strings.Contains(errStr, "server.name") {
		t.Error("expected error for server.name")
	}
	if !strings.Contains(errStr, "server.domain") {
		t.Error("expected error for server.domain")
	}
	if !strings.Contains(errStr, "smtp.port") {
		t.Error("expected error for smtp.port")
	}
	if !strings.Contains(errStr, "logging.level") {
		t.Error("expected error for logging.level")
	}
}

func TestValidationErrorType(t *testing.T) {
	cfg := Default()
	cfg.Server.Name = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should be ValidationErrors type
	_, ok := err.(ValidationErrors)
	if !ok {
		t.Errorf("expected ValidationErrors type, got %T", err)
	}
}
