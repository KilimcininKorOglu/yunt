package postgres

import (
	"context"
	"fmt"
	"strings"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// SearchRepository provides PostgreSQL full-text search capabilities using tsvector.
type SearchRepository struct {
	repo *Repository
}

// SearchResult represents a search result with ranking.
type SearchResult struct {
	ID       string  `db:"id"`
	Type     string  `db:"type"`
	Rank     float64 `db:"rank"`
	Headline string  `db:"headline"`
}

// NewSearchRepository creates a new PostgreSQL search repository.
func NewSearchRepository(repo *Repository) *SearchRepository {
	return &SearchRepository{repo: repo}
}

// SearchMessages performs a full-text search on messages using tsvector.
func (s *SearchRepository) SearchMessages(ctx context.Context, query string, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.Message], error) {
	if query == "" {
		return s.repo.Messages().List(ctx, filter, opts)
	}

	// Parse the query for PostgreSQL full-text search
	tsQuery := s.buildTSQuery(query)

	var sb strings.Builder
	args := make([]interface{}, 0)
	argIndex := 1

	// Count query with full-text search
	countQueryBuilder := strings.Builder{}
	countQueryBuilder.WriteString(`SELECT COUNT(*) FROM messages WHERE search_vector @@ to_tsquery('english', $1)`)
	countArgs := []interface{}{tsQuery}
	countArgIndex := 2

	sb.WriteString(`SELECT id, mailbox_id, message_id, from_name, from_address, subject, 
		text_body, html_body, raw_body, headers, content_type, size, attachment_count, 
		status, is_starred, is_spam, in_reply_to, references_list, 
		received_at, sent_at, created_at, updated_at,
		ts_rank(search_vector, to_tsquery('english', $1)) as rank
		FROM messages WHERE search_vector @@ to_tsquery('english', $1)`)
	args = append(args, tsQuery)
	argIndex++

	// Apply additional filters
	if filter != nil {
		if filter.MailboxID != nil {
			sb.WriteString(fmt.Sprintf(" AND mailbox_id = $%d", argIndex))
			args = append(args, string(*filter.MailboxID))
			countQueryBuilder.WriteString(fmt.Sprintf(" AND mailbox_id = $%d", countArgIndex))
			countArgs = append(countArgs, string(*filter.MailboxID))
			argIndex++
			countArgIndex++
		}

		if len(filter.MailboxIDs) > 0 {
			placeholders := make([]string, len(filter.MailboxIDs))
			for i, id := range filter.MailboxIDs {
				placeholders[i] = fmt.Sprintf("$%d", argIndex)
				args = append(args, string(id))
				countArgs = append(countArgs, string(id))
				argIndex++
				countArgIndex++
			}
			clause := fmt.Sprintf(" AND mailbox_id IN (%s)", strings.Join(placeholders, ","))
			sb.WriteString(clause)
			countQueryBuilder.WriteString(clause)
		}

		if filter.Status != nil {
			sb.WriteString(fmt.Sprintf(" AND status = $%d", argIndex))
			args = append(args, string(*filter.Status))
			countQueryBuilder.WriteString(fmt.Sprintf(" AND status = $%d", countArgIndex))
			countArgs = append(countArgs, string(*filter.Status))
			argIndex++
			countArgIndex++
		}

		if filter.IsStarred != nil {
			sb.WriteString(fmt.Sprintf(" AND is_starred = $%d", argIndex))
			args = append(args, *filter.IsStarred)
			countQueryBuilder.WriteString(fmt.Sprintf(" AND is_starred = $%d", countArgIndex))
			countArgs = append(countArgs, *filter.IsStarred)
			argIndex++
			countArgIndex++
		}

		if filter.IsSpam != nil {
			sb.WriteString(fmt.Sprintf(" AND is_spam = $%d", argIndex))
			args = append(args, *filter.IsSpam)
			countQueryBuilder.WriteString(fmt.Sprintf(" AND is_spam = $%d", countArgIndex))
			countArgs = append(countArgs, *filter.IsSpam)
			argIndex++
			countArgIndex++
		}

		if filter.ExcludeSpam {
			sb.WriteString(" AND is_spam = false")
			countQueryBuilder.WriteString(" AND is_spam = false")
		}

		if filter.ReceivedAfter != nil {
			sb.WriteString(fmt.Sprintf(" AND received_at > $%d", argIndex))
			args = append(args, filter.ReceivedAfter.Time)
			countQueryBuilder.WriteString(fmt.Sprintf(" AND received_at > $%d", countArgIndex))
			countArgs = append(countArgs, filter.ReceivedAfter.Time)
			argIndex++
			countArgIndex++
		}

		if filter.ReceivedBefore != nil {
			sb.WriteString(fmt.Sprintf(" AND received_at < $%d", argIndex))
			args = append(args, filter.ReceivedBefore.Time)
			countQueryBuilder.WriteString(fmt.Sprintf(" AND received_at < $%d", countArgIndex))
			countArgs = append(countArgs, filter.ReceivedBefore.Time)
			argIndex++
			countArgIndex++
		}
	}

	// Get total count
	var total int64
	if err := s.repo.db().GetContext(ctx, &total, countQueryBuilder.String(), countArgs...); err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// Order by rank (relevance) by default
	sb.WriteString(" ORDER BY rank DESC, received_at DESC")

	// Apply pagination
	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		sb.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset()))
	}

	var rows []messageRow
	if err := s.repo.db().SelectContext(ctx, &rows, sb.String(), args...); err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	messages := make([]*domain.Message, len(rows))
	for i, row := range rows {
		messages[i] = row.toMessage()
		if err := s.repo.messages.loadRecipients(ctx, messages[i]); err != nil {
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

// SearchWithHighlights performs a full-text search and returns highlighted results.
func (s *SearchRepository) SearchWithHighlights(ctx context.Context, query string, filter *repository.MessageFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.MessageSummary], error) {
	if query == "" {
		result, err := s.repo.Messages().List(ctx, filter, opts)
		if err != nil {
			return nil, err
		}
		summaries := make([]*domain.MessageSummary, len(result.Items))
		for i, msg := range result.Items {
			summaries[i] = msg.ToSummary(200)
		}
		return &repository.ListResult[*domain.MessageSummary]{
			Items:      summaries,
			Total:      result.Total,
			HasMore:    result.HasMore,
			Pagination: result.Pagination,
		}, nil
	}

	tsQuery := s.buildTSQuery(query)

	var sb strings.Builder
	args := make([]interface{}, 0)
	argIndex := 1

	// Count query
	countQueryBuilder := strings.Builder{}
	countQueryBuilder.WriteString(`SELECT COUNT(*) FROM messages WHERE search_vector @@ to_tsquery('english', $1)`)
	countArgs := []interface{}{tsQuery}
	countArgIndex := 2

	// Select with headline highlighting
	sb.WriteString(`SELECT 
		id, mailbox_id, from_address, from_name, subject, status, is_starred, is_spam,
		attachment_count, received_at,
		ts_headline('english', COALESCE(subject, ''), to_tsquery('english', $1), 
			'MaxWords=50, MinWords=30, StartSel=<mark>, StopSel=</mark>') as subject_highlight,
		ts_headline('english', COALESCE(text_body, ''), to_tsquery('english', $1), 
			'MaxWords=100, MinWords=50, StartSel=<mark>, StopSel=</mark>') as body_highlight,
		ts_rank(search_vector, to_tsquery('english', $1)) as rank
		FROM messages WHERE search_vector @@ to_tsquery('english', $1)`)
	args = append(args, tsQuery)
	argIndex++

	// Apply filters
	if filter != nil {
		if filter.MailboxID != nil {
			sb.WriteString(fmt.Sprintf(" AND mailbox_id = $%d", argIndex))
			args = append(args, string(*filter.MailboxID))
			countQueryBuilder.WriteString(fmt.Sprintf(" AND mailbox_id = $%d", countArgIndex))
			countArgs = append(countArgs, string(*filter.MailboxID))
			argIndex++
			countArgIndex++
		}

		if filter.ExcludeSpam {
			sb.WriteString(" AND is_spam = false")
			countQueryBuilder.WriteString(" AND is_spam = false")
		}
	}

	// Get total count
	var total int64
	if err := s.repo.db().GetContext(ctx, &total, countQueryBuilder.String(), countArgs...); err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	sb.WriteString(" ORDER BY rank DESC, received_at DESC")

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		sb.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", opts.Pagination.Limit(), opts.Pagination.Offset()))
	}

	type highlightRow struct {
		ID              string  `db:"id"`
		MailboxID       string  `db:"mailbox_id"`
		FromAddress     string  `db:"from_address"`
		FromName        *string `db:"from_name"`
		Subject         *string `db:"subject"`
		Status          string  `db:"status"`
		IsStarred       bool    `db:"is_starred"`
		IsSpam          bool    `db:"is_spam"`
		AttachmentCount int     `db:"attachment_count"`
		ReceivedAt      string  `db:"received_at"`
		SubjectHL       string  `db:"subject_highlight"`
		BodyHL          string  `db:"body_highlight"`
		Rank            float64 `db:"rank"`
	}

	var rows []highlightRow
	if err := s.repo.db().SelectContext(ctx, &rows, sb.String(), args...); err != nil {
		return nil, fmt.Errorf("failed to search messages with highlights: %w", err)
	}

	summaries := make([]*domain.MessageSummary, len(rows))
	for i, row := range rows {
		summary := &domain.MessageSummary{
			ID:             domain.ID(row.ID),
			MailboxID:      domain.ID(row.MailboxID),
			From:           domain.EmailAddress{Address: row.FromAddress},
			Status:         domain.MessageStatus(row.Status),
			IsStarred:      row.IsStarred,
			HasAttachments: row.AttachmentCount > 0,
		}
		if row.FromName != nil {
			summary.From.Name = *row.FromName
		}
		if row.Subject != nil {
			summary.Subject = *row.Subject
		}
		// Use highlighted body as preview
		summary.Preview = row.BodyHL

		summaries[i] = summary
	}

	result := &repository.ListResult[*domain.MessageSummary]{
		Items: summaries,
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

// buildTSQuery converts a user query into a PostgreSQL tsquery format.
// It handles quoted phrases and individual terms.
func (s *SearchRepository) buildTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	var parts []string
	inQuote := false
	var currentPhrase strings.Builder

	for _, char := range query {
		switch char {
		case '"':
			if inQuote {
				// End of phrase - wrap in parens with <->
				phrase := currentPhrase.String()
				if phrase != "" {
					words := strings.Fields(phrase)
					if len(words) > 1 {
						parts = append(parts, fmt.Sprintf("(%s)", strings.Join(words, " <-> ")))
					} else if len(words) == 1 {
						parts = append(parts, words[0])
					}
				}
				currentPhrase.Reset()
			}
			inQuote = !inQuote
		case ' ':
			if inQuote {
				currentPhrase.WriteRune(char)
			} else {
				word := currentPhrase.String()
				if word != "" {
					parts = append(parts, word)
				}
				currentPhrase.Reset()
			}
		default:
			currentPhrase.WriteRune(char)
		}
	}

	// Handle remaining content
	remaining := currentPhrase.String()
	if remaining != "" {
		if inQuote {
			words := strings.Fields(remaining)
			if len(words) > 1 {
				parts = append(parts, fmt.Sprintf("(%s)", strings.Join(words, " <-> ")))
			} else if len(words) == 1 {
				parts = append(parts, words[0])
			}
		} else {
			parts = append(parts, remaining)
		}
	}

	// Convert to tsquery with prefix matching for partial words
	for i, part := range parts {
		// Add prefix matching (:*) for partial word matching
		if !strings.Contains(part, "<->") && !strings.HasPrefix(part, "(") {
			parts[i] = part + ":*"
		}
	}

	return strings.Join(parts, " & ")
}

// SuggestSearchTerms provides search term suggestions based on partial input.
func (s *SearchRepository) SuggestSearchTerms(ctx context.Context, prefix string, limit int) ([]string, error) {
	if prefix == "" || limit <= 0 {
		return []string{}, nil
	}

	// Search in subjects and from addresses for suggestions
	query := `
		SELECT DISTINCT word FROM (
			SELECT unnest(string_to_array(subject, ' ')) as word FROM messages
			UNION
			SELECT from_address as word FROM messages
		) words
		WHERE LOWER(word) LIKE LOWER($1)
		ORDER BY word
		LIMIT $2`

	var suggestions []string
	if err := s.repo.db().SelectContext(ctx, &suggestions, query, prefix+"%", limit); err != nil {
		return nil, fmt.Errorf("failed to get search suggestions: %w", err)
	}

	return suggestions, nil
}

// RebuildSearchIndex rebuilds the full-text search index for all messages.
func (s *SearchRepository) RebuildSearchIndex(ctx context.Context) error {
	query := `
		UPDATE messages SET search_vector = 
			setweight(to_tsvector('english', COALESCE(subject, '')), 'A') ||
			setweight(to_tsvector('english', COALESCE(from_address, '')), 'B') ||
			setweight(to_tsvector('english', COALESCE(from_name, '')), 'B') ||
			setweight(to_tsvector('english', COALESCE(text_body, '')), 'C')`

	_, err := s.repo.db().ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to rebuild search index: %w", err)
	}

	return nil
}

