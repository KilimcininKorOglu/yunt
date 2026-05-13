package parser

import (
	"strings"
	"testing"
	"time"

	"yunt/internal/domain"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    domain.EmailAddress
		isEmpty bool
	}{
		{
			name:  "simple address",
			input: "user@example.com",
			want:  domain.EmailAddress{Address: "user@example.com"},
		},
		{
			name:  "address with name",
			input: "John Doe <john@example.com>",
			want:  domain.EmailAddress{Name: "John Doe", Address: "john@example.com"},
		},
		{
			name:  "address with quoted name",
			input: "\"Doe, John\" <john@example.com>",
			want:  domain.EmailAddress{Name: "Doe, John", Address: "john@example.com"},
		},
		{
			name:  "address with angle brackets only",
			input: "<user@example.com>",
			want:  domain.EmailAddress{Address: "user@example.com"},
		},
		{
			name:  "address with extra whitespace",
			input: "  John Doe   <  john@example.com  >  ",
			want:  domain.EmailAddress{Name: "John Doe", Address: "john@example.com"},
		},
		{
			name:    "empty string",
			input:   "",
			want:    domain.EmailAddress{},
			isEmpty: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			want:    domain.EmailAddress{},
			isEmpty: true,
		},
		{
			name:  "complex quoted name with special chars",
			input: "\"John (CEO) Doe\" <john@example.com>",
			want:  domain.EmailAddress{Name: "John (CEO) Doe", Address: "john@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAddress(tt.input)
			if got.Name != tt.want.Name {
				t.Errorf("ParseAddress().Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Address != tt.want.Address {
				t.Errorf("ParseAddress().Address = %v, want %v", got.Address, tt.want.Address)
			}
			if got.IsEmpty() != tt.isEmpty {
				t.Errorf("ParseAddress().IsEmpty() = %v, want %v", got.IsEmpty(), tt.isEmpty)
			}
		})
	}
}

