package smtp

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/rs/zerolog"

	"yunt/internal/domain"
)

func TestSessionMailFrom(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name    string
		from    string
		size    int64
		wantErr bool
		errCode int
	}{
		{
			name:    "valid address",
			from:    "sender@example.com",
			wantErr: false,
		},
		{
			name:    "valid address with angle brackets",
			from:    "<sender@example.com>",
			wantErr: false,
		},
		{
			name:    "empty address (bounce message)",
			from:    "",
			wantErr: false,
		},
		{
			name:    "invalid address - no @",
			from:    "invalid-address",
			wantErr: true,
			errCode: 553,
		},
		{
			name:    "invalid address - empty local part",
			from:    "@example.com",
			wantErr: true,
			errCode: 553,
		},
		{
			name:    "invalid address - empty domain",
			from:    "user@",
			wantErr: true,
			errCode: 553,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewSession(server.Backend(), nil, "127.0.0.1:12345")

			opts := &smtp.MailOptions{}
			if tt.size > 0 {
				opts.Size = tt.size
			}

			err := session.Mail(tt.from, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mail() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errCode > 0 {
				smtpErr, ok := err.(*smtp.SMTPError)
				if !ok {
					t.Errorf("expected SMTPError, got %T", err)
				} else if smtpErr.Code != tt.errCode {
					t.Errorf("expected error code %d, got %d", tt.errCode, smtpErr.Code)
				}
			}
		})
	}
}

func TestSessionMailFromSizeLimit(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	logger := zerolog.New(io.Discard)
	cfg := &Config{
		Host:              "127.0.0.1",
		Port:              port,
		Domain:            "test.example.com",
		MaxMessageSize:    1024, // 1KB limit
		MaxRecipients:     100,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		AllowInsecureAuth: true,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")

	// Size within limit should succeed
	err = session.Mail("sender@example.com", &smtp.MailOptions{Size: 512})
	if err != nil {
		t.Errorf("expected size within limit to succeed, got error: %v", err)
	}

	session.Reset()

	// Size exceeding limit should fail with 552
	err = session.Mail("sender@example.com", &smtp.MailOptions{Size: 2048})
	if err == nil {
		t.Error("expected size exceeding limit to fail")
	} else {
		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 552 {
			t.Errorf("expected error code 552, got %d", smtpErr.Code)
		}
	}
}

func TestSessionRcptTo(t *testing.T) {
	mailboxRepo := newMockMailboxRepository()
	mailbox := &domain.Mailbox{
		ID:      "mailbox-1",
		UserID:  "user-1",
		Name:    "Test",
		Address: "valid@example.com",
	}
	mailboxRepo.addMailbox(mailbox)

	server := createTestServer(t, WithMailboxRepo(mailboxRepo))

	tests := []struct {
		name    string
		to      string
		wantErr bool
		errCode int
	}{
		{
			name:    "valid recipient",
			to:      "valid@example.com",
			wantErr: false,
		},
		{
			name:    "valid recipient with angle brackets",
			to:      "<valid@example.com>",
			wantErr: false,
		},
		{
			name:    "invalid recipient - not in database",
			to:      "invalid@example.com",
			wantErr: true,
			errCode: 550,
		},
		{
			name:    "invalid address format",
			to:      "not-an-email",
			wantErr: true,
			errCode: 553,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
			session.Mail("sender@example.com", nil)

			err := session.Rcpt(tt.to, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Rcpt() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errCode > 0 {
				smtpErr, ok := err.(*smtp.SMTPError)
				if !ok {
					t.Errorf("expected SMTPError, got %T", err)
				} else if smtpErr.Code != tt.errCode {
					t.Errorf("expected error code %d, got %d", tt.errCode, smtpErr.Code)
				}
			}
		})
	}
}

func TestSessionRcptToMaxRecipients(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	logger := zerolog.New(io.Discard)
	cfg := &Config{
		Host:              "127.0.0.1",
		Port:              port,
		Domain:            "test.example.com",
		MaxMessageSize:    10 * 1024 * 1024,
		MaxRecipients:     3, // Only 3 recipients allowed
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		AllowInsecureAuth: true,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
	session.Mail("sender@example.com", nil)

	// Add 3 recipients (should succeed)
	recipients := []string{"recipient1@example.com", "recipient2@example.com", "recipient3@example.com"}
	for i, rcpt := range recipients {
		err := session.Rcpt(rcpt, nil)
		if err != nil {
			t.Errorf("expected recipient %d to succeed, got error: %v", i+1, err)
		}
	}

	// 4th recipient should fail with 452
	err = session.Rcpt("extra@example.com", nil)
	if err == nil {
		t.Error("expected 4th recipient to fail")
	} else {
		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 452 {
			t.Errorf("expected error code 452, got %d", smtpErr.Code)
		}
	}
}

func TestSessionDuplicateRecipients(t *testing.T) {
	server := createTestServer(t)

	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
	session.Mail("sender@example.com", nil)

	// Add the same recipient twice
	err := session.Rcpt("recipient@example.com", nil)
	if err != nil {
		t.Errorf("expected first recipient to succeed, got error: %v", err)
	}

	err = session.Rcpt("recipient@example.com", nil)
	if err != nil {
		t.Errorf("expected duplicate recipient to be silently accepted, got error: %v", err)
	}

	// Should only have 1 unique recipient
	if session.RecipientCount() != 1 {
		t.Errorf("expected 1 recipient, got %d", session.RecipientCount())
	}
}

func TestSessionDataNoRecipients(t *testing.T) {
	server := createTestServer(t)

	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
	session.Mail("sender@example.com", nil)

	// Try DATA without any recipients
	err := session.Data(strings.NewReader("test message"))
	if err == nil {
		t.Error("expected DATA to fail without recipients")
	} else {
		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 503 {
			t.Errorf("expected error code 503, got %d", smtpErr.Code)
		}
	}
}

func TestSessionDataWithSizeLimit(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	logger := zerolog.New(io.Discard)
	cfg := &Config{
		Host:              "127.0.0.1",
		Port:              port,
		Domain:            "test.example.com",
		MaxMessageSize:    100, // 100 byte limit
		MaxRecipients:     100,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		AllowInsecureAuth: true,
	}

	server, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
	session.Mail("sender@example.com", nil)
	session.Rcpt("recipient@example.com", nil)

	// Message exceeding size limit should fail
	largeMessage := strings.Repeat("X", 200)
	err = session.Data(strings.NewReader(largeMessage))
	if err == nil {
		t.Error("expected DATA to fail when message exceeds size limit")
	} else {
		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Errorf("expected SMTPError, got %T", err)
		} else if smtpErr.Code != 552 {
			t.Errorf("expected error code 552, got %d", smtpErr.Code)
		}
	}
}

func TestSessionDataSuccess(t *testing.T) {
	mailboxRepo := newMockMailboxRepository()
	messageRepo := newMockMessageRepository()

	mailbox := &domain.Mailbox{
		ID:      "mailbox-1",
		UserID:  "user-1",
		Name:    "Test",
		Address: "recipient@example.com",
	}
	mailboxRepo.addMailbox(mailbox)

	server := createTestServer(t,
		WithMailboxRepo(mailboxRepo),
		WithMessageRepo(messageRepo),
	)

	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
	session.Mail("sender@example.com", nil)
	session.Rcpt("recipient@example.com", nil)

	message := "From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: Test\r\n" +
		"\r\n" +
		"Test body"

	err := session.Data(strings.NewReader(message))
	if err != nil {
		t.Errorf("expected DATA to succeed, got error: %v", err)
	}

	// Verify message was stored
	messages := messageRepo.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message stored, got %d", len(messages))
	} else {
		if messages[0].From.Address != "sender@example.com" {
			t.Errorf("expected from = sender@example.com, got %s", messages[0].From.Address)
		}
		if len(messages[0].To) != 1 || messages[0].To[0].Address != "recipient@example.com" {
			t.Errorf("unexpected recipients: %v", messages[0].To)
		}
	}
}

