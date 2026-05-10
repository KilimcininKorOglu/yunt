package postgres

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

// MessageRepository implements the repository.MessageRepository interface for PostgreSQL.
type MessageRepository struct {
	repo *Repository
}

// messageRow is the database representation of a message.
type messageRow struct {
	ID              string         `db:"id"`
	MailboxID       string         `db:"mailbox_id"`
	MessageID       sql.NullString `db:"message_id"`
	FromName        sql.NullString `db:"from_name"`
	FromAddress     string         `db:"from_address"`
	Subject         sql.NullString `db:"subject"`
	TextBody        sql.NullString `db:"text_body"`
	HTMLBody        sql.NullString `db:"html_body"`
	RawBody         []byte         `db:"raw_body"`
	Headers         []byte         `db:"headers"`
	ContentType     string         `db:"content_type"`
	Size            int64          `db:"size"`
	AttachmentCount int            `db:"attachment_count"`
	Status          string         `db:"status"`
	IsStarred       bool           `db:"is_starred"`
	IsSpam          bool           `db:"is_spam"`
	IsDeleted       bool           `db:"is_deleted"`
	InReplyTo       sql.NullString `db:"in_reply_to"`
	ReferencesList  []byte         `db:"references_list"`
	ReceivedAt      time.Time      `db:"received_at"`
	SentAt          sql.NullTime   `db:"sent_at"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

// recipientRow is the database representation of a message recipient.
type recipientRow struct {
	ID            int64          `db:"id"`
	MessageID     string         `db:"message_id"`
	RecipientType string         `db:"recipient_type"`
	Name          sql.NullString `db:"name"`
	Address       string         `db:"address"`
}

// NewMessageRepository creates a new PostgreSQL message repository.
func NewMessageRepository(repo *Repository) *MessageRepository {
	return &MessageRepository{repo: repo}
}

// toMessage converts a messageRow to a domain.Message.
func (r *messageRow) toMessage() *domain.Message {
	msg := &domain.Message{
		ID:              domain.ID(r.ID),
		MailboxID:       domain.ID(r.MailboxID),
		From:            domain.EmailAddress{Address: r.FromAddress},
		ContentType:     domain.ContentType(r.ContentType),
		Size:            r.Size,
		AttachmentCount: r.AttachmentCount,
		Status:          domain.MessageStatus(r.Status),
		IsStarred:       r.IsStarred,
		IsSpam:          r.IsSpam,
		IsDeleted:       r.IsDeleted,
		RawBody:         r.RawBody,
		ReceivedAt:      domain.Timestamp{Time: r.ReceivedAt},
		CreatedAt:       domain.Timestamp{Time: r.CreatedAt},
		UpdatedAt:       domain.Timestamp{Time: r.UpdatedAt},
		To:              make([]domain.EmailAddress, 0),
		Cc:              make([]domain.EmailAddress, 0),
		Bcc:             make([]domain.EmailAddress, 0),
		Headers:         make(map[string]string),
		References:      make([]string, 0),
	}

	if r.MessageID.Valid {
		msg.MessageID = r.MessageID.String
	}
	if r.FromName.Valid {
		msg.From.Name = r.FromName.String
	}
	if r.Subject.Valid {
		msg.Subject = r.Subject.String
	}
	if r.TextBody.Valid {
		msg.TextBody = r.TextBody.String
	}
	if r.HTMLBody.Valid {
		msg.HTMLBody = r.HTMLBody.String
	}
	if r.InReplyTo.Valid {
		msg.InReplyTo = r.InReplyTo.String
	}
	if len(r.ReferencesList) > 0 {
		_ = json.Unmarshal(r.ReferencesList, &msg.References)
	}
	if len(r.Headers) > 0 {
		_ = json.Unmarshal(r.Headers, &msg.Headers)
	}
	if r.SentAt.Valid {
		ts := domain.Timestamp{Time: r.SentAt.Time}
		msg.SentAt = &ts
	}

	return msg
}

// GetByID retrieves a message by its unique identifier.
func (m *MessageRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Message, error) {
	query := `SELECT id, mailbox_id, message_id, from_name, from_address, subject, 
		text_body, html_body, raw_body, headers, content_type, size, attachment_count, 
		status, is_starred, is_spam, is_deleted, in_reply_to, references_list, 
		received_at, sent_at, created_at, updated_at 
		FROM messages WHERE id = $1`

	var row messageRow
	if err := m.repo.db().GetContext(ctx, &row, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("message", string(id))
		}
		return nil, fmt.Errorf("failed to get message by ID: %w", err)
	}

	msg := row.toMessage()

	// Load recipients
	if err := m.loadRecipients(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// loadRecipients loads the recipients for a message.
func (m *MessageRepository) loadRecipients(ctx context.Context, msg *domain.Message) error {
	query := `SELECT id, message_id, recipient_type, name, address 
		FROM message_recipients WHERE message_id = $1`

	var recipients []recipientRow
	if err := m.repo.db().SelectContext(ctx, &recipients, query, string(msg.ID)); err != nil {
		return fmt.Errorf("failed to load recipients: %w", err)
	}

	for _, r := range recipients {
		addr := domain.EmailAddress{Address: r.Address}
		if r.Name.Valid {
			addr.Name = r.Name.String
		}

		switch r.RecipientType {
		case "to":
			msg.To = append(msg.To, addr)
		case "cc":
			msg.Cc = append(msg.Cc, addr)
		case "bcc":
			msg.Bcc = append(msg.Bcc, addr)
		case "reply_to":
			msg.ReplyTo = &addr
		}
	}

	return nil
}

// GetByMessageID retrieves a message by its email Message-ID header.
func (m *MessageRepository) GetByMessageID(ctx context.Context, messageID string) (*domain.Message, error) {
	query := `SELECT id, mailbox_id, message_id, from_name, from_address, subject, 
		text_body, html_body, raw_body, headers, content_type, size, attachment_count, 
		status, is_starred, is_spam, is_deleted, in_reply_to, references_list, 
		received_at, sent_at, created_at, updated_at 
		FROM messages WHERE message_id = $1`

	var row messageRow
	if err := m.repo.db().GetContext(ctx, &row, query, messageID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("message", messageID)
		}
		return nil, fmt.Errorf("failed to get message by Message-ID: %w", err)
	}

	msg := row.toMessage()

	if err := m.loadRecipients(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// GetWithAttachments retrieves a message with its attachments loaded.
func (m *MessageRepository) GetWithAttachments(ctx context.Context, id domain.ID) (*domain.Message, []*domain.Attachment, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	attachments, err := m.repo.Attachments().ListByMessage(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return msg, attachments, nil
}

// List retrieves messages with optional filtering, sorting, and pagination.
func (m *MessageRepository) List(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	query, args := m.buildListQuery(filter, opts, false)
	countQuery, countArgs := m.buildListQuery(filter, opts, true)

	var total int64
	if err := m.repo.db().GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, fmt.Errorf("failed to count messages: %w", err)
	}

	var rows []messageRow
	if err := m.repo.db().SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := make([]*domain.Message, len(rows))
	for i, row := range rows {
		messages[i] = row.toMessage()
		if err := m.loadRecipients(ctx, messages[i]); err != nil {
			return nil, err
		}
	}

	result := &repository.ListResult[*domain.Message]{
		Items: messages,
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

// buildListQuery builds the SQL query for listing messages.
func (m *MessageRepository) buildListQuery(filter *repository.MessageFilter, opts *repository.ListOptions, countOnly bool) (string, []interface{}) {
	var sb strings.Builder
	args := make([]interface{}, 0)
	argIndex := 1

	if countOnly {
		sb.WriteString("SELECT COUNT(*) FROM messages WHERE 1=1")
	} else {
		sb.WriteString(`SELECT id, mailbox_id, message_id, from_name, from_address, subject, 
			text_body, html_body, raw_body, headers, content_type, size, attachment_count, 
			status, is_starred, is_spam, is_deleted, in_reply_to, references_list, 
			received_at, sent_at, created_at, updated_at FROM messages WHERE 1=1`)
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

		if filter.MailboxID != nil {
			sb.WriteString(fmt.Sprintf(" AND mailbox_id = $%d", argIndex))
			args = append(args, string(*filter.MailboxID))
			argIndex++
		}

		if len(filter.MailboxIDs) > 0 {
			placeholders := make([]string, len(filter.MailboxIDs))
			for i, id := range filter.MailboxIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, string(id))
				argIndex++
			}
			sb.WriteString(fmt.Sprintf(" AND mailbox_id IN (%s)", strings.Join(placeholders, ",")))
		}

		if filter.Status != nil {
			sb.WriteString(fmt.Sprintf(" AND status = $%d", argIndex))
			args = append(args, string(*filter.Status))
			argIndex++
		}

		if filter.IsStarred != nil {
			sb.WriteString(fmt.Sprintf(" AND is_starred = $%d", argIndex))
			args = append(args, *filter.IsStarred)
			argIndex++
		}

		if filter.IsSpam != nil {
			sb.WriteString(fmt.Sprintf(" AND is_spam = $%d", argIndex))
			args = append(args, *filter.IsSpam)
			argIndex++
		}

		if filter.HasAttachments != nil {
			if *filter.HasAttachments {
				sb.WriteString(" AND attachment_count > 0")
			} else {
				sb.WriteString(" AND attachment_count = 0")
			}
		}

		if filter.FromAddress != "" {
			sb.WriteString(fmt.Sprintf(" AND LOWER(from_address) = LOWER($%d)", argIndex))
			args = append(args, filter.FromAddress)
			argIndex++
		}

		if filter.FromAddressContains != "" {
			sb.WriteString(fmt.Sprintf(" AND LOWER(from_address) LIKE LOWER($%d)", argIndex))
			args = append(args, "%"+filter.FromAddressContains+"%")
			argIndex++
		}

		if filter.Subject != "" {
			sb.WriteString(fmt.Sprintf(" AND subject = $%d", argIndex))
			args = append(args, filter.Subject)
			argIndex++
		}

		if filter.SubjectContains != "" {
			sb.WriteString(fmt.Sprintf(" AND subject ILIKE $%d", argIndex))
			args = append(args, "%"+filter.SubjectContains+"%")
			argIndex++
		}

		if filter.BodyContains != "" {
			sb.WriteString(fmt.Sprintf(" AND (text_body ILIKE $%d OR html_body ILIKE $%d)", argIndex, argIndex+1))
			pattern := "%" + filter.BodyContains + "%"
			args = append(args, pattern, pattern)
			argIndex += 2
		}

		if filter.Search != "" {
			sb.WriteString(fmt.Sprintf(" AND (subject ILIKE $%d OR from_address ILIKE $%d OR text_body ILIKE $%d)", argIndex, argIndex+1, argIndex+2))
			pattern := "%" + filter.Search + "%"
			args = append(args, pattern, pattern, pattern)
			argIndex += 3
		}

		if filter.MessageID != "" {
			sb.WriteString(fmt.Sprintf(" AND message_id = $%d", argIndex))
			args = append(args, filter.MessageID)
			argIndex++
		}

		if filter.InReplyTo != "" {
			sb.WriteString(fmt.Sprintf(" AND in_reply_to = $%d", argIndex))
			args = append(args, filter.InReplyTo)
			argIndex++
		}

		if filter.ReceivedAfter != nil {
			sb.WriteString(fmt.Sprintf(" AND received_at > $%d", argIndex))
			args = append(args, filter.ReceivedAfter.Time)
			argIndex++
		}

		if filter.ReceivedBefore != nil {
			sb.WriteString(fmt.Sprintf(" AND received_at < $%d", argIndex))
			args = append(args, filter.ReceivedBefore.Time)
			argIndex++
		}

		if filter.SentAfter != nil {
			sb.WriteString(fmt.Sprintf(" AND sent_at > $%d", argIndex))
			args = append(args, filter.SentAfter.Time)
			argIndex++
		}

		if filter.SentBefore != nil {
			sb.WriteString(fmt.Sprintf(" AND sent_at < $%d", argIndex))
			args = append(args, filter.SentBefore.Time)
			argIndex++
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

		if filter.ContentType != nil {
			sb.WriteString(fmt.Sprintf(" AND content_type = $%d", argIndex))
			args = append(args, string(*filter.ContentType))
		}

		if filter.ExcludeSpam {
			sb.WriteString(" AND is_spam = false")
		}

		if filter.ExcludeDeleted {
			sb.WriteString(" AND is_deleted = false")
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
			sb.WriteString(" ORDER BY received_at DESC")
		}

		if opts != nil && opts.Pagination != nil {
			opts.Pagination.Normalize()
			sb.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset()))
		}
	}

	return sb.String(), args
}

// mapSortField maps repository sort field to database column.
func (m *MessageRepository) mapSortField(field string) string {
	switch field {
	case "receivedAt":
		return "received_at"
	case "sentAt":
		return "sent_at"
	case "subject":
		return "subject"
	case "from":
		return "from_address"
	case "size":
		return "size"
	case "status":
		return "status"
	case "createdAt":
		return "created_at"
	default:
		return "received_at"
	}
}

// ListByMailbox retrieves all messages in a specific mailbox.
func (m *MessageRepository) ListByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	filter := &repository.MessageFilter{MailboxID: &mailboxID}
	return m.List(ctx, filter, opts)
}

// ListSummaries retrieves message summaries for faster list rendering.
func (m *MessageRepository) ListSummaries(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	result, err := m.List(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.MessageSummary, len(result.Items))
	for i, msg := range result.Items {
		summaries[i] = msg.ToSummary(100)
	}

	return &repository.ListResult[*domain.MessageSummary]{
		Items:      summaries,
		Total:      result.Total,
		HasMore:    result.HasMore,
		Pagination: result.Pagination,
	}, nil
}

// Create creates a new message.
func (m *MessageRepository) Create(ctx context.Context, msg *domain.Message) error {
	query := `INSERT INTO messages (id, mailbox_id, message_id, from_name, from_address,
		subject, text_body, html_body, raw_body, headers, content_type, size,
		attachment_count, status, is_starred, is_spam, is_deleted, in_reply_to, references_list,
		received_at, sent_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)`

	var messageID, fromName, subject, textBody, htmlBody, inReplyTo sql.NullString
	var refsList, headersJSON []byte

	if msg.MessageID != "" {
		messageID = sql.NullString{String: msg.MessageID, Valid: true}
	}
	if msg.From.Name != "" {
		fromName = sql.NullString{String: msg.From.Name, Valid: true}
	}
	if msg.Subject != "" {
		subject = sql.NullString{String: msg.Subject, Valid: true}
	}
	if msg.TextBody != "" {
		textBody = sql.NullString{String: msg.TextBody, Valid: true}
	}
	if msg.HTMLBody != "" {
		htmlBody = sql.NullString{String: msg.HTMLBody, Valid: true}
	}
	if msg.InReplyTo != "" {
		inReplyTo = sql.NullString{String: msg.InReplyTo, Valid: true}
	}
	if len(msg.References) > 0 {
		refsList, _ = json.Marshal(msg.References)
	}
	if len(msg.Headers) > 0 {
		headersJSON, _ = json.Marshal(msg.Headers)
	}

	var sentAt sql.NullTime
	if msg.SentAt != nil {
		sentAt = sql.NullTime{Time: msg.SentAt.Time, Valid: true}
	}

	_, err := m.repo.db().ExecContext(ctx, query,
		string(msg.ID),
		string(msg.MailboxID),
		messageID,
		fromName,
		msg.From.Address,
		subject,
		textBody,
		htmlBody,
		msg.RawBody,
		headersJSON,
		string(msg.ContentType),
		msg.Size,
		msg.AttachmentCount,
		string(msg.Status),
		msg.IsStarred,
		msg.IsSpam,
		msg.IsDeleted,
		inReplyTo,
		refsList,
		msg.ReceivedAt.Time,
		sentAt,
		msg.CreatedAt.Time,
		msg.UpdatedAt.Time,
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Insert recipients
	if err := m.saveRecipients(ctx, msg); err != nil {
		return err
	}

	// Update mailbox stats
	if err := m.repo.Mailboxes().IncrementMessageCount(ctx, msg.MailboxID, msg.Size); err != nil {
		return fmt.Errorf("failed to update mailbox stats: %w", err)
	}

	return nil
}

// saveRecipients saves the recipients for a message.
func (m *MessageRepository) saveRecipients(ctx context.Context, msg *domain.Message) error {
	query := `INSERT INTO message_recipients (message_id, recipient_type, name, address) VALUES ($1, $2, $3, $4)`

	for _, addr := range msg.To {
		var name sql.NullString
		if addr.Name != "" {
			name = sql.NullString{String: addr.Name, Valid: true}
		}
		if _, err := m.repo.db().ExecContext(ctx, query, string(msg.ID), "to", name, addr.Address); err != nil {
			return fmt.Errorf("failed to save To recipient: %w", err)
		}
	}

	for _, addr := range msg.Cc {
		var name sql.NullString
		if addr.Name != "" {
			name = sql.NullString{String: addr.Name, Valid: true}
		}
		if _, err := m.repo.db().ExecContext(ctx, query, string(msg.ID), "cc", name, addr.Address); err != nil {
			return fmt.Errorf("failed to save Cc recipient: %w", err)
		}
	}

	for _, addr := range msg.Bcc {
		var name sql.NullString
		if addr.Name != "" {
			name = sql.NullString{String: addr.Name, Valid: true}
		}
		if _, err := m.repo.db().ExecContext(ctx, query, string(msg.ID), "bcc", name, addr.Address); err != nil {
			return fmt.Errorf("failed to save Bcc recipient: %w", err)
		}
	}

	if msg.ReplyTo != nil {
		var name sql.NullString
		if msg.ReplyTo.Name != "" {
			name = sql.NullString{String: msg.ReplyTo.Name, Valid: true}
		}
		if _, err := m.repo.db().ExecContext(ctx, query, string(msg.ID), "reply_to", name, msg.ReplyTo.Address); err != nil {
			return fmt.Errorf("failed to save ReplyTo: %w", err)
		}
	}

	return nil
}

// Update updates an existing message.
func (m *MessageRepository) Update(ctx context.Context, msg *domain.Message) error {
	exists, err := m.Exists(ctx, msg.ID)
	if err != nil {
		return err
	}
	if !exists {
		return domain.NewNotFoundError("message", string(msg.ID))
	}

	query := `UPDATE messages SET mailbox_id = $1, message_id = $2, from_name = $3,
		from_address = $4, subject = $5, text_body = $6, html_body = $7, headers = $8,
		content_type = $9, size = $10, attachment_count = $11, status = $12, is_starred = $13,
		is_spam = $14, is_deleted = $15, in_reply_to = $16, references_list = $17, received_at = $18, sent_at = $19,
		updated_at = $20 WHERE id = $21`

	var messageID, fromName, subject, textBody, htmlBody, inReplyTo sql.NullString
	var refsList, headersJSON []byte

	if msg.MessageID != "" {
		messageID = sql.NullString{String: msg.MessageID, Valid: true}
	}
	if msg.From.Name != "" {
		fromName = sql.NullString{String: msg.From.Name, Valid: true}
	}
	if msg.Subject != "" {
		subject = sql.NullString{String: msg.Subject, Valid: true}
	}
	if msg.TextBody != "" {
		textBody = sql.NullString{String: msg.TextBody, Valid: true}
	}
	if msg.HTMLBody != "" {
		htmlBody = sql.NullString{String: msg.HTMLBody, Valid: true}
	}
	if msg.InReplyTo != "" {
		inReplyTo = sql.NullString{String: msg.InReplyTo, Valid: true}
	}
	if len(msg.References) > 0 {
		refsList, _ = json.Marshal(msg.References)
	}
	if len(msg.Headers) > 0 {
		headersJSON, _ = json.Marshal(msg.Headers)
	}

	var sentAt sql.NullTime
	if msg.SentAt != nil {
		sentAt = sql.NullTime{Time: msg.SentAt.Time, Valid: true}
	}

	_, err = m.repo.db().ExecContext(ctx, query,
		string(msg.MailboxID),
		messageID,
		fromName,
		msg.From.Address,
		subject,
		textBody,
		htmlBody,
		headersJSON,
		string(msg.ContentType),
		msg.Size,
		msg.AttachmentCount,
		string(msg.Status),
		msg.IsStarred,
		msg.IsSpam,
		msg.IsDeleted,
		inReplyTo,
		refsList,
		msg.ReceivedAt.Time,
		sentAt,
		time.Now().UTC(),
		string(msg.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	// Update recipients (delete and re-insert)
	if _, err := m.repo.db().ExecContext(ctx, "DELETE FROM message_recipients WHERE message_id = $1", string(msg.ID)); err != nil {
		return fmt.Errorf("failed to delete recipients: %w", err)
	}

	return m.saveRecipients(ctx, msg)
}

// Delete permanently removes a message by its ID.
func (m *MessageRepository) Delete(ctx context.Context, id domain.ID) error {
	// Get message first to update mailbox stats
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	query := `DELETE FROM messages WHERE id = $1`
	result, err := m.repo.db().ExecContext(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	// Update mailbox stats
	wasUnread := msg.Status == domain.MessageUnread
	if err := m.repo.Mailboxes().DecrementMessageCount(ctx, msg.MailboxID, msg.Size, wasUnread); err != nil {
		return fmt.Errorf("failed to update mailbox stats: %w", err)
	}

	return nil
}

// DeleteByMailbox removes all messages in a mailbox.
func (m *MessageRepository) DeleteByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	query := `DELETE FROM messages WHERE mailbox_id = $1`

	result, err := m.repo.db().ExecContext(ctx, query, string(mailboxID))
	if err != nil {
		return 0, fmt.Errorf("failed to delete messages by mailbox: %w", err)
	}

	return result.RowsAffected()
}

// Exists checks if a message with the given ID exists.
func (m *MessageRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`

	var exists bool
	if err := m.repo.db().GetContext(ctx, &exists, query, string(id)); err != nil {
		return false, fmt.Errorf("failed to check message existence: %w", err)
	}

	return exists, nil
}