// GetSearchStats returns statistics about the search index.
func (s *SearchRepository) GetSearchStats(ctx context.Context) (*SearchStats, error) {
	stats := &SearchStats{}

	// Get total indexed messages
	if err := s.repo.db().GetContext(ctx, &stats.IndexedMessages,
		`SELECT COUNT(*) FROM messages WHERE search_vector IS NOT NULL`); err != nil {
		return nil, fmt.Errorf("failed to get indexed message count: %w", err)
	}

	// Get unindexed messages
	if err := s.repo.db().GetContext(ctx, &stats.UnindexedMessages,
		`SELECT COUNT(*) FROM messages WHERE search_vector IS NULL`); err != nil {
		return nil, fmt.Errorf("failed to get unindexed message count: %w", err)
	}

	// Get index size estimate
	if err := s.repo.db().GetContext(ctx, &stats.IndexSizeBytes,
		`SELECT pg_total_relation_size('idx_messages_search_vector')`); err != nil {
		// Index might not exist yet
		stats.IndexSizeBytes = 0
	}

	return stats, nil
}

// SearchStats contains statistics about the search index.
type SearchStats struct {
	IndexedMessages   int64 `db:"indexed_messages"`
	UnindexedMessages int64 `db:"unindexed_messages"`
	IndexSizeBytes    int64 `db:"index_size_bytes"`
}

