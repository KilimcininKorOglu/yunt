package domain

import (
	"strings"
)

// Message represents an email message in the Yunt mail server.
// Messages are stored in mailboxes and contain the full email content
// including headers, body, and references to attachments.
type Message struct {
	// ID is the unique identifier for the message.
	ID ID `json:"id"`

	// MailboxID is the ID of the mailbox containing this message.
	MailboxID ID `json:"mailboxId"`

	// MessageID is the email Message-ID header value.
	// This is the unique identifier assigned by the sending mail server.
	MessageID string `json:"messageId,omitempty"`

	// From is the sender's email address.
	From EmailAddress `json:"from"`

	// To is the list of primary recipients.
	To []EmailAddress `json:"to"`

	// Cc is the list of carbon copy recipients.
	Cc []EmailAddress `json:"cc,omitempty"`

	// Bcc is the list of blind carbon copy recipients.
	// Note: BCC addresses are typically stripped before delivery.
	Bcc []EmailAddress `json:"bcc,omitempty"`

	// ReplyTo is the reply-to address if different from From.
	ReplyTo *EmailAddress `json:"replyTo,omitempty"`

	// Subject is the email subject line.
	Subject string `json:"subject"`

	// TextBody is the plain text version of the message body.
	TextBody string `json:"textBody,omitempty"`

	// HTMLBody is the HTML version of the message body.
	HTMLBody string `json:"htmlBody,omitempty"`

	// RawBody is the original raw message body before parsing.
	// This field is not serialized to JSON by default.
	RawBody []byte `json:"-"`

	// Headers contains all email headers as key-value pairs.
	// Multiple values for the same header are comma-separated.
	Headers map[string]string `json:"headers,omitempty"`

	// ContentType is the MIME content type of the message.
	ContentType ContentType `json:"contentType"`

	// Size is the total size of the message in bytes.
	Size int64 `json:"size"`

	// AttachmentCount is the number of attachments in this message.
	AttachmentCount int `json:"attachmentCount"`

	// Status indicates whether the message has been read.
	Status MessageStatus `json:"status"`

	// IsStarred indicates if the message is flagged/starred.
	IsStarred bool `json:"isStarred"`

	// IsSpam indicates if the message was marked as spam.
	IsSpam bool `json:"isSpam"`

	// InReplyTo is the Message-ID of the message this is replying to.
	InReplyTo string `json:"inReplyTo,omitempty"`

	// References is a list of Message-IDs in the conversation thread.
	References []string `json:"references,omitempty"`

	// ReceivedAt is the timestamp when the message was received by the server.
	ReceivedAt Timestamp `json:"receivedAt"`

	// SentAt is the timestamp from the Date header (when the message was sent).
	SentAt *Timestamp `json:"sentAt,omitempty"`

	// CreatedAt is the timestamp when the message was created in the database.
	CreatedAt Timestamp `json:"createdAt"`

	// UpdatedAt is the timestamp when the message was last updated.
	UpdatedAt Timestamp `json:"updatedAt"`
}

