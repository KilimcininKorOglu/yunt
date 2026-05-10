package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// AttachmentRepository implements the repository.AttachmentRepository interface for PostgreSQL.
type AttachmentRepository struct {
	repo *Repository
}

// attachmentRow is the database representation of an attachment.
type attachmentRow struct {
	ID          string         `db:"id"`
	MessageID   string         `db:"message_id"`
	Filename    string         `db:"filename"`
	ContentType string         `db:"content_type"`
	Size        int64          `db:"size"`
	ContentID   sql.NullString `db:"content_id"`
	Disposition string         `db:"disposition"`
	StoragePath sql.NullString `db:"storage_path"`
	Checksum    sql.NullString `db:"checksum"`
	IsInline    bool           `db:"is_inline"`
	CreatedAt   time.Time      `db:"created_at"`
}

// NewAttachmentRepository creates a new PostgreSQL attachment repository.
func NewAttachmentRepository(repo *Repository) *AttachmentRepository {
	return &AttachmentRepository{repo: repo}
}

// toAttachment converts an attachmentRow to a domain.Attachment.
func (r *attachmentRow) toAttachment() *domain.Attachment {
	att := &domain.Attachment{
		ID:          domain.ID(r.ID),
		MessageID:   domain.ID(r.MessageID),
		Filename:    r.Filename,
		ContentType: r.ContentType,
		Size:        r.Size,
		Disposition: domain.AttachmentDisposition(r.Disposition),
		IsInline:    r.IsInline,
		CreatedAt:   domain.Timestamp{Time: r.CreatedAt},
	}

	if r.ContentID.Valid {
		att.ContentID = r.ContentID.String
	}
	if r.StoragePath.Valid {
		att.StoragePath = r.StoragePath.String
	}
	if r.Checksum.Valid {
		att.Checksum = r.Checksum.String
	}

	return att
}

// GetByID retrieves an attachment by its unique identifier.
func (a *AttachmentRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Attachment, error) {
	query := `SELECT id, message_id, filename, content_type, size, content_id, 
		disposition, storage_path, checksum, is_inline, created_at 
		FROM attachments WHERE id = $1`

	var row attachmentRow
	if err := a.repo.db().GetContext(ctx, &row, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("attachment", string(id))
		}
		return nil, fmt.Errorf("failed to get attachment by ID: %w", err)
	}

	return row.toAttachment(), nil
}

// GetByContentID retrieves an attachment by its Content-ID.
func (a *AttachmentRepository) GetByContentID(ctx context.Context, contentID string) (*domain.Attachment, error) {
	query := `SELECT id, message_id, filename, content_type, size, content_id, 
		disposition, storage_path, checksum, is_inline, created_at 
		FROM attachments WHERE content_id = $1`

	var row attachmentRow
	if err := a.repo.db().GetContext(ctx, &row, query, contentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("attachment", contentID)
		}
		return nil, fmt.Errorf("failed to get attachment by Content-ID: %w", err)
	}

	return row.toAttachment(), nil
}

