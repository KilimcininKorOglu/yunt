package imap

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

func TestMessageBuilder_Build(t *testing.T) {
	now := domain.Now()
	msg := &domain.Message{
		ID:        domain.ID("test-msg-1"),
		MailboxID: domain.ID("inbox-1"),
		MessageID: "test@example.com",
		From:      domain.EmailAddress{Name: "Sender", Address: "sender@example.com"},
		To: []domain.EmailAddress{
			{Name: "Recipient", Address: "recipient@example.com"},
		},
		Subject:    "Test Subject",
		TextBody:   "This is a test message.",
		ReceivedAt: now,
	}

	builder := NewMessageBuilder()
	result := builder.Build(msg)

	// Check that essential headers are present
	resultStr := string(result)
	if !strings.Contains(resultStr, "From: Sender <sender@example.com>") {
		t.Error("Missing or incorrect From header")
	}
	if !strings.Contains(resultStr, "Subject: Test Subject") {
		t.Error("Missing Subject header")
	}
	if !strings.Contains(resultStr, "To: Recipient <recipient@example.com>") {
		t.Error("Missing To header")
	}
	if !strings.Contains(resultStr, "Message-ID: <test@example.com>") {
		t.Error("Missing Message-ID header")
	}
	if !strings.Contains(resultStr, "This is a test message.") {
		t.Error("Missing message body")
	}
}

func TestMessageBuilder_BuildWithHTMLAndText(t *testing.T) {
	now := domain.Now()
	msg := &domain.Message{
		ID:        domain.ID("test-msg-2"),
		MailboxID: domain.ID("inbox-1"),
		From:      domain.EmailAddress{Address: "sender@example.com"},
		To: []domain.EmailAddress{
			{Address: "recipient@example.com"},
		},
		Subject:    "Multipart Test",
		TextBody:   "Plain text body",
		HTMLBody:   "<html><body>HTML body</body></html>",
		ReceivedAt: now,
	}

	builder := NewMessageBuilder()
	result := builder.Build(msg)
	resultStr := string(result)

	// Check for multipart structure
	if !strings.Contains(resultStr, "multipart/alternative") {
		t.Error("Expected multipart/alternative content type")
	}
	if !strings.Contains(resultStr, "text/plain") {
		t.Error("Missing text/plain part")
	}
	if !strings.Contains(resultStr, "text/html") {
		t.Error("Missing text/html part")
	}
	if !strings.Contains(resultStr, "Plain text body") {
		t.Error("Missing plain text content")
	}
	if !strings.Contains(resultStr, "<html><body>HTML body</body></html>") {
		t.Error("Missing HTML content")
	}
}

func TestSplitEmailAddress(t *testing.T) {
	tests := []struct {
		email        string
		wantMailbox  string
		wantHost     string
	}{
		{"user@example.com", "user", "example.com"},
		{"user", "user", ""},
		{"user@subdomain.example.com", "user", "subdomain.example.com"},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			mailbox, host := splitEmailAddress(tt.email)
			if mailbox != tt.wantMailbox {
				t.Errorf("splitEmailAddress(%q) mailbox = %q, want %q", tt.email, mailbox, tt.wantMailbox)
			}
			if host != tt.wantHost {
				t.Errorf("splitEmailAddress(%q) host = %q, want %q", tt.email, host, tt.wantHost)
			}
		})
	}
}

func TestDomainToIMAPAddress(t *testing.T) {
	addr := domain.EmailAddress{
		Name:    "John Doe",
		Address: "john@example.com",
	}

	imapAddr := domainToIMAPAddress(addr)

	if imapAddr.Name != "John Doe" {
		t.Errorf("Name = %q, want %q", imapAddr.Name, "John Doe")
	}
	if imapAddr.Mailbox != "john" {
		t.Errorf("Mailbox = %q, want %q", imapAddr.Mailbox, "john")
	}
	if imapAddr.Host != "example.com" {
		t.Errorf("Host = %q, want %q", imapAddr.Host, "example.com")
	}
}

