package smtp

import (
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/rs/zerolog"
)

func TestNewRateLimiter(t *testing.T) {
	logger := zerolog.New(io.Discard)

	t.Run("with nil config uses defaults", func(t *testing.T) {
		rl := NewRateLimiter(nil, logger)
		defer rl.Stop()

		if !rl.IsEnabled() {
			t.Error("expected rate limiter to be enabled by default")
		}

		cfg := rl.Config()
		if cfg.MessagesPerHour != 100 {
			t.Errorf("expected MessagesPerHour = 100, got %d", cfg.MessagesPerHour)
		}
		if cfg.ConnectionsPerMinute != 20 {
			t.Errorf("expected ConnectionsPerMinute = 20, got %d", cfg.ConnectionsPerMinute)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:                  true,
			MessagesPerHour:          50,
			ConnectionsPerMinute:     10,
			MaxConcurrentConnections: 5,
			MaxGlobalConnections:     100,
			CleanupInterval:          time.Minute,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		if rl.Config().MessagesPerHour != 50 {
			t.Errorf("expected MessagesPerHour = 50, got %d", rl.Config().MessagesPerHour)
		}
	})
}

func TestRateLimiterCheckConnection(t *testing.T) {
	logger := zerolog.New(io.Discard)

	t.Run("allows connections when disabled", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:              false,
			ConnectionsPerMinute: 1,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		// Should always allow when disabled
		for i := 0; i < 10; i++ {
			err := rl.CheckConnection(context.Background(), "127.0.0.1:12345")
			if err != nil {
				t.Errorf("expected nil error when disabled, got: %v", err)
			}
		}
	})

	t.Run("enforces connection rate limit", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:              true,
			ConnectionsPerMinute: 3,
			CleanupInterval:      0, // Disable cleanup
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		remoteAddr := "192.168.1.1:12345"

		// First 3 should succeed
		for i := 0; i < 3; i++ {
			err := rl.CheckConnection(context.Background(), remoteAddr)
			if err != nil {
				t.Errorf("connection %d should succeed, got error: %v", i+1, err)
			}
		}

		// 4th should fail
		err := rl.CheckConnection(context.Background(), remoteAddr)
		if err == nil {
			t.Error("expected 4th connection to be rate limited")
		}

		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 421 {
			t.Errorf("expected error code 421, got %d", smtpErr.Code)
		}
	})

	t.Run("enforces concurrent connection limit", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:                  true,
			MaxConcurrentConnections: 2,
			ConnectionsPerMinute:     100, // High limit to not interfere
			CleanupInterval:          0,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		remoteAddr := "10.0.0.1:12345"

		// First 2 connections
		rl.CheckConnection(context.Background(), remoteAddr)
		rl.OnConnectionOpened(remoteAddr)
		rl.CheckConnection(context.Background(), remoteAddr)
		rl.OnConnectionOpened(remoteAddr)

		// 3rd should fail
		err := rl.CheckConnection(context.Background(), remoteAddr)
		if err == nil {
			t.Error("expected 3rd concurrent connection to be rejected")
		}

		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 421 {
			t.Errorf("expected error code 421, got %d", smtpErr.Code)
		}

		// Close one connection
		rl.OnConnectionClosed(remoteAddr)

		// Now new connection should be allowed
		err = rl.CheckConnection(context.Background(), remoteAddr)
		if err != nil {
			t.Errorf("expected connection after close to succeed, got: %v", err)
		}
	})

	t.Run("enforces global connection limit", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:                  true,
			MaxGlobalConnections:     3,
			MaxConcurrentConnections: 100, // High per-IP limit
			ConnectionsPerMinute:     100,
			CleanupInterval:          0,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		// Open connections from different IPs
		for i := 0; i < 3; i++ {
			addr := "192.168.1." + string(rune('1'+i)) + ":12345"
			rl.CheckConnection(context.Background(), addr)
			rl.OnConnectionOpened(addr)
		}

		// 4th should fail (global limit)
		err := rl.CheckConnection(context.Background(), "10.0.0.1:12345")
		if err == nil {
			t.Error("expected global connection limit to reject connection")
		}

		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 421 {
			t.Errorf("expected error code 421, got %d", smtpErr.Code)
		}
	})
}

