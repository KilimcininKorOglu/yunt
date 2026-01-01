package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yunt/internal/domain"
)

// StatsRepository provides methods for retrieving and calculating statistics.
type StatsRepository struct {
	repo *Repository
}

// NewStatsRepository creates a new SQLite stats repository.
func NewStatsRepository(repo *Repository) *StatsRepository {
	return &StatsRepository{repo: repo}
}

// GetStats retrieves comprehensive system-wide statistics.
func (s *StatsRepository) GetStats(ctx context.Context) (*domain.Stats, error) {
	stats := domain.NewStats()

	// Get user stats
	userStats, err := s.GetUserStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	stats.Users = *userStats

	// Get mailbox aggregate stats
	mailboxStats, err := s.GetMailboxAggregateStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get mailbox stats: %w", err)
	}
	stats.Mailboxes = *mailboxStats

	// Get message aggregate stats
	messageStats, err := s.GetMessageAggregateStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get message stats: %w", err)
	}
	stats.Messages = *messageStats

	// Get storage stats
	storageStats, err := s.GetStorageStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}
	stats.Storage = *storageStats

	return stats, nil
}

// GetUserStats retrieves user-related statistics.
func (s *StatsRepository) GetUserStats(ctx context.Context) (*domain.UserStats, error) {
	query := `SELECT 
		COUNT(*) as total_users,
		SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_users,
		SUM(CASE WHEN status = 'inactive' THEN 1 ELSE 0 END) as inactive_users,
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending_users,
		SUM(CASE WHEN role = 'admin' THEN 1 ELSE 0 END) as admin_users
		FROM users WHERE deleted_at IS NULL`

	var stats struct {
		TotalUsers    int64 `db:"total_users"`
		ActiveUsers   int64 `db:"active_users"`
		InactiveUsers int64 `db:"inactive_users"`
		PendingUsers  int64 `db:"pending_users"`
		AdminUsers    int64 `db:"admin_users"`
	}

	if err := s.repo.db().GetContext(ctx, &stats, query); err != nil {
		return nil, fmt.Errorf("failed to query user stats: %w", err)
	}

	return &domain.UserStats{
		TotalUsers:    stats.TotalUsers,
		ActiveUsers:   stats.ActiveUsers,
		InactiveUsers: stats.InactiveUsers,
		PendingUsers:  stats.PendingUsers,
		AdminUsers:    stats.AdminUsers,
	}, nil
}

// GetMailboxAggregateStats retrieves mailbox-related aggregate statistics.
func (s *StatsRepository) GetMailboxAggregateStats(ctx context.Context) (*domain.MailboxAggregateStats, error) {
	query := `SELECT 
		COUNT(*) as total_mailboxes,
		SUM(CASE WHEN message_count > 0 THEN 1 ELSE 0 END) as active_mailboxes,
		SUM(CASE WHEN message_count = 0 THEN 1 ELSE 0 END) as empty_mailboxes,
		SUM(CASE WHEN is_catch_all = 1 THEN 1 ELSE 0 END) as catch_all_mailboxes,
		SUM(CASE WHEN is_default = 1 THEN 1 ELSE 0 END) as default_mailboxes,
		COALESCE(AVG(message_count), 0) as avg_messages
		FROM mailboxes`

	var stats struct {
		TotalMailboxes   int64   `db:"total_mailboxes"`
		ActiveMailboxes  int64   `db:"active_mailboxes"`
		EmptyMailboxes   int64   `db:"empty_mailboxes"`
		CatchAllMailboxes int64  `db:"catch_all_mailboxes"`
		DefaultMailboxes int64   `db:"default_mailboxes"`
		AvgMessages      float64 `db:"avg_messages"`
	}

	if err := s.repo.db().GetContext(ctx, &stats, query); err != nil {
		return nil, fmt.Errorf("failed to query mailbox stats: %w", err)
	}

	// Count unique domains
	domainQuery := `SELECT COUNT(DISTINCT SUBSTR(address, INSTR(address, '@') + 1)) FROM mailboxes`
	var uniqueDomains int64
	if err := s.repo.db().GetContext(ctx, &uniqueDomains, domainQuery); err != nil {
		uniqueDomains = 0
	}

	return &domain.MailboxAggregateStats{
		TotalMailboxes:            stats.TotalMailboxes,
		ActiveMailboxes:           stats.ActiveMailboxes,
		EmptyMailboxes:            stats.EmptyMailboxes,
		CatchAllMailboxes:         stats.CatchAllMailboxes,
		DefaultMailboxes:          stats.DefaultMailboxes,
		UniqueDomains:             uniqueDomains,
		AverageMessagesPerMailbox: stats.AvgMessages,
	}, nil
}

