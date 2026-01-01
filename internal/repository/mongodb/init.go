package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"yunt/internal/config"
)

// Initializer handles MongoDB database initialization including
// connection setup, index creation, and initial data seeding.
type Initializer struct {
	pool         *ConnectionPool
	indexManager *IndexManager
	seeder       *Seeder
	repo         *Repository
	logger       *slog.Logger
	config       *InitConfig
}

// InitConfig contains configuration for database initialization.
type InitConfig struct {
	// CreateIndexes determines if indexes should be created on startup.
	CreateIndexes bool

	// SeedDatabase determines if the database should be seeded with initial data.
	SeedDatabase bool

	// SeedConfig contains the seed configuration.
	SeedConfig *SeedConfig

	// DropIndexesFirst determines if existing indexes should be dropped before creating new ones.
	// Use with caution - only for development.
	DropIndexesFirst bool

	// ConnectionTimeout is the timeout for establishing a connection.
	ConnectionTimeout time.Duration

	// InitializationTimeout is the timeout for the entire initialization process.
	InitializationTimeout time.Duration
}

// DefaultInitConfig returns the default initialization configuration.
func DefaultInitConfig() *InitConfig {
	return &InitConfig{
		CreateIndexes:         true,
		SeedDatabase:          true,
		SeedConfig:            DefaultSeedConfig(),
		DropIndexesFirst:      false,
		ConnectionTimeout:     10 * time.Second,
		InitializationTimeout: 60 * time.Second,
	}
}

// InitResult contains the result of database initialization.
type InitResult struct {
	// Success indicates if initialization completed successfully.
	Success bool

	// ConnectionEstablished indicates if the database connection was established.
	ConnectionEstablished bool

	// IndexesCreated indicates the number of indexes created.
	IndexesCreated int

	// DatabaseSeeded indicates if the database was seeded.
	DatabaseSeeded bool

	// MongoDBVersion is the version of the connected MongoDB server.
	MongoDBVersion string

	// Duration is the time taken for initialization.
	Duration time.Duration

	// Errors contains any errors that occurred during initialization.
	Errors []string
}

// NewInitializer creates a new database initializer.
func NewInitializer(cfg *config.DatabaseConfig, logger *slog.Logger) (*Initializer, error) {
	if logger == nil {
		logger = slog.Default()
	}

	connConfig := NewConnectionConfig(cfg)
	pool, err := NewConnectionPool(connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	repo, err := New(pool)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return &Initializer{
		pool:         pool,
		indexManager: NewIndexManager(pool, logger),
		seeder:       NewSeeder(repo, logger),
		repo:         repo,
		logger:       logger,
		config:       DefaultInitConfig(),
	}, nil
}

// NewInitializerWithPool creates a new initializer with an existing connection pool.
func NewInitializerWithPool(pool *ConnectionPool, logger *slog.Logger) (*Initializer, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool is required")
	}

	if logger == nil {
		logger = slog.Default()
	}

	repo, err := New(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return &Initializer{
		pool:         pool,
		indexManager: NewIndexManager(pool, logger),
		seeder:       NewSeeder(repo, logger),
		repo:         repo,
		logger:       logger,
		config:       DefaultInitConfig(),
	}, nil
}

// WithConfig sets the initialization configuration.
func (i *Initializer) WithConfig(cfg *InitConfig) *Initializer {
	if cfg != nil {
		i.config = cfg
	}
	return i
}

// Initialize performs the complete database initialization process.
func (i *Initializer) Initialize(ctx context.Context) (*InitResult, error) {
	startTime := time.Now()
	result := &InitResult{
		Errors: make([]string, 0),
	}

	// Apply initialization timeout
	timeout := i.config.InitializationTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	i.logger.Info("starting MongoDB database initialization")

	// Step 1: Verify connection
	if err := i.verifyConnection(ctx); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("connection verification failed: %v", err))
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("connection verification failed: %w", err)
	}
	result.ConnectionEstablished = true

	// Get MongoDB version
	version, err := i.pool.Version(ctx)
	if err != nil {
		i.logger.Warn("failed to get MongoDB version", "error", err)
	} else {
		result.MongoDBVersion = version
		i.logger.Info("connected to MongoDB", "version", version)
	}

	// Step 2: Drop indexes if configured (development only)
	if i.config.DropIndexesFirst {
		i.logger.Warn("dropping existing indexes (development mode)")
		if err := i.indexManager.DropAllIndexes(ctx); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to drop indexes: %v", err))
			i.logger.Error("failed to drop indexes", "error", err)
			// Continue anyway - this is not fatal
		}
	}

	// Step 3: Create indexes
	if i.config.CreateIndexes {
		if err := i.indexManager.EnsureAllIndexes(ctx); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("index creation failed: %v", err))
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("index creation failed: %w", err)
		}
		i.logger.Info("indexes created successfully")
	}

	// Step 4: Seed database
	if i.config.SeedDatabase {
		seedConfig := i.config.SeedConfig
		if seedConfig == nil {
			seedConfig = DefaultSeedConfig()
		}

		if err := i.seeder.SeedWithConfig(ctx, seedConfig); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("database seeding failed: %v", err))
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("database seeding failed: %w", err)
		}
		result.DatabaseSeeded = true
		i.logger.Info("database seeded successfully")
	}

	result.Success = true
	result.Duration = time.Since(startTime)

	i.logger.Info("MongoDB database initialization completed",
		"duration", result.Duration,
		"success", result.Success,
		"seeded", result.DatabaseSeeded)

	return result, nil
}

