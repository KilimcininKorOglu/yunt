package domain

import (
	"strings"
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	if msg.ID != ID("msg1") {
		t.Errorf("NewMessage().ID = %v, want %v", msg.ID, "msg1")
	}
	if msg.MailboxID != ID("mb1") {
		t.Errorf("NewMessage().MailboxID = %v, want %v", msg.MailboxID, "mb1")
	}
	if msg.Status != MessageUnread {
		t.Errorf("NewMessage().Status = %v, want %v", msg.Status, MessageUnread)
	}
	if msg.IsStarred {
		t.Error("NewMessage().IsStarred should be false")
	}
	if msg.IsSpam {
		t.Error("NewMessage().IsSpam should be false")
	}
	if len(msg.To) != 0 {
		t.Error("NewMessage().To should be empty")
	}
	if len(msg.Headers) != 0 {
		t.Error("NewMessage().Headers should be empty")
	}
}

func TestMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid message",
			message: &Message{
				ID:        ID("msg1"),
				MailboxID: ID("mb1"),
				From:      EmailAddress{Address: "sender@example.com"},
				To:        []EmailAddress{{Address: "recipient@example.com"}},
				Status:    MessageUnread,
			},
			wantErr: false,
		},
		{
			name: "missing id",
			message: &Message{
				MailboxID: ID("mb1"),
				From:      EmailAddress{Address: "sender@example.com"},
				To:        []EmailAddress{{Address: "recipient@example.com"}},
				Status:    MessageUnread,
			},
			wantErr: true,
			errMsgs: []string{"id"},
		},
		{
			name: "missing mailbox id",
			message: &Message{
				ID:     ID("msg1"),
				From:   EmailAddress{Address: "sender@example.com"},
				To:     []EmailAddress{{Address: "recipient@example.com"}},
				Status: MessageUnread,
			},
			wantErr: true,
			errMsgs: []string{"mailboxId"},
		},
		{
			name: "missing from",
			message: &Message{
				ID:        ID("msg1"),
				MailboxID: ID("mb1"),
				To:        []EmailAddress{{Address: "recipient@example.com"}},
				Status:    MessageUnread,
			},
			wantErr: true,
			errMsgs: []string{"from"},
		},
		{
			name: "missing recipients",
			message: &Message{
				ID:        ID("msg1"),
				MailboxID: ID("mb1"),
				From:      EmailAddress{Address: "sender@example.com"},
				To:        []EmailAddress{},
				Status:    MessageUnread,
			},
			wantErr: true,
			errMsgs: []string{"to"},
		},
		{
			name: "invalid status",
			message: &Message{
				ID:        ID("msg1"),
				MailboxID: ID("mb1"),
				From:      EmailAddress{Address: "sender@example.com"},
				To:        []EmailAddress{{Address: "recipient@example.com"}},
				Status:    MessageStatus("invalid"),
			},
			wantErr: true,
			errMsgs: []string{"status"},
		},
		{
			name: "negative size",
			message: &Message{
				ID:        ID("msg1"),
				MailboxID: ID("mb1"),
				From:      EmailAddress{Address: "sender@example.com"},
				To:        []EmailAddress{{Address: "recipient@example.com"}},
				Status:    MessageUnread,
				Size:      -1,
			},
			wantErr: true,
			errMsgs: []string{"size"},
		},
		{
			name: "negative attachment count",
			message: &Message{
				ID:              ID("msg1"),
				MailboxID:       ID("mb1"),
				From:            EmailAddress{Address: "sender@example.com"},
				To:              []EmailAddress{{Address: "recipient@example.com"}},
				Status:          MessageUnread,
				AttachmentCount: -1,
			},
			wantErr: true,
			errMsgs: []string{"attachmentCount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				errStr := err.Error()
				for _, msg := range tt.errMsgs {
					if !strings.Contains(errStr, msg) {
						t.Errorf("Message.Validate() error should contain '%s', got %v", msg, errStr)
					}
				}
			}
		})
	}
}

func TestMessage_MarkAsRead(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	changed := msg.MarkAsRead()
	if !changed {
		t.Error("MarkAsRead() should return true for unread message")
	}
	if msg.Status != MessageRead {
		t.Errorf("MarkAsRead() Status = %v, want %v", msg.Status, MessageRead)
	}

	// Already read
	changed = msg.MarkAsRead()
	if changed {
		t.Error("MarkAsRead() should return false for already read message")
	}
}

func TestMessage_MarkAsUnread(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))
	msg.Status = MessageRead

	changed := msg.MarkAsUnread()
	if !changed {
		t.Error("MarkAsUnread() should return true for read message")
	}
	if msg.Status != MessageUnread {
		t.Errorf("MarkAsUnread() Status = %v, want %v", msg.Status, MessageUnread)
	}

	// Already unread
	changed = msg.MarkAsUnread()
	if changed {
		t.Error("MarkAsUnread() should return false for already unread message")
	}
}

func TestMessage_IsRead(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	if msg.IsRead() {
		t.Error("IsRead() should return false for new message")
	}

	msg.Status = MessageRead
	if !msg.IsRead() {
		t.Error("IsRead() should return true for read message")
	}
}

func TestMessage_Star(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	msg.Star()
	if !msg.IsStarred {
		t.Error("Star() should set IsStarred to true")
	}

	msg.Unstar()
	if msg.IsStarred {
		t.Error("Unstar() should set IsStarred to false")
	}

	msg.ToggleStar()
	if !msg.IsStarred {
		t.Error("ToggleStar() should toggle IsStarred to true")
	}

	msg.ToggleStar()
	if msg.IsStarred {
		t.Error("ToggleStar() should toggle IsStarred to false")
	}
}