// GetMessageAggregateStats retrieves message-related aggregate statistics.
func (s *StatsRepository) GetMessageAggregateStats(ctx context.Context) (*domain.MessageAggregateStats, error) {
	query := `SELECT 
		COUNT(*) as total_messages,
		SUM(CASE WHEN status = 'unread' THEN 1 ELSE 0 END) as unread_messages,
		SUM(CASE WHEN status = 'read' THEN 1 ELSE 0 END) as read_messages,
		SUM(CASE WHEN is_starred = 1 THEN 1 ELSE 0 END) as starred_messages,
		SUM(CASE WHEN is_spam = 1 THEN 1 ELSE 0 END) as spam_messages,
		SUM(CASE WHEN attachment_count > 0 THEN 1 ELSE 0 END) as messages_with_attachments,
		COALESCE(SUM(attachment_count), 0) as total_attachments,
		COALESCE(AVG(size), 0) as avg_message_size,
		COALESCE(MAX(size), 0) as largest_message
		FROM messages`

	var stats struct {
		TotalMessages           int64   `db:"total_messages"`
		UnreadMessages          int64   `db:"unread_messages"`
		ReadMessages            int64   `db:"read_messages"`
		StarredMessages         int64   `db:"starred_messages"`
		SpamMessages            int64   `db:"spam_messages"`
		MessagesWithAttachments int64   `db:"messages_with_attachments"`
		TotalAttachments        int64   `db:"total_attachments"`
		AvgMessageSize          float64 `db:"avg_message_size"`
		LargestMessage          int64   `db:"largest_message"`
	}

	if err := s.repo.db().GetContext(ctx, &stats, query); err != nil {
		return nil, fmt.Errorf("failed to query message stats: %w", err)
	}

	return &domain.MessageAggregateStats{
		TotalMessages:           stats.TotalMessages,
		UnreadMessages:          stats.UnreadMessages,
		ReadMessages:            stats.ReadMessages,
		StarredMessages:         stats.StarredMessages,
		SpamMessages:            stats.SpamMessages,
		MessagesWithAttachments: stats.MessagesWithAttachments,
		TotalAttachments:        stats.TotalAttachments,
		AverageMessageSize:      stats.AvgMessageSize,
		LargestMessage:          stats.LargestMessage,
	}, nil
}

// GetStorageStats retrieves storage-related statistics.
func (s *StatsRepository) GetStorageStats(ctx context.Context) (*domain.StorageStats, error) {
	// Get message storage stats
	messageQuery := `SELECT COALESCE(SUM(size), 0) FROM messages`
	var messageStorageSize int64
	if err := s.repo.db().GetContext(ctx, &messageStorageSize, messageQuery); err != nil {
		return nil, fmt.Errorf("failed to query message storage: %w", err)
	}

	// Get attachment storage stats
	attachmentQuery := `SELECT COALESCE(SUM(size), 0) FROM attachments`
	var attachmentStorageSize int64
	if err := s.repo.db().GetContext(ctx, &attachmentStorageSize, attachmentQuery); err != nil {
		return nil, fmt.Errorf("failed to query attachment storage: %w", err)
	}

	// Get mailbox size stats
	mailboxQuery := `SELECT 
		COALESCE(AVG(total_size), 0) as avg_size,
		COALESCE(MAX(total_size), 0) as max_size
		FROM mailboxes`

	var mailboxStats struct {
		AvgSize float64 `db:"avg_size"`
		MaxSize int64   `db:"max_size"`
	}
	if err := s.repo.db().GetContext(ctx, &mailboxStats, mailboxQuery); err != nil {
		return nil, fmt.Errorf("failed to query mailbox storage: %w", err)
	}

	return &domain.StorageStats{
		TotalSize:             messageStorageSize + attachmentStorageSize,
		MessageStorageSize:    messageStorageSize,
		AttachmentStorageSize: attachmentStorageSize,
		AverageMailboxSize:    mailboxStats.AvgSize,
		LargestMailboxSize:    mailboxStats.MaxSize,
	}, nil
}

