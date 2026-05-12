// Package postgres provides PostgreSQL-specific implementation of the repository interfaces.
// It implements connection management, schema creation, and CRUD operations for all entities.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"yunt/internal/config"
	"yunt/internal/repository"
)

// ConnectionPool manages PostgreSQL database connections with pooling support.
type ConnectionPool struct {
	db      *sqlx.DB
	mu      sync.RWMutex
	config  *ConnectionConfig
	metrics *ConnectionMetrics
	cb      *repository.CircuitBreaker
	stopCh  chan struct{}
}

// ConnectionConfig holds the configuration for the PostgreSQL connection pool.
type ConnectionConfig struct {
	// DSN is the Data Source Name for PostgreSQL connection.
	// Format: "host=localhost port=5432 user=postgres password=secret dbname=yunt sslmode=disable"
	DSN string

	// Host is the database host.
	Host string

	// Port is the database port.
	Port int

	// Database is the database name.
	Database string

	// Username is the database username.
	Username string

	// Password is the database password.
	Password string

	// SSLMode is the SSL mode (disable, require, verify-ca, verify-full).
	SSLMode string

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	ConnMaxIdleTime time.Duration

	// StatementCacheCapacity is the capacity of prepared statement cache.
	StatementCacheCapacity int

	// ApplicationName is the application name sent to PostgreSQL.
	ApplicationName string
}

// ConnectionMetrics tracks connection pool statistics.
type ConnectionMetrics struct {
	mu              sync.RWMutex
	totalOpened     int64
	totalClosed     int64
	currentOpen     int
	currentInUse    int
	totalQueries    int64
	totalExecTime   time.Duration
	lastHealthCheck time.Time
	lastError       error
}

// DefaultConnectionConfig returns a sensible default configuration for PostgreSQL.
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Host:                   "localhost",
		Port:                   5432,
		Database:               "yunt",
		Username:               "postgres",
		Password:               "",
		SSLMode:                "disable",
		MaxOpenConns:           25,
		MaxIdleConns:           5,
		ConnMaxLifetime:        30 * time.Minute,
		ConnMaxIdleTime:        5 * time.Minute,
		StatementCacheCapacity: 512,
		ApplicationName:        "yunt",
	}
}

// NewConnectionConfig creates a ConnectionConfig from the application config.
func NewConnectionConfig(cfg *config.DatabaseConfig) *ConnectionConfig {
	connConfig := &ConnectionConfig{
		DSN:             cfg.DSN,
		Host:            cfg.Host,
		Port:            cfg.Port,
		Database:        cfg.Name,
		Username:        cfg.Username,
		Password:        cfg.Password,
		SSLMode:         cfg.SSLMode,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ApplicationName: "yunt",
	}

	// Apply defaults for zero values
	if connConfig.MaxOpenConns == 0 {
		connConfig.MaxOpenConns = 25
	}
	if connConfig.MaxIdleConns == 0 {
		connConfig.MaxIdleConns = 5
	}
	if connConfig.SSLMode == "" {
		connConfig.SSLMode = "disable"
	}
	if connConfig.Port == 0 {
		connConfig.Port = 5432
	}

	return connConfig
}

// BuildDSN constructs a PostgreSQL DSN from the configuration.
func (c *ConnectionConfig) BuildDSN() string {
	if c.DSN != "" {
		return c.DSN
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Database, c.SSLMode,
	)

	if c.Password != "" {
		dsn += fmt.Sprintf(" password=%s", c.Password)
	}

	if c.ApplicationName != "" {
		dsn += fmt.Sprintf(" application_name=%s", c.ApplicationName)
	}

	return dsn
}

// NewConnectionPool creates a new PostgreSQL connection pool.
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
	dsn := p.config.BuildDSN()

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
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

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	p.mu.Lock()
	p.db = db
	p.metrics.totalOpened++
	p.metrics.currentOpen = 1
	p.mu.Unlock()

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

// CircuitBreaker returns the pool's circuit breaker.
func (p *ConnectionPool) CircuitBreaker() *repository.CircuitBreaker {
	return p.cb
}

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
		totalQueries:    p.metrics.totalQueries,
		totalExecTime:   p.metrics.totalExecTime,
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

// recordQuery records query execution metrics.
func (p *ConnectionPool) recordQuery(duration time.Duration) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.totalQueries++
	p.metrics.totalExecTime += duration
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

	start := time.Now()
	result, err := db.ExecContext(ctx, query, args...)
	p.recordQuery(time.Since(start))

	return result, err
}

// QueryContext executes a query that returns rows.
func (p *ConnectionPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	start := time.Now()
	rows, err := db.QueryContext(ctx, query, args...)
	p.recordQuery(time.Since(start))

	return rows, err
}

// QueryRowContext executes a query that returns at most one row.
func (p *ConnectionPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil
	}

	start := time.Now()
	row := db.QueryRowContext(ctx, query, args...)
	p.recordQuery(time.Since(start))

	return row
}

// SelectContext executes a query and scans results into dest.
func (p *ConnectionPool) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database connection is closed")
	}

	start := time.Now()
	err := db.SelectContext(ctx, dest, query, args...)
	p.recordQuery(time.Since(start))

	return err
}

// GetContext executes a query and scans a single row into dest.
func (p *ConnectionPool) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database connection is closed")
	}

	start := time.Now()
	err := db.GetContext(ctx, dest, query, args...)
	p.recordQuery(time.Since(start))

	return err
}

// NamedExecContext executes a named query that doesn't return rows.
func (p *ConnectionPool) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	start := time.Now()
	result, err := db.NamedExecContext(ctx, query, arg)
	p.recordQuery(time.Since(start))

	return result, err
}

// Rebind transforms a query from QUESTION to the DB's bindvar type (PostgreSQL uses $1, $2, etc.).
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

// Version returns the PostgreSQL version.
func (p *ConnectionPool) Version(ctx context.Context) (string, error) {
	var version string
	err := p.GetContext(ctx, &version, "SELECT version()")
	return version, err
}

// ServerVersion returns just the PostgreSQL server version number.
func (p *ConnectionPool) ServerVersion(ctx context.Context) (string, error) {
	var version string
	err := p.GetContext(ctx, &version, "SHOW server_version")
	return version, err
}