// ExistsByMessageID checks if a message with the given Message-ID exists.
func (m *MessageRepository) ExistsByMessageID(ctx context.Context, messageID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM messages WHERE message_id = $1)`

	var exists bool
	if err := m.repo.db().GetContext(ctx, &exists, query, messageID); err != nil {
		return false, fmt.Errorf("failed to check Message-ID existence: %w", err)
	}

	return exists, nil
}

// Count returns the total number of messages matching the filter.
func (m *MessageRepository) Count(ctx context.Context, filter *repository.MessageFilter) (int64, error) {
	query, args := m.buildListQuery(filter, nil, true)

	var count int64
	if err := m.repo.db().GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// CountByMailbox returns the number of messages in a mailbox.
func (m *MessageRepository) CountByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	query := `SELECT COUNT(*) FROM messages WHERE mailbox_id = $1`

	var count int64
	if err := m.repo.db().GetContext(ctx, &count, query, string(mailboxID)); err != nil {
		return 0, fmt.Errorf("failed to count messages by mailbox: %w", err)
	}

	return count, nil
}

// CountUnreadByMailbox returns the number of unread messages in a mailbox.
func (m *MessageRepository) CountUnreadByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	query := `SELECT COUNT(*) FROM messages WHERE mailbox_id = $1 AND status = 'unread'`

	var count int64
	if err := m.repo.db().GetContext(ctx, &count, query, string(mailboxID)); err != nil {
		return 0, fmt.Errorf("failed to count unread messages: %w", err)
	}

	return count, nil
}

// MarkAsRead marks a message as read.
func (m *MessageRepository) MarkAsRead(ctx context.Context, id domain.ID) (bool, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	if msg.Status == domain.MessageRead {
		return false, nil
	}

	query := `UPDATE messages SET status = 'read', updated_at = $1 WHERE id = $2`
	_, err = m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return false, fmt.Errorf("failed to mark message as read: %w", err)
	}

	// Update mailbox unread count
	if err := m.repo.Mailboxes().UpdateUnreadCount(ctx, msg.MailboxID, -1); err != nil {
		return false, err
	}

	return true, nil
}

// MarkAsUnread marks a message as unread.
func (m *MessageRepository) MarkAsUnread(ctx context.Context, id domain.ID) (bool, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	if msg.Status == domain.MessageUnread {
		return false, nil
	}

	query := `UPDATE messages SET status = 'unread', updated_at = $1 WHERE id = $2`
	_, err = m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return false, fmt.Errorf("failed to mark message as unread: %w", err)
	}

	// Update mailbox unread count
	if err := m.repo.Mailboxes().UpdateUnreadCount(ctx, msg.MailboxID, 1); err != nil {
		return false, err
	}

	return true, nil
}

// MarkAllAsRead marks all messages in a mailbox as read.
func (m *MessageRepository) MarkAllAsRead(ctx context.Context, mailboxID domain.ID) (int64, error) {
	query := `UPDATE messages SET status = 'read', updated_at = $1 WHERE mailbox_id = $2 AND status = 'unread'`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(mailboxID))
	if err != nil {
		return 0, fmt.Errorf("failed to mark all as read: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	// Update mailbox unread count
	if count > 0 {
		zero := int64(0)
		if err := m.repo.Mailboxes().UpdateStats(ctx, mailboxID, &repository.MailboxStatsUpdate{UnreadCount: &zero}); err != nil {
			return count, err
		}
	}

	return count, nil
}

// ToggleStar toggles the starred status of a message.
func (m *MessageRepository) ToggleStar(ctx context.Context, id domain.ID) (bool, error) {
	query := `UPDATE messages SET is_starred = NOT is_starred, updated_at = $1 WHERE id = $2`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return false, fmt.Errorf("failed to toggle star: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, domain.NewNotFoundError("message", string(id))
	}

	// Get new status
	var isStarred bool
	if err := m.repo.db().GetContext(ctx, &isStarred, "SELECT is_starred FROM messages WHERE id = $1", string(id)); err != nil {
		return false, err
	}

	return isStarred, nil
}

// Star marks a message as starred.
func (m *MessageRepository) Star(ctx context.Context, id domain.ID) error {
	query := `UPDATE messages SET is_starred = true, updated_at = $1 WHERE id = $2`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to star message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// Unstar removes the star from a message.
func (m *MessageRepository) Unstar(ctx context.Context, id domain.ID) error {
	query := `UPDATE messages SET is_starred = false, updated_at = $1 WHERE id = $2`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to unstar message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// MarkAsSpam marks a message as spam.
func (m *MessageRepository) MarkAsSpam(ctx context.Context, id domain.ID) error {
	query := `UPDATE messages SET is_spam = true, updated_at = $1 WHERE id = $2`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to mark as spam: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// MarkAsNotSpam removes the spam flag from a message.
func (m *MessageRepository) MarkAsNotSpam(ctx context.Context, id domain.ID) error {
	query := `UPDATE messages SET is_spam = false, updated_at = $1 WHERE id = $2`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to mark as not spam: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

func (m *MessageRepository) MarkAsDeleted(ctx context.Context, id domain.ID) error {
	query := `UPDATE messages SET is_deleted = true, updated_at = $1 WHERE id = $2`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to mark as deleted: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

func (m *MessageRepository) UnmarkAsDeleted(ctx context.Context, id domain.ID) error {
	query := `UPDATE messages SET is_deleted = false, updated_at = $1 WHERE id = $2`

	result, err := m.repo.db().ExecContext(ctx, query, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to unmark as deleted: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// MoveToMailbox moves a message to a different mailbox.
func (m *MessageRepository) MoveToMailbox(ctx context.Context, id domain.ID, targetMailboxID domain.ID) error {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if msg.MailboxID == targetMailboxID {
		return nil // Already in target mailbox
	}

	// Check target mailbox exists
	exists, err := m.repo.Mailboxes().Exists(ctx, targetMailboxID)
	if err != nil {
		return err
	}
	if !exists {
		return domain.NewNotFoundError("mailbox", string(targetMailboxID))
	}

	query := `UPDATE messages SET mailbox_id = $1, updated_at = $2 WHERE id = $3`
	_, err = m.repo.db().ExecContext(ctx, query, string(targetMailboxID), time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to move message: %w", err)
	}

	// Update source mailbox stats
	wasUnread := msg.Status == domain.MessageUnread
	if err := m.repo.Mailboxes().DecrementMessageCount(ctx, msg.MailboxID, msg.Size, wasUnread); err != nil {
		return err
	}

	// Update target mailbox stats
	if err := m.repo.Mailboxes().IncrementMessageCount(ctx, targetMailboxID, msg.Size); err != nil {
		return err
	}

	// Adjust unread count for target (IncrementMessageCount adds to unread, but message might be read)
	if !wasUnread {
		if err := m.repo.Mailboxes().UpdateUnreadCount(ctx, targetMailboxID, -1); err != nil {
			return err
		}
	}

	return nil
}

// Search performs a full-text search across message fields.
func (m *MessageRepository) Search(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	if filter == nil {
		filter = &repository.MessageFilter{}
	}
	if searchOpts != nil && !searchOpts.IsEmpty() {
		filter.Search = searchOpts.Query
	}
	return m.List(ctx, filter, opts)
}

// SearchSummaries performs search and returns message summaries.
func (m *MessageRepository) SearchSummaries(ctx context.Context, searchOpts *repository.SearchOptions, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	result, err := m.Search(ctx, searchOpts, filter, opts)
	if err != nil {
		return nil, err
	}

	summaries := make([]*domain.MessageSummary, len(result.Items))
	for i, msg := range result.Items {
		summaries[i] = msg.ToSummary(100)
	}

	return &repository.ListResult[*domain.MessageSummary]{
		Items:      summaries,
		Total:      result.Total,
		HasMore:    result.HasMore,
		Pagination: result.Pagination,
	}, nil
}

// GetThread retrieves all messages in a conversation thread.
func (m *MessageRepository) GetThread(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Collect all message IDs in the thread
	threadIDs := make(map[string]bool)
	threadIDs[msg.MessageID] = true
	if msg.InReplyTo != "" {
		threadIDs[msg.InReplyTo] = true
	}
	for _, ref := range msg.References {
		threadIDs[ref] = true
	}

	if len(threadIDs) == 0 {
		return []*domain.Message{msg}, nil
	}

	// Build query for all related messages
	placeholders := make([]string, 0, len(threadIDs))
	args := make([]interface{}, 0, len(threadIDs))
	argIndex := 1
	for msgID := range threadIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
		args = append(args, msgID)
		argIndex++
	}

	// Duplicate for both IN clauses
	for msgID := range threadIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
		args = append(args, msgID)
		argIndex++
	}

	query := fmt.Sprintf(`SELECT id, mailbox_id, message_id, from_name, from_address, subject, 
		text_body, html_body, raw_body, headers, content_type, size, attachment_count, 
		status, is_starred, is_spam, is_deleted, in_reply_to, references_list, 
		received_at, sent_at, created_at, updated_at 
		FROM messages WHERE message_id IN (%s) OR in_reply_to IN (%s)
		ORDER BY received_at ASC`,
		strings.Join(placeholders[:len(threadIDs)], ","),
		strings.Join(placeholders[len(threadIDs):], ","))

	var rows []messageRow
	if err := m.repo.db().SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	messages := make([]*domain.Message, len(rows))
	for i, row := range rows {
		messages[i] = row.toMessage()
		if err := m.loadRecipients(ctx, messages[i]); err != nil {
			return nil, err
		}
	}

	return messages, nil
}

// GetReplies retrieves all replies to a specific message.
func (m *MessageRepository) GetReplies(ctx context.Context, id domain.ID) ([]*domain.Message, error) {
	msg, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if msg.MessageID == "" {
		return []*domain.Message{}, nil
	}

	query := `SELECT id, mailbox_id, message_id, from_name, from_address, subject, 
		text_body, html_body, raw_body, headers, content_type, size, attachment_count, 
		status, is_starred, is_spam, is_deleted, in_reply_to, references_list, 
		received_at, sent_at, created_at, updated_at 
		FROM messages WHERE in_reply_to = $1 ORDER BY received_at ASC`

	var rows []messageRow
	if err := m.repo.db().SelectContext(ctx, &rows, query, msg.MessageID); err != nil {
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}

	messages := make([]*domain.Message, len(rows))
	for i, row := range rows {
		messages[i] = row.toMessage()
		if err := m.loadRecipients(ctx, messages[i]); err != nil {
			return nil, err
		}
	}

	return messages, nil
}

// GetStarred retrieves all starred messages.
func (m *MessageRepository) GetStarred(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	isStarred := true
	filter := &repository.MessageFilter{IsStarred: &isStarred}
	return m.List(ctx, filter, opts)
}

// GetStarredByUser retrieves starred messages from mailboxes owned by a user.
func (m *MessageRepository) GetStarredByUser(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	// Get user's mailboxes
	mailboxes, err := m.repo.Mailboxes().ListByUser(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	mailboxIDs := make([]domain.ID, len(mailboxes.Items))
	for i, mb := range mailboxes.Items {
		mailboxIDs[i] = mb.ID
	}

	isStarred := true
	filter := &repository.MessageFilter{
		MailboxIDs: mailboxIDs,
		IsStarred:  &isStarred,
	}
	return m.List(ctx, filter, opts)
}

// GetSpam retrieves all spam messages.
func (m *MessageRepository) GetSpam(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	isSpam := true
	filter := &repository.MessageFilter{IsSpam: &isSpam}
	return m.List(ctx, filter, opts)
}

// GetUnread retrieves all unread messages.
func (m *MessageRepository) GetUnread(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	status := domain.MessageUnread
	filter := &repository.MessageFilter{Status: &status}
	return m.List(ctx, filter, opts)
}

// GetUnreadByMailbox retrieves unread messages in a specific mailbox.
func (m *MessageRepository) GetUnreadByMailbox(ctx context.Context, mailboxID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	status := domain.MessageUnread
	filter := &repository.MessageFilter{
		MailboxID: &mailboxID,
		Status:    &status,
	}
	return m.List(ctx, filter, opts)
}

// GetMessagesWithAttachments retrieves messages that have attachments.
func (m *MessageRepository) GetMessagesWithAttachments(ctx context.Context, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	if filter == nil {
		filter = &repository.MessageFilter{}
	}
	hasAttachments := true
	filter.HasAttachments = &hasAttachments
	return m.List(ctx, filter, opts)
}

// GetRecent retrieves messages received in the last N hours.
func (m *MessageRepository) GetRecent(ctx context.Context, hours int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	ts := domain.Timestamp{Time: since}
	filter := &repository.MessageFilter{ReceivedAfter: &ts}
	return m.List(ctx, filter, opts)
}

// GetByDateRange retrieves messages within a date range.
func (m *MessageRepository) GetByDateRange(ctx context.Context, dateRange *repository.DateRangeFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	filter := &repository.MessageFilter{}
	if dateRange != nil {
		filter.ReceivedAfter = dateRange.From
		filter.ReceivedBefore = dateRange.To
	}
	return m.List(ctx, filter, opts)
}

// GetBySender retrieves all messages from a specific sender.
func (m *MessageRepository) GetBySender(ctx context.Context, senderAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	filter := &repository.MessageFilter{FromAddress: senderAddress}
	return m.List(ctx, filter, opts)
}

// GetByRecipient retrieves all messages sent to a specific recipient.
func (m *MessageRepository) GetByRecipient(ctx context.Context, recipientAddress string, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	// This requires joining with recipients table
	query := `SELECT DISTINCT m.id FROM messages m 
		JOIN message_recipients r ON m.id = r.message_id 
		WHERE LOWER(r.address) = LOWER($1)`

	var ids []string
	if err := m.repo.db().SelectContext(ctx, &ids, query, recipientAddress); err != nil {
		return nil, fmt.Errorf("failed to get messages by recipient: %w", err)
	}

	if len(ids) == 0 {
		return &repository.ListResult[*domain.Message]{Items: []*domain.Message{}, Total: 0}, nil
	}

	domainIDs := make([]domain.ID, len(ids))
	for i, id := range ids {
		domainIDs[i] = domain.ID(id)
	}

	filter := &repository.MessageFilter{IDs: domainIDs}
	return m.List(ctx, filter, opts)
}

// GetOldMessages retrieves messages older than the specified days.
func (m *MessageRepository) GetOldMessages(ctx context.Context, olderThanDays int, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	before := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	ts := domain.Timestamp{Time: before}
	filter := &repository.MessageFilter{ReceivedBefore: &ts}
	return m.List(ctx, filter, opts)
}

// GetLargeMessages retrieves messages larger than the specified size.
func (m *MessageRepository) GetLargeMessages(ctx context.Context, minSize int64, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	filter := &repository.MessageFilter{MinSize: &minSize}
	return m.List(ctx, filter, opts)
}

// DeleteOldMessages deletes messages older than the specified days.
func (m *MessageRepository) DeleteOldMessages(ctx context.Context, olderThanDays int) (int64, error) {
	before := time.Now().UTC().AddDate(0, 0, -olderThanDays)

	query := `DELETE FROM messages WHERE received_at < $1`
	result, err := m.repo.db().ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old messages: %w", err)
	}

	return result.RowsAffected()
}

// DeleteSpam deletes all spam messages.
func (m *MessageRepository) DeleteSpam(ctx context.Context) (int64, error) {
	query := `DELETE FROM messages WHERE is_spam = true`

	result, err := m.repo.db().ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete spam: %w", err)
	}

	return result.RowsAffected()
}

// BulkMarkAsRead marks multiple messages as read.
func (m *MessageRepository) BulkMarkAsRead(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if _, err := m.MarkAsRead(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkMarkAsUnread marks multiple messages as unread.
func (m *MessageRepository) BulkMarkAsUnread(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if _, err := m.MarkAsUnread(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkDelete permanently removes multiple messages.
func (m *MessageRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
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

// BulkMove moves multiple messages to a different mailbox.
func (m *MessageRepository) BulkMove(ctx context.Context, ids []domain.ID, targetMailboxID domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := m.MoveToMailbox(ctx, id, targetMailboxID); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkStar marks multiple messages as starred.
func (m *MessageRepository) BulkStar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := m.Star(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkUnstar removes the star from multiple messages.
func (m *MessageRepository) BulkUnstar(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := m.Unstar(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// GetSizeByMailbox calculates the total size of messages in a mailbox.
func (m *MessageRepository) GetSizeByMailbox(ctx context.Context, mailboxID domain.ID) (int64, error) {
	query := `SELECT COALESCE(SUM(size), 0) FROM messages WHERE mailbox_id = $1`

	var size int64
	if err := m.repo.db().GetContext(ctx, &size, query, string(mailboxID)); err != nil {
		return 0, fmt.Errorf("failed to get size by mailbox: %w", err)
	}

	return size, nil
}

// GetTotalSize calculates the total size of all messages.
func (m *MessageRepository) GetTotalSize(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(SUM(size), 0) FROM messages`

	var size int64
	if err := m.repo.db().GetContext(ctx, &size, query); err != nil {
		return 0, fmt.Errorf("failed to get total size: %w", err)
	}

	return size, nil
}