// GetMessageStats retrieves statistics for a filtered set of messages.
func (s *StatsRepository) GetMessageStats(ctx context.Context, filter *domain.StatsFilter) (*domain.MessageStats, error) {
	query := `SELECT 
		COUNT(*) as count,
		SUM(CASE WHEN status = 'unread' THEN 1 ELSE 0 END) as unread_count,
		SUM(CASE WHEN is_starred = 1 THEN 1 ELSE 0 END) as starred_count,
		SUM(CASE WHEN is_spam = 1 THEN 1 ELSE 0 END) as spam_count,
		COALESCE(SUM(size), 0) as total_size,
		COALESCE(SUM(attachment_count), 0) as attachment_count,
		MIN(received_at) as oldest_message,
		MAX(received_at) as newest_message
		FROM messages WHERE 1=1`

	args := make([]interface{}, 0)

	if filter != nil {
		if filter.MailboxID != nil {
			query += " AND mailbox_id = ?"
			args = append(args, string(*filter.MailboxID))
		}

		if len(filter.MailboxIDs) > 0 {
			query += " AND mailbox_id IN ("
			for i, id := range filter.MailboxIDs {
				if i > 0 {
					query += ","
				}
				query += "?"
				args = append(args, string(id))
			}
			query += ")"
		}

		if filter.DateFrom != nil {
			query += " AND received_at >= ?"
			args = append(args, filter.DateFrom.Time)
		}

		if filter.DateTo != nil {
			query += " AND received_at <= ?"
			args = append(args, filter.DateTo.Time)
		}

		if filter.ExcludeSpam {
			query += " AND is_spam = 0"
		}
	}

	var stats struct {
		Count           int64        `db:"count"`
		UnreadCount     int64        `db:"unread_count"`
		StarredCount    int64        `db:"starred_count"`
		SpamCount       int64        `db:"spam_count"`
		TotalSize       int64        `db:"total_size"`
		AttachmentCount int64        `db:"attachment_count"`
		OldestMessage   sql.NullTime `db:"oldest_message"`
		NewestMessage   sql.NullTime `db:"newest_message"`
	}

	if err := s.repo.db().GetContext(ctx, &stats, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query message stats: %w", err)
	}

	result := &domain.MessageStats{
		Count:           stats.Count,
		UnreadCount:     stats.UnreadCount,
		StarredCount:    stats.StarredCount,
		SpamCount:       stats.SpamCount,
		TotalSize:       stats.TotalSize,
		AttachmentCount: stats.AttachmentCount,
	}

	if stats.OldestMessage.Valid {
		ts := domain.Timestamp{Time: stats.OldestMessage.Time}
		result.OldestMessage = &ts
	}

	if stats.NewestMessage.Valid {
		ts := domain.Timestamp{Time: stats.NewestMessage.Time}
		result.NewestMessage = &ts
	}

	return result, nil
}

// GetMessageStatsByMailbox retrieves message statistics for a specific mailbox.
func (s *StatsRepository) GetMessageStatsByMailbox(ctx context.Context, mailboxID domain.ID) (*domain.MessageStats, error) {
	filter := &domain.StatsFilter{MailboxID: &mailboxID}
	return s.GetMessageStats(ctx, filter)
}

