package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// MailboxRepository implements the repository.MailboxRepository interface for PostgreSQL.
type MailboxRepository struct {
	repo *Repository
}

// mailboxRow is the database representation of a mailbox.
type mailboxRow struct {
	ID            string         `db:"id"`
	UserID        string         `db:"user_id"`
	Name          string         `db:"name"`
	Address       string         `db:"address"`
	Description   sql.NullString `db:"description"`
	IsCatchAll    bool           `db:"is_catch_all"`
	IsDefault     bool           `db:"is_default"`
	Type          string         `db:"mailbox_type"`
	MessageCount  int64          `db:"message_count"`
	UnreadCount   int64          `db:"unread_count"`
	TotalSize     int64          `db:"total_size"`
	RetentionDays int            `db:"retention_days"`
	UIDNext       uint32         `db:"uid_next"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
}

// NewMailboxRepository creates a new PostgreSQL mailbox repository.
func NewMailboxRepository(repo *Repository) *MailboxRepository {
	return &MailboxRepository{repo: repo}
}

// toMailbox converts a mailboxRow to a domain.Mailbox.
func (r *mailboxRow) toMailbox() *domain.Mailbox {
	mailbox := &domain.Mailbox{
		ID:            domain.ID(r.ID),
		UserID:        domain.ID(r.UserID),
		Name:          r.Name,
		Address:       r.Address,
		IsCatchAll:    r.IsCatchAll,
		IsDefault:     r.IsDefault,
		Type:          domain.MailboxType(r.Type),
		MessageCount:  r.MessageCount,
		UnreadCount:   r.UnreadCount,
		TotalSize:     r.TotalSize,
		RetentionDays: r.RetentionDays,
		UIDNext:       r.UIDNext,
		CreatedAt:     domain.Timestamp{Time: r.CreatedAt},
		UpdatedAt:     domain.Timestamp{Time: r.UpdatedAt},
	}

	if r.Description.Valid {
		mailbox.Description = r.Description.String
	}

	return mailbox
}

// GetByID retrieves a mailbox by its unique identifier.
func (m *MailboxRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Mailbox, error) {
	query := `SELECT id, user_id, name, address, description, is_catch_all, is_default, mailbox_type,
		message_count, unread_count, total_size, retention_days, uid_next, created_at, updated_at
		FROM mailboxes WHERE id = $1`

	var row mailboxRow
	if err := m.repo.db().GetContext(ctx, &row, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("mailbox", string(id))
		}
		return nil, fmt.Errorf("failed to get mailbox by ID: %w", err)
	}

	return row.toMailbox(), nil
}

// GetByAddress retrieves a mailbox by its email address.
func (m *MailboxRepository) GetByAddress(ctx context.Context, address string) (*domain.Mailbox, error) {
	query := `SELECT id, user_id, name, address, description, is_catch_all, is_default, mailbox_type,
		message_count, unread_count, total_size, retention_days, uid_next, created_at, updated_at
		FROM mailboxes WHERE LOWER(address) = LOWER($1)`

	var row mailboxRow
	if err := m.repo.db().GetContext(ctx, &row, query, strings.ToLower(address)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("mailbox", address)
		}
		return nil, fmt.Errorf("failed to get mailbox by address: %w", err)
	}

	return row.toMailbox(), nil
}

// GetCatchAll retrieves the catch-all mailbox for a domain.
func (m *MailboxRepository) GetCatchAll(ctx context.Context, domainName string) (*domain.Mailbox, error) {
	query := `SELECT id, user_id, name, address, description, is_catch_all, is_default, mailbox_type,
		message_count, unread_count, total_size, retention_days, uid_next, created_at, updated_at
		FROM mailboxes WHERE is_catch_all = true AND LOWER(address) LIKE LOWER($1)`

	pattern := "%@" + domainName
	var row mailboxRow
	if err := m.repo.db().GetContext(ctx, &row, query, pattern); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("mailbox", "catch-all for "+domainName)
		}
		return nil, fmt.Errorf("failed to get catch-all mailbox: %w", err)
	}

	return row.toMailbox(), nil
}

// GetDefault retrieves the default mailbox for a user.
func (m *MailboxRepository) GetDefault(ctx context.Context, userID domain.ID) (*domain.Mailbox, error) {
	query := `SELECT id, user_id, name, address, description, is_catch_all, is_default, mailbox_type,
		message_count, unread_count, total_size, retention_days, uid_next, created_at, updated_at
		FROM mailboxes WHERE user_id = $1 AND is_default = true`

	var row mailboxRow
	if err := m.repo.db().GetContext(ctx, &row, query, string(userID)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("mailbox", "default for user "+string(userID))
		}
		return nil, fmt.Errorf("failed to get default mailbox: %w", err)
	}

	return row.toMailbox(), nil
}

// List retrieves mailboxes with optional filtering, sorting, and pagination.
func (m *MailboxRepository) List(ctx context.Context, filter *repository.MailboxFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	query, args := m.buildListQuery(filter, opts, false)
	countQuery, countArgs := m.buildListQuery(filter, opts, true)

	var total int64
	if err := m.repo.db().GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, fmt.Errorf("failed to count mailboxes: %w", err)
	}

	var rows []mailboxRow
	if err := m.repo.db().SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list mailboxes: %w", err)
	}

	mailboxes := make([]*domain.Mailbox, len(rows))
	for i, row := range rows {
		mailboxes[i] = row.toMailbox()
	}

	result := &repository.ListResult[*domain.Mailbox]{
		Items: mailboxes,
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

// buildListQuery builds the SQL query for listing mailboxes.
func (m *MailboxRepository) buildListQuery(filter *repository.MailboxFilter, opts *repository.ListOptions, countOnly bool) (string, []interface{}) {
	var sb strings.Builder
	args := make([]interface{}, 0)
	argIndex := 1

	if countOnly {
		sb.WriteString("SELECT COUNT(*) FROM mailboxes WHERE 1=1")
	} else {
		sb.WriteString(`SELECT id, user_id, name, address, description, is_catch_all, is_default, mailbox_type,
			message_count, unread_count, total_size, retention_days, uid_next, created_at, updated_at FROM mailboxes WHERE 1=1`)
	}

	if filter != nil {
		if len(filter.IDs) > 0 {
			placeholders := make([]string, len(filter.IDs))
			for i, id := range filter.IDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, string(id))
				argIndex++
			}
			sb.WriteString(fmt.Sprintf(" AND id IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.UserID != nil {
			sb.WriteString(fmt.Sprintf(" AND user_id = $%d", argIndex))
			args = append(args, string(*filter.UserID))
			argIndex++
		}

		if len(filter.UserIDs) > 0 {
			placeholders := make([]string, len(filter.UserIDs))
			for i, id := range filter.UserIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, string(id))
				argIndex++
			}
			sb.WriteString(fmt.Sprintf(" AND user_id IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.Address != "" {
			sb.WriteString(fmt.Sprintf(" AND LOWER(address) = LOWER($%d)", argIndex))
			args = append(args, filter.Address)
			argIndex++
		}

		if filter.AddressContains != "" {
			sb.WriteString(fmt.Sprintf(" AND LOWER(address) LIKE LOWER($%d)", argIndex))
			args = append(args, "%"+filter.AddressContains+"%")
			argIndex++
		}

		if filter.Domain != "" {
			sb.WriteString(fmt.Sprintf(" AND LOWER(address) LIKE LOWER($%d)", argIndex))
			args = append(args, "%@"+filter.Domain)
			argIndex++
		}

		if filter.IsCatchAll != nil {
			sb.WriteString(fmt.Sprintf(" AND is_catch_all = $%d", argIndex))
			args = append(args, *filter.IsCatchAll)
			argIndex++
		}

		if filter.IsDefault != nil {
			sb.WriteString(fmt.Sprintf(" AND is_default = $%d", argIndex))
			args = append(args, *filter.IsDefault)
			argIndex++
		}

		if filter.HasMessages != nil {
			if *filter.HasMessages {
				sb.WriteString(" AND message_count > 0")
			} else {
				sb.WriteString(" AND message_count = 0")
			}
		}

		if filter.HasUnread != nil {
			if *filter.HasUnread {
				sb.WriteString(" AND unread_count > 0")
			} else {
				sb.WriteString(" AND unread_count = 0")
			}
		}

		if filter.Search != "" {
			sb.WriteString(fmt.Sprintf(" AND (name ILIKE $%d OR address ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1, argIndex+2))
			searchPattern := "%" + filter.Search + "%"
			args = append(args, searchPattern, searchPattern, searchPattern)
			argIndex += 3
		}

		if filter.MinMessageCount != nil {
			sb.WriteString(fmt.Sprintf(" AND message_count >= $%d", argIndex))
			args = append(args, *filter.MinMessageCount)
			argIndex++
		}

		if filter.MaxMessageCount != nil {
			sb.WriteString(fmt.Sprintf(" AND message_count <= $%d", argIndex))
			args = append(args, *filter.MaxMessageCount)
			argIndex++
		}

		if filter.MinSize != nil {
			sb.WriteString(fmt.Sprintf(" AND total_size >= $%d", argIndex))
			args = append(args, *filter.MinSize)
			argIndex++
		}

		if filter.MaxSize != nil {
			sb.WriteString(fmt.Sprintf(" AND total_size <= $%d", argIndex))
			args = append(args, *filter.MaxSize)
			argIndex++
		}

		if filter.CreatedBefore != nil {
			sb.WriteString(fmt.Sprintf(" AND created_at < $%d", argIndex))
			args = append(args, filter.CreatedBefore.Time)
			argIndex++
		}

		if filter.CreatedAfter != nil {
			sb.WriteString(fmt.Sprintf(" AND created_at > $%d", argIndex))
			args = append(args, filter.CreatedAfter.Time)
			argIndex++
		}

		if filter.RetentionDays != nil {
			if *filter.RetentionDays == -1 {
				sb.WriteString(" AND retention_days = 0")
			} else {
				sb.WriteString(fmt.Sprintf(" AND retention_days = $%d", argIndex))
				args = append(args, *filter.RetentionDays)
			}
		}
	}

	if !countOnly {
		if opts != nil && opts.Sort != nil {
			field := m.mapSortField(opts.Sort.Field)
			order := "ASC"
			if opts.Sort.Order == domain.SortDesc {
				order = "DESC"
			}
			sb.WriteString(fmt.Sprintf(" ORDER BY %s %s", field, order))
		} else {
			sb.WriteString(" ORDER BY created_at DESC")
		}

		if opts != nil && opts.Pagination != nil {
			opts.Pagination.Normalize()
			sb.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset()))
		}
	}

	return sb.String(), args
}

// mapSortField maps repository sort field to database column.
func (m *MailboxRepository) mapSortField(field string) string {
	switch field {
	case "name":
		return "name"
	case "address":
		return "address"
	case "messageCount":
		return "message_count"
	case "unreadCount":
		return "unread_count"
	case "totalSize":
		return "total_size"
	case "createdAt":
		return "created_at"
	case "updatedAt":
		return "updated_at"
	default:
		return "created_at"
	}
}

// ListByUser retrieves all mailboxes owned by a specific user.
func (m *MailboxRepository) ListByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	filter := &repository.MailboxFilter{UserID: &userID}
	return m.List(ctx, filter, opts)
}

// Create creates a new mailbox.
func (m *MailboxRepository) Create(ctx context.Context, mailbox *domain.Mailbox) error {
	exists, err := m.ExistsByAddress(ctx, mailbox.Address)
	if err != nil {
		return fmt.Errorf("failed to check address existence: %w", err)
	}
	if exists {
		return domain.NewAlreadyExistsError("mailbox", "address", mailbox.Address)
	}

	if mailbox.UIDNext == 0 {
		mailbox.UIDNext = 1
	}

	query := `INSERT INTO mailboxes (id, user_id, name, address, description, is_catch_all, is_default,
		message_count, unread_count, total_size, retention_days, uid_next, created_at, updated_at, mailbox_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	var description sql.NullString
	if mailbox.Description != "" {
		description = sql.NullString{String: mailbox.Description, Valid: true}
	}

	_, err = m.repo.db().ExecContext(ctx, query,
		string(mailbox.ID),
		string(mailbox.UserID),
		mailbox.Name,
		strings.ToLower(mailbox.Address),
		description,
		mailbox.IsCatchAll,
		mailbox.IsDefault,
		mailbox.MessageCount,
		mailbox.UnreadCount,
		mailbox.TotalSize,
		mailbox.RetentionDays,
		mailbox.UIDNext,
		mailbox.CreatedAt.Time,
		mailbox.UpdatedAt.Time,
		string(mailbox.Type),
	)
	if err != nil {
		return fmt.Errorf("failed to create mailbox: %w", err)
	}

	return nil
}

// Update updates an existing mailbox.
func (m *MailboxRepository) Update(ctx context.Context, mailbox *domain.Mailbox) error {
	exists, err := m.Exists(ctx, mailbox.ID)
	if err != nil {
		return fmt.Errorf("failed to check mailbox existence: %w", err)
	}
	if !exists {
		return domain.NewNotFoundError("mailbox", string(mailbox.ID))
	}

	query := `UPDATE mailboxes SET user_id = $1, name = $2, address = $3, description = $4,
		is_catch_all = $5, is_default = $6, mailbox_type = $7, message_count = $8, unread_count = $9,
		total_size = $10, retention_days = $11, updated_at = $12 WHERE id = $13`

	var description sql.NullString
	if mailbox.Description != "" {
		description = sql.NullString{String: mailbox.Description, Valid: true}
	}

	_, err = m.repo.db().ExecContext(ctx, query,
		string(mailbox.UserID),
		mailbox.Name,
		strings.ToLower(mailbox.Address),
		description,
		mailbox.IsCatchAll,
		mailbox.IsDefault,
		string(mailbox.Type),
		mailbox.MessageCount,
		mailbox.UnreadCount,
		mailbox.TotalSize,
		mailbox.RetentionDays,
		time.Now().UTC(),
		string(mailbox.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to update mailbox: %w", err)
	}

	return nil
}

// Delete permanently removes a mailbox by its ID.
func (m *MailboxRepository) Delete(ctx context.Context, id domain.ID) error {
	query := `DELETE FROM mailboxes WHERE id = $1`

	result, err := m.repo.db().ExecContext(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("failed to delete mailbox: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// DeleteWithMessages removes a mailbox and all its messages.
func (m *MailboxRepository) DeleteWithMessages(ctx context.Context, id domain.ID) error {
	// Due to CASCADE, deleting mailbox will delete messages and attachments
	return m.Delete(ctx, id)
}

// DeleteByUser removes all mailboxes owned by a user.
func (m *MailboxRepository) DeleteByUser(ctx context.Context, userID domain.ID) (int64, error) {
	query := `DELETE FROM mailboxes WHERE user_id = $1`

	result, err := m.repo.db().ExecContext(ctx, query, string(userID))
	if err != nil {
		return 0, fmt.Errorf("failed to delete mailboxes by user: %w", err)
	}

	return result.RowsAffected()
}

// Exists checks if a mailbox with the given ID exists.
func (m *MailboxRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM mailboxes WHERE id = $1)`

	var exists bool
	if err := m.repo.db().GetContext(ctx, &exists, query, string(id)); err != nil {
		return false, fmt.Errorf("failed to check mailbox existence: %w", err)
	}

	return exists, nil
}

// ExistsByAddress checks if a mailbox with the given address exists.
func (m *MailboxRepository) ExistsByAddress(ctx context.Context, address string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM mailboxes WHERE LOWER(address) = LOWER($1))`

	var exists bool
	if err := m.repo.db().GetContext(ctx, &exists, query, strings.ToLower(address)); err != nil {
		return false, fmt.Errorf("failed to check address existence: %w", err)
	}

	return exists, nil
}

// Count returns the total number of mailboxes matching the filter.
func (m *MailboxRepository) Count(ctx context.Context, filter *repository.MailboxFilter) (int64, error) {
	query, args := m.buildListQuery(filter, nil, true)

	var count int64
	if err := m.repo.db().GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count mailboxes: %w", err)
	}

	return count, nil
}

// CountByUser returns the number of mailboxes owned by a user.
func (m *MailboxRepository) CountByUser(ctx context.Context, userID domain.ID) (int64, error) {
	query := `SELECT COUNT(*) FROM mailboxes WHERE user_id = $1`

	var count int64
	if err := m.repo.db().GetContext(ctx, &count, query, string(userID)); err != nil {
		return 0, fmt.Errorf("failed to count mailboxes by user: %w", err)
	}

	return count, nil
}

// SetDefault sets a mailbox as the default for its owner.
func (m *MailboxRepository) SetDefault(ctx context.Context, id domain.ID) error {
	mailbox, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Clear current default for the user
	if err := m.ClearDefault(ctx, mailbox.UserID); err != nil {
		return err
	}

	query := `UPDATE mailboxes SET is_default = true, updated_at = $1 WHERE id = $2`
	_, err = m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to set default mailbox: %w", err)
	}

	return nil
}

// ClearDefault removes the default flag from all mailboxes for a user.
func (m *MailboxRepository) ClearDefault(ctx context.Context, userID domain.ID) error {
	query := `UPDATE mailboxes SET is_default = false, updated_at = $1 WHERE user_id = $2 AND is_default = true`
	_, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(userID))
	if err != nil {
		return fmt.Errorf("failed to clear default mailbox: %w", err)
	}

	return nil
}

// SetCatchAll sets a mailbox as the catch-all for its domain.
func (m *MailboxRepository) SetCatchAll(ctx context.Context, id domain.ID) error {
	mailbox, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	domainName := mailbox.GetDomain()

	// Clear existing catch-all for the domain
	clearQuery := `UPDATE mailboxes SET is_catch_all = false, updated_at = $1 
		WHERE is_catch_all = true AND LOWER(address) LIKE LOWER($2)`
	_, err = m.repo.db().ExecContext(ctx, clearQuery, time.Now().UTC(), "%@"+domainName)
	if err != nil {
		return fmt.Errorf("failed to clear catch-all: %w", err)
	}

	// Set new catch-all
	query := `UPDATE mailboxes SET is_catch_all = true, updated_at = $1 WHERE id = $2`
	_, err = m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to set catch-all: %w", err)
	}

	return nil
}