// List retrieves attachments with optional filtering, sorting, and pagination.
func (a *AttachmentRepository) List(ctx context.Context, filter *repository.AttachmentFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	query, args := a.buildListQuery(filter, opts, false)
	countQuery, countArgs := a.buildListQuery(filter, opts, true)

	var total int64
	if err := a.repo.db().GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, fmt.Errorf("failed to count attachments: %w", err)
	}

	var rows []attachmentRow
	if err := a.repo.db().SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}

	attachments := make([]*domain.Attachment, len(rows))
	for i, row := range rows {
		attachments[i] = row.toAttachment()
	}

	result := &repository.ListResult[*domain.Attachment]{
		Items: attachments,
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

// buildListQuery builds the SQL query for listing attachments.
func (a *AttachmentRepository) buildListQuery(filter *repository.AttachmentFilter, opts *repository.ListOptions, countOnly bool) (string, []interface{}) {
	var sb strings.Builder
	args := make([]interface{}, 0)
	argIndex := 1

	if countOnly {
		sb.WriteString("SELECT COUNT(*) FROM attachments WHERE 1=1")
	} else {
		sb.WriteString(`SELECT id, message_id, filename, content_type, size, content_id, 
			disposition, storage_path, checksum, is_inline, created_at FROM attachments WHERE 1=1`)
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

		if filter.MessageID != nil {
			sb.WriteString(fmt.Sprintf(" AND message_id = $%d", argIndex))
			args = append(args, string(*filter.MessageID))
			argIndex++
		}

		if len(filter.MessageIDs) > 0 {
			placeholders := make([]string, len(filter.MessageIDs))
			for i, id := range filter.MessageIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, string(id))
				argIndex++
			}
			sb.WriteString(fmt.Sprintf(" AND message_id IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.IsInline != nil {
			sb.WriteString(fmt.Sprintf(" AND is_inline = $%d", argIndex))
			args = append(args, *filter.IsInline)
			argIndex++
		}

		if filter.ContentType != "" {
			sb.WriteString(fmt.Sprintf(" AND content_type = $%d", argIndex))
			args = append(args, filter.ContentType)
			argIndex++
		}

		if filter.ContentTypePrefix != "" {
			sb.WriteString(fmt.Sprintf(" AND content_type LIKE $%d", argIndex))
			args = append(args, filter.ContentTypePrefix+"%")
			argIndex++
		}

		if filter.Filename != "" {
			sb.WriteString(fmt.Sprintf(" AND filename = $%d", argIndex))
			args = append(args, filter.Filename)
			argIndex++
		}

		if filter.FilenameContains != "" {
			sb.WriteString(fmt.Sprintf(" AND filename ILIKE $%d", argIndex))
			args = append(args, "%"+filter.FilenameContains+"%")
			argIndex++
		}

		if filter.Extension != "" {
			sb.WriteString(fmt.Sprintf(" AND filename ILIKE $%d", argIndex))
			args = append(args, "%."+filter.Extension)
			argIndex++
		}

		if len(filter.Extensions) > 0 {
			patterns := make([]string, len(filter.Extensions))
			for i, ext := range filter.Extensions {
				patterns[i] = fmt.Sprintf("filename ILIKE $%d", argIndex)
				args = append(args, "%."+ext)
				argIndex++
			}
			sb.WriteString(fmt.Sprintf(" AND (%s)", strings.Join(patterns, " OR ")))
		}

		if filter.MinSize != nil {
			sb.WriteString(fmt.Sprintf(" AND size >= $%d", argIndex))
			args = append(args, *filter.MinSize)
			argIndex++
		}

		if filter.MaxSize != nil {
			sb.WriteString(fmt.Sprintf(" AND size <= $%d", argIndex))
			args = append(args, *filter.MaxSize)
			argIndex++
		}

		if filter.Checksum != "" {
			sb.WriteString(fmt.Sprintf(" AND checksum = $%d", argIndex))
			args = append(args, filter.Checksum)
			argIndex++
		}

		if filter.CreatedAfter != nil {
			sb.WriteString(fmt.Sprintf(" AND created_at > $%d", argIndex))
			args = append(args, filter.CreatedAfter.Time)
			argIndex++
		}

		if filter.CreatedBefore != nil {
			sb.WriteString(fmt.Sprintf(" AND created_at < $%d", argIndex))
			args = append(args, filter.CreatedBefore.Time)
		}
	}

	if !countOnly {
		if opts != nil && opts.Sort != nil {
			field := a.mapSortField(opts.Sort.Field)
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
func (a *AttachmentRepository) mapSortField(field string) string {
	switch field {
	case "filename":
		return "filename"
	case "size":
		return "size"
	case "contentType":
		return "content_type"
	case "createdAt":
		return "created_at"
	default:
		return "created_at"
	}
}

// ListByMessage retrieves all attachments for a specific message.
func (a *AttachmentRepository) ListByMessage(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	query := `SELECT id, message_id, filename, content_type, size, content_id, 
		disposition, storage_path, checksum, is_inline, created_at 
		FROM attachments WHERE message_id = $1 ORDER BY created_at`

	var rows []attachmentRow
	if err := a.repo.db().SelectContext(ctx, &rows, query, string(messageID)); err != nil {
		return nil, fmt.Errorf("failed to list attachments by message: %w", err)
	}

	attachments := make([]*domain.Attachment, len(rows))
	for i, row := range rows {
		attachments[i] = row.toAttachment()
	}

	return attachments, nil
}

// ListByMessages retrieves attachments for multiple messages.
func (a *AttachmentRepository) ListByMessages(ctx context.Context, messageIDs []domain.ID) (map[domain.ID][]*domain.Attachment, error) {
	if len(messageIDs) == 0 {
		return make(map[domain.ID][]*domain.Attachment), nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = string(id)
	}

	query := fmt.Sprintf(`SELECT id, message_id, filename, content_type, size, content_id, 
		disposition, storage_path, checksum, is_inline, created_at 
		FROM attachments WHERE message_id IN (%s) ORDER BY created_at`, strings.Join(placeholders, ","))

	var rows []attachmentRow
	if err := a.repo.db().SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list attachments by messages: %w", err)
	}

	result := make(map[domain.ID][]*domain.Attachment)
	for _, row := range rows {
		att := row.toAttachment()
		result[att.MessageID] = append(result[att.MessageID], att)
	}

	return result, nil
}

// ListSummaries retrieves attachment summaries for faster list rendering.
func (a *AttachmentRepository) ListSummaries(ctx context.Context, filter *repository.AttachmentFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.AttachmentSummary], error) {
	result, err := a.List(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.AttachmentSummary, len(result.Items))
	for i, att := range result.Items {
		summaries[i] = att.ToSummary()
	}

	return &repository.ListResult[*domain.AttachmentSummary]{
		Items:      summaries,
		Total:      result.Total,
		HasMore:    result.HasMore,
		Pagination: result.Pagination,
	}, nil
}

// ListSummariesByMessage retrieves attachment summaries for a specific message.
func (a *AttachmentRepository) ListSummariesByMessage(ctx context.Context, messageID domain.ID) ([]*domain.AttachmentSummary, error) {
	attachments, err := a.ListByMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.AttachmentSummary, len(attachments))
	for i, att := range attachments {
		summaries[i] = att.ToSummary()
	}

	return summaries, nil
}

// Create creates a new attachment record.
func (a *AttachmentRepository) Create(ctx context.Context, attachment *domain.Attachment) error {
	query := `INSERT INTO attachments (id, message_id, filename, content_type, size, 
		content_id, disposition, storage_path, checksum, is_inline, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	var contentID, storagePath, checksum sql.NullString
	if attachment.ContentID != "" {
		contentID = sql.NullString{String: attachment.ContentID, Valid: true}
	}
	if attachment.StoragePath != "" {
		storagePath = sql.NullString{String: attachment.StoragePath, Valid: true}
	}
	if attachment.Checksum != "" {
		checksum = sql.NullString{String: attachment.Checksum, Valid: true}
	}

	_, err := a.repo.db().ExecContext(ctx, query,
		string(attachment.ID),
		string(attachment.MessageID),
		attachment.Filename,
		attachment.ContentType,
		attachment.Size,
		contentID,
		string(attachment.Disposition),
		storagePath,
		checksum,
		attachment.IsInline,
		attachment.CreatedAt.Time,
	)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	return nil
}

// CreateWithContent creates an attachment and stores its content.
func (a *AttachmentRepository) CreateWithContent(ctx context.Context, attachment *domain.Attachment, content io.Reader) error {
	if err := a.Create(ctx, attachment); err != nil {
		return err
	}

	return a.StoreContent(ctx, attachment.ID, content)
}

// Update updates an existing attachment's metadata.
func (a *AttachmentRepository) Update(ctx context.Context, attachment *domain.Attachment) error {
	exists, err := a.Exists(ctx, attachment.ID)
	if err != nil {
		return err
	}
	if !exists {
		return domain.NewNotFoundError("attachment", string(attachment.ID))
	}

	query := `UPDATE attachments SET message_id = $1, filename = $2, content_type = $3, 
		size = $4, content_id = $5, disposition = $6, storage_path = $7, checksum = $8, 
		is_inline = $9 WHERE id = $10`

	var contentID, storagePath, checksum sql.NullString
	if attachment.ContentID != "" {
		contentID = sql.NullString{String: attachment.ContentID, Valid: true}
	}
	if attachment.StoragePath != "" {
		storagePath = sql.NullString{String: attachment.StoragePath, Valid: true}
	}
	if attachment.Checksum != "" {
		checksum = sql.NullString{String: attachment.Checksum, Valid: true}
	}

	_, err = a.repo.db().ExecContext(ctx, query,
		string(attachment.MessageID),
		attachment.Filename,
		attachment.ContentType,
		attachment.Size,
		contentID,
		string(attachment.Disposition),
		storagePath,
		checksum,
		attachment.IsInline,
		string(attachment.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to update attachment: %w", err)
	}

	return nil
}

// Delete removes an attachment and its content.
func (a *AttachmentRepository) Delete(ctx context.Context, id domain.ID) error {
	// Delete content first
	if _, err := a.repo.db().ExecContext(ctx, "DELETE FROM attachment_content WHERE attachment_id = $1", string(id)); err != nil {
		return fmt.Errorf("failed to delete attachment content: %w", err)
	}

	// Delete attachment record
	result, err := a.repo.db().ExecContext(ctx, "DELETE FROM attachments WHERE id = $1", string(id))
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("attachment", string(id))
	}

	return nil
}

// DeleteByMessage removes all attachments for a message.
func (a *AttachmentRepository) DeleteByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	// Delete content first
	deleteContentQuery := `DELETE FROM attachment_content WHERE attachment_id IN 
		(SELECT id FROM attachments WHERE message_id = $1)`
	if _, err := a.repo.db().ExecContext(ctx, deleteContentQuery, string(messageID)); err != nil {
		return 0, fmt.Errorf("failed to delete attachment content: %w", err)
	}

	// Delete attachments
	result, err := a.repo.db().ExecContext(ctx, "DELETE FROM attachments WHERE message_id = $1", string(messageID))
	if err != nil {
		return 0, fmt.Errorf("failed to delete attachments by message: %w", err)
	}

	return result.RowsAffected()
}

// DeleteByMessages removes all attachments for multiple messages.
func (a *AttachmentRepository) DeleteByMessages(ctx context.Context, messageIDs []domain.ID) (int64, error) {
	if len(messageIDs) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = string(id)
	}

	// Delete content first
	deleteContentQuery := fmt.Sprintf(`DELETE FROM attachment_content WHERE attachment_id IN 
		(SELECT id FROM attachments WHERE message_id IN (%s))`, strings.Join(placeholders, ","))
	if _, err := a.repo.db().ExecContext(ctx, deleteContentQuery, args...); err != nil {
		return 0, fmt.Errorf("failed to delete attachment content: %w", err)
	}

	// Delete attachments
	deleteQuery := fmt.Sprintf("DELETE FROM attachments WHERE message_id IN (%s)", strings.Join(placeholders, ","))
	result, err := a.repo.db().ExecContext(ctx, deleteQuery, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete attachments by messages: %w", err)
	}

	return result.RowsAffected()
}

// Exists checks if an attachment with the given ID exists.
func (a *AttachmentRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM attachments WHERE id = $1)`

	var exists bool
	if err := a.repo.db().GetContext(ctx, &exists, query, string(id)); err != nil {
		return false, fmt.Errorf("failed to check attachment existence: %w", err)
	}

	return exists, nil
}

// ExistsByContentID checks if an attachment with the given Content-ID exists.
func (a *AttachmentRepository) ExistsByContentID(ctx context.Context, contentID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM attachments WHERE content_id = $1)`

	var exists bool
	if err := a.repo.db().GetContext(ctx, &exists, query, contentID); err != nil {
		return false, fmt.Errorf("failed to check Content-ID existence: %w", err)
	}

	return exists, nil
}

// Count returns the total number of attachments matching the filter.
func (a *AttachmentRepository) Count(ctx context.Context, filter *repository.AttachmentFilter) (int64, error) {
	query, args := a.buildListQuery(filter, nil, true)

	var count int64
	if err := a.repo.db().GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count attachments: %w", err)
	}

	return count, nil
}

// CountByMessage returns the number of attachments for a message.
func (a *AttachmentRepository) CountByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	query := `SELECT COUNT(*) FROM attachments WHERE message_id = $1`

	var count int64
	if err := a.repo.db().GetContext(ctx, &count, query, string(messageID)); err != nil {
		return 0, fmt.Errorf("failed to count attachments by message: %w", err)
	}

	return count, nil
}

// StoreContent stores the content of an attachment.
func (a *AttachmentRepository) StoreContent(ctx context.Context, id domain.ID, content io.Reader) error {
	exists, err := a.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domain.NewNotFoundError("attachment", string(id))
	}

	data, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	query := `INSERT INTO attachment_content (attachment_id, content) VALUES ($1, $2)
		ON CONFLICT (attachment_id) DO UPDATE SET content = EXCLUDED.content`
	if _, err := a.repo.db().ExecContext(ctx, query, string(id), data); err != nil {
		return fmt.Errorf("failed to store content: %w", err)
	}

	return nil
}

// GetContent retrieves the content of an attachment.
func (a *AttachmentRepository) GetContent(ctx context.Context, id domain.ID) (io.ReadCloser, error) {
	query := `SELECT content FROM attachment_content WHERE attachment_id = $1`

	var content []byte
	if err := a.repo.db().GetContext(ctx, &content, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("attachment content", string(id))
		}
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	return io.NopCloser(bytes.NewReader(content)), nil
}

// GetContentWithMetadata retrieves both the attachment metadata and content.
func (a *AttachmentRepository) GetContentWithMetadata(ctx context.Context, id domain.ID) (*domain.Attachment, io.ReadCloser, error) {
	attachment, err := a.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	content, err := a.GetContent(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return attachment, content, nil
}

// GetContentSize retrieves the size of the attachment content.
func (a *AttachmentRepository) GetContentSize(ctx context.Context, id domain.ID) (int64, error) {
	query := `SELECT LENGTH(content) FROM attachment_content WHERE attachment_id = $1`

	var size int64
	if err := a.repo.db().GetContext(ctx, &size, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.NewNotFoundError("attachment content", string(id))
		}
		return 0, fmt.Errorf("failed to get content size: %w", err)
	}

	return size, nil
}

// VerifyContent verifies the integrity of the attachment content.
func (a *AttachmentRepository) VerifyContent(ctx context.Context, id domain.ID) (bool, error) {
	// Get stored checksum
	attachment, err := a.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	if attachment.Checksum == "" {
		return true, nil // No checksum to verify
	}

	// Get content and calculate checksum
	content, err := a.GetContent(ctx, id)
	if err != nil {
		return false, err
	}
	defer content.Close()

	data, err := io.ReadAll(content)
	if err != nil {
		return false, fmt.Errorf("failed to read content for verification: %w", err)
	}

	// Simple size check (for real implementation, use crypto hash)
	storedSize := attachment.Size
	actualSize := int64(len(data))

	return storedSize == actualSize, nil
}

// GetTotalSize calculates the total size of all attachments.
func (a *AttachmentRepository) GetTotalSize(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(SUM(size), 0) FROM attachments`

	var size int64
	if err := a.repo.db().GetContext(ctx, &size, query); err != nil {
		return 0, fmt.Errorf("failed to get total size: %w", err)
	}

	return size, nil
}