// GetMessageStatsByUser retrieves message statistics for all mailboxes owned by a user.
func (s *StatsRepository) GetMessageStatsByUser(ctx context.Context, userID domain.ID) (*domain.MessageStats, error) {
	// First get the user's mailbox IDs
	mailboxQuery := `SELECT id FROM mailboxes WHERE user_id = ?`
	var mailboxIDs []string
	if err := s.repo.db().SelectContext(ctx, &mailboxIDs, mailboxQuery, string(userID)); err != nil {
		return nil, fmt.Errorf("failed to get user mailboxes: %w", err)
	}

	if len(mailboxIDs) == 0 {
		return domain.NewMessageStats(), nil
	}

	ids := make([]domain.ID, len(mailboxIDs))
	for i, id := range mailboxIDs {
		ids[i] = domain.ID(id)
	}

	filter := &domain.StatsFilter{MailboxIDs: ids}
	return s.GetMessageStats(ctx, filter)
}

// GetDailyStats retrieves daily statistics for a date range.
func (s *StatsRepository) GetDailyStats(ctx context.Context, from, to time.Time) ([]domain.DailyStats, error) {
	query := `SELECT 
		DATE(received_at) as date,
		COUNT(*) as received_count,
		COALESCE(SUM(size), 0) as total_size,
		SUM(CASE WHEN is_spam = 1 THEN 1 ELSE 0 END) as spam_count,
		COALESCE(SUM(attachment_count), 0) as attachment_count
		FROM messages
		WHERE received_at >= ? AND received_at <= ?
		GROUP BY DATE(received_at)
		ORDER BY date`

	var rows []struct {
		Date            string `db:"date"`
		ReceivedCount   int64  `db:"received_count"`
		TotalSize       int64  `db:"total_size"`
		SpamCount       int64  `db:"spam_count"`
		AttachmentCount int64  `db:"attachment_count"`
	}

	if err := s.repo.db().SelectContext(ctx, &rows, query, from, to); err != nil {
		return nil, fmt.Errorf("failed to query daily stats: %w", err)
	}

	stats := make([]domain.DailyStats, len(rows))
	for i, row := range rows {
		stats[i] = domain.DailyStats{
			Date:            row.Date,
			ReceivedCount:   row.ReceivedCount,
			TotalSize:       row.TotalSize,
			SpamCount:       row.SpamCount,
			AttachmentCount: row.AttachmentCount,
		}
	}

	return stats, nil
}

// GetTopSenders retrieves the top senders by message count.
func (s *StatsRepository) GetTopSenders(ctx context.Context, limit int) ([]domain.SenderStats, error) {
	query := `SELECT 
		from_address as address,
		from_name as name,
		COUNT(*) as message_count,
		COALESCE(SUM(size), 0) as total_size,
		SUM(CASE WHEN is_spam = 1 THEN 1 ELSE 0 END) as spam_count,
		MIN(received_at) as first_seen,
		MAX(received_at) as last_seen
		FROM messages
		GROUP BY from_address
		ORDER BY message_count DESC
		LIMIT ?`

	var rows []struct {
		Address      string         `db:"address"`
		Name         sql.NullString `db:"name"`
		MessageCount int64          `db:"message_count"`
		TotalSize    int64          `db:"total_size"`
		SpamCount    int64          `db:"spam_count"`
		FirstSeen    sql.NullTime   `db:"first_seen"`
		LastSeen     sql.NullTime   `db:"last_seen"`
	}

	if err := s.repo.db().SelectContext(ctx, &rows, query, limit); err != nil {
		return nil, fmt.Errorf("failed to query top senders: %w", err)
	}

	stats := make([]domain.SenderStats, len(rows))
	for i, row := range rows {
		stats[i] = domain.SenderStats{
			Address:      row.Address,
			MessageCount: row.MessageCount,
			TotalSize:    row.TotalSize,
			SpamCount:    row.SpamCount,
		}
		if row.Name.Valid {
			stats[i].Name = row.Name.String
		}
		if row.FirstSeen.Valid {
			ts := domain.Timestamp{Time: row.FirstSeen.Time}
			stats[i].FirstSeen = &ts
		}
		if row.LastSeen.Valid {
			ts := domain.Timestamp{Time: row.LastSeen.Time}
			stats[i].LastSeen = &ts
		}
	}

	return stats, nil
}

