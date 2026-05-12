// Package sqlite provides SQLite-specific implementation of the repository interfaces.
// It implements connection management, schema creation, and CRUD operations for all entities.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"yunt/internal/config"
	"yunt/internal/repository"
)

// ConnectionPool manages SQLite database connections with pooling support.
type ConnectionPool struct {
	db      *sqlx.DB
	mu      sync.RWMutex
	config  *ConnectionConfig
	metrics *ConnectionMetrics
	cb      *repository.CircuitBreaker
	stopCh  chan struct{}
}

// ConnectionConfig holds the configuration for the SQLite connection pool.
type ConnectionConfig struct {
	// DSN is the Data Source Name for SQLite connection.
	// For file-based: "file:path/to/db.sqlite?mode=rwc&_journal_mode=WAL"
	// For in-memory: ":memory:" or "file::memory:?cache=shared"
	DSN string

	// MaxOpenConns is the maximum number of open connections.
	// SQLite works best with a single writer, so this is typically 1 for write
	// operations, but can be higher for read-only scenarios.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	ConnMaxIdleTime time.Duration

	// BusyTimeout is the SQLite busy timeout in milliseconds.
	// This helps handle concurrent access.
	BusyTimeout int

	// EnableForeignKeys enables foreign key constraint enforcement.
	EnableForeignKeys bool

	// JournalMode sets the SQLite journal mode (e.g., WAL, DELETE, MEMORY).
	JournalMode string

	// SynchronousMode sets the SQLite synchronous mode (e.g., OFF, NORMAL, FULL).
	SynchronousMode string

	// CacheSize sets the SQLite cache size in pages (negative for KB).
	CacheSize int
}

// ConnectionMetrics tracks connection pool statistics.
type ConnectionMetrics struct {
	mu              sync.RWMutex
	totalOpened     int64
	totalClosed     int64
	currentOpen     int
	currentInUse    int
	lastHealthCheck time.Time
	lastError       error
}

// DefaultConnectionConfig returns a sensible default configuration for SQLite.
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		DSN:               "file:yunt.db?mode=rwc&_journal_mode=WAL",
		MaxOpenConns:      1,      // SQLite works best with single writer
		MaxIdleConns:      1,      // Keep one connection idle
		ConnMaxLifetime:   0,      // No maximum lifetime
		ConnMaxIdleTime:   0,      // No maximum idle time
		BusyTimeout:       5000,   // 5 seconds
		EnableForeignKeys: true,   // Enable FK constraints
		JournalMode:       "WAL",  // Write-Ahead Logging for better concurrency
		SynchronousMode:   "NORMAL",
		CacheSize:         -64000, // 64MB cache
	}
}

// NewConnectionConfig creates a ConnectionConfig from the application config.
func NewConnectionConfig(cfg *config.DatabaseConfig) *ConnectionConfig {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = "file:yunt.db?mode=rwc"
	}

	maxOpen := cfg.MaxOpenConns
	maxIdle := cfg.MaxIdleConns
	if maxOpen <= 0 {
		maxOpen = 1
	}
	if maxIdle <= 0 {
		maxIdle = 1
	}

	return &ConnectionConfig{
		DSN:               dsn,
		MaxOpenConns:      maxOpen,
		MaxIdleConns:      maxIdle,
		ConnMaxLifetime:   cfg.ConnMaxLifetime,
		ConnMaxIdleTime:   cfg.ConnMaxIdleTime,
		BusyTimeout:       5000,
		EnableForeignKeys: true,
		JournalMode:       "WAL",
		SynchronousMode:   "NORMAL",
		CacheSize:         -64000,
	}
}

// NewConnectionPool creates a new SQLite connection pool.
func NewConnectionPool(cfg *ConnectionConfig) (*ConnectionPool, error) {
	if cfg == nil {
		cfg = DefaultConnectionConfig()
	}

	pool := &ConnectionPool{
		config:  cfg,
		metrics: &ConnectionMetrics{},
		cb:      repository.NewCircuitBreaker(repository.DefaultCircuitBreakerConfig()),
		stopCh:  make(chan struct{}),
	}

	if err := pool.connect(); err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	go pool.healthMonitor()

	return pool, nil
}

// connect establishes the database connection and configures the pool.
func (p *ConnectionPool) connect() error {
	db, err := sqlx.Open("sqlite", p.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(p.config.MaxOpenConns)
	db.SetMaxIdleConns(p.config.MaxIdleConns)

	if p.config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(p.config.ConnMaxLifetime)
	}
	if p.config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(p.config.ConnMaxIdleTime)
	}

	// Configure SQLite pragmas
	if err := p.configurePragmas(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to configure SQLite pragmas: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	p.mu.Lock()
	p.db = db
	p.metrics.totalOpened++
	p.metrics.currentOpen = 1
	p.mu.Unlock()

	return nil
}

