package parser

import (
	"strings"
	"testing"
)

func TestBuildRawMessage_PlainText(t *testing.T) {
	opts := BuildMessageOpts{
		From:     "sender@example.com",
		FromName: "Test Sender",
		To:       []string{"recipient@example.com"},
		Subject:  "Test Subject",
		TextBody: "Hello, World!",
		Domain:   "example.com",
	}

	forSent, forRelay := BuildRawMessage(opts)

	p := NewParser()
	parsed, err := p.Parse(forRelay)
	if err != nil {
		t.Fatalf("failed to parse built message: %v", err)
	}
	if parsed.Subject != "Test Subject" {
		t.Errorf("subject = %q, want %q", parsed.Subject, "Test Subject")
	}
	if parsed.From.Address != "sender@example.com" {
		t.Errorf("from = %q, want sender@example.com", parsed.From.Address)
	}
	if len(parsed.To) == 0 || parsed.To[0].Address != "recipient@example.com" {
		t.Errorf("to = %v, want [recipient@example.com]", parsed.To)
	}
	if !strings.Contains(parsed.TextBody, "Hello, World!") {
		t.Errorf("text body = %q, want to contain 'Hello, World!'", parsed.TextBody)
	}
	if len(forSent) == 0 {
		t.Error("forSent should not be empty")
	}
}

func TestBuildRawMessage_TurkishSubject(t *testing.T) {
	opts := BuildMessageOpts{
		From:     "gonder@test.com",
		FromName: "Gönderici Şükrü",
		To:       []string{"alici@test.com"},
		Subject:  "Merhaba Dünya — Türkçe ğüşöçİ",
		TextBody: "Bu bir test mesajıdır.",
		Domain:   "test.com",
	}

	_, forRelay := BuildRawMessage(opts)

	p := NewParser()
	parsed, err := p.Parse(forRelay)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if !strings.Contains(parsed.Subject, "Merhaba") {
		t.Errorf("subject = %q, want Turkish content", parsed.Subject)
	}
	if parsed.From.Name == "" {
		t.Error("from name should not be empty")
	}
}

func TestBuildRawMessage_HTMLAndText(t *testing.T) {
	opts := BuildMessageOpts{
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Subject:  "HTML Test",
		TextBody: "Plain text version",
		HTMLBody: "<html><body><h1>Hello</h1></body></html>",
		Domain:   "example.com",
	}

	_, forRelay := BuildRawMessage(opts)

	p := NewParser()
	parsed, err := p.Parse(forRelay)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if parsed.TextBody == "" {
		t.Error("text body should not be empty")
	}
	if parsed.HTMLBody == "" {
		t.Error("html body should not be empty")
	}
}

func TestBuildRawMessage_WithAttachments(t *testing.T) {
	opts := BuildMessageOpts{
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Subject:  "With Attachment",
		TextBody: "See attached.",
		Attachments: []AttachmentInput{
			{Filename: "report.txt", ContentType: "application/octet-stream", Data: []byte("report content here")},
			{Filename: "data.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.4 fake")},
		},
		Domain: "example.com",
	}

	_, forRelay := BuildRawMessage(opts)

	p := NewParser()
	parsed, err := p.Parse(forRelay)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(parsed.Attachments) != 2 {
		t.Errorf("attachments = %d, want 2", len(parsed.Attachments))
	}
	if parsed.TextBody == "" {
		t.Error("text body should not be empty")
	}
}

func TestBuildRawMessage_BCC(t *testing.T) {
	opts := BuildMessageOpts{
		From:     "sender@example.com",
		To:       []string{"to@example.com"},
		Bcc:      []string{"secret@example.com"},
		Subject:  "BCC Test",
		TextBody: "Secret copy.",
		Domain:   "example.com",
	}

	forSent, forRelay := BuildRawMessage(opts)

	// forRelay should NOT contain BCC
	if strings.Contains(string(forRelay), "secret@example.com") {
		t.Error("forRelay should not contain BCC address")
	}

	// forSent should contain BCC
	if !strings.Contains(string(forSent), "secret@example.com") {
		t.Error("forSent should contain BCC address")
	}
}

func TestBuildRawMessage_RequiredHeaders(t *testing.T) {
	opts := BuildMessageOpts{
		From:     "sender@example.com",
		To:       []string{"to@example.com"},
		Subject:  "Headers Test",
		TextBody: "Body.",
		Domain:   "example.com",
	}

	_, forRelay := BuildRawMessage(opts)
	raw := string(forRelay)

	required := []string{"Date:", "From:", "To:", "Subject:", "Message-ID:", "MIME-Version:"}
	for _, h := range required {
		if !strings.Contains(raw, h) {
			t.Errorf("missing required header: %s", h)
		}
	}
}