// GetDailyCounts returns message counts grouped by day within a date range.
func (m *MessageRepository) GetDailyCounts(ctx context.Context, dateRange *repository.DateRangeFilter) ([]repository.DateCount, error) {
	var sb strings.Builder
	args := make([]interface{}, 0)
	argIndex := 1

	sb.WriteString(`SELECT DATE(received_at) as date, COUNT(*) as count FROM messages WHERE 1=1`)

	if dateRange != nil {
		if dateRange.From != nil {
			sb.WriteString(fmt.Sprintf(" AND received_at >= $%d", argIndex))
			args = append(args, dateRange.From.Time)
			argIndex++
		}
		if dateRange.To != nil {
			sb.WriteString(fmt.Sprintf(" AND received_at <= $%d", argIndex))
			args = append(args, dateRange.To.Time)
		}
	}

	sb.WriteString(" GROUP BY DATE(received_at) ORDER BY date")

	var counts []repository.DateCount
	if err := m.repo.db().SelectContext(ctx, &counts, sb.String(), args...); err != nil {
		return nil, fmt.Errorf("failed to get daily counts: %w", err)
	}

	return counts, nil
}

// GetHourlyCounts returns message counts grouped by hour within a date range.
func (m *MessageRepository) GetHourlyCounts(ctx context.Context, dateRange *repository.DateRangeFilter) ([]repository.HourCount, error) {
	var sb strings.Builder
	args := make([]interface{}, 0)
	argIndex := 1

	sb.WriteString(`SELECT to_char(received_at, 'YYYY-MM-DD HH24:00') as hour, COUNT(*) as count FROM messages WHERE 1=1`)

	if dateRange != nil {
		if dateRange.From != nil {
			sb.WriteString(fmt.Sprintf(" AND received_at >= $%d", argIndex))
			args = append(args, dateRange.From.Time)
			argIndex++
		}
		if dateRange.To != nil {
			sb.WriteString(fmt.Sprintf(" AND received_at <= $%d", argIndex))
			args = append(args, dateRange.To.Time)
		}
	}

	sb.WriteString(" GROUP BY to_char(received_at, 'YYYY-MM-DD HH24:00') ORDER BY hour")

	var counts []repository.HourCount
	if err := m.repo.db().SelectContext(ctx, &counts, sb.String(), args...); err != nil {
		return nil, fmt.Errorf("failed to get hourly counts: %w", err)
	}

	return counts, nil
}