// NewMessage creates a new Message with default values.
func NewMessage(id, mailboxID ID) *Message {
	now := Now()
	return &Message{
		ID:              id,
		MailboxID:       mailboxID,
		To:              make([]EmailAddress, 0),
		Cc:              make([]EmailAddress, 0),
		Bcc:             make([]EmailAddress, 0),
		Headers:         make(map[string]string),
		References:      make([]string, 0),
		ContentType:     ContentTypePlain,
		Status:          MessageUnread,
		IsStarred:       false,
		IsSpam:          false,
		AttachmentCount: 0,
		ReceivedAt:      now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// Validate checks if the message has valid field values.
func (m *Message) Validate() error {
	errs := NewValidationErrors()

	// Validate ID
	if m.ID.IsEmpty() {
		errs.Add("id", "id is required")
	}

	// Validate MailboxID
	if m.MailboxID.IsEmpty() {
		errs.Add("mailboxId", "mailbox id is required")
	}

	// Validate From
	if m.From.IsEmpty() {
		errs.Add("from", "from address is required")
	}

	// Validate To (at least one recipient)
	if len(m.To) == 0 {
		errs.Add("to", "at least one recipient is required")
	}

	// Validate Status
	if !m.Status.IsValid() {
		errs.Add("status", "invalid status")
	}

	// Validate Size
	if m.Size < 0 {
		errs.Add("size", "size cannot be negative")
	}

	// Validate AttachmentCount
	if m.AttachmentCount < 0 {
		errs.Add("attachmentCount", "attachment count cannot be negative")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// MarkAsRead marks the message as read.
func (m *Message) MarkAsRead() bool {
	if m.Status == MessageUnread {
		m.Status = MessageRead
		m.UpdatedAt = Now()
		return true
	}
	return false
}

// MarkAsUnread marks the message as unread.
func (m *Message) MarkAsUnread() bool {
	if m.Status == MessageRead {
		m.Status = MessageUnread
		m.UpdatedAt = Now()
		return true
	}
	return false
}

// IsRead returns true if the message has been read.
func (m *Message) IsRead() bool {
	return m.Status == MessageRead
}

// ToggleStar toggles the starred status.
func (m *Message) ToggleStar() {
	m.IsStarred = !m.IsStarred
	m.UpdatedAt = Now()
}

// Star marks the message as starred.
func (m *Message) Star() {
	m.IsStarred = true
	m.UpdatedAt = Now()
}

// Unstar removes the starred flag.
func (m *Message) Unstar() {
	m.IsStarred = false
	m.UpdatedAt = Now()
}

// MarkAsSpam marks the message as spam.
func (m *Message) MarkAsSpam() {
	m.IsSpam = true
	m.UpdatedAt = Now()
}

// MarkAsNotSpam removes the spam flag.
func (m *Message) MarkAsNotSpam() {
	m.IsSpam = false
	m.UpdatedAt = Now()
}

// HasAttachments returns true if the message has attachments.
func (m *Message) HasAttachments() bool {
	return m.AttachmentCount > 0
}

// HasHTMLBody returns true if the message has an HTML body.
func (m *Message) HasHTMLBody() bool {
	return m.HTMLBody != ""
}

// HasTextBody returns true if the message has a plain text body.
func (m *Message) HasTextBody() bool {
	return m.TextBody != ""
}

// GetPreview returns a preview of the message body (first N characters).
func (m *Message) GetPreview(maxLength int) string {
	body := m.TextBody
	if body == "" {
		// Strip HTML tags for preview (simple approach)
		body = stripHTMLTags(m.HTMLBody)
	}

	// Normalize whitespace
	body = strings.Join(strings.Fields(body), " ")

	if len(body) <= maxLength {
		return body
	}
	return body[:maxLength] + "..."
}

// GetHeader returns a header value by name (case-insensitive).
func (m *Message) GetHeader(name string) string {
	// Try exact match first
	if val, ok := m.Headers[name]; ok {
		return val
	}
	// Try case-insensitive match
	nameLower := strings.ToLower(name)
	for key, val := range m.Headers {
		if strings.ToLower(key) == nameLower {
			return val
		}
	}
	return ""
}

// SetHeader sets a header value.
func (m *Message) SetHeader(name, value string) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[name] = value
}

// AddRecipient adds a recipient to the To list.
func (m *Message) AddRecipient(name, address string) {
	m.To = append(m.To, EmailAddress{Name: name, Address: address})
}

// AddCc adds a CC recipient.
func (m *Message) AddCc(name, address string) {
	m.Cc = append(m.Cc, EmailAddress{Name: name, Address: address})
}

// AddBcc adds a BCC recipient.
func (m *Message) AddBcc(name, address string) {
	m.Bcc = append(m.Bcc, EmailAddress{Name: name, Address: address})
}

// GetAllRecipients returns all recipients (To, Cc, Bcc).
func (m *Message) GetAllRecipients() []EmailAddress {
	recipients := make([]EmailAddress, 0, len(m.To)+len(m.Cc)+len(m.Bcc))
	recipients = append(recipients, m.To...)
	recipients = append(recipients, m.Cc...)
	recipients = append(recipients, m.Bcc...)
	return recipients
}

// stripHTMLTags removes HTML tags from a string (simple implementation).
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// MessageSummary represents a lightweight message summary for listings.
type MessageSummary struct {
	// ID is the unique identifier for the message.
	ID ID `json:"id"`

	// MailboxID is the ID of the mailbox containing this message.
	MailboxID ID `json:"mailboxId"`

	// From is the sender's email address.
	From EmailAddress `json:"from"`

	// Subject is the email subject line.
	Subject string `json:"subject"`

	// Preview is a short preview of the message body.
	Preview string `json:"preview"`

	// Status indicates whether the message has been read.
	Status MessageStatus `json:"status"`

	// IsStarred indicates if the message is flagged/starred.
	IsStarred bool `json:"isStarred"`

	// HasAttachments indicates if the message has attachments.
	HasAttachments bool `json:"hasAttachments"`

	// ReceivedAt is the timestamp when the message was received.
	ReceivedAt Timestamp `json:"receivedAt"`
}

// ToSummary converts a Message to a MessageSummary.
func (m *Message) ToSummary(previewLength int) *MessageSummary {
	return &MessageSummary{
		ID:             m.ID,
		MailboxID:      m.MailboxID,
		From:           m.From,
		Subject:        m.Subject,
		Preview:        m.GetPreview(previewLength),
		Status:         m.Status,
		IsStarred:      m.IsStarred,
		HasAttachments: m.HasAttachments(),
		ReceivedAt:     m.ReceivedAt,
	}
}

// MessageFilter represents filtering options for listing messages.
type MessageFilter struct {
	// MailboxID filters by mailbox ID.
	MailboxID *ID `json:"mailboxId,omitempty"`

	// Status filters by read/unread status.
	Status *MessageStatus `json:"status,omitempty"`

	// IsStarred filters by starred status.
	IsStarred *bool `json:"isStarred,omitempty"`

	// IsSpam filters by spam status.
	IsSpam *bool `json:"isSpam,omitempty"`

	// HasAttachments filters by attachment presence.
	HasAttachments *bool `json:"hasAttachments,omitempty"`

	// FromAddress filters by sender address.
	FromAddress string `json:"fromAddress,omitempty"`

	// ToAddress filters by recipient address.
	ToAddress string `json:"toAddress,omitempty"`

	// Subject filters by subject (partial match).
	Subject string `json:"subject,omitempty"`

	// Search is a full-text search on subject and body.
	Search string `json:"search,omitempty"`

	// ReceivedAfter filters messages received after this timestamp.
	ReceivedAfter *Timestamp `json:"receivedAfter,omitempty"`

	// ReceivedBefore filters messages received before this timestamp.
	ReceivedBefore *Timestamp `json:"receivedBefore,omitempty"`
}

// MessageSort represents sorting options for listing messages.
type MessageSort struct {
	// Field is the field to sort by.
	Field MessageSortField `json:"field"`

	// Order is the sort direction.
	Order SortOrder `json:"order"`
}

// MessageSortField represents the available fields for sorting messages.
type MessageSortField string

const (
	// MessageSortByReceivedAt sorts by received timestamp.
	MessageSortByReceivedAt MessageSortField = "receivedAt"
	// MessageSortBySubject sorts by subject.
	MessageSortBySubject MessageSortField = "subject"
	// MessageSortByFrom sorts by sender address.
	MessageSortByFrom MessageSortField = "from"
	// MessageSortBySize sorts by message size.
	MessageSortBySize MessageSortField = "size"
)

// IsValid returns true if the sort field is a recognized value.
func (f MessageSortField) IsValid() bool {
	switch f {
	case MessageSortByReceivedAt, MessageSortBySubject, MessageSortByFrom, MessageSortBySize:
		return true
	default:
		return false
	}
}
