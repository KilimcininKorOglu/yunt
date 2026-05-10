// Package domain contains the core domain models and entities for the Yunt mail server.
// These models represent the fundamental concepts and data structures used throughout
// the application, including users, mailboxes, messages, attachments, webhooks, and settings.
package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ID represents a unique identifier for domain entities.
// It uses string to support various ID formats (UUID, ULID, etc.).
type ID string

// String returns the string representation of the ID.
func (id ID) String() string {
	return string(id)
}

// IsEmpty returns true if the ID is empty or not set.
func (id ID) IsEmpty() bool {
	return id == ""
}

// Value implements the driver.Valuer interface for database serialization.
func (id ID) Value() (driver.Value, error) {
	return string(id), nil
}

// Scan implements the sql.Scanner interface for database deserialization.
func (id *ID) Scan(value interface{}) error {
	if value == nil {
		*id = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*id = ID(v)
	case []byte:
		*id = ID(v)
	default:
		return fmt.Errorf("cannot scan type %T into ID", value)
	}
	return nil
}

// EmailAddress represents a validated email address.
type EmailAddress struct {
	// Name is the display name part of the email address (e.g., "John Doe").
	Name string `json:"name,omitempty"`
	// Address is the actual email address (e.g., "john@example.com").
	Address string `json:"address"`
}

// String returns the formatted email address.
// If Name is set, it returns "Name <address>", otherwise just the address.
func (e EmailAddress) String() string {
	if e.Name == "" {
		return e.Address
	}
	return fmt.Sprintf("%s <%s>", e.Name, e.Address)
}

// IsEmpty returns true if the email address is not set.
func (e EmailAddress) IsEmpty() bool {
	return e.Address == ""
}

// Timestamp represents a point in time with JSON serialization support.
// It wraps time.Time to provide consistent formatting across the application.
type Timestamp struct {
	time.Time
}

// Now returns the current time as a Timestamp.
func Now() Timestamp {
	return Timestamp{Time: time.Now().UTC()}
}

// MarshalJSON implements the json.Marshaler interface.
// Timestamps are serialized in RFC3339 format.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Format(time.RFC3339))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Supports RFC3339 format for deserialization.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

// Value implements the driver.Valuer interface for database serialization.
func (t Timestamp) Value() (driver.Value, error) {
	return t.Time, nil
}

// Scan implements the sql.Scanner interface for database deserialization.
func (t *Timestamp) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v
	default:
		return fmt.Errorf("cannot scan type %T into Timestamp", value)
	}
	return nil
}

// UserRole represents the role of a user in the system.
type UserRole string

const (
	// RoleAdmin has full access to all features and settings.
	RoleAdmin UserRole = "admin"
	// RoleUser has standard access to their own mailboxes and messages.
	RoleUser UserRole = "user"
	// RoleViewer has read-only access to mailboxes and messages.
	RoleViewer UserRole = "viewer"
)

// IsValid returns true if the role is a recognized value.
func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleUser, RoleViewer:
		return true
	default:
		return false
	}
}

// String returns the string representation of the role.
func (r UserRole) String() string {
	return string(r)
}

// UserStatus represents the current status of a user account.
type UserStatus string

const (
	// StatusActive indicates the user account is active and can log in.
	StatusActive UserStatus = "active"
	// StatusInactive indicates the user account is disabled.
	StatusInactive UserStatus = "inactive"
	// StatusPending indicates the user account is awaiting activation.
	StatusPending UserStatus = "pending"
)

