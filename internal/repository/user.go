package repository

import (
	"context"

	"yunt/internal/domain"
)

// UserRepository provides data access operations for User entities.
// It supports CRUD operations, authentication-related queries, and user management.
type UserRepository interface {
	// GetByID retrieves a user by their unique identifier.
	// Returns domain.ErrNotFound if the user does not exist.
	GetByID(ctx context.Context, id domain.ID) (*domain.User, error)

	// GetByUsername retrieves a user by their username.
	// Username lookup is case-insensitive.
	// Returns domain.ErrNotFound if no user with the username exists.
	GetByUsername(ctx context.Context, username string) (*domain.User, error)

	// GetByEmail retrieves a user by their email address.
	// Email lookup is case-insensitive.
	// Returns domain.ErrNotFound if no user with the email exists.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// List retrieves users with optional filtering, sorting, and pagination.
	// Returns an empty slice if no users match the criteria.
	List(ctx context.Context, filter *UserFilter, opts *ListOptions) (*ListResult[*domain.User], error)

	// Create creates a new user.
	// Returns domain.ErrAlreadyExists if a user with the same username or email exists.
	Create(ctx context.Context, user *domain.User) error

	// Update updates an existing user.
	// Returns domain.ErrNotFound if the user does not exist.
	// Returns domain.ErrAlreadyExists if the new username or email conflicts with another user.
	Update(ctx context.Context, user *domain.User) error

	// Delete permanently removes a user by their ID.
	// Returns domain.ErrNotFound if the user does not exist.
	// Note: This may fail if the user has associated mailboxes or other resources.
	// Consider using SoftDelete for safer deletion.
	Delete(ctx context.Context, id domain.ID) error

	// SoftDelete marks a user as deleted without removing the record.
	// The user will be excluded from normal queries but can be restored.
	// Returns domain.ErrNotFound if the user does not exist.
	SoftDelete(ctx context.Context, id domain.ID) error

	// Restore restores a soft-deleted user.
	// Returns domain.ErrNotFound if the user does not exist or was not soft-deleted.
	Restore(ctx context.Context, id domain.ID) error

	// Exists checks if a user with the given ID exists.
	Exists(ctx context.Context, id domain.ID) (bool, error)

	// ExistsByUsername checks if a user with the given username exists.
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// ExistsByEmail checks if a user with the given email exists.
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// Count returns the total number of users matching the filter.
	// If filter is nil, returns the total count of all users.
	Count(ctx context.Context, filter *UserFilter) (int64, error)

	// CountByRole returns the count of users grouped by role.
	CountByRole(ctx context.Context) (map[domain.UserRole]int64, error)

	// CountByStatus returns the count of users grouped by status.
	CountByStatus(ctx context.Context) (map[domain.UserStatus]int64, error)

	// UpdatePassword updates a user's password hash.
	// Returns domain.ErrNotFound if the user does not exist.
	UpdatePassword(ctx context.Context, id domain.ID, passwordHash string) error

	// UpdateLastLogin updates the user's last login timestamp.
	// Returns domain.ErrNotFound if the user does not exist.
	UpdateLastLogin(ctx context.Context, id domain.ID) error

	// UpdateStatus updates a user's status.
	// Returns domain.ErrNotFound if the user does not exist.
	UpdateStatus(ctx context.Context, id domain.ID, status domain.UserStatus) error

	// UpdateRole updates a user's role.
	// Returns domain.ErrNotFound if the user does not exist.
	UpdateRole(ctx context.Context, id domain.ID, role domain.UserRole) error

	// GetActiveUsers retrieves all active users.
	// This is a convenience method equivalent to List with Status=Active filter.
	GetActiveUsers(ctx context.Context, opts *ListOptions) (*ListResult[*domain.User], error)

	// GetAdmins retrieves all admin users.
	// This is a convenience method equivalent to List with Role=Admin filter.
	GetAdmins(ctx context.Context) ([]*domain.User, error)

	// Search performs a text search across user fields.
	// Searches in username, email, and display name.
	Search(ctx context.Context, query string, opts *ListOptions) (*ListResult[*domain.User], error)

	// BulkUpdateStatus updates the status of multiple users.
	// Returns a BulkOperation result with success/failure counts.
	BulkUpdateStatus(ctx context.Context, ids []domain.ID, status domain.UserStatus) (*BulkOperation, error)

	// BulkDelete permanently removes multiple users.
	// Returns a BulkOperation result with success/failure counts.
	BulkDelete(ctx context.Context, ids []domain.ID) (*BulkOperation, error)

	// GetUsersCreatedBetween retrieves users created within the date range.
	GetUsersCreatedBetween(ctx context.Context, dateRange *DateRangeFilter, opts *ListOptions) (*ListResult[*domain.User], error)

	// GetUsersWithRecentLogin retrieves users who logged in within the specified days.
	GetUsersWithRecentLogin(ctx context.Context, days int, opts *ListOptions) (*ListResult[*domain.User], error)

	// GetInactiveUsers retrieves users who haven't logged in for the specified days.
	GetInactiveUsers(ctx context.Context, days int, opts *ListOptions) (*ListResult[*domain.User], error)
}