// verifyConnection verifies the database connection is working.
func (i *Initializer) verifyConnection(ctx context.Context) error {
	if err := i.pool.Health(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// Repository returns the initialized repository.
func (i *Initializer) Repository() *Repository {
	return i.repo
}

// Seeder returns the database seeder.
func (i *Initializer) Seeder() *Seeder {
	return i.seeder
}

// IndexManager returns the index manager.
func (i *Initializer) IndexManager() *IndexManager {
	return i.indexManager
}

// Close closes the database connection.
func (i *Initializer) Close() error {
	return i.pool.Close()
}

// InitializeFromConfig performs database initialization using application config.
func InitializeFromConfig(ctx context.Context, cfg *config.DatabaseConfig, logger *slog.Logger) (*Repository, error) {
	if logger == nil {
		logger = slog.Default()
	}

	initializer, err := NewInitializer(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create initializer: %w", err)
	}

	result, err := initializer.Initialize(ctx)
	if err != nil {
		initializer.Close()
		return nil, fmt.Errorf("initialization failed: %w", err)
	}

	if !result.Success {
		initializer.Close()
		return nil, fmt.Errorf("initialization failed: %v", result.Errors)
	}

	return initializer.Repository(), nil
}

// QuickInit provides a simple initialization function for common use cases.
// It creates indexes and seeds the database with default configuration.
func QuickInit(ctx context.Context, dsn string, database string, logger *slog.Logger) (*Repository, error) {
	cfg := &config.DatabaseConfig{
		DSN:  dsn,
		Name: database,
	}
	return InitializeFromConfig(ctx, cfg, logger)
}

// MustQuickInit is like QuickInit but panics on error.
// Useful for test setup and simple applications.
func MustQuickInit(ctx context.Context, dsn string, database string, logger *slog.Logger) *Repository {
	repo, err := QuickInit(ctx, dsn, database, logger)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize MongoDB: %v", err))
	}
	return repo
}

// HealthCheck performs a health check on the database connection.
func (i *Initializer) HealthCheck(ctx context.Context) error {
	return i.pool.Health(ctx)
}

// GetStatus returns the current status of the database.
func (i *Initializer) GetStatus(ctx context.Context) (*DatabaseStatus, error) {
	status := &DatabaseStatus{
		CheckedAt: time.Now().UTC(),
	}

	// Check connection health
	if err := i.pool.Health(ctx); err != nil {
		status.Connected = false
		status.Error = err.Error()
		return status, nil
	}
	status.Connected = true

	// Get MongoDB version
	version, err := i.pool.Version(ctx)
	if err != nil {
		status.Error = fmt.Sprintf("failed to get version: %v", err)
	} else {
		status.Version = version
	}

	// Get seed status
	seedStatus, err := i.seeder.GetSeedStatus(ctx)
	if err != nil {
		status.Error = fmt.Sprintf("failed to get seed status: %v", err)
	} else {
		status.SeedStatus = seedStatus
	}

	// Get collection stats
	collections, err := i.pool.ListCollections(ctx)
	if err != nil {
		status.Error = fmt.Sprintf("failed to list collections: %v", err)
	} else {
		status.Collections = collections
	}

	return status, nil
}

// DatabaseStatus contains information about the database status.
type DatabaseStatus struct {
	Connected   bool        `json:"connected"`
	Version     string      `json:"version,omitempty"`
	Collections []string    `json:"collections,omitempty"`
	SeedStatus  *SeedStatus `json:"seedStatus,omitempty"`
	CheckedAt   time.Time   `json:"checkedAt"`
	Error       string      `json:"error,omitempty"`
}

// EnsureIndexes re-creates all indexes (useful after schema changes).
func (i *Initializer) EnsureIndexes(ctx context.Context) error {
	return i.indexManager.EnsureAllIndexes(ctx)
}

// EnsureSeed ensures the database is seeded (idempotent).
func (i *Initializer) EnsureSeed(ctx context.Context) error {
	return i.seeder.Seed(ctx)
}

// Reset drops all data and re-initializes the database.
// Use with extreme caution - this deletes all data!
func (i *Initializer) Reset(ctx context.Context) error {
	i.logger.Warn("resetting database - all data will be deleted")
	return i.seeder.ResetDatabase(ctx)
}