func TestMessage_Spam(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	msg.MarkAsSpam()
	if !msg.IsSpam {
		t.Error("MarkAsSpam() should set IsSpam to true")
	}

	msg.MarkAsNotSpam()
	if msg.IsSpam {
		t.Error("MarkAsNotSpam() should set IsSpam to false")
	}
}

func TestMessage_HasAttachments(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	if msg.HasAttachments() {
		t.Error("HasAttachments() should return false for message with no attachments")
	}

	msg.AttachmentCount = 1
	if !msg.HasAttachments() {
		t.Error("HasAttachments() should return true for message with attachments")
	}
}

func TestMessage_HasBody(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	if msg.HasHTMLBody() {
		t.Error("HasHTMLBody() should return false for new message")
	}
	if msg.HasTextBody() {
		t.Error("HasTextBody() should return false for new message")
	}

	msg.HTMLBody = "<p>Hello</p>"
	if !msg.HasHTMLBody() {
		t.Error("HasHTMLBody() should return true when HTML body is set")
	}

	msg.TextBody = "Hello"
	if !msg.HasTextBody() {
		t.Error("HasTextBody() should return true when text body is set")
	}
}

func TestMessage_GetPreview(t *testing.T) {
	tests := []struct {
		name      string
		textBody  string
		htmlBody  string
		maxLength int
		want      string
	}{
		{
			name:      "text body short",
			textBody:  "Hello world",
			maxLength: 50,
			want:      "Hello world",
		},
		{
			name:      "text body truncated",
			textBody:  "This is a very long message that should be truncated",
			maxLength: 20,
			want:      "This is a very long ...",
		},
		{
			name:      "html body stripped",
			htmlBody:  "<p>Hello <strong>world</strong></p>",
			maxLength: 50,
			want:      "Hello world",
		},
		{
			name:      "whitespace normalized",
			textBody:  "Hello   world\n\ntest",
			maxLength: 50,
			want:      "Hello world test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{TextBody: tt.textBody, HTMLBody: tt.htmlBody}
			if got := msg.GetPreview(tt.maxLength); got != tt.want {
				t.Errorf("GetPreview() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_GetHeader(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))
	msg.Headers = map[string]string{
		"Content-Type": "text/plain",
		"X-Custom":     "value",
	}

	tests := []struct {
		name string
		want string
	}{
		{"Content-Type", "text/plain"},
		{"content-type", "text/plain"}, // case insensitive
		{"X-Custom", "value"},
		{"Non-Existent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := msg.GetHeader(tt.name); got != tt.want {
				t.Errorf("GetHeader(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestMessage_SetHeader(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	msg.SetHeader("Content-Type", "text/html")

	if got := msg.GetHeader("Content-Type"); got != "text/html" {
		t.Errorf("SetHeader() then GetHeader() = %v, want %v", got, "text/html")
	}
}

func TestMessage_AddRecipient(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))

	msg.AddRecipient("John", "john@example.com")
	msg.AddCc("Jane", "jane@example.com")
	msg.AddBcc("Secret", "secret@example.com")

	if len(msg.To) != 1 {
		t.Errorf("AddRecipient() To length = %v, want 1", len(msg.To))
	}
	if len(msg.Cc) != 1 {
		t.Errorf("AddCc() Cc length = %v, want 1", len(msg.Cc))
	}
	if len(msg.Bcc) != 1 {
		t.Errorf("AddBcc() Bcc length = %v, want 1", len(msg.Bcc))
	}
}

func TestMessage_GetAllRecipients(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))
	msg.To = []EmailAddress{{Address: "to@example.com"}}
	msg.Cc = []EmailAddress{{Address: "cc@example.com"}}
	msg.Bcc = []EmailAddress{{Address: "bcc@example.com"}}

	recipients := msg.GetAllRecipients()

	if len(recipients) != 3 {
		t.Errorf("GetAllRecipients() length = %v, want 3", len(recipients))
	}
}

func TestMessage_ToSummary(t *testing.T) {
	msg := NewMessage(ID("msg1"), ID("mb1"))
	msg.From = EmailAddress{Name: "Sender", Address: "sender@example.com"}
	msg.Subject = "Test Subject"
	msg.TextBody = "This is the message body"
	msg.AttachmentCount = 2

	summary := msg.ToSummary(20)

	if summary.ID != msg.ID {
		t.Errorf("ToSummary().ID = %v, want %v", summary.ID, msg.ID)
	}
	if summary.MailboxID != msg.MailboxID {
		t.Errorf("ToSummary().MailboxID = %v, want %v", summary.MailboxID, msg.MailboxID)
	}
	if summary.From.Address != msg.From.Address {
		t.Errorf("ToSummary().From = %v, want %v", summary.From, msg.From)
	}
	if summary.Subject != msg.Subject {
		t.Errorf("ToSummary().Subject = %v, want %v", summary.Subject, msg.Subject)
	}
	if !summary.HasAttachments {
		t.Error("ToSummary().HasAttachments should be true")
	}
}

func TestMessageSortField_IsValid(t *testing.T) {
	tests := []struct {
		field MessageSortField
		want  bool
	}{
		{MessageSortByReceivedAt, true},
		{MessageSortBySubject, true},
		{MessageSortByFrom, true},
		{MessageSortBySize, true},
		{MessageSortField("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.field), func(t *testing.T) {
			if got := tt.field.IsValid(); got != tt.want {
				t.Errorf("MessageSortField.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<div><span>Test</span></div>", "Test"},
		{"No tags here", "No tags here"},
		{"<a href='#'>Link</a>", "Link"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := stripHTMLTags(tt.input); got != tt.want {
				t.Errorf("stripHTMLTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