// UserFilter provides filtering options for user queries.
type UserFilter struct {
	// IDs filters by specific user IDs.
	IDs []domain.ID

	// Status filters by user status.
	Status *domain.UserStatus

	// Statuses filters by multiple statuses (OR condition).
	Statuses []domain.UserStatus

	// Role filters by user role.
	Role *domain.UserRole

	// Roles filters by multiple roles (OR condition).
	Roles []domain.UserRole

	// Search performs text search on username, email, and display name.
	Search string

	// Username filters by exact username match.
	Username string

	// Email filters by exact email match.
	Email string

	// CreatedBefore filters users created before this timestamp.
	CreatedBefore *domain.Timestamp

	// CreatedAfter filters users created after this timestamp.
	CreatedAfter *domain.Timestamp

	// LastLoginBefore filters users who last logged in before this timestamp.
	LastLoginBefore *domain.Timestamp

	// LastLoginAfter filters users who last logged in after this timestamp.
	LastLoginAfter *domain.Timestamp

	// HasNeverLoggedIn filters users who have never logged in.
	HasNeverLoggedIn *bool

	// IncludeDeleted includes soft-deleted users in results.
	IncludeDeleted bool
}

// IsEmpty returns true if no filter criteria are set.
func (f *UserFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return len(f.IDs) == 0 &&
		f.Status == nil &&
		len(f.Statuses) == 0 &&
		f.Role == nil &&
		len(f.Roles) == 0 &&
		f.Search == "" &&
		f.Username == "" &&
		f.Email == "" &&
		f.CreatedBefore == nil &&
		f.CreatedAfter == nil &&
		f.LastLoginBefore == nil &&
		f.LastLoginAfter == nil &&
		f.HasNeverLoggedIn == nil &&
		!f.IncludeDeleted
}

// UserSortField represents the available fields for sorting users.
type UserSortField string

const (
	// UserSortByUsername sorts by username.
	UserSortByUsername UserSortField = "username"

	// UserSortByEmail sorts by email.
	UserSortByEmail UserSortField = "email"

	// UserSortByDisplayName sorts by display name.
	UserSortByDisplayName UserSortField = "displayName"

	// UserSortByCreatedAt sorts by creation timestamp.
	UserSortByCreatedAt UserSortField = "createdAt"

	// UserSortByUpdatedAt sorts by update timestamp.
	UserSortByUpdatedAt UserSortField = "updatedAt"

	// UserSortByLastLogin sorts by last login timestamp.
	UserSortByLastLogin UserSortField = "lastLoginAt"

	// UserSortByRole sorts by role.
	UserSortByRole UserSortField = "role"

	// UserSortByStatus sorts by status.
	UserSortByStatus UserSortField = "status"
)

// IsValid returns true if the sort field is a recognized value.
func (f UserSortField) IsValid() bool {
	switch f {
	case UserSortByUsername, UserSortByEmail, UserSortByDisplayName,
		UserSortByCreatedAt, UserSortByUpdatedAt, UserSortByLastLogin,
		UserSortByRole, UserSortByStatus:
		return true
	default:
		return false
	}
}

// String returns the string representation of the sort field.
func (f UserSortField) String() string {
	return string(f)
}