// GetSenderCounts returns message counts grouped by sender address.
func (m *MessageRepository) GetSenderCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	query := `SELECT from_address as address, from_name as name, COUNT(*) as count
		FROM messages GROUP BY from_address, from_name ORDER BY count DESC LIMIT $1`

	var counts []repository.AddressCount
	if err := m.repo.db().SelectContext(ctx, &counts, query, limit); err != nil {
		return nil, fmt.Errorf("failed to get sender counts: %w", err)
	}

	return counts, nil
}

// GetRecipientCounts returns message counts grouped by recipient address.
func (m *MessageRepository) GetRecipientCounts(ctx context.Context, limit int) ([]repository.AddressCount, error) {
	query := `SELECT r.address, r.name, COUNT(*) as count 
		FROM message_recipients r 
		GROUP BY r.address, r.name ORDER BY count DESC LIMIT $1`

	var counts []repository.AddressCount
	if err := m.repo.db().SelectContext(ctx, &counts, query, limit); err != nil {
		return nil, fmt.Errorf("failed to get recipient counts: %w", err)
	}

	return counts, nil
}

// StoreRawBody stores the raw message body for a message.
func (m *MessageRepository) StoreRawBody(ctx context.Context, id domain.ID, rawBody []byte) error {
	query := `UPDATE messages SET raw_body = $1, updated_at = $2 WHERE id = $3`

	result, err := m.repo.db().ExecContext(ctx, query, rawBody, time.Now().UTC(), string(id))
	if err != nil {
		return fmt.Errorf("failed to store raw body: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("message", string(id))
	}

	return nil
}

// GetRawBody retrieves the raw message body.
func (m *MessageRepository) GetRawBody(ctx context.Context, id domain.ID) ([]byte, error) {
	query := `SELECT raw_body FROM messages WHERE id = $1`

	var rawBody []byte
	if err := m.repo.db().GetContext(ctx, &rawBody, query, string(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("message", string(id))
		}
		return nil, fmt.Errorf("failed to get raw body: %w", err)
	}

	return rawBody, nil
}

// Ensure MessageRepository implements repository.MessageRepository
var _ repository.MessageRepository = (*MessageRepository)(nil)
