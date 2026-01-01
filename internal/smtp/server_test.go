package smtp

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	logger := zerolog.New(io.Discard)

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
		},
		{
			name: "invalid config - empty domain",
			cfg: &Config{
				Port:          1025,
				Domain:        "",
				MaxRecipients: 100,
			},
			wantErr: true,
			errMsg:  "invalid config",
		},
		{
			name: "valid config",
			cfg: &Config{
				Host:           "127.0.0.1",
				Port:           1025,
				Domain:         "localhost",
				MaxMessageSize: 10 * 1024 * 1024,
				MaxRecipients:  100,
				ReadTimeout:    60 * time.Second,
				WriteTimeout:   60 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(tt.cfg, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("New() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
			if !tt.wantErr && s == nil {
				t.Error("New() returned nil server without error")
			}
		})
	}
}

func TestServerStartStop(t *testing.T) {
	logger := zerolog.New(io.Discard)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg := &Config{
		Host:            "127.0.0.1",
		Port:            port,
		Domain:          "localhost",
		MaxMessageSize:  10 * 1024 * 1024,
		MaxRecipients:   100,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		GracefulTimeout: 5 * time.Second,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test IsRunning before start
	if server.IsRunning() {
		t.Error("server should not be running before Start()")
	}

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Test IsRunning after start
	if !server.IsRunning() {
		t.Error("server should be running after Start()")
	}

	// Verify server is listening
	addr := server.Addr()
	if addr == "" {
		t.Error("server address should not be empty when running")
	}

	// Test connection
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Errorf("failed to connect to server: %v", err)
	} else {
		// Read greeting
		buf := make([]byte, 256)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("failed to read greeting: %v", err)
		} else {
			greeting := string(buf[:n])
			if !strings.HasPrefix(greeting, "220 ") {
				t.Errorf("unexpected greeting: %s", greeting)
			}
		}
		conn.Close()
	}

	// Test double start
	if err := server.Start(); err == nil {
		t.Error("double Start() should return error")
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		t.Errorf("failed to stop server: %v", err)
	}

	// Test IsRunning after stop
	if server.IsRunning() {
		t.Error("server should not be running after Stop()")
	}

	// Test double stop (should not error)
	if err := server.Stop(ctx); err != nil {
		t.Errorf("double Stop() should not return error: %v", err)
	}
}

func TestServerEHLOResponse(t *testing.T) {
	logger := zerolog.New(io.Discard)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg := &Config{
		Host:            "127.0.0.1",
		Port:            port,
		Domain:          "test.example.com",
		MaxMessageSize:  5 * 1024 * 1024,
		MaxRecipients:   50,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		GracefulTimeout: 5 * time.Second,
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
	greeting := readFullResponse(t, conn)
	if !strings.Contains(greeting, "220") {
		t.Errorf("expected 220 greeting, got: %s", greeting)
	}

	// Send EHLO
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte("EHLO client.example.com\r\n"))
	if err != nil {
		t.Fatalf("failed to send EHLO: %v", err)
	}

	// Read EHLO response (multi-line)
	response := readFullResponse(t, conn)

	// Verify EHLO response contains expected extensions
	if !strings.Contains(response, "250") {
		t.Errorf("expected 250 response, got: %s", response)
	}
	// The go-smtp library echoes the client hostname, not the server domain
	if !strings.Contains(response, "client.example.com") {
		t.Errorf("expected client hostname in response, got: %s", response)
	}
	if !strings.Contains(response, "PIPELINING") {
		t.Errorf("expected PIPELINING extension in response, got: %s", response)
	}
	if !strings.Contains(response, "8BITMIME") {
		t.Errorf("expected 8BITMIME extension in response, got: %s", response)
	}

	// Send QUIT
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte("QUIT\r\n"))
	if err != nil {
		t.Fatalf("failed to send QUIT: %v", err)
	}
}