// GetTopRecipients retrieves the top recipients by message count.
func (s *StatsRepository) GetTopRecipients(ctx context.Context, limit int) ([]domain.RecipientStats, error) {
	query := `SELECT 
		r.address as address,
		r.name as name,
		COUNT(*) as message_count,
		r.recipient_type as type
		FROM message_recipients r
		GROUP BY r.address
		ORDER BY message_count DESC
		LIMIT ?`

	var rows []struct {
		Address      string         `db:"address"`
		Name         sql.NullString `db:"name"`
		MessageCount int64          `db:"message_count"`
		Type         string         `db:"type"`
	}

	if err := s.repo.db().SelectContext(ctx, &rows, query, limit); err != nil {
		return nil, fmt.Errorf("failed to query top recipients: %w", err)
	}

	stats := make([]domain.RecipientStats, len(rows))
	for i, row := range rows {
		stats[i] = domain.RecipientStats{
			Address:      row.Address,
			MessageCount: row.MessageCount,
			Type:         row.Type,
		}
		if row.Name.Valid {
			stats[i].Name = row.Name.String
		}
	}

	return stats, nil
}

// GetContentTypeStats retrieves statistics grouped by content type.
func (s *StatsRepository) GetContentTypeStats(ctx context.Context) ([]domain.ContentTypeStats, error) {
	query := `SELECT 
		content_type as content_type,
		COUNT(*) as count,
		COALESCE(SUM(size), 0) as total_size
		FROM attachments
		GROUP BY content_type
		ORDER BY count DESC`

	var rows []struct {
		ContentType string `db:"content_type"`
		Count       int64  `db:"count"`
		TotalSize   int64  `db:"total_size"`
	}

	if err := s.repo.db().SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to query content type stats: %w", err)
	}

	stats := make([]domain.ContentTypeStats, len(rows))
	for i, row := range rows {
		stats[i] = domain.ContentTypeStats{
			ContentType: row.ContentType,
			Count:       row.Count,
			TotalSize:   row.TotalSize,
		}
	}

	return stats, nil
}

// GetTrendData retrieves trend data for a specified time range.
func (s *StatsRepository) GetTrendData(ctx context.Context, timeRange domain.StatsTimeRange) (*domain.TrendData, error) {
	var from time.Time
	var groupFormat string
	now := time.Now().UTC()

	switch timeRange {
	case domain.StatsTimeRange24Hours:
		from = now.Add(-24 * time.Hour)
		groupFormat = "strftime('%Y-%m-%d %H:00', received_at)"
	case domain.StatsTimeRange7Days:
		from = now.AddDate(0, 0, -7)
		groupFormat = "DATE(received_at)"
	case domain.StatsTimeRange30Days:
		from = now.AddDate(0, 0, -30)
		groupFormat = "DATE(received_at)"
	case domain.StatsTimeRange90Days:
		from = now.AddDate(0, 0, -90)
		groupFormat = "strftime('%Y-%W', received_at)"
	default:
		from = time.Time{}
		groupFormat = "strftime('%Y-%m', received_at)"
	}

	var whereClause string
	var args []interface{}
	if !from.IsZero() {
		whereClause = " WHERE received_at >= ?"
		args = append(args, from)
	}

	query := fmt.Sprintf(`SELECT 
		%s as label,
		COUNT(*) as message_count,
		COALESCE(SUM(size), 0) as size_total
		FROM messages
		%s
		GROUP BY label
		ORDER BY label`, groupFormat, whereClause)

	var rows []struct {
		Label        string `db:"label"`
		MessageCount int64  `db:"message_count"`
		SizeTotal    int64  `db:"size_total"`
	}

	if err := s.repo.db().SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query trend data: %w", err)
	}

	result := &domain.TrendData{
		Labels:        make([]string, len(rows)),
		MessageCounts: make([]int64, len(rows)),
		SizeTotals:    make([]int64, len(rows)),
	}

	for i, row := range rows {
		result.Labels[i] = row.Label
		result.MessageCounts[i] = row.MessageCount
		result.SizeTotals[i] = row.SizeTotal
	}

	return result, nil
}

