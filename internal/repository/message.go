package repository

import (
	"context"

	"yunt/internal/domain"
)

// MessageRepository provides data access operations for Message entities.
// It supports CRUD operations, message status management, search, and threading.
type MessageRepository interface {
	// GetByID retrieves a message by its unique identifier.
	// Returns domain.ErrNotFound if the message does not exist.
	GetByID(ctx context.Context, id domain.ID) (*domain.Message, error)

	// GetByMessageID retrieves a message by its email Message-ID header.
	// Returns domain.ErrNotFound if no message with the Message-ID exists.
	GetByMessageID(ctx context.Context, messageID string) (*domain.Message, error)

	// GetWithAttachments retrieves a message with its attachments loaded.
	// Returns domain.ErrNotFound if the message does not exist.
	GetWithAttachments(ctx context.Context, id domain.ID) (*domain.Message, []*domain.Attachment, error)

	// List retrieves messages with optional filtering, sorting, and pagination.
	// Returns an empty slice if no messages match the criteria.
	List(ctx context.Context, filter *MessageFilter, opts *ListOptions) (*ListResult[*domain.Message], error)

	// ListByMailbox retrieves all messages in a specific mailbox.
	ListByMailbox(ctx context.Context, mailboxID domain.ID, opts *ListOptions) (*ListResult[*domain.Message], error)

	// ListSummaries retrieves message summaries for faster list rendering.
	// Summaries contain only essential fields for displaying message lists.
	ListSummaries(ctx context.Context, filter *MessageFilter, opts *ListOptions) (*ListResult[*domain.MessageSummary], error)

	// Create creates a new message.
	// This also updates the mailbox statistics (message count, size).
	Create(ctx context.Context, message *domain.Message) error

	// Update updates an existing message.
	// Returns domain.ErrNotFound if the message does not exist.
	Update(ctx context.Context, message *domain.Message) error

	// Delete permanently removes a message by its ID.
	// This also updates the mailbox statistics and removes associated attachments.
	// Returns domain.ErrNotFound if the message does not exist.
	Delete(ctx context.Context, id domain.ID) error

	// DeleteByMailbox removes all messages in a mailbox.
	// Returns the number of deleted messages.
	DeleteByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error)

	// Exists checks if a message with the given ID exists.
	Exists(ctx context.Context, id domain.ID) (bool, error)

	// ExistsByMessageID checks if a message with the given Message-ID exists.
	ExistsByMessageID(ctx context.Context, messageID string) (bool, error)

	// Count returns the total number of messages matching the filter.
	Count(ctx context.Context, filter *MessageFilter) (int64, error)

	// CountByMailbox returns the number of messages in a mailbox.
	CountByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error)

	// CountUnreadByMailbox returns the number of unread messages in a mailbox.
	CountUnreadByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error)

	// MarkAsRead marks a message as read.
	// Returns true if the status was changed, false if already read.
	// Returns domain.ErrNotFound if the message does not exist.
	MarkAsRead(ctx context.Context, id domain.ID) (bool, error)

	// MarkAsUnread marks a message as unread.
	// Returns true if the status was changed, false if already unread.
	// Returns domain.ErrNotFound if the message does not exist.
	MarkAsUnread(ctx context.Context, id domain.ID) (bool, error)

	// MarkAllAsRead marks all messages in a mailbox as read.
	// Returns the number of messages updated.
	MarkAllAsRead(ctx context.Context, mailboxID domain.ID) (int64, error)

	// ToggleStar toggles the starred status of a message.
	// Returns the new starred status.
	// Returns domain.ErrNotFound if the message does not exist.
	ToggleStar(ctx context.Context, id domain.ID) (bool, error)

	// Star marks a message as starred.
	// Returns domain.ErrNotFound if the message does not exist.
	Star(ctx context.Context, id domain.ID) error

	// Unstar removes the star from a message.
	// Returns domain.ErrNotFound if the message does not exist.
	Unstar(ctx context.Context, id domain.ID) error

	// MarkAsSpam marks a message as spam.
	// Returns domain.ErrNotFound if the message does not exist.
	MarkAsSpam(ctx context.Context, id domain.ID) error

	// MarkAsNotSpam removes the spam flag from a message.
	// Returns domain.ErrNotFound if the message does not exist.
	MarkAsNotSpam(ctx context.Context, id domain.ID) error

	// MarkAsDeleted sets the \Deleted flag on a message (pending EXPUNGE).
	MarkAsDeleted(ctx context.Context, id domain.ID) error

	// UnmarkAsDeleted removes the \Deleted flag from a message.
	UnmarkAsDeleted(ctx context.Context, id domain.ID) error

	// MoveToMailbox moves a message to a different mailbox.
	// This updates statistics for both source and destination mailboxes.
	// Returns domain.ErrNotFound if the message or target mailbox does not exist.
	MoveToMailbox(ctx context.Context, id domain.ID, targetMailboxID domain.ID) error

	// Search performs a full-text search across message fields.
	// Searches in subject, sender, recipients, and body content.
	Search(ctx context.Context, searchOpts *SearchOptions, filter *MessageFilter, opts *ListOptions) (*ListResult[*domain.Message], error)

	// SearchSummaries performs search and returns message summaries.
	SearchSummaries(ctx context.Context, searchOpts *SearchOptions, filter *MessageFilter, opts *ListOptions) (*ListResult[*domain.MessageSummary], error)

	// GetThread retrieves all messages in a conversation thread.
	// Uses References and In-Reply-To headers to find related messages.
	GetThread(ctx context.Context, id domain.ID) ([]*domain.Message, error)

	// GetReplies retrieves all replies to a specific message.
	GetReplies(ctx context.Context, id domain.ID) ([]*domain.Message, error)

	// GetStarred retrieves all starred messages.
	GetStarred(ctx context.Context, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetStarredByUser retrieves starred messages from mailboxes owned by a user.
	GetStarredByUser(ctx context.Context, userID domain.ID, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetSpam retrieves all spam messages.
	GetSpam(ctx context.Context, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetUnread retrieves all unread messages.
	GetUnread(ctx context.Context, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetUnreadByMailbox retrieves unread messages in a specific mailbox.
	GetUnreadByMailbox(ctx context.Context, mailboxID domain.ID, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetWithAttachments retrieves messages that have attachments.
	GetMessagesWithAttachments(ctx context.Context, filter *MessageFilter, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetRecent retrieves messages received in the last N hours.
	GetRecent(ctx context.Context, hours int, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetByDateRange retrieves messages within a date range.
	GetByDateRange(ctx context.Context, dateRange *DateRangeFilter, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetBySender retrieves all messages from a specific sender.
	GetBySender(ctx context.Context, senderAddress string, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetByRecipient retrieves all messages sent to a specific recipient.
	GetByRecipient(ctx context.Context, recipientAddress string, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetOldMessages retrieves messages older than the specified days.
	// Useful for retention policy enforcement.
	GetOldMessages(ctx context.Context, olderThanDays int, opts *ListOptions) (*ListResult[*domain.Message], error)

	// GetLargeMessages retrieves messages larger than the specified size.
	GetLargeMessages(ctx context.Context, minSize int64, opts *ListOptions) (*ListResult[*domain.Message], error)

	// DeleteOldMessages deletes messages older than the specified days.
	// Returns the number of deleted messages.
	DeleteOldMessages(ctx context.Context, olderThanDays int) (int64, error)

	// DeleteSpam deletes all spam messages.
	// Returns the number of deleted messages.
	DeleteSpam(ctx context.Context) (int64, error)

	// BulkMarkAsRead marks multiple messages as read.
	// Returns a BulkOperation result with success/failure counts.
	BulkMarkAsRead(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// BulkMarkAsUnread marks multiple messages as unread.
	BulkMarkAsUnread(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// BulkDelete permanently removes multiple messages.
	BulkDelete(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// BulkMove moves multiple messages to a different mailbox.
	BulkMove(ctx context.Context, ids []domain.ID, targetMailboxID domain.ID) (*BulkOperation, error)

	// BulkStar marks multiple messages as starred.
	BulkStar(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// BulkUnstar removes the star from multiple messages.
	BulkUnstar(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// GetSizeByMailbox calculates the total size of messages in a mailbox.
	GetSizeByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error)

	// GetTotalSize calculates the total size of all messages.
	GetTotalSize(ctx context.Context) (int64, error)

	// GetDailyCounts returns message counts grouped by day within a date range.
	GetDailyCounts(ctx context.Context, dateRange *DateRangeFilter) ([]DateCount, error)

	// GetSenderCounts returns message counts grouped by sender address.
	GetSenderCounts(ctx context.Context, limit int) ([]AddressCount, error)

	// GetRecipientCounts returns message counts grouped by recipient address.
	GetRecipientCounts(ctx context.Context, limit int) ([]AddressCount, error)

	// StoreRawBody stores the raw message body for a message.
	// This may store in the database or a separate file storage.
	StoreRawBody(ctx context.Context, id domain.ID, rawBody []byte) error

	// GetRawBody retrieves the raw message body.
	// Returns domain.ErrNotFound if not found.
	GetRawBody(ctx context.Context, id domain.ID) ([]byte, error)
}

// MessageFilter provides filtering options for message queries.
type MessageFilter struct {
	// IDs filters by specific message IDs.
	IDs []domain.ID

	// MailboxID filters by mailbox ID.
	MailboxID *domain.ID

	// MailboxIDs filters by multiple mailbox IDs (OR condition).
	MailboxIDs []domain.ID

	// Status filters by read/unread status.
	Status *domain.MessageStatus

	// IsStarred filters by starred status.
	IsStarred *bool

	// IsSpam filters by spam status.
	IsSpam *bool

	// HasAttachments filters by attachment presence.
	HasAttachments *bool

	// FromAddress filters by sender address (exact match).
	FromAddress string

	// FromAddressContains filters by partial sender address match.
	FromAddressContains string

	// ToAddress filters by recipient address (exact match).
	ToAddress string

	// ToAddressContains filters by partial recipient address match.
	ToAddressContains string

	// Subject filters by exact subject match.
	Subject string

	// SubjectContains filters by partial subject match.
	SubjectContains string

	// BodyContains filters by body content (searches both text and HTML).
	BodyContains string

	// Search performs full-text search on subject, from, to, and body.
	Search string

	// MessageID filters by email Message-ID header.
	MessageID string

	// InReplyTo filters by In-Reply-To header.
	InReplyTo string

	// ReceivedAfter filters messages received after this timestamp.
	ReceivedAfter *domain.Timestamp

	// ReceivedBefore filters messages received before this timestamp.
	ReceivedBefore *domain.Timestamp

	// SentAfter filters messages sent after this timestamp.
	SentAfter *domain.Timestamp

	// SentBefore filters messages sent before this timestamp.
	SentBefore *domain.Timestamp

	// MinSize filters messages larger than this size (bytes).
	MinSize *int64

	// MaxSize filters messages smaller than this size (bytes).
	MaxSize *int64

	// ContentType filters by message content type.
	ContentType *domain.ContentType

	// ExcludeSpam excludes spam messages from results.
	ExcludeSpam bool

	// ExcludeDeleted excludes messages marked with \Deleted flag from results.
	ExcludeDeleted bool
}

// IsEmpty returns true if no filter criteria are set.
func (f *MessageFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.IDs) == 0 &&
		f.MailboxID == nil &&
		len(f.MailboxIDs) == 0 &&
		f.Status == nil &&
		f.IsStarred == nil &&
		f.IsSpam == nil &&
		f.HasAttachments == nil &&
		f.FromAddress == "" &&
		f.FromAddressContains == "" &&
		f.ToAddress == "" &&
		f.ToAddressContains == "" &&
		f.Subject == "" &&
		f.SubjectContains == "" &&
		f.BodyContains == "" &&
		f.Search == "" &&
		f.MessageID == "" &&
		f.InReplyTo == "" &&
		f.ReceivedAfter == nil &&
		f.ReceivedBefore == nil &&
		f.SentAfter == nil &&
		f.SentBefore == nil &&
		f.MinSize == nil &&
		f.MaxSize == nil &&
		f.ContentType == nil &&
		!f.ExcludeSpam
}

// MessageSortField represents the available fields for sorting messages.
type MessageSortField string

const (
	// MessageSortByReceivedAt sorts by received timestamp.
	MessageSortByReceivedAt MessageSortField = "receivedAt"

	// MessageSortBySentAt sorts by sent timestamp.
	MessageSortBySentAt MessageSortField = "sentAt"

	// MessageSortBySubject sorts by subject.
	MessageSortBySubject MessageSortField = "subject"

	// MessageSortByFrom sorts by sender address.
	MessageSortByFrom MessageSortField = "from"

	// MessageSortBySize sorts by message size.
	MessageSortBySize MessageSortField = "size"

	// MessageSortByStatus sorts by read/unread status.
	MessageSortByStatus MessageSortField = "status"

	// MessageSortByCreatedAt sorts by creation timestamp.
	MessageSortByCreatedAt MessageSortField = "createdAt"
)

// IsValid returns true if the sort field is a recognized value.
func (f MessageSortField) IsValid() bool {
	switch f {
	case MessageSortByReceivedAt, MessageSortBySentAt, MessageSortBySubject,
		MessageSortByFrom, MessageSortBySize, MessageSortByStatus, MessageSortByCreatedAt:
		return true
	default:
		return false
	}
}

// String returns the string representation of the sort field.
func (f MessageSortField) String() string {
	return string(f)
}

// DateCount represents a count grouped by date.
type DateCount struct {
	// Date is the date (YYYY-MM-DD format).
	Date string `json:"date"`

	// Count is the number of items on that date.
	Count int64 `json:"count"`
}

// AddressCount represents a count grouped by email address.
type AddressCount struct {
	// Address is the email address.
	Address string

	// Name is the display name (if available).
	Name string

	// Count is the number of items for this address.
	Count int64
}