func TestSessionReset(t *testing.T) {
	server := createTestServer(t)

	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
	session.Mail("sender@example.com", nil)
	session.Rcpt("recipient1@example.com", nil)
	session.Rcpt("recipient2@example.com", nil)

	// Verify state before reset
	if session.From() != "sender@example.com" {
		t.Errorf("expected from = sender@example.com, got %s", session.From())
	}
	if session.RecipientCount() != 2 {
		t.Errorf("expected 2 recipients, got %d", session.RecipientCount())
	}

	// Reset
	session.Reset()

	// Verify state after reset
	if session.From() != "" {
		t.Errorf("expected from to be empty after reset, got %s", session.From())
	}
	if session.RecipientCount() != 0 {
		t.Errorf("expected 0 recipients after reset, got %d", session.RecipientCount())
	}
}

func TestSessionLogout(t *testing.T) {
	server := createTestServer(t)

	// Start the server to track stats
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	// Create session (which increments connection count via NewSession in Backend)
	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
	server.stats.ConnectionOpened() // Simulate backend NewSession

	_, open, _, _ := server.Stats().GetStats()
	if open != 1 {
		t.Errorf("expected 1 open connection, got %d", open)
	}

	// Logout should decrement connection count
	session.Logout()

	_, open, _, _ = server.Stats().GetStats()
	if open != 0 {
		t.Errorf("expected 0 open connections after logout, got %d", open)
	}
}

func TestExtractAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<user@example.com>", "user@example.com"},
		{"user@example.com", "user@example.com"},
		{"  <user@example.com>  ", "user@example.com"},
		{"  user@example.com  ", "user@example.com"},
		{"<>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractAddress(tt.input)
			if result != tt.expected {
				t.Errorf("extractAddress(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidEmailFormat(t *testing.T) {
	tests := []struct {
		email   string
		isValid bool
	}{
		{"user@example.com", true},
		{"user@localhost", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"", false},
		{"user", false},
		{"@example.com", false},
		{"user@", false},
		{"user@@example.com", false},
		{"user @example.com", false},
		{"user\t@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmailFormat(tt.email)
			if result != tt.isValid {
				t.Errorf("isValidEmailFormat(%q) = %v, want %v", tt.email, result, tt.isValid)
			}
		})
	}
}

func TestMultipleRecipientsToSameMailbox(t *testing.T) {
	mailboxRepo := newMockMailboxRepository()
	messageRepo := newMockMessageRepository()

	// Create a catch-all mailbox
	mailbox := &domain.Mailbox{
		ID:         "mailbox-1",
		UserID:     "user-1",
		Name:       "Catch-All",
		Address:    "*@example.com",
		IsCatchAll: true,
	}
	mailboxRepo.addMailbox(mailbox)

	server := createTestServer(t,
		WithMailboxRepo(mailboxRepo),
		WithMessageRepo(messageRepo),
	)

	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")
	session.Mail("sender@example.com", nil)

	// Add multiple recipients (all will map to the catch-all mailbox)
	session.Rcpt("user1@example.com", nil)
	session.Rcpt("user2@example.com", nil)
	session.Rcpt("user3@example.com", nil)

	message := "Subject: Test\r\n\r\nBody"
	err := session.Data(strings.NewReader(message))
	if err != nil {
		t.Errorf("expected DATA to succeed, got error: %v", err)
	}

	// Should only store one message (since all go to the same mailbox)
	messages := messageRepo.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message stored (deduped by mailbox), got %d", len(messages))
	}

	// But the message should have all 3 recipients
	if len(messages) > 0 && len(messages[0].To) != 3 {
		t.Errorf("expected 3 recipients in message, got %d", len(messages[0].To))
	}
}

func TestSessionWithRealSMTPConnection(t *testing.T) {
	mailboxRepo := newMockMailboxRepository()
	messageRepo := newMockMessageRepository()

	mailbox := &domain.Mailbox{
		ID:      "mailbox-1",
		UserID:  "user-1",
		Name:    "Test",
		Address: "test@example.com",
	}
	mailboxRepo.addMailbox(mailbox)

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	logger := zerolog.New(io.Discard)
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
	}

	server, err := New(cfg, logger,
		WithMailboxRepo(mailboxRepo),
		WithMessageRepo(messageRepo),
	)
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

	// Connect and send a complete transaction
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

	// RCPT TO - valid recipient
	sendCmd(t, conn, "RCPT TO:<test@example.com>\r\n", "250")

	// DATA
	sendCmd(t, conn, "DATA\r\n", "354")

	// Send message
	message := "From: sender@test.com\r\n" +
		"To: test@example.com\r\n" +
		"Subject: Integration Test\r\n" +
		"\r\n" +
		"This is a test message.\r\n" +
		".\r\n"

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte(message))
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	response := readFullResponse(t, conn)
	if !strings.Contains(response, "250") {
		t.Errorf("expected 250 after message, got: %s", response)
	}

	// Verify message was stored
	messages := messageRepo.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message stored, got %d", len(messages))
	}

	// QUIT
	sendCmd(t, conn, "QUIT\r\n", "221")
}

func TestSessionInvalidRecipientRejection(t *testing.T) {
	mailboxRepo := newMockMailboxRepository()

	// Only add one valid mailbox
	mailbox := &domain.Mailbox{
		ID:      "mailbox-1",
		UserID:  "user-1",
		Name:    "Valid",
		Address: "valid@example.com",
	}
	mailboxRepo.addMailbox(mailbox)

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	logger := zerolog.New(io.Discard)
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
	}

	server, err := New(cfg, logger, WithMailboxRepo(mailboxRepo))
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
	_ = readFullResponse(t, conn)

	// EHLO
	sendCmd(t, conn, "EHLO client.test\r\n", "250")

	// MAIL FROM
	sendCmd(t, conn, "MAIL FROM:<sender@test.com>\r\n", "250")

	// RCPT TO - invalid recipient (not in database)
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte("RCPT TO:<invalid@example.com>\r\n"))
	if err != nil {
		t.Fatalf("failed to send RCPT: %v", err)
	}

	response := readFullResponse(t, conn)
	if !strings.Contains(response, "550") {
		t.Errorf("expected 550 for invalid recipient, got: %s", response)
	}

	// QUIT
	sendCmd(t, conn, "QUIT\r\n", "221")
}