func TestParseAddressList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // expected count
	}{
		{
			name:  "single address",
			input: "user@example.com",
			want:  1,
		},
		{
			name:  "two addresses",
			input: "user1@example.com, user2@example.com",
			want:  2,
		},
		{
			name:  "multiple with names",
			input: "John <john@example.com>, Jane <jane@example.com>, Bob <bob@example.com>",
			want:  3,
		},
		{
			name:  "quoted names with commas",
			input: "\"Doe, John\" <john@example.com>, \"Smith, Jane\" <jane@example.com>",
			want:  2,
		},
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAddressList(tt.input)
			if len(got) != tt.want {
				t.Errorf("ParseAddressList() count = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name string
		addr domain.EmailAddress
		want string
	}{
		{
			name: "simple address",
			addr: domain.EmailAddress{Address: "user@example.com"},
			want: "user@example.com",
		},
		{
			name: "address with name",
			addr: domain.EmailAddress{Name: "John Doe", Address: "john@example.com"},
			want: "John Doe <john@example.com>",
		},
		{
			name: "name with special chars",
			addr: domain.EmailAddress{Name: "Doe, John", Address: "john@example.com"},
			want: "\"Doe, John\" <john@example.com>",
		},
		{
			name: "empty address",
			addr: domain.EmailAddress{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAddress(tt.addr)
			if got != tt.want {
				t.Errorf("FormatAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatAddressList(t *testing.T) {
	addresses := []domain.EmailAddress{
		{Name: "John", Address: "john@example.com"},
		{Address: "jane@example.com"},
	}
	got := FormatAddressList(addresses)
	if !strings.Contains(got, "john@example.com") || !strings.Contains(got, "jane@example.com") {
		t.Errorf("FormatAddressList() = %v, missing addresses", got)
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user@example.com", "example.com"},
		{"user@EXAMPLE.COM", "example.com"},
		{"user", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ExtractDomain(tt.input); got != tt.want {
				t.Errorf("ExtractDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractLocalPart(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user@example.com", "user"},
		{"john.doe@example.com", "john.doe"},
		{"user", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ExtractLocalPart(tt.input); got != tt.want {
				t.Errorf("ExtractLocalPart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidAddress(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"user@example.com", true},
		{"john.doe@example.com", true},
		{"user@sub.example.com", true},
		{"", false},
		{"user", false},
		{"@example.com", false},
		{"user@", false},
		{"user@@example.com", false},
		{"user@example", true}, // RFC 5321 §2.3.5: dotless domains are valid for local delivery
		{"user@.example.com", false},
		{"user@example.", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidAddress(tt.input); got != tt.want {
				t.Errorf("IsValidAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ParsePlainText(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Test Subject
Content-Type: text/plain; charset=utf-8

Hello, this is the message body.`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.From.Address != "sender@example.com" {
		t.Errorf("From = %v, want sender@example.com", msg.From.Address)
	}
	if len(msg.To) != 1 || msg.To[0].Address != "recipient@example.com" {
		t.Errorf("To = %v, want [recipient@example.com]", msg.To)
	}
	if msg.Subject != "Test Subject" {
		t.Errorf("Subject = %v, want Test Subject", msg.Subject)
	}
	if !strings.Contains(msg.TextBody, "Hello, this is the message body") {
		t.Errorf("TextBody = %v, should contain 'Hello, this is the message body'", msg.TextBody)
	}
}

func TestParser_ParseHTML(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: HTML Test
Content-Type: text/html; charset=utf-8

<html><body><h1>Hello</h1><p>This is HTML content.</p></body></html>`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.HTMLBody == "" {
		t.Error("HTMLBody should not be empty")
	}
	if !strings.Contains(msg.HTMLBody, "<h1>Hello</h1>") {
		t.Errorf("HTMLBody = %v, should contain '<h1>Hello</h1>'", msg.HTMLBody)
	}
}

func TestParser_ParseMultipartAlternative(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Multipart Test
Content-Type: multipart/alternative; boundary="boundary123"

--boundary123
Content-Type: text/plain; charset=utf-8

Plain text version.
--boundary123
Content-Type: text/html; charset=utf-8

<html><body><p>HTML version.</p></body></html>
--boundary123--`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.TextBody == "" {
		t.Error("TextBody should not be empty")
	}
	if msg.HTMLBody == "" {
		t.Error("HTMLBody should not be empty")
	}
	if !strings.Contains(msg.TextBody, "Plain text version") {
		t.Errorf("TextBody = %v, should contain 'Plain text version'", msg.TextBody)
	}
	if !strings.Contains(msg.HTMLBody, "HTML version") {
		t.Errorf("HTMLBody = %v, should contain 'HTML version'", msg.HTMLBody)
	}
}

func TestParser_ParseWithAttachment(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: With Attachment
Content-Type: multipart/mixed; boundary="boundary456"

--boundary456
Content-Type: text/plain; charset=utf-8

Message with attachment.
--boundary456
Content-Type: application/pdf
Content-Disposition: attachment; filename="document.pdf"
Content-Transfer-Encoding: base64

SGVsbG8gV29ybGQh
--boundary456--`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(msg.Attachments) != 1 {
		t.Fatalf("Attachments count = %v, want 1", len(msg.Attachments))
	}

	att := msg.Attachments[0]
	if att.Filename != "document.pdf" {
		t.Errorf("Attachment.Filename = %v, want document.pdf", att.Filename)
	}
	if att.ContentType != "application/pdf" {
		t.Errorf("Attachment.ContentType = %v, want application/pdf", att.ContentType)
	}
	if att.IsInline {
		t.Error("Attachment.IsInline should be false")
	}
	// "Hello World!" base64 decoded
	expected := "Hello World!"
	if string(att.Data) != expected {
		t.Errorf("Attachment.Data = %v, want %v", string(att.Data), expected)
	}
}

func TestParser_ParseInlineImage(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: With Inline Image
Content-Type: multipart/related; boundary="boundary789"

--boundary789
Content-Type: text/html; charset=utf-8

<html><body><img src="cid:image123"></body></html>
--boundary789
Content-Type: image/png
Content-ID: <image123>
Content-Transfer-Encoding: base64
Content-Disposition: inline; filename="logo.png"

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==
--boundary789--`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.HTMLBody == "" {
		t.Error("HTMLBody should not be empty")
	}

	if len(msg.Attachments) != 1 {
		t.Fatalf("Attachments count = %v, want 1", len(msg.Attachments))
	}

	att := msg.Attachments[0]
	if att.ContentID != "image123" {
		t.Errorf("Attachment.ContentID = %v, want image123", att.ContentID)
	}
	if !att.IsInline {
		t.Error("Attachment.IsInline should be true")
	}
	if att.Disposition != domain.DispositionInline {
		t.Errorf("Attachment.Disposition = %v, want inline", att.Disposition)
	}
}

func TestParser_ParseQuotedPrintable(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Quoted-Printable Test
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

Hello=20World!
This=20is=20a=20test=20message=20with=20special=20chars:=20=C3=A9=C3=A0=C3=BC`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !strings.Contains(msg.TextBody, "Hello World!") {
		t.Errorf("TextBody = %v, should contain 'Hello World!'", msg.TextBody)
	}
}

func TestParser_ParseBase64Body(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Base64 Test
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: base64

SGVsbG8gV29ybGQhIFRoaXMgaXMgYSBiYXNlNjQgZW5jb2RlZCBtZXNzYWdlLg==`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !strings.Contains(msg.TextBody, "Hello World!") {
		t.Errorf("TextBody = %v, should contain 'Hello World!'", msg.TextBody)
	}
}

func TestParser_ParseEncodedSubject(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: =?UTF-8?B?SGVsbG8gV29ybGQh?=
Content-Type: text/plain

Test`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.Subject != "Hello World!" {
		t.Errorf("Subject = %v, want 'Hello World!'", msg.Subject)
	}
}

func TestParser_ParseEncodedSubjectQuotedPrintable(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: =?UTF-8?Q?Hello_World!?=
Content-Type: text/plain

Test`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.Subject != "Hello World!" {
		t.Errorf("Subject = %v, want 'Hello World!'", msg.Subject)
	}
}

func TestParser_ParseMultipleRecipients(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient1@example.com, recipient2@example.com
Cc: cc1@example.com, cc2@example.com
Bcc: bcc@example.com
Subject: Multiple Recipients
Content-Type: text/plain

Test`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(msg.To) != 2 {
		t.Errorf("To count = %v, want 2", len(msg.To))
	}
	if len(msg.Cc) != 2 {
		t.Errorf("Cc count = %v, want 2", len(msg.Cc))
	}
	if len(msg.Bcc) != 1 {
		t.Errorf("Bcc count = %v, want 1", len(msg.Bcc))
	}
}

func TestParser_ParseReplyTo(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Reply-To: reply@example.com
Subject: With Reply-To
Content-Type: text/plain

Test`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.ReplyTo == nil {
		t.Fatal("ReplyTo should not be nil")
	}
	if msg.ReplyTo.Address != "reply@example.com" {
		t.Errorf("ReplyTo.Address = %v, want reply@example.com", msg.ReplyTo.Address)
	}
}

func TestParser_ParseMessageID(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Message-ID: <unique123@example.com>
Subject: With Message-ID
Content-Type: text/plain

Test`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.MessageID != "unique123@example.com" {
		t.Errorf("MessageID = %v, want unique123@example.com", msg.MessageID)
	}
}

func TestParser_ParseReferences(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
In-Reply-To: <original@example.com>
References: <thread1@example.com> <thread2@example.com>
Subject: Reply
Content-Type: text/plain

Test`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.InReplyTo != "original@example.com" {
		t.Errorf("InReplyTo = %v, want original@example.com", msg.InReplyTo)
	}
	if len(msg.References) != 2 {
		t.Errorf("References count = %v, want 2", len(msg.References))
	}
}

func TestParser_ParseDate(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Date: Mon, 02 Jan 2006 15:04:05 -0700
Subject: With Date
Content-Type: text/plain

Test`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.Date == nil {
		t.Fatal("Date should not be nil")
	}
	if msg.Date.Year() != 2006 {
		t.Errorf("Date.Year() = %v, want 2006", msg.Date.Year())
	}
}

func TestParser_EmptyMessage(t *testing.T) {
	p := NewParser()
	_, err := p.Parse([]byte{})
	if err == nil {
		t.Error("Parse() should return error for empty message")
	}
}

func TestParser_MalformedMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "no headers",
			input: "Just some text without headers",
		},
		{
			name:  "multipart without boundary",
			input: "Content-Type: multipart/mixed\n\nBody",
		},
	}

	p := NewParser()
	p.StrictMode = false // Non-strict mode should not crash

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse([]byte(tt.input))
			// Should not crash even with malformed input
			_ = err
		})
	}
}

func TestParser_MaxMessageSize(t *testing.T) {
	p := NewParser()
	p.MaxMessageSize = 100 // Very small limit

	largeMessage := `From: sender@example.com
To: recipient@example.com
Subject: Large Message

` + strings.Repeat("x", 200)

	_, err := p.Parse([]byte(largeMessage))
	if err == nil {
		t.Error("Parse() should return error for oversized message")
	}
}

func TestParser_StrictMode(t *testing.T) {
	p := NewParser()
	p.StrictMode = true

	// Multipart without boundary
	malformed := `From: sender@example.com
To: recipient@example.com
Content-Type: multipart/mixed

Body without boundary`

	_, err := p.Parse([]byte(malformed))
	if err == nil {
		t.Error("Parse() in strict mode should return error for malformed multipart")
	}
}

func TestParsedMessage_ToMessage(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Test
Date: Mon, 02 Jan 2006 15:04:05 -0700
Content-Type: text/plain

Body text`

	p := NewParser()
	parsed, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	msg := parsed.ToMessage(domain.ID("msg1"), domain.ID("mb1"))

	if msg.ID != domain.ID("msg1") {
		t.Errorf("ToMessage().ID = %v, want msg1", msg.ID)
	}
	if msg.MailboxID != domain.ID("mb1") {
		t.Errorf("ToMessage().MailboxID = %v, want mb1", msg.MailboxID)
	}
	if msg.Subject != "Test" {
		t.Errorf("ToMessage().Subject = %v, want Test", msg.Subject)
	}
	if msg.ContentType != domain.ContentTypePlain {
		t.Errorf("ToMessage().ContentType = %v, want text/plain", msg.ContentType)
	}
}

func TestParsedMessage_GetInlineAttachments(t *testing.T) {
	pm := &ParsedMessage{
		Attachments: []*AttachmentData{
			{Filename: "doc.pdf", IsInline: false},
			{Filename: "image.png", IsInline: true},
			{Filename: "photo.jpg", IsInline: true},
		},
	}

	inline := pm.GetInlineAttachments()
	if len(inline) != 2 {
		t.Errorf("GetInlineAttachments() count = %v, want 2", len(inline))
	}
}

func TestParsedMessage_GetRegularAttachments(t *testing.T) {
	pm := &ParsedMessage{
		Attachments: []*AttachmentData{
			{Filename: "doc.pdf", IsInline: false},
			{Filename: "image.png", IsInline: true},
			{Filename: "report.xlsx", IsInline: false},
		},
	}

	regular := pm.GetRegularAttachments()
	if len(regular) != 2 {
		t.Errorf("GetRegularAttachments() count = %v, want 2", len(regular))
	}
}

func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple",
			input: "SGVsbG8gV29ybGQh",
			want:  "Hello World!",
		},
		{
			name:  "with line breaks",
			input: "SGVs\nbG8g\nV29y\nbGQh",
			want:  "Hello World!",
		},
		{
			name:  "with carriage returns",
			input: "SGVs\r\nbG8g\r\nV29y\r\nbGQh",
			want:  "Hello World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeBase64([]byte(tt.input))
			if string(got) != tt.want {
				t.Errorf("decodeBase64() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestDecodeQuotedPrintable(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple",
			input: "Hello=20World!",
			want:  "Hello World!",
		},
		{
			name:  "with soft line break",
			input: "Hello=\r\nWorld",
			want:  "HelloWorld",
		},
		{
			name:  "hex encoded",
			input: "=48=65=6C=6C=6F",
			want:  "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeQuotedPrintable([]byte(tt.input))
			if string(got) != tt.want {
				t.Errorf("decodeQuotedPrintable() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestCanonicalHeaderKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"content-type", "Content-Type"},
		{"CONTENT-TYPE", "Content-Type"},
		{"Content-Type", "Content-Type"},
		{"x-custom-header", "X-Custom-Header"},
		{"message-id", "Message-Id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := canonicalHeaderKey(tt.input); got != tt.want {
				t.Errorf("canonicalHeaderKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMediaType(t *testing.T) {
	tests := []struct {
		input     string
		wantType  string
		wantParam string
	}{
		{
			input:     "text/plain",
			wantType:  "text/plain",
			wantParam: "",
		},
		{
			input:     "text/plain; charset=utf-8",
			wantType:  "text/plain",
			wantParam: "utf-8",
		},
		{
			input:     "multipart/mixed; boundary=\"----boundary\"",
			wantType:  "multipart/mixed",
			wantParam: "----boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mediaType, params := parseMediaType(tt.input)
			if mediaType != tt.wantType {
				t.Errorf("parseMediaType() mediaType = %v, want %v", mediaType, tt.wantType)
			}
			if tt.wantParam != "" {
				charset := params["charset"]
				boundary := params["boundary"]
				if charset != tt.wantParam && boundary != tt.wantParam {
					t.Errorf("parseMediaType() param = %v, want %v", params, tt.wantParam)
				}
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "RFC1123Z",
			input:   "Mon, 02 Jan 2006 15:04:05 -0700",
			wantErr: false,
		},
		{
			name:    "without day name",
			input:   "02 Jan 2006 15:04:05 -0700",
			wantErr: false,
		},
		{
			name:    "with timezone name",
			input:   "02 Jan 2006 15:04:05 MST",
			wantErr: false,
		},
		{
			name:    "with parenthetical timezone",
			input:   "02 Jan 2006 15:04:05 -0700 (MST)",
			wantErr: false,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"document.pdf", "document.pdf"},
		{"../../../etc/passwd", "______etc_passwd"},
		{".hidden", "hidden"},
		{"path/to/file.txt", "path_to_file.txt"},
		{"", "attachment"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeFilename(tt.input); got != tt.want {
				t.Errorf("sanitizeFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeHeaderValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "base64 encoded",
			input: "=?UTF-8?B?SGVsbG8gV29ybGQ=?=",
			want:  "Hello World",
		},
		{
			name:  "quoted-printable encoded",
			input: "=?UTF-8?Q?Hello_World?=",
			want:  "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeHeaderValue(tt.input); got != tt.want {
				t.Errorf("decodeHeaderValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ParseNestedMultipart(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Nested Multipart
Content-Type: multipart/mixed; boundary="outer"

--outer
Content-Type: multipart/alternative; boundary="inner"

--inner
Content-Type: text/plain

Plain text content
--inner
Content-Type: text/html

<p>HTML content</p>
--inner--
--outer
Content-Type: application/pdf
Content-Disposition: attachment; filename="doc.pdf"
Content-Transfer-Encoding: base64

SGVsbG8=
--outer--`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.TextBody == "" {
		t.Error("TextBody should not be empty")
	}
	if msg.HTMLBody == "" {
		t.Error("HTMLBody should not be empty")
	}
	if len(msg.Attachments) != 1 {
		t.Errorf("Attachments count = %v, want 1", len(msg.Attachments))
	}
}

func TestParser_ParseCRLFLineEndings(t *testing.T) {
	rawEmail := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: CRLF Test\r\n\r\nBody with CRLF"

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if msg.From.Address != "sender@example.com" {
		t.Errorf("From = %v, want sender@example.com", msg.From.Address)
	}
}

func TestParser_ParseFoldedHeaders(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: This is a very long subject line
 that continues on the next line
 and keeps going
Content-Type: text/plain

Body`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !strings.Contains(msg.Subject, "continues on the next line") {
		t.Errorf("Subject = %v, should contain folded content", msg.Subject)
	}
}

func TestExtractMessageIDs(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"<id1@example.com>", 1},
		{"<id1@example.com> <id2@example.com>", 2},
		{"<id1@example.com> <id2@example.com> <id3@example.com>", 3},
		{"", 0},
		{"malformed", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractMessageIDs(tt.input)
			if len(got) != tt.want {
				t.Errorf("extractMessageIDs() count = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user@EXAMPLE.COM", "user@example.com"},
		{"User@Example.Com", "User@example.com"},
		{"user", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeAddress(tt.input); got != tt.want {
				t.Errorf("NormalizeAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	err := NewParseError("test error")
	if err.Error() != "mime parse error: test error" {
		t.Errorf("ParseError.Error() = %v, want 'mime parse error: test error'", err.Error())
	}
}

func TestParser_ParseISO8859(t *testing.T) {
	// ISO-8859-1 encoded content
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: ISO-8859-1 Test
Content-Type: text/plain; charset=iso-8859-1

Hello World`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !strings.Contains(msg.TextBody, "Hello World") {
		t.Errorf("TextBody = %v, should contain 'Hello World'", msg.TextBody)
	}
}

func TestAttachmentData_Properties(t *testing.T) {
	att := &AttachmentData{
		Filename:    "test.pdf",
		ContentType: "application/pdf",
		ContentID:   "content123",
		Disposition: domain.DispositionAttachment,
		IsInline:    false,
		Data:        []byte("test data"),
		Encoding:    "base64",
	}

	if att.Filename != "test.pdf" {
		t.Errorf("Filename = %v, want test.pdf", att.Filename)
	}
	if att.ContentType != "application/pdf" {
		t.Errorf("ContentType = %v, want application/pdf", att.ContentType)
	}
	if len(att.Data) != 9 {
		t.Errorf("Data length = %v, want 9", len(att.Data))
	}
}

func TestParser_ParseContentIDWithoutDisposition(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Content-ID Without Disposition
Content-Type: multipart/related; boundary="boundary"

--boundary
Content-Type: text/html

<html><body><img src="cid:image1"></body></html>
--boundary
Content-Type: image/png
Content-ID: <image1>
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==
--boundary--`

	p := NewParser()
	msg, err := p.Parse([]byte(rawEmail))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(msg.Attachments) != 1 {
		t.Fatalf("Attachments count = %v, want 1", len(msg.Attachments))
	}

	// Should be treated as inline because of Content-ID
	if !msg.Attachments[0].IsInline {
		t.Error("Attachment with Content-ID should be treated as inline")
	}
}

// Benchmark tests
func BenchmarkParser_ParseSimple(b *testing.B) {
	rawEmail := []byte(`From: sender@example.com
To: recipient@example.com
Subject: Simple Test
Content-Type: text/plain

This is a simple test message.`)

	p := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = p.Parse(rawEmail)
	}
}

func BenchmarkParser_ParseMultipart(b *testing.B) {
	rawEmail := []byte(`From: sender@example.com
To: recipient@example.com
Subject: Multipart Test
Content-Type: multipart/alternative; boundary="boundary"

--boundary
Content-Type: text/plain

Plain text version.
--boundary
Content-Type: text/html

<html><body><p>HTML version.</p></body></html>
--boundary--`)

	p := NewParser()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = p.Parse(rawEmail)
	}
}

func BenchmarkParseAddress(b *testing.B) {
	input := "John Doe <john.doe@example.com>"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ParseAddress(input)
	}
}

func BenchmarkParseAddressList(b *testing.B) {
	input := "John <john@example.com>, Jane <jane@example.com>, Bob <bob@example.com>"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ParseAddressList(input)
	}
}

// Test for time package usage check
func TestTimePackageUsed(t *testing.T) {
	// Ensure time.Time is usable (this just verifies import works)
	now := time.Now()
	if now.IsZero() {
		t.Error("time.Now() should not be zero")
	}
}
