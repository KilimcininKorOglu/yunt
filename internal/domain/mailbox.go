package domain

import (
	"regexp"
	"strings"
)

// mailboxNameRegex validates mailbox names (alphanumeric, dots, underscores, hyphens).
var mailboxNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// Mailbox represents an email mailbox in the Yunt mail server.
// A mailbox is a container for messages and is associated with a user.
// Multiple mailboxes can be created per user for organization.
type Mailbox struct {
	// ID is the unique identifier for the mailbox.
	ID ID `json:"id"`

	// UserID is the ID of the user who owns this mailbox.
	UserID ID `json:"userId"`

	// Name is the display name of the mailbox (e.g., "Inbox", "Work", "Testing").
	Name string `json:"name"`

	// Address is the email address for this mailbox (e.g., "test@localhost").
	// For catch-all mailboxes, this may contain a wildcard pattern.
	Address string `json:"address"`

	// Description is an optional description of the mailbox purpose.
	Description string `json:"description,omitempty"`

	// IsCatchAll indicates if this mailbox catches all unmatched emails.
	IsCatchAll bool `json:"isCatchAll"`

	// IsDefault indicates if this is the user's default mailbox.
	IsDefault bool `json:"isDefault"`

	// MessageCount is the total number of messages in the mailbox.
	MessageCount int64 `json:"messageCount"`

	// UnreadCount is the number of unread messages in the mailbox.
	UnreadCount int64 `json:"unreadCount"`

	// TotalSize is the total size of all messages in bytes.
	TotalSize int64 `json:"totalSize"`

	// RetentionDays is the number of days to retain messages (0 = forever).
	RetentionDays int `json:"retentionDays"`

	// CreatedAt is the timestamp when the mailbox was created.
	CreatedAt Timestamp `json:"createdAt"`

	// UpdatedAt is the timestamp when the mailbox was last updated.
	UpdatedAt Timestamp `json:"updatedAt"`
}

