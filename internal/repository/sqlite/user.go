package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// UserRepository implements the repository.UserRepository interface for SQLite.
type UserRepository struct {
	repo *Repository
}

// userRow is the database representation of a user.
type userRow struct {
	ID           string         `db:"id"`
	Username     string         `db:"username"`
	Email        string         `db:"email"`
	PasswordHash string         `db:"password_hash"`
	DisplayName  sql.NullString `db:"display_name"`
	Role         string         `db:"role"`
	Status       string         `db:"status"`
	AvatarURL    sql.NullString `db:"avatar_url"`
	LastLoginAt  sql.NullTime   `db:"last_login_at"`
	DeletedAt    sql.NullTime   `db:"deleted_at"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

// NewUserRepository creates a new SQLite user repository.
func NewUserRepository(repo *Repository) *UserRepository {
	return &UserRepository{repo: repo}
}

// toUser converts a userRow to a domain.User.
func (r *userRow) toUser() *domain.User {
	user := &domain.User{
		ID:           domain.ID(r.ID),
		Username:     r.Username,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		Role:         domain.UserRole(r.Role),
		Status:       domain.UserStatus(r.Status),
		CreatedAt:    domain.Timestamp{Time: r.CreatedAt},
		UpdatedAt:    domain.Timestamp{Time: r.UpdatedAt},
	}

	if r.DisplayName.Valid {
		user.DisplayName = r.DisplayName.String
	}
	if r.AvatarURL.Valid {
		user.AvatarURL = r.AvatarURL.String
	}
	if r.LastLoginAt.Valid {
		ts := domain.Timestamp{Time: r.LastLoginAt.Time}
		user.LastLoginAt = &ts
	}

	return user
}

// GetByID retrieves a user by their unique identifier.
func (u *UserRepository) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	query := `SELECT id, username, email, password_hash, display_name, role, status, 
		avatar_url, last_login_at, deleted_at, created_at, updated_at 
		FROM users WHERE id = ? AND deleted_at IS NULL`

	var row userRow
	if err := u.repo.db().GetContext(ctx, &row, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("user", string(id))
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return row.toUser(), nil
}

// GetByUsername retrieves a user by their username.
func (u *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, username, email, password_hash, display_name, role, status, 
		avatar_url, last_login_at, deleted_at, created_at, updated_at 
		FROM users WHERE username = ? COLLATE NOCASE AND deleted_at IS NULL`

	var row userRow
	if err := u.repo.db().GetContext(ctx, &row, query, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("user", username)
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return row.toUser(), nil
}

// GetByEmail retrieves a user by their email address.
func (u *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, username, email, password_hash, display_name, role, status, 
		avatar_url, last_login_at, deleted_at, created_at, updated_at 
		FROM users WHERE email = ? COLLATE NOCASE AND deleted_at IS NULL`

	var row userRow
	if err := u.repo.db().GetContext(ctx, &row, query, strings.ToLower(email)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("user", email)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return row.toUser(), nil
}

// List retrieves users with optional filtering, sorting, and pagination.
func (u *UserRepository) List(ctx context.Context, filter *repository.UserFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	query, args := u.buildListQuery(filter, opts, false)
	countQuery, countArgs := u.buildListQuery(filter, opts, true)

	var total int64
	if err := u.repo.db().GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	var rows []userRow
	if err := u.repo.db().SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*domain.User, len(rows))
	for i, row := range rows {
		users[i] = row.toUser()
	}

	result := &repository.ListResult[*domain.User]{
		Items: users,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// buildListQuery builds the SQL query for listing users.
func (u *UserRepository) buildListQuery(filter *repository.UserFilter, opts *repository.ListOptions, countOnly bool) (string, []interface{}) {
	var sb strings.Builder
	args := make([]interface{}, 0)

	if countOnly {
		sb.WriteString("SELECT COUNT(*) FROM users WHERE 1=1")
	} else {
		sb.WriteString(`SELECT id, username, email, password_hash, display_name, role, status, 
			avatar_url, last_login_at, deleted_at, created_at, updated_at FROM users WHERE 1=1`)
	}

	// Apply filters
	if filter != nil {
		if !filter.IncludeDeleted {
			sb.WriteString(" AND deleted_at IS NULL")
		}

		if len(filter.IDs) > 0 {
			placeholders := make([]string, len(filter.IDs))
			for i, id := range filter.IDs {
				placeholders[i] = "?"
				args = append(args, string(id))
			}
			sb.WriteString(fmt.Sprintf(" AND id IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.Status != nil {
			sb.WriteString(" AND status = ?")
			args = append(args, string(*filter.Status))
		}

		if len(filter.Statuses) > 0 {
			placeholders := make([]string, len(filter.Statuses))
			for i, status := range filter.Statuses {
				placeholders[i] = "?"
				args = append(args, string(status))
			}
			sb.WriteString(fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.Role != nil {
			sb.WriteString(" AND role = ?")
			args = append(args, string(*filter.Role))
		}

		if len(filter.Roles) > 0 {
			placeholders := make([]string, len(filter.Roles))
			for i, role := range filter.Roles {
				placeholders[i] = "?"
				args = append(args, string(role))
			}
			sb.WriteString(fmt.Sprintf(" AND role IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.Username != "" {
			sb.WriteString(" AND username = ? COLLATE NOCASE")
			args = append(args, filter.Username)
		}

		if filter.Email != "" {
			sb.WriteString(" AND email = ? COLLATE NOCASE")
			args = append(args, filter.Email)
		}

		if filter.Search != "" {
			sb.WriteString(" AND (username LIKE ? OR email LIKE ? OR display_name LIKE ?)")
			searchPattern := "%" + filter.Search + "%"
			args = append(args, searchPattern, searchPattern, searchPattern)
		}

		if filter.CreatedBefore != nil {
			sb.WriteString(" AND created_at < ?")
			args = append(args, filter.CreatedBefore.Time)
		}

		if filter.CreatedAfter != nil {
			sb.WriteString(" AND created_at > ?")
			args = append(args, filter.CreatedAfter.Time)
		}

		if filter.LastLoginBefore != nil {
			sb.WriteString(" AND last_login_at < ?")
			args = append(args, filter.LastLoginBefore.Time)
		}

		if filter.LastLoginAfter != nil {
			sb.WriteString(" AND last_login_at > ?")
			args = append(args, filter.LastLoginAfter.Time)
		}

		if filter.HasNeverLoggedIn != nil && *filter.HasNeverLoggedIn {
			sb.WriteString(" AND last_login_at IS NULL")
		}
	} else {
		sb.WriteString(" AND deleted_at IS NULL")
	}

	if !countOnly {
		// Apply sorting
		if opts != nil && opts.Sort != nil {
			field := u.mapSortField(opts.Sort.Field)
			order := "ASC"
			if opts.Sort.Order == domain.SortDesc {
				order = "DESC"
			}
			sb.WriteString(fmt.Sprintf(" ORDER BY %s %s", field, order))
		} else {
			sb.WriteString(" ORDER BY created_at DESC")
		}

		// Apply pagination
		if opts != nil && opts.Pagination != nil {
			opts.Pagination.Normalize()
			sb.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset()))
		}
	}

	return sb.String(), args
}

// mapSortField maps repository sort field to database column.
func (u *UserRepository) mapSortField(field string) string {
	switch field {
	case "username":
		return "username"
	case "email":
		return "email"
	case "displayName":
		return "display_name"
	case "createdAt":
		return "created_at"
	case "updatedAt":
		return "updated_at"
	case "lastLoginAt":
		return "last_login_at"
	case "role":
		return "role"
	case "status":
		return "status"
	default:
		return "created_at"
	}
}

// Create creates a new user.
func (u *UserRepository) Create(ctx context.Context, user *domain.User) error {
	// Check for existing username
	exists, err := u.ExistsByUsername(ctx, user.Username)
	if err != nil {
		return fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return domain.NewAlreadyExistsError("user", "username", user.Username)
	}

	// Check for existing email
	exists, err = u.ExistsByEmail(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return domain.NewAlreadyExistsError("user", "email", user.Email)
	}

	query := `INSERT INTO users (id, username, email, password_hash, display_name, role, status, 
		avatar_url, last_login_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var displayName, avatarURL sql.NullString
	if user.DisplayName != "" {
		displayName = sql.NullString{String: user.DisplayName, Valid: true}
	}
	if user.AvatarURL != "" {
		avatarURL = sql.NullString{String: user.AvatarURL, Valid: true}
	}

	var lastLoginAt sql.NullTime
	if user.LastLoginAt != nil {
		lastLoginAt = sql.NullTime{Time: user.LastLoginAt.Time, Valid: true}
	}

	_, err = u.repo.db().ExecContext(ctx, query,
		string(user.ID),
		user.Username,
		user.Email,
		user.PasswordHash,
		displayName,
		string(user.Role),
		string(user.Status),
		avatarURL,
		lastLoginAt,
		user.CreatedAt.Time,
		user.UpdatedAt.Time,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Update updates an existing user.
func (u *UserRepository) Update(ctx context.Context, user *domain.User) error {
	// Check if user exists
	exists, err := u.Exists(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if !exists {
		return domain.NewNotFoundError("user", string(user.ID))
	}

	query := `UPDATE users SET username = ?, email = ?, password_hash = ?, display_name = ?, 
		role = ?, status = ?, avatar_url = ?, last_login_at = ?, updated_at = ? WHERE id = ?`

	var displayName, avatarURL sql.NullString
	if user.DisplayName != "" {
		displayName = sql.NullString{String: user.DisplayName, Valid: true}
	}
	if user.AvatarURL != "" {
		avatarURL = sql.NullString{String: user.AvatarURL, Valid: true}
	}

	var lastLoginAt sql.NullTime
	if user.LastLoginAt != nil {
		lastLoginAt = sql.NullTime{Time: user.LastLoginAt.Time, Valid: true}
	}

	_, err = u.repo.db().ExecContext(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		displayName,
		string(user.Role),
		string(user.Status),
		avatarURL,
		lastLoginAt,
		time.Now().UTC(),
		string(user.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete permanently removes a user by their ID.
func (u *UserRepository) Delete(ctx context.Context, id domain.ID) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := u.repo.db().ExecContext(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// SoftDelete marks a user as deleted without removing the record.
func (u *UserRepository) SoftDelete(ctx context.Context, id domain.ID) error {
	query := `UPDATE users SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`

	now := time.Now().UTC()
	result, err := u.repo.db().ExecContext(ctx, query, now, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// Restore restores a soft-deleted user.
func (u *UserRepository) Restore(ctx context.Context, id domain.ID) error {
	query := `UPDATE users SET deleted_at = NULL, updated_at = ? WHERE id = ? AND deleted_at IS NOT NULL`

	result, err := u.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to restore user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// Exists checks if a user with the given ID exists.
func (u *UserRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = ? AND deleted_at IS NULL)`

	var exists bool
	if err := u.repo.db().GetContext(ctx, &exists, query, string(id)); err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

// ExistsByUsername checks if a user with the given username exists.
func (u *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = ? COLLATE NOCASE AND deleted_at IS NULL)`

	var exists bool
	if err := u.repo.db().GetContext(ctx, &exists, query, username); err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return exists, nil
}

// ExistsByEmail checks if a user with the given email exists.
func (u *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = ? COLLATE NOCASE AND deleted_at IS NULL)`

	var exists bool
	if err := u.repo.db().GetContext(ctx, &exists, query, strings.ToLower(email)); err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// Count returns the total number of users matching the filter.
func (u *UserRepository) Count(ctx context.Context, filter *repository.UserFilter) (int64, error) {
	query, args := u.buildListQuery(filter, nil, true)

	var count int64
	if err := u.repo.db().GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// CountByRole returns the count of users grouped by role.
func (u *UserRepository) CountByRole(ctx context.Context) (map[domain.UserRole]int64, error) {
	query := `SELECT role, COUNT(*) as count FROM users WHERE deleted_at IS NULL GROUP BY role`

	type roleCount struct {
		Role  string `db:"role"`
		Count int64  `db:"count"`
	}

	var counts []roleCount
	if err := u.repo.db().SelectContext(ctx, &counts, query); err != nil {
		return nil, fmt.Errorf("failed to count users by role: %w", err)
	}

	result := make(map[domain.UserRole]int64)
	for _, rc := range counts {
		result[domain.UserRole(rc.Role)] = rc.Count
	}

	return result, nil
}

// CountByStatus returns the count of users grouped by status.
func (u *UserRepository) CountByStatus(ctx context.Context) (map[domain.UserStatus]int64, error) {
	query := `SELECT status, COUNT(*) as count FROM users WHERE deleted_at IS NULL GROUP BY status`

	type statusCount struct {
		Status string `db:"status"`
		Count  int64  `db:"count"`
	}

	var counts []statusCount
	if err := u.repo.db().SelectContext(ctx, &counts, query); err != nil {
		return nil, fmt.Errorf("failed to count users by status: %w", err)
	}

	result := make(map[domain.UserStatus]int64)
	for _, sc := range counts {
		result[domain.UserStatus(sc.Status)] = sc.Count
	}

	return result, nil
}

// UpdatePassword updates a user's password hash.
func (u *UserRepository) UpdatePassword(ctx context.Context, id domain.ID, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`

	result, err := u.repo.db().ExecContext(ctx, query, passwordHash, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// UpdateLastLogin updates the user's last login timestamp.
func (u *UserRepository) UpdateLastLogin(ctx context.Context, id domain.ID) error {
	query := `UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`

	now := time.Now().UTC()
	result, err := u.repo.db().ExecContext(ctx, query, now, now, string(id))
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// UpdateStatus updates a user's status.
func (u *UserRepository) UpdateStatus(ctx context.Context, id domain.ID, status domain.UserStatus) error {
	query := `UPDATE users SET status = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`

	result, err := u.repo.db().ExecContext(ctx, query, string(status), time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// UpdateRole updates a user's role.
func (u *UserRepository) UpdateRole(ctx context.Context, id domain.ID, role domain.UserRole) error {
	query := `UPDATE users SET role = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`

	result, err := u.repo.db().ExecContext(ctx, query, string(role), time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// GetActiveUsers retrieves all active users.
func (u *UserRepository) GetActiveUsers(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	status := domain.StatusActive
	filter := &repository.UserFilter{Status: &status}
	return u.List(ctx, filter, opts)
}

// GetAdmins retrieves all admin users.
func (u *UserRepository) GetAdmins(ctx context.Context) ([]*domain.User, error) {
	role := domain.RoleAdmin
	filter := &repository.UserFilter{Role: &role}
	result, err := u.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// Search performs a text search across user fields.
func (u *UserRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	filter := &repository.UserFilter{Search: query}
	return u.List(ctx, filter, opts)
}

// BulkUpdateStatus updates the status of multiple users.
func (u *UserRepository) BulkUpdateStatus(ctx context.Context, ids []domain.ID, status domain.UserStatus) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := u.UpdateStatus(ctx, id, status); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkDelete permanently removes multiple users.
func (u *UserRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := u.Delete(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// GetUsersCreatedBetween retrieves users created within the date range.
func (u *UserRepository) GetUsersCreatedBetween(ctx context.Context, dateRange *repository.DateRangeFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	filter := &repository.UserFilter{}
	if dateRange != nil {
		filter.CreatedAfter = dateRange.From
		filter.CreatedBefore = dateRange.To
	}
	return u.List(ctx, filter, opts)
}

// GetUsersWithRecentLogin retrieves users who logged in within the specified days.
func (u *UserRepository) GetUsersWithRecentLogin(ctx context.Context, days int, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	ts := domain.Timestamp{Time: since}
	filter := &repository.UserFilter{LastLoginAfter: &ts}
	return u.List(ctx, filter, opts)
}

// GetInactiveUsers retrieves users who haven't logged in for the specified days.
func (u *UserRepository) GetInactiveUsers(ctx context.Context, days int, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	before := time.Now().UTC().AddDate(0, 0, -days)
	ts := domain.Timestamp{Time: before}
	filter := &repository.UserFilter{LastLoginBefore: &ts}
	return u.List(ctx, filter, opts)
}

// Ensure UserRepository implements repository.UserRepository
var _ repository.UserRepository = (*UserRepository)(nil)

// Unused import prevention
var _ = json.Marshal
