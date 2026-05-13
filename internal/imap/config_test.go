package imap

import (
	"testing"
	"time"

	"yunt/internal/config"
)

func TestConfig_Address(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "default localhost",
			host:     "localhost",
			port:     143,
			expected: "localhost:143",
		},
		{
			name:     "all interfaces",
			host:     "0.0.0.0",
			port:     1143,
			expected: "0.0.0.0:1143",
		},
		{
			name:     "specific IP",
			host:     "192.168.1.100",
			port:     993,
			expected: "192.168.1.100:993",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Host: tt.host,
				Port: tt.port,
			}
			if got := cfg.Address(); got != tt.expected {
				t.Errorf("Address() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         1143,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  30 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "disabled config always valid",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "empty host",
			config: &Config{
				Enabled:      true,
				Host:         "",
				Port:         1143,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  30 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid port zero",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         0,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  30 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         70000,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  30 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero read timeout",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         1143,
				ReadTimeout:  0,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  30 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero write timeout",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         1143,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 0,
				IdleTimeout:  30 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero idle timeout",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         1143,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  0,
			},
			wantErr: true,
		},
		{
			name: "TLS enabled without cert auto-disables",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         993,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  30 * time.Minute,
				TLS: TLSConfig{
					Enabled:  true,
					CertFile: "",
					KeyFile:  "server.key",
				},
			},
			wantErr: false,
		},
		{
			name: "TLS enabled without key auto-disables",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         993,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  30 * time.Minute,
				TLS: TLSConfig{
					Enabled:  true,
					CertFile: "server.crt",
					KeyFile:  "",
				},
			},
			wantErr: false,
		},
		{
			name: "STARTTLS without cert auto-disables",
			config: &Config{
				Enabled:      true,
				Host:         "0.0.0.0",
				Port:         1143,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 60 * time.Second,
				IdleTimeout:  30 * time.Minute,
				TLS: TLSConfig{
					Enabled:  false,
					StartTLS: true,
					CertFile: "",
					KeyFile:  "server.key",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if !cfg.Enabled {
		t.Error("DefaultConfig() should be enabled by default")
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("DefaultConfig() Host = %v, want 0.0.0.0", cfg.Host)
	}

	if cfg.Port != 1143 {
		t.Errorf("DefaultConfig() Port = %v, want 1143", cfg.Port)
	}

	if cfg.ReadTimeout != 60*time.Second {
		t.Errorf("DefaultConfig() ReadTimeout = %v, want 60s", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout != 60*time.Second {
		t.Errorf("DefaultConfig() WriteTimeout = %v, want 60s", cfg.WriteTimeout)
	}

	if cfg.IdleTimeout != 30*time.Minute {
		t.Errorf("DefaultConfig() IdleTimeout = %v, want 30m", cfg.IdleTimeout)
	}

	if cfg.TLS.Enabled {
		t.Error("DefaultConfig() TLS should be disabled by default")
	}

	if !cfg.TLS.StartTLS {
		t.Error("DefaultConfig() STARTTLS should be enabled by default")
	}
}

func TestNewConfigFromApp(t *testing.T) {
	appCfg := &config.Config{
		Server: config.ServerConfig{
			Name: "mail.example.com",
		},
		IMAP: config.IMAPConfig{
			Enabled:      true,
			Host:         "192.168.1.1",
			Port:         993,
			ReadTimeout:  120 * time.Second,
			WriteTimeout: 120 * time.Second,
			IdleTimeout:  60 * time.Minute,
			TLS: config.TLSConfig{
				Enabled:  true,
				CertFile: "/path/to/cert.pem",
				KeyFile:  "/path/to/key.pem",
				StartTLS: false,
			},
		},
	}

	cfg := NewConfigFromApp(appCfg)

	if cfg.Enabled != appCfg.IMAP.Enabled {
		t.Errorf("Enabled = %v, want %v", cfg.Enabled, appCfg.IMAP.Enabled)
	}

	if cfg.Host != appCfg.IMAP.Host {
		t.Errorf("Host = %v, want %v", cfg.Host, appCfg.IMAP.Host)
	}

	if cfg.Port != appCfg.IMAP.Port {
		t.Errorf("Port = %v, want %v", cfg.Port, appCfg.IMAP.Port)
	}

	if cfg.ReadTimeout != appCfg.IMAP.ReadTimeout {
		t.Errorf("ReadTimeout = %v, want %v", cfg.ReadTimeout, appCfg.IMAP.ReadTimeout)
	}

	if cfg.WriteTimeout != appCfg.IMAP.WriteTimeout {
		t.Errorf("WriteTimeout = %v, want %v", cfg.WriteTimeout, appCfg.IMAP.WriteTimeout)
	}

	if cfg.IdleTimeout != appCfg.IMAP.IdleTimeout {
		t.Errorf("IdleTimeout = %v, want %v", cfg.IdleTimeout, appCfg.IMAP.IdleTimeout)
	}

	if cfg.ServerName != appCfg.Server.Name {
		t.Errorf("ServerName = %v, want %v", cfg.ServerName, appCfg.Server.Name)
	}

	if cfg.TLS.Enabled != appCfg.IMAP.TLS.Enabled {
		t.Errorf("TLS.Enabled = %v, want %v", cfg.TLS.Enabled, appCfg.IMAP.TLS.Enabled)
	}

	if cfg.TLS.CertFile != appCfg.IMAP.TLS.CertFile {
		t.Errorf("TLS.CertFile = %v, want %v", cfg.TLS.CertFile, appCfg.IMAP.TLS.CertFile)
	}

	if cfg.TLS.KeyFile != appCfg.IMAP.TLS.KeyFile {
		t.Errorf("TLS.KeyFile = %v, want %v", cfg.TLS.KeyFile, appCfg.IMAP.TLS.KeyFile)
	}

	if cfg.TLS.StartTLS != appCfg.IMAP.TLS.StartTLS {
		t.Errorf("TLS.StartTLS = %v, want %v", cfg.TLS.StartTLS, appCfg.IMAP.TLS.StartTLS)
	}
}

func TestConfig_LoadTLSConfig_NoTLS(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		TLS: TLSConfig{
			Enabled:  false,
			StartTLS: false,
		},
	}

	tlsCfg, err := cfg.LoadTLSConfig()
	if err != nil {
		t.Errorf("LoadTLSConfig() error = %v, want nil", err)
	}
	if tlsCfg != nil {
		t.Error("LoadTLSConfig() should return nil when TLS is disabled")
	}
}