// IsValid returns true if the status is a recognized value.
func (s UserStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusPending:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status.
func (s UserStatus) String() string {
	return string(s)
}

// MessageStatus represents the read/unread status of a message.
type MessageStatus string

const (
	// MessageUnread indicates the message has not been read.
	MessageUnread MessageStatus = "unread"
	// MessageRead indicates the message has been read.
	MessageRead MessageStatus = "read"
)

// IsValid returns true if the status is a recognized value.
func (s MessageStatus) IsValid() bool {
	switch s {
	case MessageUnread, MessageRead:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status.
func (s MessageStatus) String() string {
	return string(s)
}

// ContentType represents the MIME content type of a message or attachment.
type ContentType string

const (
	// ContentTypePlain represents plain text content.
	ContentTypePlain ContentType = "text/plain"
	// ContentTypeHTML represents HTML content.
	ContentTypeHTML ContentType = "text/html"
	// ContentTypeMultipart represents multipart content.
	ContentTypeMultipart ContentType = "multipart/mixed"
)

// String returns the string representation of the content type.
func (c ContentType) String() string {
	return string(c)
}

// WebhookEvent represents the type of event that triggers a webhook.
type WebhookEvent string

const (
	// WebhookEventMessageReceived is triggered when a new message is received.
	WebhookEventMessageReceived WebhookEvent = "message.received"
	// WebhookEventMessageDeleted is triggered when a message is deleted.
	WebhookEventMessageDeleted WebhookEvent = "message.deleted"
	// WebhookEventMailboxCreated is triggered when a new mailbox is created.
	WebhookEventMailboxCreated WebhookEvent = "mailbox.created"
	// WebhookEventMailboxDeleted is triggered when a mailbox is deleted.
	WebhookEventMailboxDeleted WebhookEvent = "mailbox.deleted"
	// WebhookEventUserCreated is triggered when a new user is created.
	WebhookEventUserCreated WebhookEvent = "user.created"
)

// IsValid returns true if the event is a recognized value.
func (e WebhookEvent) IsValid() bool {
	switch e {
	case WebhookEventMessageReceived, WebhookEventMessageDeleted,
		WebhookEventMailboxCreated, WebhookEventMailboxDeleted,
		WebhookEventUserCreated:
		return true
	default:
		return false
	}
}

// String returns the string representation of the event.
func (e WebhookEvent) String() string {
	return string(e)
}

// WebhookStatus represents the current status of a webhook.
type WebhookStatus string

const (
	// WebhookStatusActive indicates the webhook is enabled and will receive events.
	WebhookStatusActive WebhookStatus = "active"
	// WebhookStatusInactive indicates the webhook is disabled.
	WebhookStatusInactive WebhookStatus = "inactive"
	// WebhookStatusFailed indicates the webhook has failed too many times.
	WebhookStatusFailed WebhookStatus = "failed"
)

// IsValid returns true if the status is a recognized value.
func (s WebhookStatus) IsValid() bool {
	switch s {
	case WebhookStatusActive, WebhookStatusInactive, WebhookStatusFailed:
		return true
	default:
		return false
	}
}

// String returns the string representation of the status.
func (s WebhookStatus) String() string {
	return string(s)
}

// DatabaseDriver represents the supported database drivers.
type DatabaseDriver string

const (
	// DatabaseDriverSQLite uses SQLite as the database backend.
	DatabaseDriverSQLite DatabaseDriver = "sqlite"
	// DatabaseDriverPostgres uses PostgreSQL as the database backend.
	DatabaseDriverPostgres DatabaseDriver = "postgres"
	// DatabaseDriverMySQL uses MySQL as the database backend.
	DatabaseDriverMySQL DatabaseDriver = "mysql"
	// DatabaseDriverMongoDB uses MongoDB as the database backend.
	DatabaseDriverMongoDB DatabaseDriver = "mongodb"
)

// IsValid returns true if the driver is a recognized value.
func (d DatabaseDriver) IsValid() bool {
	switch d {
	case DatabaseDriverSQLite, DatabaseDriverPostgres, DatabaseDriverMySQL, DatabaseDriverMongoDB:
		return true
	default:
		return false
	}
}

// String returns the string representation of the driver.
func (d DatabaseDriver) String() string {
	return string(d)
}

// Pagination contains pagination parameters for list queries.
type Pagination struct {
	// Page is the current page number (1-indexed).
	Page int `json:"page"`
	// PerPage is the number of items per page.
	PerPage int `json:"perPage"`
	// Total is the total number of items available.
	Total int64 `json:"total"`
}

// Offset returns the database offset for the current page.
func (p Pagination) Offset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

// TotalPages returns the total number of pages.
func (p Pagination) TotalPages() int {
	if p.PerPage <= 0 {
		return 0
	}
	pages := int(p.Total) / p.PerPage
	if int(p.Total)%p.PerPage > 0 {
		pages++
	}
	return pages
}

// HasNext returns true if there is a next page.
func (p Pagination) HasNext() bool {
	return p.Page < p.TotalPages()
}

// HasPrev returns true if there is a previous page.
func (p Pagination) HasPrev() bool {
	return p.Page > 1
}

// SortOrder represents the sort direction for list queries.
type SortOrder string

const (
	// SortAsc sorts in ascending order.
	SortAsc SortOrder = "asc"
	// SortDesc sorts in descending order.
	SortDesc SortOrder = "desc"
)

// IsValid returns true if the sort order is a recognized value.
func (s SortOrder) IsValid() bool {
	switch s {
	case SortAsc, SortDesc:
		return true
	default:
		return false
	}
}

// String returns the string representation of the sort order.
func (s SortOrder) String() string {
	return string(s)
}