// RecalculateAllMailboxStats recalculates statistics for all mailboxes from their messages.
// This is useful for repairing denormalized counters.
func (s *StatsRepository) RecalculateAllMailboxStats(ctx context.Context) (int64, error) {
	query := `UPDATE mailboxes SET 
		message_count = (SELECT COUNT(*) FROM messages WHERE messages.mailbox_id = mailboxes.id),
		unread_count = (SELECT COUNT(*) FROM messages WHERE messages.mailbox_id = mailboxes.id AND messages.status = 'unread'),
		total_size = (SELECT COALESCE(SUM(size), 0) FROM messages WHERE messages.mailbox_id = mailboxes.id),
		updated_at = ?`

	result, err := s.repo.db().ExecContext(ctx, query, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("failed to recalculate mailbox stats: %w", err)
	}

	return result.RowsAffected()
}

// VerifyMailboxStats checks if mailbox denormalized counters are accurate.
// Returns mailbox IDs with mismatched statistics.
func (s *StatsRepository) VerifyMailboxStats(ctx context.Context) ([]domain.ID, error) {
	query := `SELECT m.id FROM mailboxes m
		WHERE m.message_count != (SELECT COUNT(*) FROM messages WHERE mailbox_id = m.id)
		   OR m.unread_count != (SELECT COUNT(*) FROM messages WHERE mailbox_id = m.id AND status = 'unread')
		   OR m.total_size != (SELECT COALESCE(SUM(size), 0) FROM messages WHERE mailbox_id = m.id)`

	var ids []string
	if err := s.repo.db().SelectContext(ctx, &ids, query); err != nil {
		return nil, fmt.Errorf("failed to verify mailbox stats: %w", err)
	}

	result := make([]domain.ID, len(ids))
	for i, id := range ids {
		result[i] = domain.ID(id)
	}

	return result, nil
}

// GetStarredCount returns the count of starred messages.
func (s *StatsRepository) GetStarredCount(ctx context.Context, filter *domain.StatsFilter) (int64, error) {
	query := "SELECT COUNT(*) FROM messages WHERE is_starred = 1"
	args := make([]interface{}, 0)

	if filter != nil && filter.MailboxID != nil {
		query += " AND mailbox_id = ?"
		args = append(args, string(*filter.MailboxID))
	}

	var count int64
	if err := s.repo.db().GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count starred messages: %w", err)
	}

	return count, nil
}

// GetSpamCount returns the count of spam messages.
func (s *StatsRepository) GetSpamCount(ctx context.Context, filter *domain.StatsFilter) (int64, error) {
	query := "SELECT COUNT(*) FROM messages WHERE is_spam = 1"
	args := make([]interface{}, 0)

	if filter != nil && filter.MailboxID != nil {
		query += " AND mailbox_id = ?"
		args = append(args, string(*filter.MailboxID))
	}

	var count int64
	if err := s.repo.db().GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count spam messages: %w", err)
	}

	return count, nil
}

// GetUnreadCount returns the count of unread messages.
func (s *StatsRepository) GetUnreadCount(ctx context.Context, filter *domain.StatsFilter) (int64, error) {
	query := "SELECT COUNT(*) FROM messages WHERE status = 'unread'"
	args := make([]interface{}, 0)

	if filter != nil && filter.MailboxID != nil {
		query += " AND mailbox_id = ?"
		args = append(args, string(*filter.MailboxID))
	}

	var count int64
	if err := s.repo.db().GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count unread messages: %w", err)
	}

	return count, nil
}

// GetAttachmentStats returns statistics about attachments.
func (s *StatsRepository) GetAttachmentStats(ctx context.Context) (*domain.ContentTypeStats, error) {
	query := `SELECT 
		COUNT(*) as count,
		COALESCE(SUM(size), 0) as total_size
		FROM attachments`

	var stats struct {
		Count     int64 `db:"count"`
		TotalSize int64 `db:"total_size"`
	}

	if err := s.repo.db().GetContext(ctx, &stats, query); err != nil {
		return nil, fmt.Errorf("failed to query attachment stats: %w", err)
	}

	return &domain.ContentTypeStats{
		ContentType: "all",
		Count:       stats.Count,
		TotalSize:   stats.TotalSize,
	}, nil
}