func TestRateLimiterCheckMessage(t *testing.T) {
	logger := zerolog.New(io.Discard)

	t.Run("allows messages when disabled", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:         false,
			MessagesPerHour: 1,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		for i := 0; i < 10; i++ {
			err := rl.CheckMessage(context.Background(), "127.0.0.1:12345")
			if err != nil {
				t.Errorf("expected nil error when disabled, got: %v", err)
			}
		}
	})

	t.Run("enforces messages per connection limit", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:               true,
			MessagesPerConnection: 2,
			MessagesPerHour:       100, // High limit to not interfere
			CleanupInterval:       0,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		remoteAddr := "192.168.1.1:12345"

		// First 2 messages
		for i := 0; i < 2; i++ {
			rl.CheckMessage(context.Background(), remoteAddr)
			rl.OnMessageSent(remoteAddr)
		}

		// 3rd should fail
		err := rl.CheckMessage(context.Background(), remoteAddr)
		if err == nil {
			t.Error("expected 3rd message to be rate limited")
		}

		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 452 {
			t.Errorf("expected error code 452, got %d", smtpErr.Code)
		}
	})

	t.Run("enforces hourly message limit", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:               true,
			MessagesPerHour:       3,
			MessagesPerConnection: 100, // High limit to not interfere
			CleanupInterval:       0,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		ip := "192.168.1.1"
		remoteAddr := ip + ":12345"

		// First 3 messages
		for i := 0; i < 3; i++ {
			err := rl.CheckMessage(context.Background(), remoteAddr)
			if err != nil {
				t.Errorf("message %d should succeed, got error: %v", i+1, err)
			}
			rl.OnMessageSent(remoteAddr)
		}

		// 4th should fail
		err := rl.CheckMessage(context.Background(), remoteAddr)
		if err == nil {
			t.Error("expected 4th message to be rate limited")
		}

		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 452 {
			t.Errorf("expected error code 452, got %d", smtpErr.Code)
		}

		// Different IP should still work
		err = rl.CheckMessage(context.Background(), "10.0.0.1:54321")
		if err != nil {
			t.Errorf("different IP should succeed, got: %v", err)
		}
	})
}

func TestRateLimiterCheckRecipients(t *testing.T) {
	logger := zerolog.New(io.Discard)

	t.Run("allows within limit", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:              true,
			RecipientsPerMessage: 10,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		err := rl.CheckRecipients(5)
		if err != nil {
			t.Errorf("expected nil error for 5 recipients, got: %v", err)
		}
	})

	t.Run("rejects over limit", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:              true,
			RecipientsPerMessage: 5,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		err := rl.CheckRecipients(10)
		if err == nil {
			t.Error("expected error for 10 recipients")
		}

		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 452 {
			t.Errorf("expected error code 452, got %d", smtpErr.Code)
		}
	})

	t.Run("allows any when disabled", func(t *testing.T) {
		cfg := &RateLimitConfig{
			Enabled:              false,
			RecipientsPerMessage: 1,
		}
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		err := rl.CheckRecipients(1000)
		if err != nil {
			t.Errorf("expected nil error when disabled, got: %v", err)
		}
	})
}

