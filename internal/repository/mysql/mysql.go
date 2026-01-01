package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// Repository is the main MySQL repository implementation.
// It aggregates all entity repositories and provides transaction support.
type Repository struct {
	pool *ConnectionPool

	users       *UserRepository
	mailboxes   *MailboxRepository
	messages    *MessageRepository
	attachments *AttachmentRepository
	webhooks    *WebhookRepository
	settings    *SettingsRepository

	// For transaction support
	tx   *sqlx.Tx
	isTx bool
	mu   sync.RWMutex
}

// New creates a new MySQL repository with the given connection pool.
func New(pool *ConnectionPool) (*Repository, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool is required")
	}

	repo := &Repository{
		pool: pool,
	}

	// Initialize entity repositories
	repo.users = NewUserRepository(repo)
	repo.mailboxes = NewMailboxRepository(repo)
	repo.messages = NewMessageRepository(repo)
	repo.attachments = NewAttachmentRepository(repo)
	repo.webhooks = NewWebhookRepository(repo)
	repo.settings = NewSettingsRepository(repo)

	// Create schema if needed
	if err := repo.createSchema(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return repo, nil
}

// Users returns the user repository.
func (r *Repository) Users() repository.UserRepository {
	return r.users
}

// Mailboxes returns the mailbox repository.
func (r *Repository) Mailboxes() repository.MailboxRepository {
	return r.mailboxes
}

// Messages returns the message repository.
func (r *Repository) Messages() repository.MessageRepository {
	return r.messages
}

// Attachments returns the attachment repository.
func (r *Repository) Attachments() repository.AttachmentRepository {
	return r.attachments
}

// Webhooks returns the webhook repository.
func (r *Repository) Webhooks() repository.WebhookRepository {
	return r.webhooks
}

// Settings returns the settings repository.
func (r *Repository) Settings() repository.SettingsRepository {
	return r.settings
}

// Transaction executes the given function within a database transaction.
func (r *Repository) Transaction(ctx context.Context, fn func(tx repository.Repository) error) error {
	return r.TransactionWithOptions(ctx, repository.TransactionOptions{}, fn)
}

// TransactionWithOptions executes the function within a transaction with custom options.
func (r *Repository) TransactionWithOptions(ctx context.Context, opts repository.TransactionOptions, fn func(tx repository.Repository) error) error {
	if r.isTx {
		// Already in a transaction, just execute the function
		return fn(r)
	}

	sqlOpts := &sql.TxOptions{
		ReadOnly: opts.ReadOnly,
	}

	switch opts.IsolationLevel {
	case repository.IsolationSerializable:
		sqlOpts.Isolation = sql.LevelSerializable
	case repository.IsolationRepeatableRead:
		sqlOpts.Isolation = sql.LevelRepeatableRead
	case repository.IsolationReadCommitted:
		sqlOpts.Isolation = sql.LevelReadCommitted
	case repository.IsolationReadUncommitted:
		sqlOpts.Isolation = sql.LevelReadUncommitted
	}

	tx, err := r.pool.BeginTx(ctx, sqlOpts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txRepo := r.withTx(tx)
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// withTx creates a new repository instance that uses the given transaction.
func (r *Repository) withTx(tx *sqlx.Tx) *Repository {
	txRepo := &Repository{
		pool: r.pool,
		tx:   tx,
		isTx: true,
	}

	// Initialize entity repositories with the transaction
	txRepo.users = NewUserRepository(txRepo)
	txRepo.mailboxes = NewMailboxRepository(txRepo)
	txRepo.messages = NewMessageRepository(txRepo)
	txRepo.attachments = NewAttachmentRepository(txRepo)
	txRepo.webhooks = NewWebhookRepository(txRepo)
	txRepo.settings = NewSettingsRepository(txRepo)

	return txRepo
}

// Health checks the health of the database connection.
func (r *Repository) Health(ctx context.Context) error {
	return r.pool.Health(ctx)
}

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.pool.Close()
}

// db returns the database connection or transaction.
func (r *Repository) db() sqlxDB {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.isTx && r.tx != nil {
		return r.tx
	}
	return r.pool.DB()
}

// sqlxDB is an interface that both *sqlx.DB and *sqlx.Tx implement.
type sqlxDB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	Rebind(query string) string
}

