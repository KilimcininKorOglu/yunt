package repository

import (
	"context"

	"yunt/internal/domain"
)

// MailboxRepository provides data access operations for Mailbox entities.
// It supports CRUD operations, ownership management, and mailbox statistics.
type MailboxRepository interface {
	// GetByID retrieves a mailbox by its unique identifier.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	GetByID(ctx context.Context, id domain.ID) (*domain.Mailbox, error)

	// GetByAddress retrieves a mailbox by its email address.
	// Address lookup is case-insensitive.
	// Returns domain.ErrNotFound if no mailbox with the address exists.
	GetByAddress(ctx context.Context, address string) (*domain.Mailbox, error)

	// GetCatchAll retrieves the catch-all mailbox for a domain.
	// Returns domain.ErrNotFound if no catch-all mailbox exists for the domain.
	GetCatchAll(ctx context.Context, domainName string) (*domain.Mailbox, error)

	// GetDefault retrieves the default mailbox for a user.
	// Returns domain.ErrNotFound if the user has no default mailbox.
	GetDefault(ctx context.Context, userID domain.ID) (*domain.Mailbox, error)

	// List retrieves mailboxes with optional filtering, sorting, and pagination.
	// Returns an empty slice if no mailboxes match the criteria.
	List(ctx context.Context, filter *MailboxFilter, opts *ListOptions) (*ListResult[*domain.Mailbox], error)

	// ListByUser retrieves all mailboxes owned by a specific user.
	ListByUser(ctx context.Context, userID domain.ID, opts *ListOptions) (*ListResult[*domain.Mailbox], error)

	// Create creates a new mailbox.
	// Returns domain.ErrAlreadyExists if a mailbox with the same address exists.
	Create(ctx context.Context, mailbox *domain.Mailbox) error

	// Update updates an existing mailbox.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	// Returns domain.ErrAlreadyExists if the new address conflicts with another mailbox.
	Update(ctx context.Context, mailbox *domain.Mailbox) error

	// Delete permanently removes a mailbox by its ID.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	// Note: This may fail if the mailbox contains messages. Use DeleteWithMessages for cascade delete.
	Delete(ctx context.Context, id domain.ID) error

	// DeleteWithMessages removes a mailbox and all its messages.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	DeleteWithMessages(ctx context.Context, id domain.ID) error

	// DeleteByUser removes all mailboxes owned by a user.
	// Returns the number of deleted mailboxes.
	DeleteByUser(ctx context.Context, userID domain.ID) (int64, error)

	// Exists checks if a mailbox with the given ID exists.
	Exists(ctx context.Context, id domain.ID) (bool, error)

	// ExistsByAddress checks if a mailbox with the given address exists.
	ExistsByAddress(ctx context.Context, address string) (bool, error)

	// Count returns the total number of mailboxes matching the filter.
	Count(ctx context.Context, filter *MailboxFilter) (int64, error)

	// CountByUser returns the number of mailboxes owned by a user.
	CountByUser(ctx context.Context, userID domain.ID) (int64, error)

	// SetDefault sets a mailbox as the default for its owner.
	// This also clears the default flag from any other mailbox owned by the same user.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	SetDefault(ctx context.Context, id domain.ID) error

	// ClearDefault removes the default flag from all mailboxes for a user.
	ClearDefault(ctx context.Context, userID domain.ID) error

	// SetCatchAll sets a mailbox as the catch-all for its domain.
	// This also clears the catch-all flag from any other mailbox in the same domain.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	SetCatchAll(ctx context.Context, id domain.ID) error

	// ClearCatchAll removes the catch-all flag from a mailbox.
	ClearCatchAll(ctx context.Context, id domain.ID) error

	// UpdateStats updates the mailbox statistics (message count, unread count, size).
	// Returns domain.ErrNotFound if the mailbox does not exist.
	UpdateStats(ctx context.Context, id domain.ID, stats *MailboxStatsUpdate) error

	// IncrementMessageCount atomically increments message counters and assigns the next IMAP UID.
	// Returns the assigned UID and domain.ErrNotFound if the mailbox does not exist.
	IncrementMessageCount(ctx context.Context, id domain.ID, size int64) (uint32, error)

	// DecrementMessageCount atomically decrements message counters.
	// wasUnread indicates if the removed message was unread.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	DecrementMessageCount(ctx context.Context, id domain.ID, size int64, wasUnread bool) error

	// UpdateUnreadCount atomically updates the unread count.
	// delta can be positive or negative.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	UpdateUnreadCount(ctx context.Context, id domain.ID, delta int) error

	// RecalculateStats recalculates mailbox statistics from messages.
	// This is useful for repairing corrupted stats.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	RecalculateStats(ctx context.Context, id domain.ID) error

	// GetStats retrieves detailed statistics for a mailbox.
	// Returns domain.ErrNotFound if the mailbox does not exist.
	GetStats(ctx context.Context, id domain.ID) (*domain.MailboxStats, error)

	// GetStatsByUser retrieves aggregated statistics for all mailboxes owned by a user.
	GetStatsByUser(ctx context.Context, userID domain.ID) (*domain.MailboxStats, error)

	// GetTotalStats retrieves aggregated statistics for all mailboxes.
	GetTotalStats(ctx context.Context) (*domain.MailboxStats, error)

	// FindMatchingMailbox finds the mailbox that should receive a message for the given address.
	// It first looks for an exact match, then falls back to catch-all mailboxes.
	// Returns domain.ErrNotFound if no matching mailbox exists.
	FindMatchingMailbox(ctx context.Context, address string) (*domain.Mailbox, error)

	// Search performs a text search across mailbox fields.
	// Searches in name, address, and description.
	Search(ctx context.Context, query string, opts *ListOptions) (*ListResult[*domain.Mailbox], error)

	// GetMailboxesWithMessages retrieves mailboxes that have at least one message.
	GetMailboxesWithMessages(ctx context.Context, opts *ListOptions) (*ListResult[*domain.Mailbox], error)

	// GetMailboxesWithUnread retrieves mailboxes that have unread messages.
	GetMailboxesWithUnread(ctx context.Context, opts *ListOptions) (*ListResult[*domain.Mailbox], error)

	// TransferOwnership transfers all mailboxes from one user to another.
	// Returns the number of transferred mailboxes.
	TransferOwnership(ctx context.Context, fromUserID, toUserID domain.ID) (int64, error)

	// BulkDelete permanently removes multiple mailboxes.
	// Returns a BulkOperation result with success/failure counts.
	BulkDelete(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// GetDomains retrieves all unique domains from mailbox addresses.
	GetDomains(ctx context.Context) ([]string, error)

	// GetMailboxesByDomain retrieves all mailboxes for a specific domain.
	GetMailboxesByDomain(ctx context.Context, domainName string, opts *ListOptions) (*ListResult[*domain.Mailbox], error)
}

// MailboxFilter provides filtering options for mailbox queries.
type MailboxFilter struct {
	// IDs filters by specific mailbox IDs.
	IDs []domain.ID

	// UserID filters by owner user ID.
	UserID *domain.ID

	// UserIDs filters by multiple owner user IDs (OR condition).
	UserIDs []domain.ID

	// Address filters by exact email address match.
	Address string

	// AddressContains filters by partial address match.
	AddressContains string

	// Domain filters mailboxes by domain part of the address.
	Domain string

	// IsCatchAll filters by catch-all status.
	IsCatchAll *bool

	// IsDefault filters by default status.
	IsDefault *bool

	// HasMessages filters mailboxes that have at least one message.
	HasMessages *bool

	// HasUnread filters mailboxes that have unread messages.
	HasUnread *bool

	// Search performs text search on name, address, and description.
	Search string

	// MinMessageCount filters mailboxes with at least this many messages.
	MinMessageCount *int64

	// MaxMessageCount filters mailboxes with at most this many messages.
	MaxMessageCount *int64

	// MinSize filters mailboxes with total size at least this value (bytes).
	MinSize *int64

	// MaxSize filters mailboxes with total size at most this value (bytes).
	MaxSize *int64

	// CreatedBefore filters mailboxes created before this timestamp.
	CreatedBefore *domain.Timestamp

	// CreatedAfter filters mailboxes created after this timestamp.
	CreatedAfter *domain.Timestamp

	// RetentionDays filters mailboxes with specific retention settings.
	// Use -1 to filter mailboxes with no retention (0 = forever).
	RetentionDays *int
}

// IsEmpty returns true if no filter criteria are set.
func (f *MailboxFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.IDs) == 0 &&
		f.UserID == nil &&
		len(f.UserIDs) == 0 &&
		f.Address == "" &&
		f.AddressContains == "" &&
		f.Domain == "" &&
		f.IsCatchAll == nil &&
		f.IsDefault == nil &&
		f.HasMessages == nil &&
		f.HasUnread == nil &&
		f.Search == "" &&
		f.MinMessageCount == nil &&
		f.MaxMessageCount == nil &&
		f.MinSize == nil &&
		f.MaxSize == nil &&
		f.CreatedBefore == nil &&
		f.CreatedAfter == nil &&
		f.RetentionDays == nil
}

