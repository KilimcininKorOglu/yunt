package mongodb

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// Repository is the main MongoDB repository implementation.
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
	session mongo.Session
	isTx    bool
	mu      sync.RWMutex
}

// New creates a new MongoDB repository with the given connection pool.
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

	// Create indexes
	if err := pool.EnsureIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
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

	// Configure session options
	sessionOpts := options.Session().
		SetDefaultReadConcern(readconcern.Snapshot()).
		SetDefaultWriteConcern(writeconcern.Majority())

	session, err := r.pool.StartSession(sessionOpts)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Configure transaction options
	txOpts := options.Transaction().
		SetWriteConcern(writeconcern.Majority())

	if opts.ReadOnly {
		txOpts.SetReadConcern(readconcern.Snapshot())
	}

	// Execute the transaction
	_, err = session.WithTransaction(ctx, func(_ mongo.SessionContext) (interface{}, error) {
		txRepo := r.withSession(session)
		if txErr := fn(txRepo); txErr != nil {
			return nil, txErr
		}
		return nil, nil
	}, txOpts)

	if err != nil {
		return err
	}

	return nil
}

// withSession creates a new repository instance that uses the given session.
func (r *Repository) withSession(session mongo.Session) *Repository {
	txRepo := &Repository{
		pool:    r.pool,
		session: session,
		isTx:    true,
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

// collection returns the collection by name, wrapped with metrics instrumentation.
func (r *Repository) collection(name string) mongoCollection {
	return &metricsCollection{inner: r.pool.Collection(name)}
}

// getSessionContext returns a session context if in a transaction, otherwise returns the original context.
func (r *Repository) getSessionContext(ctx context.Context) context.Context {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.isTx && r.session != nil {
		return mongo.NewSessionContext(ctx, r.session)
	}
	return ctx
}

// Migrate runs database migrations (creates indexes).
func (r *Repository) Migrate(ctx context.Context) error {
	return r.pool.EnsureIndexes(ctx)
}

// DatabaseInfo returns information about the MongoDB database.
func (r *Repository) DatabaseInfo(ctx context.Context) (*repository.DatabaseInfo, error) {
	version, err := r.pool.Version(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get MongoDB version: %w", err)
	}

	info := &repository.DatabaseInfo{
		Driver:   domain.DatabaseDriverMongoDB,
		Version:  version,
		Database: r.pool.config.Database,
	}

	// Get collection stats
	tableStats, err := r.getTableStats(ctx)
	if err == nil {
		info.TableStats = tableStats
	}

	return info, nil
}

// getTableStats retrieves statistics for all collections.
func (r *Repository) getTableStats(ctx context.Context) ([]repository.TableStats, error) {
	collections := []string{
		CollectionUsers,
		CollectionMailboxes,
		CollectionMessages,
		CollectionAttachments,
		CollectionWebhooks,
		CollectionWebhookDeliveries,
		CollectionSettings,
	}
	stats := make([]repository.TableStats, 0, len(collections))

	for _, collName := range collections {
		coll := r.collection(collName)
		if coll == nil {
			continue
		}

		count, err := coll.CountDocuments(ctx, map[string]interface{}{})
		if err != nil {
			continue
		}

		collStats, err := r.pool.CollectionStats(ctx, collName)
		var size int64
		if err == nil {
			if s, ok := collStats["size"].(int64); ok {
				size = s
			} else if s, ok := collStats["size"].(int32); ok {
				size = int64(s)
			}
		}

		stats = append(stats, repository.TableStats{
			Name:     collName,
			RowCount: count,
			Size:     size,
		})
	}

	return stats, nil
}

// Ensure Repository implements repository.Repository
var _ repository.Repository = (*Repository)(nil)