// configurePragmas sets SQLite pragmas for optimal performance and safety.
func (p *ConnectionPool) configurePragmas(db *sqlx.DB) error {
	pragmas := []struct {
		name  string
		value interface{}
	}{
		{"busy_timeout", p.config.BusyTimeout},
		{"journal_mode", p.config.JournalMode},
		{"synchronous", p.config.SynchronousMode},
		{"cache_size", p.config.CacheSize},
		{"temp_store", "MEMORY"},
		{"mmap_size", 268435456}, // 256MB memory-mapped I/O
	}

	if p.config.EnableForeignKeys {
		pragmas = append(pragmas, struct {
			name  string
			value interface{}
		}{"foreign_keys", "ON"})
	}

	for _, pragma := range pragmas {
		query := fmt.Sprintf("PRAGMA %s = %v", pragma.name, pragma.value)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to set pragma %s: %w", pragma.name, err)
		}
	}

	return nil
}

// DB returns the underlying sqlx.DB instance.
func (p *ConnectionPool) DB() *sqlx.DB {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db
}

// Close closes all connections in the pool.
func (p *ConnectionPool) Close() error {
	close(p.stopCh)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return nil
	}

	err := p.db.Close()
	p.db = nil
	p.metrics.totalClosed++
	p.metrics.currentOpen = 0
	p.metrics.currentInUse = 0

	return err
}

// Health checks the health of the database connection.
func (p *ConnectionPool) Health(ctx context.Context) error {
	if !p.cb.Allow() {
		return repository.ErrCircuitOpen
	}

	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		p.cb.RecordFailure()
		return fmt.Errorf("database connection is closed")
	}

	if err := db.PingContext(ctx); err != nil {
		p.recordError(err)
		p.cb.RecordFailure()
		return fmt.Errorf("database ping failed: %w", err)
	}

	p.cb.RecordSuccess()
	p.recordHealthCheck()
	return nil
}

// CircuitBreaker returns the pool's circuit breaker for external inspection.
func (p *ConnectionPool) CircuitBreaker() *repository.CircuitBreaker {
	return p.cb
}

// healthMonitor runs periodic health checks in the background.
func (p *ConnectionPool) healthMonitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			p.Health(ctx)
			cancel()
		}
	}
}

// Stats returns current connection pool statistics.
func (p *ConnectionPool) Stats() sql.DBStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.db == nil {
		return sql.DBStats{}
	}

	return p.db.Stats()
}

// Metrics returns connection metrics.
func (p *ConnectionPool) Metrics() ConnectionMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	return ConnectionMetrics{
		totalOpened:     p.metrics.totalOpened,
		totalClosed:     p.metrics.totalClosed,
		currentOpen:     p.metrics.currentOpen,
		currentInUse:    p.metrics.currentInUse,
		lastHealthCheck: p.metrics.lastHealthCheck,
		lastError:       p.metrics.lastError,
	}
}

// recordHealthCheck records a successful health check.
func (p *ConnectionPool) recordHealthCheck() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.lastHealthCheck = time.Now()
}

// recordError records an error occurrence.
func (p *ConnectionPool) recordError(err error) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.lastError = err
}

// BeginTx starts a new transaction.
func (p *ConnectionPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	return db.BeginTxx(ctx, opts)
}

// ExecContext executes a query that doesn't return rows.
func (p *ConnectionPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	return db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows.
func (p *ConnectionPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	return db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns at most one row.
func (p *ConnectionPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil
	}

	return db.QueryRowContext(ctx, query, args...)
}

// SelectContext executes a query and scans results into dest.
func (p *ConnectionPool) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database connection is closed")
	}

	return db.SelectContext(ctx, dest, query, args...)
}

// GetContext executes a query and scans a single row into dest.
func (p *ConnectionPool) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database connection is closed")
	}

	return db.GetContext(ctx, dest, query, args...)
}

// NamedExecContext executes a named query that doesn't return rows.
func (p *ConnectionPool) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	return db.NamedExecContext(ctx, query, arg)
}

// Rebind transforms a query from QUESTION to the DB's bindvar type.
func (p *ConnectionPool) Rebind(query string) string {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return query
	}

	return db.Rebind(query)
}

// In expands slice values in args, returning the modified query string and a new arg list.
func (p *ConnectionPool) In(query string, args ...interface{}) (string, []interface{}, error) {
	return sqlx.In(query, args...)
}

// Reconnect closes the existing connection and establishes a new one.
func (p *ConnectionPool) Reconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db != nil {
		p.db.Close()
		p.db = nil
		p.metrics.totalClosed++
	}

	return p.connect()
}

// Version returns the SQLite version.
func (p *ConnectionPool) Version(ctx context.Context) (string, error) {
	var version string
	err := p.GetContext(ctx, &version, "SELECT sqlite_version()")
	return version, err
}