// MailboxStatsUpdate represents a stats update for a mailbox.
type MailboxStatsUpdate struct {
	// MessageCount sets the new message count.
	MessageCount *int64

	// UnreadCount sets the new unread count.
	UnreadCount *int64

	// TotalSize sets the new total size in bytes.
	TotalSize *int64
}

// MailboxSortField represents the available fields for sorting mailboxes.
type MailboxSortField string

const (
	// MailboxSortByName sorts by mailbox name.
	MailboxSortByName MailboxSortField = "name"

	// MailboxSortByAddress sorts by email address.
	MailboxSortByAddress MailboxSortField = "address"

	// MailboxSortByMessageCount sorts by message count.
	MailboxSortByMessageCount MailboxSortField = "messageCount"

	// MailboxSortByUnreadCount sorts by unread count.
	MailboxSortByUnreadCount MailboxSortField = "unreadCount"

	// MailboxSortByTotalSize sorts by total size.
	MailboxSortByTotalSize MailboxSortField = "totalSize"

	// MailboxSortByCreatedAt sorts by creation timestamp.
	MailboxSortByCreatedAt MailboxSortField = "createdAt"

	// MailboxSortByUpdatedAt sorts by update timestamp.
	MailboxSortByUpdatedAt MailboxSortField = "updatedAt"
)

// IsValid returns true if the sort field is a recognized value.
func (f MailboxSortField) IsValid() bool {
	switch f {
	case MailboxSortByName, MailboxSortByAddress, MailboxSortByMessageCount,
		MailboxSortByUnreadCount, MailboxSortByTotalSize,
		MailboxSortByCreatedAt, MailboxSortByUpdatedAt:
		return true
	default:
		return false
	}
}

// String returns the string representation of the sort field.
func (f MailboxSortField) String() string {
	return string(f)
}
