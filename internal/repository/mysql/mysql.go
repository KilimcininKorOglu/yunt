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
	jmap        *JMAPRepo

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
	repo.jmap = NewJMAPRepo(repo)

	// Run migrations using the Migrator
	migrator, err := NewMigrator(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := migrator.Migrate(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
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

// JMAP returns the JMAP-specific repository sub-aggregate.
func (r *Repository) JMAP() repository.JMAPRepository {
	return r.jmap
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

// DB returns the underlying sqlx.DB connection for direct access (e.g., session store).
func (r *Repository) DB() *sqlx.DB {
	return r.pool.DB()
}

// db returns the database connection or transaction.
func (r *Repository) db() sqlxDB {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.isTx && r.tx != nil {
		return &metricsDB{inner: r.tx}
	}
	return &metricsDB{inner: r.pool.DB()}
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

// Migrate runs database migrations.
func (r *Repository) Migrate(ctx context.Context) error {
	migrator, err := NewMigrator(r.pool)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	return migrator.Migrate(ctx)
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
