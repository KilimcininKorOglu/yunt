package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// Repository is the main PostgreSQL repository implementation.
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

// New creates a new PostgreSQL repository with the given connection pool.
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
	schema := `
	-- Enable pg_trgm extension for full-text search
	CREATE EXTENSION IF NOT EXISTS pg_trgm;

	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(36) PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		display_name VARCHAR(255),
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		avatar_url TEXT,
		last_login_at TIMESTAMP WITH TIME ZONE,
		deleted_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Index for user lookups
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(LOWER(username));
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(LOWER(email));
	CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
	CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

	-- Mailboxes table
	CREATE TABLE IF NOT EXISTS mailboxes (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		address VARCHAR(255) UNIQUE NOT NULL,
		description TEXT,
		is_catch_all BOOLEAN NOT NULL DEFAULT FALSE,
		is_default BOOLEAN NOT NULL DEFAULT FALSE,
		message_count BIGINT NOT NULL DEFAULT 0,
		unread_count BIGINT NOT NULL DEFAULT 0,
		total_size BIGINT NOT NULL DEFAULT 0,
		retention_days INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for mailbox lookups
	CREATE INDEX IF NOT EXISTS idx_mailboxes_user_id ON mailboxes(user_id);
	CREATE INDEX IF NOT EXISTS idx_mailboxes_address ON mailboxes(LOWER(address));
	CREATE INDEX IF NOT EXISTS idx_mailboxes_is_catch_all ON mailboxes(is_catch_all);
	CREATE INDEX IF NOT EXISTS idx_mailboxes_is_default ON mailboxes(is_default);

	-- Messages table
	CREATE TABLE IF NOT EXISTS messages (
		id VARCHAR(36) PRIMARY KEY,
		mailbox_id VARCHAR(36) NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
		message_id VARCHAR(255),
		from_name VARCHAR(255),
		from_address VARCHAR(255) NOT NULL,
		subject TEXT,
		text_body TEXT,
		html_body TEXT,
		raw_body BYTEA,
		headers JSONB,
		content_type VARCHAR(100) NOT NULL DEFAULT 'text/plain',
		size BIGINT NOT NULL DEFAULT 0,
		attachment_count INTEGER NOT NULL DEFAULT 0,
		status VARCHAR(50) NOT NULL DEFAULT 'unread',
		is_starred BOOLEAN NOT NULL DEFAULT FALSE,
		is_spam BOOLEAN NOT NULL DEFAULT FALSE,
		in_reply_to VARCHAR(255),
		references_list JSONB,
		received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		sent_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		search_vector TSVECTOR
	);

	-- Indexes for message lookups
	CREATE INDEX IF NOT EXISTS idx_messages_mailbox_id ON messages(mailbox_id);
	CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);
	CREATE INDEX IF NOT EXISTS idx_messages_from_address ON messages(LOWER(from_address));
	CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
	CREATE INDEX IF NOT EXISTS idx_messages_is_starred ON messages(is_starred);
	CREATE INDEX IF NOT EXISTS idx_messages_is_spam ON messages(is_spam);
	CREATE INDEX IF NOT EXISTS idx_messages_received_at ON messages(received_at);
	CREATE INDEX IF NOT EXISTS idx_messages_subject ON messages USING gin(subject gin_trgm_ops);
	CREATE INDEX IF NOT EXISTS idx_messages_search_vector ON messages USING gin(search_vector);

	-- Message recipients table (for To, Cc, Bcc)
	CREATE TABLE IF NOT EXISTS message_recipients (
		id SERIAL PRIMARY KEY,
		message_id VARCHAR(36) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
		recipient_type VARCHAR(20) NOT NULL,
		name VARCHAR(255),
		address VARCHAR(255) NOT NULL
	);

	-- Index for recipient lookups
	CREATE INDEX IF NOT EXISTS idx_message_recipients_message_id ON message_recipients(message_id);
	CREATE INDEX IF NOT EXISTS idx_message_recipients_address ON message_recipients(LOWER(address));

	-- Attachments table
	CREATE TABLE IF NOT EXISTS attachments (
		id VARCHAR(36) PRIMARY KEY,
		message_id VARCHAR(36) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
		filename VARCHAR(255) NOT NULL,
		content_type VARCHAR(100) NOT NULL,
		size BIGINT NOT NULL DEFAULT 0,
		content_id VARCHAR(255),
		disposition VARCHAR(50) NOT NULL DEFAULT 'attachment',
		storage_path TEXT,
		checksum VARCHAR(64),
		is_inline BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for attachment lookups
	CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);
	CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
	CREATE INDEX IF NOT EXISTS idx_attachments_content_type ON attachments(content_type);
	CREATE INDEX IF NOT EXISTS idx_attachments_checksum ON attachments(checksum);

	-- Attachment content table (for storing binary data)
	CREATE TABLE IF NOT EXISTS attachment_content (
		attachment_id VARCHAR(36) PRIMARY KEY REFERENCES attachments(id) ON DELETE CASCADE,
		content BYTEA
	);

	-- Webhooks table
	CREATE TABLE IF NOT EXISTS webhooks (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		url TEXT NOT NULL,
		secret TEXT,
		events JSONB NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		headers JSONB,
		retry_count INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		timeout_seconds INTEGER NOT NULL DEFAULT 30,
		last_triggered_at TIMESTAMP WITH TIME ZONE,
		last_success_at TIMESTAMP WITH TIME ZONE,
		last_failure_at TIMESTAMP WITH TIME ZONE,
		last_error TEXT,
		success_count BIGINT NOT NULL DEFAULT 0,
		failure_count BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for webhook lookups
	CREATE INDEX IF NOT EXISTS idx_webhooks_user_id ON webhooks(user_id);
	CREATE INDEX IF NOT EXISTS idx_webhooks_status ON webhooks(status);
	CREATE INDEX IF NOT EXISTS idx_webhooks_url ON webhooks(url);

	-- Webhook deliveries table
	CREATE TABLE IF NOT EXISTS webhook_deliveries (
		id VARCHAR(36) PRIMARY KEY,
		webhook_id VARCHAR(36) NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
		event VARCHAR(100) NOT NULL,
		payload TEXT NOT NULL,
		status_code INTEGER,
		response TEXT,
		error TEXT,
		success BOOLEAN NOT NULL DEFAULT FALSE,
		duration BIGINT NOT NULL DEFAULT 0,
		attempt_number INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Index for delivery lookups
	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event ON webhook_deliveries(event);
	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_success ON webhook_deliveries(success);
	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created_at ON webhook_deliveries(created_at);

	-- Settings table
	CREATE TABLE IF NOT EXISTS settings (
		id VARCHAR(36) PRIMARY KEY,
		data JSONB NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Settings change history table
	CREATE TABLE IF NOT EXISTS settings_history (
		id VARCHAR(36) PRIMARY KEY,
		field_path VARCHAR(255) NOT NULL,
		old_value TEXT,
		new_value TEXT,
		changed_by VARCHAR(36) REFERENCES users(id) ON DELETE SET NULL,
		reason TEXT,
		changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Index for settings history
	CREATE INDEX IF NOT EXISTS idx_settings_history_field_path ON settings_history(field_path);
	CREATE INDEX IF NOT EXISTS idx_settings_history_changed_at ON settings_history(changed_at);

	-- Create function to update search_vector
	CREATE OR REPLACE FUNCTION messages_search_vector_update() RETURNS trigger AS $$
	BEGIN
		NEW.search_vector :=
			setweight(to_tsvector('english', COALESCE(NEW.subject, '')), 'A') ||
			setweight(to_tsvector('english', COALESCE(NEW.from_address, '')), 'B') ||
			setweight(to_tsvector('english', COALESCE(NEW.from_name, '')), 'B') ||
			setweight(to_tsvector('english', COALESCE(NEW.text_body, '')), 'C');
		RETURN NEW;
	END
	$$ LANGUAGE plpgsql;

	-- Create trigger for search_vector update
	DROP TRIGGER IF EXISTS messages_search_vector_trigger ON messages;
	CREATE TRIGGER messages_search_vector_trigger
		BEFORE INSERT OR UPDATE ON messages
		FOR EACH ROW EXECUTE FUNCTION messages_search_vector_update();
	`

	_, err := r.pool.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Migrate runs database migrations.
func (r *Repository) Migrate(ctx context.Context) error {
	return r.createSchema(ctx)
}

// DatabaseInfo returns information about the PostgreSQL database.
func (r *Repository) DatabaseInfo(ctx context.Context) (*repository.DatabaseInfo, error) {
	version, err := r.pool.ServerVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get PostgreSQL version: %w", err)
	}

	stats := r.pool.Stats()

	info := &repository.DatabaseInfo{
		Driver:          domain.DatabaseDriverPostgres,
		Version:         version,
		Database:        "PostgreSQL",
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

	// Get table stats
	tableStats, err := r.getTableStats(ctx)
	if err == nil {
		info.TableStats = tableStats
	}

	return info, nil
}

// getTableStats retrieves statistics for all tables.
func (r *Repository) getTableStats(ctx context.Context) ([]repository.TableStats, error) {
	tables := []string{"users", "mailboxes", "messages", "attachments", "webhooks", "webhook_deliveries", "settings"}
	stats := make([]repository.TableStats, 0, len(tables))

	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := r.pool.GetContext(ctx, &count, query); err != nil {
			continue
		}

		stats = append(stats, repository.TableStats{
			Name:     table,
			RowCount: count,
		})
	}

	return stats, nil
}