// ClearCatchAll removes the catch-all flag from a mailbox.
func (m *MailboxRepository) ClearCatchAll(ctx context.Context, id domain.ID) error {
	query := `UPDATE mailboxes SET is_catch_all = false, updated_at = $1 WHERE id = $2`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to clear catch-all: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// UpdateStats updates the mailbox statistics.
func (m *MailboxRepository) UpdateStats(ctx context.Context, id domain.ID, stats *repository.MailboxStatsUpdate) error {
	var updates []string
	args := make([]interface{}, 0)
	argIndex := 1

	if stats.MessageCount != nil {
		updates = append(updates, fmt.Sprintf("message_count = $%d", argIndex))
		args = append(args, *stats.MessageCount)
		argIndex++
	}
	if stats.UnreadCount != nil {
		updates = append(updates, fmt.Sprintf("unread_count = $%d", argIndex))
		args = append(args, *stats.UnreadCount)
		argIndex++
	}
	if stats.TotalSize != nil {
		updates = append(updates, fmt.Sprintf("total_size = $%d", argIndex))
		args = append(args, *stats.TotalSize)
		argIndex++
	}

	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now().UTC())
	argIndex++
	args = append(args, string(id))

	query := fmt.Sprintf("UPDATE mailboxes SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)

	result, err := m.repo.db().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update mailbox stats: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// IncrementMessageCount atomically increments message counters and assigns the next IMAP UID.
func (m *MailboxRepository) IncrementMessageCount(ctx context.Context, id domain.ID, size int64) (uint32, error) {
	query := `UPDATE mailboxes SET message_count = message_count + 1,
		unread_count = unread_count + 1, total_size = total_size + $1,
		uid_next = uid_next + 1, updated_at = $2
		WHERE id = $3
		RETURNING uid_next - 1`

	var assignedUID uint32
	err := m.repo.db().QueryRowContext(ctx, query, size, time.Now().UTC(), string(id)).Scan(&assignedUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.NewNotFoundError("mailbox", string(id))
		}
		return 0, fmt.Errorf("failed to increment message count: %w", err)
	}

	return assignedUID, nil
}

// DecrementMessageCount atomically decrements message counters.
func (m *MailboxRepository) DecrementMessageCount(ctx context.Context, id domain.ID, size int64, wasUnread bool) error {
	var query string
	if wasUnread {
		query = `UPDATE mailboxes SET message_count = GREATEST(0, message_count - 1), 
			unread_count = GREATEST(0, unread_count - 1), total_size = GREATEST(0, total_size - $1), updated_at = $2 WHERE id = $3`
	} else {
		query = `UPDATE mailboxes SET message_count = GREATEST(0, message_count - 1), 
			total_size = GREATEST(0, total_size - $1), updated_at = $2 WHERE id = $3`
	}

	result, err := m.repo.db().ExecContext(ctx, query, size, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to decrement message count: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// UpdateUnreadCount atomically updates the unread count.
func (m *MailboxRepository) UpdateUnreadCount(ctx context.Context, id domain.ID, delta int) error {
	var query string
	if delta >= 0 {
		query = `UPDATE mailboxes SET unread_count = unread_count + $1, updated_at = $2 WHERE id = $3`
	} else {
		query = `UPDATE mailboxes SET unread_count = GREATEST(0, unread_count + $1), updated_at = $2 WHERE id = $3`
	}

	result, err := m.repo.db().ExecContext(ctx, query, delta, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to update unread count: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// RecalculateStats recalculates mailbox statistics from messages.
func (m *MailboxRepository) RecalculateStats(ctx context.Context, id domain.ID) error {
	statsQuery := `SELECT COUNT(*) as message_count, 
		SUM(CASE WHEN status = 'unread' THEN 1 ELSE 0 END) as unread_count,
		COALESCE(SUM(size), 0) as total_size
		FROM messages WHERE mailbox_id = $1`

	var stats struct {
		MessageCount int64 `db:"message_count"`
		UnreadCount  int64 `db:"unread_count"`
		TotalSize    int64 `db:"total_size"`
	}

	if err := m.repo.db().GetContext(ctx, &stats, statsQuery, string(id)); err != nil {
		return fmt.Errorf("failed to calculate stats: %w", err)
	}

	updateQuery := `UPDATE mailboxes SET message_count = $1, unread_count = $2, 
		total_size = $3, updated_at = $4 WHERE id = $5`

	result, err := m.repo.db().ExecContext(ctx, updateQuery,
		stats.MessageCount, stats.UnreadCount, stats.TotalSize, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to update stats: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("mailbox", string(id))
	}

	return nil
}

// GetStats retrieves detailed statistics for a mailbox.
func (m *MailboxRepository) GetStats(ctx context.Context, id domain.ID) (*domain.MailboxStats, error) {
	mailbox, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	stats := &domain.MailboxStats{
		TotalMessages:  mailbox.MessageCount,
		UnreadMessages: mailbox.UnreadCount,
		TotalSize:      mailbox.TotalSize,
	}

	// Get oldest and newest message timestamps
	timeQuery := `SELECT MIN(received_at) as oldest, MAX(received_at) as newest 
		FROM messages WHERE mailbox_id = $1`

	var times struct {
		Oldest sql.NullTime `db:"oldest"`
		Newest sql.NullTime `db:"newest"`
	}

	if err := m.repo.db().GetContext(ctx, &times, timeQuery, string(id)); err == nil {
		if times.Oldest.Valid {
			ts := domain.Timestamp{Time: times.Oldest.Time}
			stats.OldestMessage = &ts
		}
		if times.Newest.Valid {
			ts := domain.Timestamp{Time: times.Newest.Time}
			stats.NewestMessage = &ts
		}
	}

	return stats, nil
}

// GetStatsByUser retrieves aggregated statistics for all mailboxes owned by a user.
func (m *MailboxRepository) GetStatsByUser(ctx context.Context, userID domain.ID) (*domain.MailboxStats, error) {
	query := `SELECT COALESCE(SUM(message_count), 0) as total_messages,
		COALESCE(SUM(unread_count), 0) as unread_messages,
		COALESCE(SUM(total_size), 0) as total_size
		FROM mailboxes WHERE user_id = $1`

	var stats struct {
		TotalMessages  int64 `db:"total_messages"`
		UnreadMessages int64 `db:"unread_messages"`
		TotalSize      int64 `db:"total_size"`
	}

	if err := m.repo.db().GetContext(ctx, &stats, query, string(userID)); err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return &domain.MailboxStats{
		TotalMessages:  stats.TotalMessages,
		UnreadMessages: stats.UnreadMessages,
		TotalSize:      stats.TotalSize,
	}, nil
}

// GetTotalStats retrieves aggregated statistics for all mailboxes.
func (m *MailboxRepository) GetTotalStats(ctx context.Context) (*domain.MailboxStats, error) {
	query := `SELECT COALESCE(SUM(message_count), 0) as total_messages,
		COALESCE(SUM(unread_count), 0) as unread_messages,
		COALESCE(SUM(total_size), 0) as total_size
		FROM mailboxes`

	var stats struct {
		TotalMessages  int64 `db:"total_messages"`
		UnreadMessages int64 `db:"unread_messages"`
		TotalSize      int64 `db:"total_size"`
	}

	if err := m.repo.db().GetContext(ctx, &stats, query); err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	return &domain.MailboxStats{
		TotalMessages:  stats.TotalMessages,
		UnreadMessages: stats.UnreadMessages,
		TotalSize:      stats.TotalSize,
	}, nil
}

// FindMatchingMailbox finds the mailbox that should receive a message for the given address.
func (m *MailboxRepository) FindMatchingMailbox(ctx context.Context, address string) (*domain.Mailbox, error) {
	// Try exact match first
	mailbox, err := m.GetByAddress(ctx, address)
	if err == nil {
		return mailbox, nil
	}
	if !domain.IsNotFound(err) {
		return nil, err
	}

	// Try catch-all for the domain
	parts := strings.Split(address, "@")
	if len(parts) == 2 {
		mailbox, err = m.GetCatchAll(ctx, parts[1])
		if err == nil {
			return mailbox, nil
		}
		if !domain.IsNotFound(err) {
			return nil, err
		}
	}

	return nil, domain.NewNotFoundError("mailbox", address)
}

// Search performs a text search across mailbox fields.
func (m *MailboxRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	filter := &repository.MailboxFilter{Search: query}
	return m.List(ctx, filter, opts)
}

// GetMailboxesWithMessages retrieves mailboxes that have at least one message.
func (m *MailboxRepository) GetMailboxesWithMessages(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	hasMessages := true
	filter := &repository.MailboxFilter{HasMessages: &hasMessages}
	return m.List(ctx, filter, opts)
}

// GetMailboxesWithUnread retrieves mailboxes that have unread messages.
func (m *MailboxRepository) GetMailboxesWithUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	hasUnread := true
	filter := &repository.MailboxFilter{HasUnread: &hasUnread}
	return m.List(ctx, filter, opts)
}

// TransferOwnership transfers all mailboxes from one user to another.
func (m *MailboxRepository) TransferOwnership(ctx context.Context, fromUserID, toUserID domain.ID) (int64, error) {
	query := `UPDATE mailboxes SET user_id = $1, updated_at = $2 WHERE user_id = $3`

	result, err := m.repo.db().ExecContext(ctx, query, string(toUserID), time.Now().UTC(), string(fromUserID))
	if err != nil {
		return 0, fmt.Errorf("failed to transfer ownership: %w", err)
	}

	return result.RowsAffected()
}

// BulkDelete permanently removes multiple mailboxes.
func (m *MailboxRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := m.Delete(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// GetDomains retrieves all unique domains from mailbox addresses.
func (m *MailboxRepository) GetDomains(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT SPLIT_PART(address, '@', 2) as domain FROM mailboxes ORDER BY domain`

	var domains []string
	if err := m.repo.db().SelectContext(ctx, &domains, query); err != nil {
		return nil, fmt.Errorf("failed to get domains: %w", err)
	}

	return domains, nil
}

// GetMailboxesByDomain retrieves all mailboxes for a specific domain.
func (m *MailboxRepository) GetMailboxesByDomain(ctx context.Context, domainName string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	filter := &repository.MailboxFilter{Domain: domainName}
	return m.List(ctx, filter, opts)
}

// Ensure MailboxRepository implements repository.MailboxRepository
var _ repository.MailboxRepository = (*MailboxRepository)(nil)