func TestApplyPartial(t *testing.T) {
	data := []byte("Hello, World!")

	tests := []struct {
		name    string
		partial *imap.SectionPartial
		want    string
	}{
		{
			name:    "nil partial",
			partial: nil,
			want:    "Hello, World!",
		},
		{
			name:    "offset only",
			partial: &imap.SectionPartial{Offset: 7, Size: 0},
			want:    "World!",
		},
		{
			name:    "offset and size",
			partial: &imap.SectionPartial{Offset: 0, Size: 5},
			want:    "Hello",
		},
		{
			name:    "offset and size in middle",
			partial: &imap.SectionPartial{Offset: 7, Size: 5},
			want:    "World",
		},
		{
			name:    "size exceeds data",
			partial: &imap.SectionPartial{Offset: 7, Size: 100},
			want:    "World!",
		},
		{
			name:    "offset exceeds data",
			partial: &imap.SectionPartial{Offset: 100, Size: 10},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyPartial(data, tt.partial)
			if string(result) != tt.want {
				t.Errorf("applyPartial() = %q, want %q", string(result), tt.want)
			}
		})
	}
}

func TestFilterHeaders(t *testing.T) {
	headers := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\nDate: Mon, 1 Jan 2024 00:00:00 +0000\r\n")

	tests := []struct {
		name      string
		fields    []string
		notFields []string
		wantIncl  []string
		wantExcl  []string
	}{
		{
			name:     "include From and Subject",
			fields:   []string{"From", "Subject"},
			wantIncl: []string{"From:", "Subject:"},
			wantExcl: []string{"To:", "Date:"},
		},
		{
			name:      "exclude Date",
			notFields: []string{"Date"},
			wantIncl:  []string{"From:", "To:", "Subject:"},
			wantExcl:  []string{"Date:"},
		},
		{
			name:     "case insensitive include",
			fields:   []string{"from"},
			wantIncl: []string{"From:"},
			wantExcl: []string{"To:", "Subject:", "Date:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterHeaders(headers, tt.fields, tt.notFields)
			resultStr := string(result)

			for _, incl := range tt.wantIncl {
				if !strings.Contains(resultStr, incl) {
					t.Errorf("Expected %q to be included, but it wasn't. Result: %q", incl, resultStr)
				}
			}

			for _, excl := range tt.wantExcl {
				if strings.Contains(resultStr, excl) {
					t.Errorf("Expected %q to be excluded, but it was found. Result: %q", excl, resultStr)
				}
			}
		})
	}
}

func TestPartExtractor_ExtractPart(t *testing.T) {
	// Simple multipart message
	rawMessage := []byte(`Content-Type: multipart/mixed; boundary="boundary1"

--boundary1
Content-Type: text/plain

This is part 1.
--boundary1
Content-Type: text/html

<p>This is part 2.</p>
--boundary1--`)

	extractor := NewPartExtractor(rawMessage)

	tests := []struct {
		name     string
		partPath []int
		wantIncl string
	}{
		{
			name:     "whole message",
			partPath: []int{},
			wantIncl: "multipart/mixed",
		},
		{
			name:     "part 1",
			partPath: []int{1},
			wantIncl: "This is part 1",
		},
		{
			name:     "part 2",
			partPath: []int{2},
			wantIncl: "This is part 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.ExtractPart(tt.partPath)
			if err != nil {
				t.Fatalf("ExtractPart() error = %v", err)
			}
			if !bytes.Contains(result, []byte(tt.wantIncl)) {
				t.Errorf("ExtractPart() result does not contain %q. Got: %q", tt.wantIncl, string(result))
			}
		})
	}
}

func TestPartExtractor_ExtractBoundary(t *testing.T) {
	tests := []struct {
		name         string
		rawBody      []byte
		wantBoundary string
	}{
		{
			name:         "simple boundary",
			rawBody:      []byte("Content-Type: multipart/mixed; boundary=myboundary\r\n\r\n"),
			wantBoundary: "myboundary",
		},
		{
			name:         "quoted boundary",
			rawBody:      []byte(`Content-Type: multipart/mixed; boundary="my-boundary-123"` + "\r\n\r\n"),
			wantBoundary: "my-boundary-123",
		},
		{
			name:         "no boundary",
			rawBody:      []byte("Content-Type: text/plain\r\n\r\n"),
			wantBoundary: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewPartExtractor(tt.rawBody)
			boundary := extractor.extractBoundary(tt.rawBody)
			if boundary != tt.wantBoundary {
				t.Errorf("extractBoundary() = %q, want %q", boundary, tt.wantBoundary)
			}
		})
	}
}