func TestRateLimiterConcurrency(t *testing.T) {
	logger := zerolog.New(io.Discard)
	cfg := &RateLimitConfig{
		Enabled:                  true,
		MessagesPerHour:          1000,
		ConnectionsPerMinute:     1000,
		MaxConcurrentConnections: 100,
		MaxGlobalConnections:     1000,
		MessagesPerConnection:    100,
		CleanupInterval:          0,
	}
	rl := NewRateLimiter(cfg, logger)
	defer rl.Stop()

	var wg sync.WaitGroup
	numGoroutines := 50
	operationsPerGoroutine := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			remoteAddr := "192.168.1." + string(rune('1'+id%255)) + ":12345"

			for j := 0; j < operationsPerGoroutine; j++ {
				_ = rl.CheckConnection(context.Background(), remoteAddr)
				rl.OnConnectionOpened(remoteAddr)
				_ = rl.CheckMessage(context.Background(), remoteAddr)
				rl.OnMessageSent(remoteAddr)
				rl.OnConnectionClosed(remoteAddr)
			}
		}(i)
	}

	wg.Wait()

	// Verify no data races or panics occurred
	stats := rl.GetStats()
	if stats.GlobalConnections < 0 {
		t.Errorf("negative global connections: %d", stats.GlobalConnections)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	logger := zerolog.New(io.Discard)
	cfg := &RateLimitConfig{
		Enabled:              true,
		MessagesPerHour:      100,
		ConnectionsPerMinute: 100,
		CleanupInterval:      50 * time.Millisecond,
	}
	rl := NewRateLimiter(cfg, logger)
	defer rl.Stop()

	// Add some entries
	remoteAddr := "192.168.1.1:12345"
	rl.CheckConnection(context.Background(), remoteAddr)
	rl.CheckMessage(context.Background(), remoteAddr)
	rl.OnMessageSent(remoteAddr)

	// Wait for cleanup to run
	time.Sleep(150 * time.Millisecond)

	// Verify cleanup ran without panics
	stats := rl.GetStats()
	_ = stats // Just verify we can get stats after cleanup
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1:12345", "192.168.1.1"},
		{"10.0.0.1:80", "10.0.0.1"},
		{"[::1]:8080", "::1"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"localhost:25", "localhost"},
		{"no-port", "no-port"}, // Fallback when no port
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractIP(tt.input)
			if result != tt.expected {
				t.Errorf("extractIP(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRateLimiterStats(t *testing.T) {
	logger := zerolog.New(io.Discard)
	cfg := &RateLimitConfig{
		Enabled:              true,
		MessagesPerHour:      100,
		ConnectionsPerMinute: 100,
		CleanupInterval:      0,
	}
	rl := NewRateLimiter(cfg, logger)
	defer rl.Stop()

	// Initial stats should be zero
	stats := rl.GetStats()
	if stats.GlobalConnections != 0 {
		t.Errorf("expected 0 global connections, got %d", stats.GlobalConnections)
	}

	// Add some activity
	rl.OnConnectionOpened("192.168.1.1:12345")
	rl.OnConnectionOpened("192.168.1.2:12345")
	rl.OnMessageSent("192.168.1.1:12345")

	stats = rl.GetStats()
	if stats.GlobalConnections != 2 {
		t.Errorf("expected 2 global connections, got %d", stats.GlobalConnections)
	}
	if stats.UniqueIPsWithMessages != 1 {
		t.Errorf("expected 1 unique IP with messages, got %d", stats.UniqueIPsWithMessages)
	}
}

func TestRateLimitingIntegration(t *testing.T) {
	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	logger := zerolog.New(io.Discard)

	// Create rate limit config with very low limits for testing
	rateLimitCfg := &RateLimitConfig{
		Enabled:               true,
		MessagesPerHour:       2,
		ConnectionsPerMinute:  10,
		MaxConcurrentConnections: 5,
		MaxGlobalConnections:  10,
		MessagesPerConnection: 1,
		CleanupInterval:       0,
	}

	cfg := &Config{
		Host:              "127.0.0.1",
		Port:              port,
		Domain:            "test.example.com",
		MaxMessageSize:    10 * 1024 * 1024,
		MaxRecipients:     100,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		GracefulTimeout:   5 * time.Second,
		AllowInsecureAuth: true,
		RateLimitEnabled:  true,
		RateLimitConfig:   rateLimitCfg,
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

	// Verify rate limiter is configured
	if server.RateLimiter() == nil {
		t.Fatal("expected rate limiter to be configured")
	}

	// Connect and send first message (should succeed)
	conn, err := net.DialTimeout("tcp", server.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Read greeting
	_ = readFullResponse(t, conn)

	// EHLO
	sendCmd(t, conn, "EHLO client.test\r\n", "250")

	// MAIL FROM
	sendCmd(t, conn, "MAIL FROM:<sender@test.com>\r\n", "250")

	// RCPT TO
	sendCmd(t, conn, "RCPT TO:<recipient@example.com>\r\n", "250")

	// DATA
	sendCmd(t, conn, "DATA\r\n", "354")

	// Send message
	message := "From: sender@test.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: Test\r\n" +
		"\r\n" +
		"Test body.\r\n" +
		".\r\n"

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte(message))
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	response := readFullResponse(t, conn)
	if !strings.Contains(response, "250") {
		t.Errorf("expected 250 after first message, got: %s", response)
	}

	// Try to send second message (should be rate limited due to MessagesPerConnection=1)
	sendCmd(t, conn, "MAIL FROM:<sender2@test.com>\r\n", "250")
	sendCmd(t, conn, "RCPT TO:<recipient@example.com>\r\n", "250")
	sendCmd(t, conn, "DATA\r\n", "354")

	message2 := "Subject: Test 2\r\n\r\nBody 2\r\n.\r\n"
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	conn.Write([]byte(message2))

	response = readFullResponse(t, conn)
	if !strings.Contains(response, "452") {
		t.Errorf("expected 452 for rate limited message, got: %s", response)
	}

	// QUIT
	sendCmd(t, conn, "QUIT\r\n", "221")
}

// Note: readFullResponse and sendCmd are defined in server_test.go
