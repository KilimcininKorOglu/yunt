package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// Repository is the main SQLite repository implementation.
// It aggregates all entity repositories and provides transaction support.
type Repository struct {
	pool     *ConnectionPool
	migrator *Migrator
	seeder   *Seeder

	users       *UserRepository
	mailboxes   *MailboxRepository
	messages    *MessageRepository
	attachments *AttachmentRepository
	webhooks    *WebhookRepository
	settings    *SettingsRepository
	stats       *StatsRepository

	// For transaction support
	tx   *sqlx.Tx
	isTx bool
	mu   sync.RWMutex
}

// New creates a new SQLite repository with the given connection pool.
// It automatically runs pending migrations and seeds initial data if needed.
func New(pool *ConnectionPool) (*Repository, error) {
	return NewWithOptions(pool, true, true)
}

// NewWithOptions creates a new SQLite repository with custom options.
// autoMigrate determines if migrations should run automatically.
// autoSeed determines if initial data should be seeded automatically.
func NewWithOptions(pool *ConnectionPool, autoMigrate, autoSeed bool) (*Repository, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool is required")
	}

	// Create migrator
	migrator, err := NewMigrator(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	repo := &Repository{
		pool:     pool,
		migrator: migrator,
	}

	// Initialize entity repositories
	repo.users = NewUserRepository(repo)
	repo.mailboxes = NewMailboxRepository(repo)
	repo.messages = NewMessageRepository(repo)
	repo.attachments = NewAttachmentRepository(repo)
	repo.webhooks = NewWebhookRepository(repo)
	repo.settings = NewSettingsRepository(repo)
	repo.stats = NewStatsRepository(repo)

	// Create seeder
	repo.seeder = NewSeeder(repo)

	// Run migrations if auto-migrate is enabled
	if autoMigrate {
		if err := repo.Migrate(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	// Seed initial data if auto-seed is enabled
	if autoSeed {
		if err := repo.Seed(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to seed database: %w", err)
		}
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

// Stats returns the stats repository.
func (r *Repository) Stats() *StatsRepository {
	return r.stats
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
	txRepo.stats = NewStatsRepository(txRepo)

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
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL COLLATE NOCASE,
		email TEXT UNIQUE NOT NULL COLLATE NOCASE,
		password_hash TEXT NOT NULL,
		display_name TEXT,
		role TEXT NOT NULL DEFAULT 'user',
		status TEXT NOT NULL DEFAULT 'pending',
		avatar_url TEXT,
		last_login_at DATETIME,
		deleted_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Index for user lookups
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
	CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

	-- Mailboxes table
	CREATE TABLE IF NOT EXISTS mailboxes (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		address TEXT UNIQUE NOT NULL COLLATE NOCASE,
		description TEXT,
		is_catch_all INTEGER NOT NULL DEFAULT 0,
		is_default INTEGER NOT NULL DEFAULT 0,
		message_count INTEGER NOT NULL DEFAULT 0,
		unread_count INTEGER NOT NULL DEFAULT 0,
		total_size INTEGER NOT NULL DEFAULT 0,
		retention_days INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Indexes for mailbox lookups
	CREATE INDEX IF NOT EXISTS idx_mailboxes_user_id ON mailboxes(user_id);
	CREATE INDEX IF NOT EXISTS idx_mailboxes_address ON mailboxes(address);
	CREATE INDEX IF NOT EXISTS idx_mailboxes_is_catch_all ON mailboxes(is_catch_all);
	CREATE INDEX IF NOT EXISTS idx_mailboxes_is_default ON mailboxes(is_default);

	-- Messages table
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		mailbox_id TEXT NOT NULL,
		message_id TEXT,
		from_name TEXT,
		from_address TEXT NOT NULL,
		subject TEXT,
		text_body TEXT,
		html_body TEXT,
		raw_body BLOB,
		headers TEXT,
		content_type TEXT NOT NULL DEFAULT 'text/plain',
		size INTEGER NOT NULL DEFAULT 0,
		attachment_count INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'unread',
		is_starred INTEGER NOT NULL DEFAULT 0,
		is_spam INTEGER NOT NULL DEFAULT 0,
		in_reply_to TEXT,
		references_list TEXT,
		received_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		sent_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
	);

	-- Indexes for message lookups
	CREATE INDEX IF NOT EXISTS idx_messages_mailbox_id ON messages(mailbox_id);
	CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);
	CREATE INDEX IF NOT EXISTS idx_messages_from_address ON messages(from_address);
	CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
	CREATE INDEX IF NOT EXISTS idx_messages_is_starred ON messages(is_starred);
	CREATE INDEX IF NOT EXISTS idx_messages_is_spam ON messages(is_spam);
	CREATE INDEX IF NOT EXISTS idx_messages_received_at ON messages(received_at);
	CREATE INDEX IF NOT EXISTS idx_messages_subject ON messages(subject);

	-- Message recipients table (for To, Cc, Bcc)
	CREATE TABLE IF NOT EXISTS message_recipients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id TEXT NOT NULL,
		recipient_type TEXT NOT NULL,
		name TEXT,
		address TEXT NOT NULL,
		FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
	);

	-- Index for recipient lookups
	CREATE INDEX IF NOT EXISTS idx_message_recipients_message_id ON message_recipients(message_id);
	CREATE INDEX IF NOT EXISTS idx_message_recipients_address ON message_recipients(address);

	-- Attachments table
	CREATE TABLE IF NOT EXISTS attachments (
		id TEXT PRIMARY KEY,
		message_id TEXT NOT NULL,
		filename TEXT NOT NULL,
		content_type TEXT NOT NULL,
		size INTEGER NOT NULL DEFAULT 0,
		content_id TEXT,
		disposition TEXT NOT NULL DEFAULT 'attachment',
		storage_path TEXT,
		checksum TEXT,
		is_inline INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
	);

	-- Indexes for attachment lookups
	CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);
	CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
	CREATE INDEX IF NOT EXISTS idx_attachments_content_type ON attachments(content_type);
	CREATE INDEX IF NOT EXISTS idx_attachments_checksum ON attachments(checksum);

	-- Attachment content table (for storing binary data)
	CREATE TABLE IF NOT EXISTS attachment_content (
		attachment_id TEXT PRIMARY KEY,
		content BLOB,
		FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE
	);

	-- Webhooks table
	CREATE TABLE IF NOT EXISTS webhooks (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		secret TEXT,
		events TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		headers TEXT,
		retry_count INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		timeout_seconds INTEGER NOT NULL DEFAULT 30,
		last_triggered_at DATETIME,
		last_success_at DATETIME,
		last_failure_at DATETIME,
		last_error TEXT,
		success_count INTEGER NOT NULL DEFAULT 0,
		failure_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Indexes for webhook lookups
	CREATE INDEX IF NOT EXISTS idx_webhooks_user_id ON webhooks(user_id);
	CREATE INDEX IF NOT EXISTS idx_webhooks_status ON webhooks(status);
	CREATE INDEX IF NOT EXISTS idx_webhooks_url ON webhooks(url);

	-- Webhook deliveries table
	CREATE TABLE IF NOT EXISTS webhook_deliveries (
		id TEXT PRIMARY KEY,
		webhook_id TEXT NOT NULL,
		event TEXT NOT NULL,
		payload TEXT NOT NULL,
		status_code INTEGER,
		response TEXT,
		error TEXT,
		success INTEGER NOT NULL DEFAULT 0,
		duration INTEGER NOT NULL DEFAULT 0,
		attempt_number INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
	);

	-- Index for delivery lookups
	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event ON webhook_deliveries(event);
	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_success ON webhook_deliveries(success);
	CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created_at ON webhook_deliveries(created_at);

	-- Settings table
	CREATE TABLE IF NOT EXISTS settings (
		id TEXT PRIMARY KEY,
		data TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Settings change history table
	CREATE TABLE IF NOT EXISTS settings_history (
		id TEXT PRIMARY KEY,
		field_path TEXT NOT NULL,
		old_value TEXT,
		new_value TEXT,
		changed_by TEXT,
		reason TEXT,
		changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE SET NULL
	);

	-- Index for settings history
	CREATE INDEX IF NOT EXISTS idx_settings_history_field_path ON settings_history(field_path);
	CREATE INDEX IF NOT EXISTS idx_settings_history_changed_at ON settings_history(changed_at);
	`

	_, err := r.pool.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Migrate runs all pending database migrations.
func (r *Repository) Migrate(ctx context.Context) error {
	if r.migrator == nil {
		// Fallback to legacy schema creation for backward compatibility
		return r.createSchema(ctx)
	}
	return r.migrator.Migrate(ctx)
}

// MigrateUp runs a specific number of pending migrations.
func (r *Repository) MigrateUp(ctx context.Context, steps int) error {
	if r.migrator == nil {
		return fmt.Errorf("migrator not initialized")
	}
	return r.migrator.MigrateUp(ctx, steps)
}

// MigrateDown rolls back a specific number of migrations.
func (r *Repository) MigrateDown(ctx context.Context, steps int) error {
	if r.migrator == nil {
		return fmt.Errorf("migrator not initialized")
	}
	return r.migrator.MigrateDown(ctx, steps)
}

// MigrationVersion returns the current migration version.
func (r *Repository) MigrationVersion(ctx context.Context) (int64, error) {
	if r.migrator == nil {
		return 0, fmt.Errorf("migrator not initialized")
	}
	return r.migrator.MigrationVersion(ctx)
}

// MigrationStatus returns the status of all migrations.
func (r *Repository) MigrationStatus(ctx context.Context) ([]repository.MigrationInfo, error) {
	if r.migrator == nil {
		return nil, fmt.Errorf("migrator not initialized")
	}
	return r.migrator.MigrationStatus(ctx)
}

// Migrator returns the underlying migrator instance.
func (r *Repository) Migrator() *Migrator {
	return r.migrator
}

// Seed populates the database with initial data.
func (r *Repository) Seed(ctx context.Context) error {
	if r.seeder == nil {
		return fmt.Errorf("seeder not initialized")
	}
	return r.seeder.Seed(ctx)
}

// SeedWithConfig populates the database with initial data using custom configuration.
func (r *Repository) SeedWithConfig(ctx context.Context, config *SeedConfig) error {
	if r.seeder == nil {
		return fmt.Errorf("seeder not initialized")
	}
	return r.seeder.SeedWithConfig(ctx, config)
}

// Seeder returns the underlying seeder instance.
func (r *Repository) Seeder() *Seeder {
	return r.seeder
}

// IsSeeded checks if the database has been seeded with initial data.
func (r *Repository) IsSeeded(ctx context.Context) (bool, error) {
	if r.seeder == nil {
		return false, fmt.Errorf("seeder not initialized")
	}
	return r.seeder.IsSeeded(ctx)
}

// DatabaseInfo returns information about the SQLite database.
func (r *Repository) DatabaseInfo(ctx context.Context) (*repository.DatabaseInfo, error) {
	version, err := r.pool.Version(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get SQLite version: %w", err)
	}

	stats := r.pool.Stats()

	info := &repository.DatabaseInfo{
		Driver:          domain.DatabaseDriverSQLite,
		Version:         version,
		Database:        "SQLite",
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