func TestFetchHandler_GetMessageFlags(t *testing.T) {
	handler := &FetchHandler{}

	tests := []struct {
		name     string
		msg      *domain.Message
		wantFlag imap.Flag
		hasFlag  bool
	}{
		{
			name: "read message has Seen flag",
			msg: &domain.Message{
				Status: domain.MessageRead,
			},
			wantFlag: imap.FlagSeen,
			hasFlag:  true,
		},
		{
			name: "unread message lacks Seen flag",
			msg: &domain.Message{
				Status: domain.MessageUnread,
			},
			wantFlag: imap.FlagSeen,
			hasFlag:  false,
		},
		{
			name: "starred message has Flagged flag",
			msg: &domain.Message{
				IsStarred: true,
			},
			wantFlag: imap.FlagFlagged,
			hasFlag:  true,
		},
		{
			name: "message with IsAnswered has Answered flag",
			msg: &domain.Message{
				IsAnswered: true,
			},
			wantFlag: imap.FlagAnswered,
			hasFlag:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := handler.getMessageFlags(tt.msg)

			found := false
			for _, f := range flags {
				if f == tt.wantFlag {
					found = true
					break
				}
			}

			if found != tt.hasFlag {
				t.Errorf("getMessageFlags() flag %v found = %v, want %v", tt.wantFlag, found, tt.hasFlag)
			}
		})
	}
}

func TestFetchHandler_BuildEnvelope(t *testing.T) {
	handler := &FetchHandler{}

	sentTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	sentTimestamp := domain.Timestamp{Time: sentTime}

	msg := &domain.Message{
		Subject:   "Test Subject",
		MessageID: "test@example.com",
		From:      domain.EmailAddress{Name: "Sender", Address: "sender@example.com"},
		To: []domain.EmailAddress{
			{Name: "Recipient", Address: "recipient@example.com"},
		},
		Cc: []domain.EmailAddress{
			{Address: "cc@example.com"},
		},
		ReplyTo:   &domain.EmailAddress{Address: "reply@example.com"},
		InReplyTo: "original@example.com",
		SentAt:    &sentTimestamp,
	}

	envelope := handler.buildEnvelope(msg)

	if envelope.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", envelope.Subject, "Test Subject")
	}

	if envelope.MessageID != "test@example.com" {
		t.Errorf("MessageID = %q, want %q", envelope.MessageID, "test@example.com")
	}

	if !envelope.Date.Equal(sentTime) {
		t.Errorf("Date = %v, want %v", envelope.Date, sentTime)
	}

	if len(envelope.From) != 1 || envelope.From[0].Name != "Sender" {
		t.Error("From address not set correctly")
	}

	if len(envelope.To) != 1 || envelope.To[0].Mailbox != "recipient" {
		t.Error("To address not set correctly")
	}

	if len(envelope.Cc) != 1 || envelope.Cc[0].Mailbox != "cc" {
		t.Error("Cc address not set correctly")
	}

	if len(envelope.ReplyTo) != 1 || envelope.ReplyTo[0].Mailbox != "reply" {
		t.Error("ReplyTo address not set correctly")
	}

	if len(envelope.InReplyTo) != 1 || envelope.InReplyTo[0] != "original@example.com" {
		t.Error("InReplyTo not set correctly")
	}
}

func TestGenerateBoundary(t *testing.T) {
	b1 := generateBoundary()

	if b1 == "" {
		t.Error("generateBoundary() returned empty string")
	}

	if !strings.HasPrefix(b1, "=_Part_") {
		t.Errorf("generateBoundary() = %q, want prefix =_Part_", b1)
	}

	// Wait a bit to ensure different timestamps
	time.Sleep(2 * time.Millisecond)
	b2 := generateBoundary()

	if b1 == b2 {
		t.Logf("b1=%q, b2=%q", b1, b2)
		t.Error("generateBoundary() should generate unique boundaries")
	}
}