// GetTotalSizeByMessage calculates the total size of attachments for a message.
func (a *AttachmentRepository) GetTotalSizeByMessage(ctx context.Context, messageID domain.ID) (int64, error) {
	query := `SELECT COALESCE(SUM(size), 0) FROM attachments WHERE message_id = $1`

	var size int64
	if err := a.repo.db().GetContext(ctx, &size, query, string(messageID)); err != nil {
		return 0, fmt.Errorf("failed to get total size by message: %w", err)
	}

	return size, nil
}

// GetByChecksum retrieves attachments with a specific checksum.
func (a *AttachmentRepository) GetByChecksum(ctx context.Context, checksum string) ([]*domain.Attachment, error) {
	query := `SELECT id, message_id, filename, content_type, size, content_id, 
		disposition, storage_path, checksum, is_inline, created_at 
		FROM attachments WHERE checksum = $1`

	var rows []attachmentRow
	if err := a.repo.db().SelectContext(ctx, &rows, query, checksum); err != nil {
		return nil, fmt.Errorf("failed to get attachments by checksum: %w", err)
	}

	attachments := make([]*domain.Attachment, len(rows))
	for i, row := range rows {
		attachments[i] = row.toAttachment()
	}

	return attachments, nil
}

// GetInlineAttachments retrieves inline attachments for a message.
func (a *AttachmentRepository) GetInlineAttachments(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	isInline := true
	filter := &repository.AttachmentFilter{
		MessageID: &messageID,
		IsInline:  &isInline,
	}
	result, err := a.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// GetNonInlineAttachments retrieves non-inline attachments for a message.
func (a *AttachmentRepository) GetNonInlineAttachments(ctx context.Context, messageID domain.ID) ([]*domain.Attachment, error) {
	isInline := false
	filter := &repository.AttachmentFilter{
		MessageID: &messageID,
		IsInline:  &isInline,
	}
	result, err := a.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// GetByContentType retrieves attachments with a specific content type.
func (a *AttachmentRepository) GetByContentType(ctx context.Context, contentType string, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	filter := &repository.AttachmentFilter{ContentType: contentType}
	return a.List(ctx, filter, opts)
}

// GetImages retrieves all image attachments.
func (a *AttachmentRepository) GetImages(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	filter := &repository.AttachmentFilter{ContentTypePrefix: "image/"}
	return a.List(ctx, filter, opts)
}

// GetLargeAttachments retrieves attachments larger than the specified size.
func (a *AttachmentRepository) GetLargeAttachments(ctx context.Context, minSize int64, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	filter := &repository.AttachmentFilter{MinSize: &minSize}
	return a.List(ctx, filter, opts)
}

// Search performs a text search on attachment filenames.
func (a *AttachmentRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Attachment], error) {
	filter := &repository.AttachmentFilter{FilenameContains: query}
	return a.List(ctx, filter, opts)
}

// BulkDelete permanently removes multiple attachments and their content.
func (a *AttachmentRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := a.Delete(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// CleanupOrphaned removes attachments that are not linked to any message.
func (a *AttachmentRepository) CleanupOrphaned(ctx context.Context) (int64, error) {
	// Delete orphaned content
	deleteContentQuery := `DELETE FROM attachment_content WHERE attachment_id IN 
		(SELECT a.id FROM attachments a LEFT JOIN messages m ON a.message_id = m.id WHERE m.id IS NULL)`
	if _, err := a.repo.db().ExecContext(ctx, deleteContentQuery); err != nil {
		return 0, fmt.Errorf("failed to delete orphaned content: %w", err)
	}

	// Delete orphaned attachments
	deleteQuery := `DELETE FROM attachments WHERE id IN 
		(SELECT a.id FROM attachments a LEFT JOIN messages m ON a.message_id = m.id WHERE m.id IS NULL)`
	result, err := a.repo.db().ExecContext(ctx, deleteQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to delete orphaned attachments: %w", err)
	}

	return result.RowsAffected()
}

// GetStorageStats retrieves storage statistics for attachments.
func (a *AttachmentRepository) GetStorageStats(ctx context.Context) (*repository.AttachmentStorageStats, error) {
	query := `SELECT 
		COUNT(*) as total_count,
		COALESCE(SUM(size), 0) as total_size,
		SUM(CASE WHEN is_inline = true THEN 1 ELSE 0 END) as inline_count,
		COALESCE(SUM(CASE WHEN is_inline = true THEN size ELSE 0 END), 0) as inline_size,
		SUM(CASE WHEN is_inline = false THEN 1 ELSE 0 END) as regular_count,
		COALESCE(SUM(CASE WHEN is_inline = false THEN size ELSE 0 END), 0) as regular_size,
		COALESCE(AVG(size), 0) as average_size,
		COALESCE(MAX(size), 0) as largest_size,
		COALESCE(MIN(size), 0) as smallest_size
		FROM attachments`

	var stats repository.AttachmentStorageStats
	if err := a.repo.db().GetContext(ctx, &stats, query); err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	return &stats, nil
}

// GetContentTypeStats retrieves storage statistics grouped by content type.
func (a *AttachmentRepository) GetContentTypeStats(ctx context.Context) ([]repository.ContentTypeStats, error) {
	query := `SELECT content_type, COUNT(*) as count, COALESCE(SUM(size), 0) as total_size
		FROM attachments GROUP BY content_type ORDER BY total_size DESC`

	var stats []repository.ContentTypeStats
	if err := a.repo.db().SelectContext(ctx, &stats, query); err != nil {
		return nil, fmt.Errorf("failed to get content type stats: %w", err)
	}

	// Calculate percentages
	var totalSize int64
	for _, s := range stats {
		totalSize += s.TotalSize
	}
	for i := range stats {
		if totalSize > 0 {
			stats[i].Percentage = float64(stats[i].TotalSize) / float64(totalSize) * 100
		}
	}

	return stats, nil
}

// Ensure AttachmentRepository implements repository.AttachmentRepository
var _ repository.AttachmentRepository = (*AttachmentRepository)(nil)