// createSchema creates all necessary database tables.
func (r *Repository) createSchema(ctx context.Context) error {
	// MySQL-specific schema with UTF8MB4 encoding and FULLTEXT indexes
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(36) PRIMARY KEY,
		username VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		display_name VARCHAR(255),
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		avatar_url TEXT,
		last_login_at DATETIME(6),
		deleted_at DATETIME(6),
		created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
		UNIQUE KEY idx_users_username (username),
		UNIQUE KEY idx_users_email (email),
		KEY idx_users_status (status),
		KEY idx_users_role (role),
		KEY idx_users_deleted_at (deleted_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Mailboxes table
	CREATE TABLE IF NOT EXISTS mailboxes (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL,
		name VARCHAR(255) NOT NULL,
		address VARCHAR(255) NOT NULL,
		description TEXT,
		is_catch_all TINYINT(1) NOT NULL DEFAULT 0,
		is_default TINYINT(1) NOT NULL DEFAULT 0,
		message_count BIGINT NOT NULL DEFAULT 0,
		unread_count BIGINT NOT NULL DEFAULT 0,
		total_size BIGINT NOT NULL DEFAULT 0,
		retention_days INT NOT NULL DEFAULT 0,
		created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
		UNIQUE KEY idx_mailboxes_address (address),
		KEY idx_mailboxes_user_id (user_id),
		KEY idx_mailboxes_is_catch_all (is_catch_all),
		KEY idx_mailboxes_is_default (is_default),
		CONSTRAINT fk_mailboxes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Messages table with FULLTEXT index for search
	CREATE TABLE IF NOT EXISTS messages (
		id VARCHAR(36) PRIMARY KEY,
		mailbox_id VARCHAR(36) NOT NULL,
		message_id VARCHAR(255),
		from_name VARCHAR(255),
		from_address VARCHAR(255) NOT NULL,
		subject TEXT,
		text_body MEDIUMTEXT,
		html_body MEDIUMTEXT,
		raw_body MEDIUMBLOB,
		headers JSON,
		content_type VARCHAR(100) NOT NULL DEFAULT 'text/plain',
		size BIGINT NOT NULL DEFAULT 0,
		attachment_count INT NOT NULL DEFAULT 0,
		status VARCHAR(50) NOT NULL DEFAULT 'unread',
		is_starred TINYINT(1) NOT NULL DEFAULT 0,
		is_spam TINYINT(1) NOT NULL DEFAULT 0,
		in_reply_to VARCHAR(255),
		references_list JSON,
		received_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		sent_at DATETIME(6),
		created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
		KEY idx_messages_mailbox_id (mailbox_id),
		KEY idx_messages_message_id (message_id),
		KEY idx_messages_from_address (from_address),
		KEY idx_messages_status (status),
		KEY idx_messages_is_starred (is_starred),
		KEY idx_messages_is_spam (is_spam),
		KEY idx_messages_received_at (received_at),
		FULLTEXT KEY ft_messages_search (subject, from_address, text_body),
		CONSTRAINT fk_messages_mailbox FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Message recipients table (for To, Cc, Bcc)
	CREATE TABLE IF NOT EXISTS message_recipients (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		message_id VARCHAR(36) NOT NULL,
		recipient_type VARCHAR(20) NOT NULL,
		name VARCHAR(255),
		address VARCHAR(255) NOT NULL,
		KEY idx_message_recipients_message_id (message_id),
		KEY idx_message_recipients_address (address),
		CONSTRAINT fk_recipients_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Attachments table
	CREATE TABLE IF NOT EXISTS attachments (
		id VARCHAR(36) PRIMARY KEY,
		message_id VARCHAR(36) NOT NULL,
		filename VARCHAR(255) NOT NULL,
		content_type VARCHAR(100) NOT NULL,
		size BIGINT NOT NULL DEFAULT 0,
		content_id VARCHAR(255),
		disposition VARCHAR(50) NOT NULL DEFAULT 'attachment',
		storage_path TEXT,
		checksum VARCHAR(64),
		is_inline TINYINT(1) NOT NULL DEFAULT 0,
		created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		KEY idx_attachments_message_id (message_id),
		KEY idx_attachments_content_id (content_id),
		KEY idx_attachments_content_type (content_type),
		KEY idx_attachments_checksum (checksum),
		CONSTRAINT fk_attachments_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Attachment content table (for storing binary data)
	CREATE TABLE IF NOT EXISTS attachment_content (
		attachment_id VARCHAR(36) PRIMARY KEY,
		content MEDIUMBLOB,
		CONSTRAINT fk_content_attachment FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Webhooks table
	CREATE TABLE IF NOT EXISTS webhooks (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL,
		name VARCHAR(255) NOT NULL,
		url TEXT NOT NULL,
		secret VARCHAR(255),
		events JSON NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		headers JSON,
		retry_count INT NOT NULL DEFAULT 0,
		max_retries INT NOT NULL DEFAULT 3,
		timeout_seconds INT NOT NULL DEFAULT 30,
		last_triggered_at DATETIME(6),
		last_success_at DATETIME(6),
		last_failure_at DATETIME(6),
		last_error TEXT,
		success_count BIGINT NOT NULL DEFAULT 0,
		failure_count BIGINT NOT NULL DEFAULT 0,
		created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
		KEY idx_webhooks_user_id (user_id),
		KEY idx_webhooks_status (status),
		CONSTRAINT fk_webhooks_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Webhook deliveries table
	CREATE TABLE IF NOT EXISTS webhook_deliveries (
		id VARCHAR(36) PRIMARY KEY,
		webhook_id VARCHAR(36) NOT NULL,
		event VARCHAR(100) NOT NULL,
		payload MEDIUMTEXT NOT NULL,
		status_code INT,
		response MEDIUMTEXT,
		error TEXT,
		success TINYINT(1) NOT NULL DEFAULT 0,
		duration BIGINT NOT NULL DEFAULT 0,
		attempt_number INT NOT NULL DEFAULT 1,
		created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		KEY idx_webhook_deliveries_webhook_id (webhook_id),
		KEY idx_webhook_deliveries_event (event),
		KEY idx_webhook_deliveries_success (success),
		KEY idx_webhook_deliveries_created_at (created_at),
		CONSTRAINT fk_deliveries_webhook FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Settings table
	CREATE TABLE IF NOT EXISTS settings (
		id VARCHAR(36) PRIMARY KEY,
		data JSON NOT NULL,
		updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

	-- Settings change history table
	CREATE TABLE IF NOT EXISTS settings_history (
		id VARCHAR(36) PRIMARY KEY,
		field_path VARCHAR(255) NOT NULL,
		old_value MEDIUMTEXT,
		new_value MEDIUMTEXT,
		changed_by VARCHAR(36),
		reason TEXT,
		changed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		KEY idx_settings_history_field_path (field_path),
		KEY idx_settings_history_changed_at (changed_at),
		CONSTRAINT fk_settings_changed_by FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE SET NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	// Execute schema creation
	// MySQL requires executing each statement separately when using multiStatements=true
	statements := splitStatements(schema)
	for _, stmt := range statements {
		if stmt = trimStatement(stmt); stmt != "" {
			if _, err := r.pool.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to execute schema statement: %w\nStatement: %s", err, stmt)
			}
		}
	}

	return nil
}

// splitStatements splits a SQL string containing multiple statements.
func splitStatements(sql string) []string {
	var statements []string
	var current []rune
	inString := false
	stringChar := rune(0)
	inComment := false
	prevChar := rune(0)

	for i, char := range sql {
		// Handle string literals
		if !inComment && (char == '\'' || char == '"') {
			if !inString {
				inString = true
				stringChar = char
			} else if char == stringChar && prevChar != '\\' {
				inString = false
			}
		}

		// Handle -- comments
		if !inString && !inComment && char == '-' && prevChar == '-' {
			inComment = true
		}

		// Handle end of line comments
		if inComment && char == '\n' {
			inComment = false
		}

		// Handle statement separator
		if !inString && !inComment && char == ';' {
			current = append(current, char)
			statements = append(statements, string(current))
			current = nil
		} else {
			current = append(current, char)
		}

		prevChar = char
		_ = i // silence unused variable warning
	}

	// Add any remaining content
	if len(current) > 0 {
		statements = append(statements, string(current))
	}

	return statements
}

// trimStatement removes leading/trailing whitespace and comments from a SQL statement.
func trimStatement(stmt string) string {
	// Simple trim - in production, would need more sophisticated comment removal
	result := ""
	lines := []rune(stmt)
	start := 0
	for i := 0; i < len(lines); i++ {
		if lines[i] != ' ' && lines[i] != '\t' && lines[i] != '\n' && lines[i] != '\r' {
			start = i
			break
		}
	}
	end := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != ' ' && lines[i] != '\t' && lines[i] != '\n' && lines[i] != '\r' && lines[i] != ';' {
			end = i + 1
			break
		}
	}
	if start < end {
		result = string(lines[start:end])
	}

	// Skip pure comment lines
	if len(result) >= 2 && result[:2] == "--" {
		return ""
	}

	return result
}

// Migrate runs database migrations.
func (r *Repository) Migrate(ctx context.Context) error {
	return r.createSchema(ctx)
}

// DatabaseInfo returns information about the MySQL database.
func (r *Repository) DatabaseInfo(ctx context.Context) (*repository.DatabaseInfo, error) {
	version, err := r.pool.Version(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get MySQL version: %w", err)
	}

	stats := r.pool.Stats()

	info := &repository.DatabaseInfo{
		Driver:          domain.DatabaseDriverMySQL,
		Version:         version,
		Database:        "MySQL",
		ConnectionCount: stats.OpenConnections,
		MaxConnections:  stats.MaxOpenConnections,
		PoolStats: &repository.ConnectionPoolStats{
			OpenConnections:   stats.OpenConnections,
			InUse:             stats.InUse,
			Idle:              stats.Idle,
			WaitCount:         stats.WaitCount,
			WaitDuration:      stats.WaitDuration.Nanoseconds(),
			MaxIdleClosed:     stats.MaxIdleClosed,
			MaxLifetimeClosed: stats.MaxLifetimeClosed,
		},
	}

	// Get database size
	size, err := r.pool.DatabaseSize(ctx)
	if err == nil {
		info.Size = size
	}

	// Get table stats
	tableStats, err := r.getTableStats(ctx)
	if err == nil {
		info.TableStats = tableStats
	}

	return info, nil
}

// getTableStats retrieves statistics for all tables.
func (r *Repository) getTableStats(ctx context.Context) ([]repository.TableStats, error) {
	query := `SELECT 
		table_name as name, 
		table_rows as row_count,
		data_length as size,
		index_length as index_size
		FROM information_schema.TABLES 
		WHERE table_schema = DATABASE()
		ORDER BY table_name`

	var stats []repository.TableStats
	if err := r.pool.SelectContext(ctx, &stats, query); err != nil {
		return nil, err
	}

	return stats, nil
}

// Ensure Repository implements repository.Repository
var _ repository.Repository = (*Repository)(nil)