// readFullResponse reads a full SMTP response from the connection
// It handles multi-line responses (250-continuation lines followed by 250 final line)
func readFullResponse(t *testing.T, conn net.Conn) string {
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

func TestServerHELOResponse(t *testing.T) {
	logger := zerolog.New(io.Discard)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg := &Config{
		Host:            "127.0.0.1",
		Port:            port,
		Domain:          "helo.example.com",
		MaxMessageSize:  10 * 1024 * 1024,
		MaxRecipients:   100,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		GracefulTimeout: 5 * time.Second,
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

	// Connect and send HELO
	conn, err := net.DialTimeout("tcp", server.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	reader := make([]byte, 4096)

	// Read greeting
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(reader)
	if err != nil {
		t.Fatalf("failed to read greeting: %v", err)
	}

	// Send HELO
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte("HELO client.example.com\r\n"))
	if err != nil {
		t.Fatalf("failed to send HELO: %v", err)
	}

	// Read HELO response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = conn.Read(reader)
	if err != nil {
		t.Fatalf("failed to read HELO response: %v", err)
	}
	response := string(reader[:n])

	// Verify HELO response
	if !strings.Contains(response, "250") {
		t.Errorf("expected 250 response, got: %s", response)
	}
}

func TestServerTimeouts(t *testing.T) {
	logger := zerolog.New(io.Discard)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Test with very short timeouts
	cfg := &Config{
		Host:            "127.0.0.1",
		Port:            port,
		Domain:          "localhost",
		MaxMessageSize:  10 * 1024 * 1024,
		MaxRecipients:   100,
		ReadTimeout:     100 * time.Millisecond,
		WriteTimeout:    100 * time.Millisecond,
		GracefulTimeout: 1 * time.Second,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Verify timeouts are set on config
	if server.Config().ReadTimeout != cfg.ReadTimeout {
		t.Errorf("ReadTimeout = %v, want %v", server.Config().ReadTimeout, cfg.ReadTimeout)
	}
	if server.Config().WriteTimeout != cfg.WriteTimeout {
		t.Errorf("WriteTimeout = %v, want %v", server.Config().WriteTimeout, cfg.WriteTimeout)
	}
}

func TestStats(t *testing.T) {
	stats := NewStats()

	// Initial stats
	uptime, open, total, messages := stats.GetStats()
	if uptime < 0 {
		t.Error("uptime should not be negative")
	}
	if open != 0 {
		t.Errorf("initial connectionsOpen = %d, want 0", open)
	}
	if total != 0 {
		t.Errorf("initial connectionsTotal = %d, want 0", total)
	}
	if messages != 0 {
		t.Errorf("initial messagesTotal = %d, want 0", messages)
	}

	// Open connection
	stats.ConnectionOpened()
	_, open, total, _ = stats.GetStats()
	if open != 1 {
		t.Errorf("after open connectionsOpen = %d, want 1", open)
	}
	if total != 1 {
		t.Errorf("after open connectionsTotal = %d, want 1", total)
	}

	// Open another
	stats.ConnectionOpened()
	_, open, total, _ = stats.GetStats()
	if open != 2 {
		t.Errorf("after second open connectionsOpen = %d, want 2", open)
	}
	if total != 2 {
		t.Errorf("after second open connectionsTotal = %d, want 2", total)
	}

	// Close connection
	stats.ConnectionClosed()
	_, open, total, _ = stats.GetStats()
	if open != 1 {
		t.Errorf("after close connectionsOpen = %d, want 1", open)
	}
	if total != 2 {
		t.Errorf("after close connectionsTotal = %d, want 2", total)
	}

	// Receive message
	stats.MessageReceived()
	_, _, _, messages = stats.GetStats()
	if messages != 1 {
		t.Errorf("after message messagesTotal = %d, want 1", messages)
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	logger := zerolog.New(io.Discard)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg := &Config{
		Host:            "127.0.0.1",
		Port:            port,
		Domain:          "localhost",
		MaxMessageSize:  10 * 1024 * 1024,
		MaxRecipients:   100,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		GracefulTimeout: 5 * time.Second,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Establish a connection before shutdown
	conn, err := net.DialTimeout("tcp", server.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Read greeting
	reader := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Read(reader)
	if err != nil {
		t.Fatalf("failed to read greeting: %v", err)
	}

	// Start graceful shutdown
	shutdownComplete := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulTimeout)
		defer cancel()
		shutdownComplete <- server.Stop(ctx)
	}()

	// Close client connection to allow shutdown
	conn.Close()

	// Wait for shutdown
	select {
	case err := <-shutdownComplete:
		if err != nil {
			t.Errorf("shutdown error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("shutdown timed out")
	}

	if server.IsRunning() {
		t.Error("server should not be running after shutdown")
	}
}

func TestMailTransaction(t *testing.T) {
	logger := zerolog.New(io.Discard)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cfg := &Config{
		Host:              "127.0.0.1",
		Port:              port,
		Domain:            "mail.example.com",
		MaxMessageSize:    10 * 1024 * 1024,
		MaxRecipients:     100,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		GracefulTimeout:   5 * time.Second,
		AllowInsecureAuth: true,
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

	// Connect
	conn, err := net.DialTimeout("tcp", server.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Read greeting
	greeting := readFullResponse(t, conn)
	if !strings.Contains(greeting, "220") {
		t.Fatalf("expected 220 greeting, got: %s", greeting)
	}

	// EHLO
	sendCmd(t, conn, "EHLO test.client.com\r\n", "250")

	// MAIL FROM
	sendCmd(t, conn, "MAIL FROM:<sender@example.com>\r\n", "250")

	// RCPT TO
	sendCmd(t, conn, "RCPT TO:<recipient@example.com>\r\n", "250")

	// DATA
	sendCmd(t, conn, "DATA\r\n", "354")

	// Send message content
	message := "From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: Test Message\r\n" +
		"\r\n" +
		"This is a test message.\r\n" +
		".\r\n"

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte(message))
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// Read response after message data
	response := readFullResponse(t, conn)
	if !strings.Contains(response, "250") {
		t.Errorf("expected 250 response after message, got: %s", response)
	}

	// Verify stats
	_, _, _, messages := server.Stats().GetStats()
	if messages != 1 {
		t.Errorf("expected 1 message, got %d", messages)
	}

	// QUIT
	sendCmd(t, conn, "QUIT\r\n", "221")
}

func sendCmd(t *testing.T, conn net.Conn, cmd, expectedPrefix string) string {
	t.Helper()

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := conn.Write([]byte(cmd))
	if err != nil {
		t.Fatalf("failed to send %s: %v", cmd, err)
	}

	response := readFullResponse(t, conn)
	if !strings.Contains(response, expectedPrefix) {
		t.Fatalf("expected %s response for %s, got: %s", expectedPrefix, cmd, response)
	}
	return response
}

func TestServerConfig(t *testing.T) {
	logger := zerolog.New(io.Discard)

	cfg := &Config{
		Host:              "127.0.0.1",
		Port:              1025,
		Domain:            "localhost",
		MaxMessageSize:    5 * 1024 * 1024,
		MaxRecipients:     50,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      45 * time.Second,
		GracefulTimeout:   10 * time.Second,
		AuthRequired:      false, // No auth required for this test
		AllowInsecureAuth: true,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	returnedCfg := server.Config()
	if returnedCfg != cfg {
		t.Error("Config() should return the same config pointer")
	}

	if returnedCfg.Port != 1025 {
		t.Errorf("Port = %d, want 1025", returnedCfg.Port)
	}
	if returnedCfg.Domain != "localhost" {
		t.Errorf("Domain = %s, want localhost", returnedCfg.Domain)
	}
	if returnedCfg.MaxMessageSize != 5*1024*1024 {
		t.Errorf("MaxMessageSize = %d, want %d", returnedCfg.MaxMessageSize, 5*1024*1024)
	}
}

func TestConnectionLogging(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Use a buffer to capture logs (with synchronization)
	var logBuffer safeBuffer
	logger := zerolog.New(&logBuffer).Level(zerolog.InfoLevel)

	cfg := &Config{
		Host:            "127.0.0.1",
		Port:            port,
		Domain:          "localhost",
		MaxMessageSize:  10 * 1024 * 1024,
		MaxRecipients:   100,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		GracefulTimeout: 5 * time.Second,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Wait for server to be ready
	time.Sleep(50 * time.Millisecond)

	// Verify start log
	logs := logBuffer.String()
	if !strings.Contains(logs, "starting SMTP server") {
		t.Error("expected 'starting SMTP server' log message")
	}
	if !strings.Contains(logs, "SMTP server started") {
		t.Error("expected 'SMTP server started' log message")
	}
	if !strings.Contains(logs, fmt.Sprintf(":%d", port)) {
		t.Errorf("expected port %d in log message", port)
	}

	// Connect and disconnect to trigger connection logs
	conn, err := net.DialTimeout("tcp", server.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Read greeting
	reader := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.Read(reader)

	// Send HELO first (required before QUIT in some servers)
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	conn.Write([]byte("HELO test\r\n"))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.Read(reader)

	// Send QUIT
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	conn.Write([]byte("QUIT\r\n"))

	// Read response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.Read(reader)
	conn.Close()

	// Wait for logs to be written
	time.Sleep(200 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Stop(ctx)

	// Verify connection logs
	logs = logBuffer.String()
	if !strings.Contains(logs, "new connection") {
		t.Logf("logs: %s", logs)
		t.Error("expected 'new connection' log message")
	}
	if !strings.Contains(logs, "connection closed") {
		t.Logf("logs: %s", logs)
		t.Error("expected 'connection closed' log message")
	}
}

// safeBuffer is a thread-safe buffer for log capture
type safeBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *safeBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