// SearchUsers performs a search on users.
func (s *SearchRepository) SearchUsers(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	filter := &repository.UserFilter{Search: query}
	return s.repo.Users().List(ctx, filter, opts)
}

// SearchMailboxes performs a search on mailboxes.
func (s *SearchRepository) SearchMailboxes(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.Mailbox], error) {
	filter := &repository.MailboxFilter{Search: query}
	return s.repo.Mailboxes().List(ctx, filter, opts)
}

// GlobalSearch performs a search across all entity types.
func (s *SearchRepository) GlobalSearch(ctx context.Context, query string, limit int) (*GlobalSearchResults, error) {
	if query == "" {
		return &GlobalSearchResults{}, nil
	}

	results := &GlobalSearchResults{}
	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: limit,
		},
	}

	// Search messages
	msgFilter := &repository.MessageFilter{Search: query, ExcludeSpam: true}
	msgResult, err := s.repo.Messages().List(ctx, msgFilter, opts)
	if err == nil {
		results.Messages = msgResult.Items
		results.MessageCount = msgResult.Total
	}

	// Search users
	userResult, err := s.SearchUsers(ctx, query, opts)
	if err == nil {
		results.Users = userResult.Items
		results.UserCount = userResult.Total
	}

	// Search mailboxes
	mailboxResult, err := s.SearchMailboxes(ctx, query, opts)
	if err == nil {
		results.Mailboxes = mailboxResult.Items
		results.MailboxCount = mailboxResult.Total
	}

	return results, nil
}

// GlobalSearchResults contains results from a global search across all entity types.
type GlobalSearchResults struct {
	Messages     []*domain.Message
	MessageCount int64
	Users        []*domain.User
	UserCount    int64
	Mailboxes    []*domain.Mailbox
	MailboxCount int64
}
