package repository

import (
	"context"
	"io"
)

// Repository is the aggregate interface that combines all repository interfaces.
// It provides a single entry point for accessing all data access operations
// and supports transactional operations across multiple repositories.
type Repository interface {
	// Users returns the user repository.
	Users() UserRepository

	// Mailboxes returns the mailbox repository.
	Mailboxes() MailboxRepository

	// Messages returns the message repository.
	Messages() MessageRepository

	// Attachments returns the attachment repository.
	Attachments() AttachmentRepository

	// Webhooks returns the webhook repository.
	Webhooks() WebhookRepository

	// Settings returns the settings repository.
	Settings() SettingsRepository

	// Transaction executes the given function within a database transaction.
	// If the function returns an error, the transaction is rolled back.
	// If the function returns nil, the transaction is committed.
	Transaction(ctx context.Context, fn func(tx Repository) error) error

	// TransactionWithOptions executes the function within a transaction with custom options.
	TransactionWithOptions(ctx context.Context, opts TransactionOptions, fn func(tx Repository) error) error

	// Health checks the health of the database connection.
	Health(ctx context.Context) error

	// Close closes the database connection and releases resources.
	Close() error
}

// Transactor provides transaction management capabilities.
// Repositories that support transactions should implement this interface.
type Transactor interface {
	// BeginTx starts a new transaction with the given options.
	BeginTx(ctx context.Context, opts TransactionOptions) (Transaction, error)
}

// Transaction represents an active database transaction.
type Transaction interface {
	// Commit commits the transaction.
	Commit() error

	// Rollback aborts the transaction.
	Rollback() error

	// Repository returns a repository instance that operates within this transaction.
	Repository() Repository
}

// BaseRepository defines common operations shared by all repositories.
type BaseRepository[T any, ID comparable] interface {
	// GetByID retrieves an entity by its unique identifier.
	// Returns domain.ErrNotFound if the entity does not exist.
	GetByID(ctx context.Context, id ID) (*T, error)

	// Exists checks if an entity with the given ID exists.
	Exists(ctx context.Context, id ID) (bool, error)

	// Create creates a new entity.
	// Returns domain.ErrAlreadyExists if an entity with the same ID already exists.
	Create(ctx context.Context, entity *T) error

	// Update updates an existing entity.
	// Returns domain.ErrNotFound if the entity does not exist.
	Update(ctx context.Context, entity *T) error

	// Delete removes an entity by its ID.
	// Returns domain.ErrNotFound if the entity does not exist.
	Delete(ctx context.Context, id ID) error

	// Count returns the total number of entities.
	Count(ctx context.Context) (int64, error)
}

// Migrator provides database migration capabilities.
type Migrator interface {
	// Migrate runs pending database migrations.
	Migrate(ctx context.Context) error

	// MigrateUp runs a specific number of up migrations.
	MigrateUp(ctx context.Context, steps int) error

	// MigrateDown rolls back a specific number of migrations.
	MigrateDown(ctx context.Context, steps int) error

	// MigrationVersion returns the current migration version.
	MigrationVersion(ctx context.Context) (int64, error)

	// MigrationStatus returns the status of all migrations.
	MigrationStatus(ctx context.Context) ([]MigrationInfo, error)
}

// MigrationInfo contains information about a migration.
type MigrationInfo struct {
	// Version is the migration version number.
	Version int64

	// Name is the migration name or description.
	Name string

	// Applied indicates if the migration has been applied.
	Applied bool

	// AppliedAt is when the migration was applied (if applied).
	AppliedAt *string
}

// Seeder provides database seeding capabilities.
type Seeder interface {
	// Seed populates the database with initial/test data.
	Seed(ctx context.Context) error

	// SeedFromFile seeds data from a file.
	SeedFromFile(ctx context.Context, path string) error

	// SeedFromReader seeds data from a reader.
	SeedFromReader(ctx context.Context, r io.Reader) error
}

// QueryBuilder provides a fluent interface for building queries.
// This is optional for implementations that want to expose query building.
type QueryBuilder interface {
	// Where adds a condition to the query.
	Where(field string, operator string, value interface{}) QueryBuilder

	// OrderBy adds a sort clause to the query.
	OrderBy(field string, order string) QueryBuilder

	// Limit sets the maximum number of results.
	Limit(limit int) QueryBuilder

	// Offset sets the starting offset for results.
	Offset(offset int) QueryBuilder

	// Build returns the final query and arguments.
	Build() (query string, args []interface{})
}

// ChangeSet represents a set of changes for optimistic locking scenarios.
type ChangeSet struct {
	// Fields contains the fields that were modified and their new values.
	Fields map[string]interface{}

	// Version is the expected version for optimistic locking.
	Version *int64
}

// NewChangeSet creates a new empty ChangeSet.
func NewChangeSet() *ChangeSet {
	return &ChangeSet{
		Fields: make(map[string]interface{}),
	}
}

// Set adds a field change to the changeset.
func (c *ChangeSet) Set(field string, value interface{}) *ChangeSet {
	c.Fields[field] = value
	return c
}

// WithVersion sets the expected version for optimistic locking.
func (c *ChangeSet) WithVersion(version int64) *ChangeSet {
	c.Version = &version
	return c
}

// HasChanges returns true if the changeset has any changes.
func (c *ChangeSet) HasChanges() bool {
	return len(c.Fields) > 0
}
