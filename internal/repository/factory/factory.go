// Package factory provides repository instantiation based on configuration.
// It implements the factory pattern for creating database backend instances
// at runtime based on the configured database driver.
package factory

import (
	"fmt"

	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/repository/sqlite"
)

// Factory creates repository instances based on configuration.
// It supports multiple database backends and validates configuration
// before creating the repository.
type Factory struct {
	// config holds the database configuration.
	config *config.DatabaseConfig
}

// Option is a functional option for configuring the factory.
type Option func(*Factory)

// New creates a new repository factory with the given configuration.
// Returns an error if the configuration is nil or invalid.
func New(cfg *config.DatabaseConfig, opts ...Option) (*Factory, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database configuration is required")
	}

	if err := ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	factory := &Factory{
		config: cfg,
	}

	for _, opt := range opts {
		opt(factory)
	}

	return factory, nil
}

// Create creates and returns a repository instance based on the configured driver.
// It returns the Repository interface, not the concrete implementation.
func (f *Factory) Create() (repository.Repository, error) {
	driver := domain.DatabaseDriver(f.config.Driver)

	switch driver {
	case domain.DatabaseDriverSQLite:
		return f.createSQLite()
	case domain.DatabaseDriverPostgres:
		return nil, fmt.Errorf("database driver %q is not yet implemented", driver)
	case domain.DatabaseDriverMySQL:
		return nil, fmt.Errorf("database driver %q is not yet implemented", driver)
	case domain.DatabaseDriverMongoDB:
		return nil, fmt.Errorf("database driver %q is not yet implemented", driver)
	default:
		return nil, fmt.Errorf("unsupported database driver: %q. Supported drivers are: sqlite, postgres, mysql, mongodb", f.config.Driver)
	}
}

// createSQLite creates a SQLite repository instance.
func (f *Factory) createSQLite() (repository.Repository, error) {
	connConfig := sqlite.NewConnectionConfig(f.config)

	pool, err := sqlite.NewConnectionPool(connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite connection pool: %w", err)
	}

	repo, err := sqlite.New(pool)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create SQLite repository: %w", err)
	}

	return repo, nil
}

// Driver returns the database driver from the configuration.
func (f *Factory) Driver() domain.DatabaseDriver {
	return domain.DatabaseDriver(f.config.Driver)
}

// Config returns the database configuration.
func (f *Factory) Config() *config.DatabaseConfig {
	return f.config
}

// ValidateConfig validates the database configuration.
func ValidateConfig(cfg *config.DatabaseConfig) error {
	if cfg.Driver == "" {
		return fmt.Errorf("driver is required")
	}

	driver := domain.DatabaseDriver(cfg.Driver)
	if !driver.IsValid() {
		return fmt.Errorf("invalid driver %q: supported drivers are sqlite, postgres, mysql, mongodb", cfg.Driver)
	}

	// Driver-specific validation
	switch driver {
	case domain.DatabaseDriverSQLite:
		// SQLite can work with empty DSN (uses default)
		// No additional validation needed
	case domain.DatabaseDriverPostgres, domain.DatabaseDriverMySQL:
		// For these drivers, we need connection info
		if cfg.DSN == "" && cfg.Host == "" {
			return fmt.Errorf("either DSN or Host is required for %s driver", driver)
		}
	case domain.DatabaseDriverMongoDB:
		// MongoDB requires connection info
		if cfg.DSN == "" && cfg.Host == "" {
			return fmt.Errorf("either DSN or Host is required for MongoDB driver")
		}
	}

	// Validate connection pool settings
	if cfg.MaxOpenConns < 0 {
		return fmt.Errorf("maxOpenConns cannot be negative")
	}
	if cfg.MaxIdleConns < 0 {
		return fmt.Errorf("maxIdleConns cannot be negative")
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns && cfg.MaxOpenConns > 0 {
		return fmt.Errorf("maxIdleConns (%d) cannot be greater than maxOpenConns (%d)", cfg.MaxIdleConns, cfg.MaxOpenConns)
	}

	return nil
}

// SupportedDrivers returns a list of supported database drivers.
func SupportedDrivers() []domain.DatabaseDriver {
	return []domain.DatabaseDriver{
		domain.DatabaseDriverSQLite,
		domain.DatabaseDriverPostgres,
		domain.DatabaseDriverMySQL,
		domain.DatabaseDriverMongoDB,
	}
}

// IsDriverSupported checks if the given driver string is supported.
func IsDriverSupported(driver string) bool {
	return domain.DatabaseDriver(driver).IsValid()
}

// ImplementedDrivers returns a list of currently implemented database drivers.
func ImplementedDrivers() []domain.DatabaseDriver {
	return []domain.DatabaseDriver{
		domain.DatabaseDriverSQLite,
	}
}

// IsDriverImplemented checks if the given driver is currently implemented.
func IsDriverImplemented(driver string) bool {
	d := domain.DatabaseDriver(driver)
	for _, implemented := range ImplementedDrivers() {
		if d == implemented {
			return true
		}
	}
	return false
}
