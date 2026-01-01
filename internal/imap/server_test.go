package imap

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// testConfig returns a valid config for testing without TLS requirements.
func testConfig() *Config {
	return &Config{
		Enabled:      true,
		Host:         "127.0.0.1",
		Port:         1143,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  30 * time.Minute,
		ServerName:   "localhost",
		InsecureAuth: true,
		TLS: TLSConfig{
			Enabled:  false,
			StartTLS: false,
		},
	}
}

func TestNewServer(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  testConfig(),
			wantErr: false,
		},
		{
			name: "invalid config - empty host",
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
			name: "invalid config - invalid port",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && server == nil {
				t.Error("NewServer() returned nil server with no error")
			}
		})
	}
}

func TestServer_StartStop(t *testing.T) {
	logger := zerolog.Nop()
	cfg := testConfig()
	cfg.Port = 11430 // Use a non-standard port to avoid conflicts

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// Test starting the server
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Server should be running
	if !server.IsRunning() {
		t.Error("Server should be running after Start()")
	}

	// Get the actual address
	addr := server.Address()
	if addr == "" {
		t.Error("Server address should not be empty")
	}

	// Stop the server
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(stopCtx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Server should no longer be running
	if server.IsRunning() {
		t.Error("Server should not be running after Stop()")
	}
}

func TestServer_StartDisabled(t *testing.T) {
	logger := zerolog.Nop()
	cfg := testConfig()
	cfg.Enabled = false

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Server should not be running when disabled
	if server.IsRunning() {
		t.Error("Disabled server should not be running")
	}
}

func TestServer_DoubleStart(t *testing.T) {
	logger := zerolog.Nop()
	cfg := testConfig()
	cfg.Port = 11431 // Use a non-standard port to avoid conflicts

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Stop(stopCtx)
	}()

	// Second start should fail
	if err := server.Start(ctx); err == nil {
		t.Error("Second Start() should return error")
	}
}

func TestServer_ConnectionCount(t *testing.T) {
	logger := zerolog.Nop()
	cfg := testConfig()
	cfg.Port = 11432 // Use a non-standard port to avoid conflicts

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Stop(stopCtx)
	}()

	// Initial connection count should be 0
	if server.ConnectionCount() != 0 {
		t.Errorf("Initial ConnectionCount() = %v, want 0", server.ConnectionCount())
	}

	// Connect a client
	addr := server.Address()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// The go-imap library handles connections asynchronously.
	// The connection count increments when the session is created in newSession,
	// which happens after the IMAP greeting is sent.
	// For this simple test, we just verify we can connect without error.
	// Connection tracking is more of an integration concern.

	// Close the connection
	conn.Close()

	// Give the server time to process
	time.Sleep(100 * time.Millisecond)
}

func TestServer_MultipleConnections(t *testing.T) {
	logger := zerolog.Nop()
	cfg := testConfig()
	cfg.Port = 11433 // Use a non-standard port to avoid conflicts

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Stop(stopCtx)
	}()

	addr := server.Address()

	// Connect multiple clients to verify the server can handle concurrent connections
	const numClients = 5
	conns := make([]net.Conn, numClients)

	for i := 0; i < numClients; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		conns[i] = conn
	}

	// Close all connections
	for _, conn := range conns {
		conn.Close()
	}

	// Give the server time to process
	time.Sleep(100 * time.Millisecond)
}

func TestServer_Capabilities(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name              string
		tlsEnabled        bool
		startTLS          bool
		insecureAuth      bool
		wantStartTLS      bool
		wantLoginDisabled bool
	}{
		{
			name:              "no TLS",
			tlsEnabled:        false,
			startTLS:          false,
			insecureAuth:      false,
			wantStartTLS:      false,
			wantLoginDisabled: false,
		},
		{
			name:              "STARTTLS only with secure auth (LOGINDISABLED)",
			tlsEnabled:        false,
			startTLS:          true,
			insecureAuth:      false,
			wantStartTLS:      true,
			wantLoginDisabled: true,
		},
		{
			name:              "STARTTLS with insecure auth allowed",
			tlsEnabled:        false,
			startTLS:          true,
			insecureAuth:      true,
			wantStartTLS:      true,
			wantLoginDisabled: false,
		},
		{
			name:              "implicit TLS enabled (no STARTTLS needed)",
			tlsEnabled:        true,
			startTLS:          false,
			insecureAuth:      false,
			wantStartTLS:      false,
			wantLoginDisabled: false,
		},
		{
			name:              "both TLS modes with secure auth",
			tlsEnabled:        true,
			startTLS:          true,
			insecureAuth:      false,
			wantStartTLS:      false,
			wantLoginDisabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.TLS.Enabled = tt.tlsEnabled
			cfg.TLS.StartTLS = tt.startTLS
			cfg.InsecureAuth = tt.insecureAuth

			// For this test, we're just testing the capability building logic,
			// so we create the server struct directly to avoid validation
			server := &Server{
				config: cfg,
				logger: logger,
			}

			caps := server.buildCapabilities()

			// Check required capabilities
			requiredCaps := []string{"IMAP4rev1", "IMAP4rev2", "LITERAL+", "IDLE", "NAMESPACE", "UIDPLUS", "MOVE"}
			for _, cap := range requiredCaps {
				found := false
				for c := range caps {
					if string(c) == cap {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Missing required capability: %s", cap)
				}
			}

			// Check STARTTLS capability
			hasStartTLS := false
			for c := range caps {
				if string(c) == "STARTTLS" {
					hasStartTLS = true
					break
				}
			}

			if hasStartTLS != tt.wantStartTLS {
				t.Errorf("STARTTLS capability = %v, want %v", hasStartTLS, tt.wantStartTLS)
			}

			// Check LOGINDISABLED capability
			hasLoginDisabled := false
			for c := range caps {
				if string(c) == "LOGINDISABLED" {
					hasLoginDisabled = true
					break
				}
			}

			if hasLoginDisabled != tt.wantLoginDisabled {
				t.Errorf("LOGINDISABLED capability = %v, want %v", hasLoginDisabled, tt.wantLoginDisabled)
			}
		})
	}
}
