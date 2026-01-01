package smtp

import (
	"testing"
	"time"

	"yunt/internal/config"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &config.Config{
				Server: config.ServerConfig{
					Domain:          "localhost",
					GracefulTimeout: 30 * time.Second,
				},
				SMTP: config.SMTPConfig{
					Enabled:        true,
					Host:           "0.0.0.0",
					Port:           1025,
					MaxMessageSize: 10 * 1024 * 1024,
					MaxRecipients:  100,
					ReadTimeout:    60 * time.Second,
					WriteTimeout:   60 * time.Second,
					AuthRequired:   false,
					TLS: config.TLSConfig{
						Enabled:  false,
						StartTLS: true,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg == nil {
				t.Error("NewConfig() returned nil config without error")
			}
		})
	}
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("NewDefaultConfig() returned nil")
	}

	if !cfg.Enabled {
		t.Error("default config should have Enabled = true")
	}

	if cfg.Host != config.DefaultSMTPHost {
		t.Errorf("default Host = %v, want %v", cfg.Host, config.DefaultSMTPHost)
	}

	if cfg.Port != config.DefaultSMTPPort {
		t.Errorf("default Port = %v, want %v", cfg.Port, config.DefaultSMTPPort)
	}

	if cfg.Domain != config.DefaultServerDomain {
		t.Errorf("default Domain = %v, want %v", cfg.Domain, config.DefaultServerDomain)
	}

	if cfg.MaxMessageSize != config.DefaultSMTPMaxMessageSize {
		t.Errorf("default MaxMessageSize = %v, want %v", cfg.MaxMessageSize, config.DefaultSMTPMaxMessageSize)
	}

	if cfg.MaxRecipients != config.DefaultSMTPMaxRecipients {
		t.Errorf("default MaxRecipients = %v, want %v", cfg.MaxRecipients, config.DefaultSMTPMaxRecipients)
	}
}

func TestConfigAddr(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{
			name: "localhost:1025",
			host: "localhost",
			port: 1025,
			want: "localhost:1025",
		},
		{
			name: "0.0.0.0:25",
			host: "0.0.0.0",
			port: 25,
			want: "0.0.0.0:25",
		},
		{
			name: "127.0.0.1:587",
			host: "127.0.0.1",
			port: 587,
			want: "127.0.0.1:587",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Host: tt.host,
				Port: tt.port,
			}
			if got := cfg.Addr(); got != tt.want {
				t.Errorf("Config.Addr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Port:            1025,
				Domain:          "localhost",
				MaxMessageSize:  10 * 1024 * 1024,
				MaxRecipients:   100,
				ReadTimeout:     60 * time.Second,
				WriteTimeout:    60 * time.Second,
				GracefulTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Port:          0,
				Domain:        "localhost",
				MaxRecipients: 100,
			},
			wantErr: true,
			errMsg:  "invalid port number",
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Port:          65536,
				Domain:        "localhost",
				MaxRecipients: 100,
			},
			wantErr: true,
			errMsg:  "invalid port number",
		},
		{
			name: "empty domain",
			config: &Config{
				Port:          1025,
				Domain:        "",
				MaxRecipients: 100,
			},
			wantErr: true,
			errMsg:  "domain cannot be empty",
		},
		{
			name: "negative max message size",
			config: &Config{
				Port:           1025,
				Domain:         "localhost",
				MaxMessageSize: -1,
				MaxRecipients:  100,
			},
			wantErr: true,
			errMsg:  "max message size cannot be negative",
		},
		{
			name: "zero max recipients",
			config: &Config{
				Port:          1025,
				Domain:        "localhost",
				MaxRecipients: 0,
			},
			wantErr: true,
			errMsg:  "max recipients must be at least 1",
		},
		{
			name: "negative read timeout",
			config: &Config{
				Port:          1025,
				Domain:        "localhost",
				MaxRecipients: 100,
				ReadTimeout:   -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "read timeout cannot be negative",
		},
		{
			name: "negative write timeout",
			config: &Config{
				Port:          1025,
				Domain:        "localhost",
				MaxRecipients: 100,
				WriteTimeout:  -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "write timeout cannot be negative",
		},
		{
			name: "negative graceful timeout",
			config: &Config{
				Port:            1025,
				Domain:          "localhost",
				MaxRecipients:   100,
				GracefulTimeout: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "graceful timeout cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