// NewMailbox creates a new Mailbox with default values.
func NewMailbox(id, userID ID, name, address string) *Mailbox {
	now := Now()
	return &Mailbox{
		ID:            id,
		UserID:        userID,
		Name:          name,
		Address:       address,
		IsCatchAll:    false,
		IsDefault:     false,
		MessageCount:  0,
		UnreadCount:   0,
		TotalSize:     0,
		RetentionDays: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// Validate checks if the mailbox has valid field values.
func (m *Mailbox) Validate() error {
	errs := NewValidationErrors()

	// Validate ID
	if m.ID.IsEmpty() {
		errs.Add("id", "id is required")
	}

	// Validate UserID
	if m.UserID.IsEmpty() {
		errs.Add("userId", "user id is required")
	}

	// Validate Name
	if m.Name == "" {
		errs.Add("name", "name is required")
	} else if len(m.Name) > 100 {
		errs.Add("name", "name must be at most 100 characters")
	}

	// Validate Address
	if m.Address == "" {
		errs.Add("address", "address is required")
	} else if !m.IsCatchAll && !isValidEmail(m.Address) {
		errs.Add("address", "address must be a valid email format")
	}

	// Validate RetentionDays
	if m.RetentionDays < 0 {
		errs.Add("retentionDays", "retention days cannot be negative")
	}

	// Validate counts (should not be negative)
	if m.MessageCount < 0 {
		errs.Add("messageCount", "message count cannot be negative")
	}
	if m.UnreadCount < 0 {
		errs.Add("unreadCount", "unread count cannot be negative")
	}
	if m.TotalSize < 0 {
		errs.Add("totalSize", "total size cannot be negative")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// IncrementMessageCount increases the message and unread counts.
func (m *Mailbox) IncrementMessageCount(size int64) {
	m.MessageCount++
	m.UnreadCount++
	m.TotalSize += size
	m.UpdatedAt = Now()
}

// DecrementMessageCount decreases the message count.
func (m *Mailbox) DecrementMessageCount(size int64, wasUnread bool) {
	if m.MessageCount > 0 {
		m.MessageCount--
	}
	if wasUnread && m.UnreadCount > 0 {
		m.UnreadCount--
	}
	m.TotalSize -= size
	if m.TotalSize < 0 {
		m.TotalSize = 0
	}
	m.UpdatedAt = Now()
}

// MarkMessageRead decreases the unread count.
func (m *Mailbox) MarkMessageRead() {
	if m.UnreadCount > 0 {
		m.UnreadCount--
	}
	m.UpdatedAt = Now()
}

// MarkMessageUnread increases the unread count.
func (m *Mailbox) MarkMessageUnread() {
	if m.UnreadCount < m.MessageCount {
		m.UnreadCount++
	}
	m.UpdatedAt = Now()
}

// SetAsDefault marks this mailbox as the default.
func (m *Mailbox) SetAsDefault() {
	m.IsDefault = true
	m.UpdatedAt = Now()
}

// UnsetAsDefault removes the default flag.
func (m *Mailbox) UnsetAsDefault() {
	m.IsDefault = false
	m.UpdatedAt = Now()
}

// SetCatchAll marks this mailbox as a catch-all.
func (m *Mailbox) SetCatchAll() {
	m.IsCatchAll = true
	m.UpdatedAt = Now()
}

// UnsetCatchAll removes the catch-all flag.
func (m *Mailbox) UnsetCatchAll() {
	m.IsCatchAll = false
	m.UpdatedAt = Now()
}

// HasMessages returns true if the mailbox contains messages.
func (m *Mailbox) HasMessages() bool {
	return m.MessageCount > 0
}

// HasUnread returns true if the mailbox has unread messages.
func (m *Mailbox) HasUnread() bool {
	return m.UnreadCount > 0
}

// GetLocalPart returns the local part of the email address (before @).
func (m *Mailbox) GetLocalPart() string {
	parts := strings.Split(m.Address, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// GetDomain returns the domain part of the email address (after @).
func (m *Mailbox) GetDomain() string {
	parts := strings.Split(m.Address, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// MailboxCreateInput represents the input for creating a new mailbox.
type MailboxCreateInput struct {
	// Name is the display name of the mailbox.
	Name string `json:"name"`

	// Address is the email address for this mailbox.
	Address string `json:"address"`

	// Description is an optional description of the mailbox.
	Description string `json:"description,omitempty"`

	// IsCatchAll indicates if this mailbox should catch all unmatched emails.
	IsCatchAll bool `json:"isCatchAll,omitempty"`

	// IsDefault indicates if this should be the default mailbox.
	IsDefault bool `json:"isDefault,omitempty"`

	// RetentionDays is the number of days to retain messages (0 = forever).
	RetentionDays int `json:"retentionDays,omitempty"`
}

// Validate checks if the create input is valid.
func (i *MailboxCreateInput) Validate() error {
	errs := NewValidationErrors()

	// Validate Name
	name := strings.TrimSpace(i.Name)
	if name == "" {
		errs.Add("name", "name is required")
	} else if len(name) > 100 {
		errs.Add("name", "name must be at most 100 characters")
	}

	// Validate Address
	address := strings.TrimSpace(i.Address)
	if address == "" {
		errs.Add("address", "address is required")
	} else if !i.IsCatchAll && !isValidEmail(address) {
		errs.Add("address", "address must be a valid email format")
	}

	// Validate RetentionDays
	if i.RetentionDays < 0 {
		errs.Add("retentionDays", "retention days cannot be negative")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Normalize trims and normalizes the input fields.
func (i *MailboxCreateInput) Normalize() {
	i.Name = strings.TrimSpace(i.Name)
	i.Address = strings.TrimSpace(strings.ToLower(i.Address))
	i.Description = strings.TrimSpace(i.Description)
}

// MailboxUpdateInput represents the input for updating a mailbox.
type MailboxUpdateInput struct {
	// Name is the new display name (optional).
	Name *string `json:"name,omitempty"`

	// Description is the new description (optional).
	Description *string `json:"description,omitempty"`

	// IsDefault indicates if this should be the default mailbox (optional).
	IsDefault *bool `json:"isDefault,omitempty"`

	// RetentionDays is the new retention period (optional).
	RetentionDays *int `json:"retentionDays,omitempty"`
}

// Validate checks if the update input is valid.
func (i *MailboxUpdateInput) Validate() error {
	errs := NewValidationErrors()

	// Validate Name if provided
	if i.Name != nil {
		name := strings.TrimSpace(*i.Name)
		if name == "" {
			errs.Add("name", "name cannot be empty")
		} else if len(name) > 100 {
			errs.Add("name", "name must be at most 100 characters")
		}
	}

	// Validate RetentionDays if provided
	if i.RetentionDays != nil && *i.RetentionDays < 0 {
		errs.Add("retentionDays", "retention days cannot be negative")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Apply applies the update to the given mailbox.
func (i *MailboxUpdateInput) Apply(mailbox *Mailbox) {
	if i.Name != nil {
		mailbox.Name = strings.TrimSpace(*i.Name)
	}
	if i.Description != nil {
		mailbox.Description = strings.TrimSpace(*i.Description)
	}
	if i.IsDefault != nil {
		mailbox.IsDefault = *i.IsDefault
	}
	if i.RetentionDays != nil {
		mailbox.RetentionDays = *i.RetentionDays
	}
	mailbox.UpdatedAt = Now()
}

// MailboxFilter represents filtering options for listing mailboxes.
type MailboxFilter struct {
	// UserID filters by owner user ID.
	UserID *ID `json:"userId,omitempty"`

	// IsCatchAll filters by catch-all status.
	IsCatchAll *bool `json:"isCatchAll,omitempty"`

	// IsDefault filters by default status.
	IsDefault *bool `json:"isDefault,omitempty"`

	// Search is a text search on name and address.
	Search string `json:"search,omitempty"`
}

// MailboxStats represents statistics for a mailbox.
type MailboxStats struct {
	// TotalMessages is the total number of messages.
	TotalMessages int64 `json:"totalMessages"`

	// UnreadMessages is the number of unread messages.
	UnreadMessages int64 `json:"unreadMessages"`

	// TotalSize is the total size of all messages in bytes.
	TotalSize int64 `json:"totalSize"`

	// OldestMessage is the timestamp of the oldest message.
	OldestMessage *Timestamp `json:"oldestMessage,omitempty"`

	// NewestMessage is the timestamp of the newest message.
	NewestMessage *Timestamp `json:"newestMessage,omitempty"`
}
