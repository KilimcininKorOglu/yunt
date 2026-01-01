// Package mysql provides MySQL-specific implementation of the repository interfaces.
// It implements connection management, schema creation, and CRUD operations for all entities.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/go-sql-driver/mysql"

	"yunt/internal/config"
)

// ConnectionPool manages MySQL database connections with pooling support.
type ConnectionPool struct {
	db      *sqlx.DB
	mu      sync.RWMutex
	config  *ConnectionConfig
	metrics *ConnectionMetrics
}

// ConnectionConfig holds the configuration for the MySQL connection pool.
type ConnectionConfig struct {
	// DSN is the Data Source Name for MySQL connection.
	// Format: user:password@tcp(host:port)/dbname?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci
	DSN string

	// Host is the MySQL server host.
	Host string

	// Port is the MySQL server port.
	Port int

	// Username is the MySQL user.
	Username string

	// Password is the MySQL password.
	Password string

	// Database is the database name.
	Database string

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	ConnMaxIdleTime time.Duration

	// ParseTime enables automatic parsing of DATE and DATETIME values.
	ParseTime bool

	// Charset sets the character set for the connection.
	Charset string

	// Collation sets the collation for the connection.
	Collation string

	// Location sets the time zone for the connection.
	Location string

	// MultiStatements allows executing multiple statements in one query.
	MultiStatements bool

	// InterpolateParams enables client-side parameter interpolation.
	InterpolateParams bool
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

// DefaultConnectionConfig returns a sensible default configuration for MySQL.
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Host:              "localhost",
		Port:              3306,
		Username:          "root",
		Password:          "",
		Database:          "yunt",
		MaxOpenConns:      25,
		MaxIdleConns:      10,
		ConnMaxLifetime:   5 * time.Minute,
		ConnMaxIdleTime:   5 * time.Minute,
		ParseTime:         true,
		Charset:           "utf8mb4",
		Collation:         "utf8mb4_unicode_ci",
		Location:          "UTC",
		MultiStatements:   true,
		InterpolateParams: true,
	}
}

// NewConnectionConfig creates a ConnectionConfig from the application config.
func NewConnectionConfig(cfg *config.DatabaseConfig) *ConnectionConfig {
	connCfg := &ConnectionConfig{
		DSN:               cfg.DSN,
		Host:              cfg.Host,
		Port:              cfg.Port,
		Username:          cfg.Username,
		Password:          cfg.Password,
		Database:          cfg.Name,
		MaxOpenConns:      cfg.MaxOpenConns,
		MaxIdleConns:      cfg.MaxIdleConns,
		ConnMaxLifetime:   cfg.ConnMaxLifetime,
		ConnMaxIdleTime:   cfg.ConnMaxIdleTime,
		ParseTime:         true,
		Charset:           "utf8mb4",
		Collation:         "utf8mb4_unicode_ci",
		Location:          "UTC",
		MultiStatements:   true,
		InterpolateParams: true,
	}

	// Apply defaults if not specified
	if connCfg.Host == "" {
		connCfg.Host = "localhost"
	}
	if connCfg.Port == 0 {
		connCfg.Port = 3306
	}
	if connCfg.Database == "" {
		connCfg.Database = "yunt"
	}
	if connCfg.MaxOpenConns == 0 {
		connCfg.MaxOpenConns = 25
	}
	if connCfg.MaxIdleConns == 0 {
		connCfg.MaxIdleConns = 10
	}
	if connCfg.ConnMaxLifetime == 0 {
		connCfg.ConnMaxLifetime = 5 * time.Minute
	}
	if connCfg.ConnMaxIdleTime == 0 {
		connCfg.ConnMaxIdleTime = 5 * time.Minute
	}

	return connCfg
}

// BuildDSN constructs a DSN string from the configuration.
func (c *ConnectionConfig) BuildDSN() string {
	if c.DSN != "" {
		return c.DSN
	}

	// Build DSN from individual components
	// Format: user:password@tcp(host:port)/dbname?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?",
		c.Username, c.Password, c.Host, c.Port, c.Database)

	params := make([]string, 0)
	if c.ParseTime {
		params = append(params, "parseTime=true")
	}
	if c.Charset != "" {
		params = append(params, fmt.Sprintf("charset=%s", c.Charset))
	}
	if c.Collation != "" {
		params = append(params, fmt.Sprintf("collation=%s", c.Collation))
	}
	if c.Location != "" {
		params = append(params, fmt.Sprintf("loc=%s", c.Location))
	}
	if c.MultiStatements {
		params = append(params, "multiStatements=true")
	}
	if c.InterpolateParams {
		params = append(params, "interpolateParams=true")
	}

	for i, param := range params {
		if i > 0 {
			dsn += "&"
		}
		dsn += param
	}

	return dsn
}

// NewConnectionPool creates a new MySQL connection pool.
func NewConnectionPool(cfg *ConnectionConfig) (*ConnectionPool, error) {
	if cfg == nil {
		cfg = DefaultConnectionConfig()
	}

	pool := &ConnectionPool{
		config:  cfg,
		metrics: &ConnectionMetrics{},
	}

	if err := pool.connect(); err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return pool, nil
}

// connect establishes the database connection and configures the pool.
func (p *ConnectionPool) connect() error {
	dsn := p.config.BuildDSN()

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL database: %w", err)
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
		return fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	// Set session variables for optimal behavior
	if err := p.configureSessions(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to configure MySQL sessions: %w", err)
	}

	p.mu.Lock()
	p.db = db
	p.metrics.totalOpened++
	p.metrics.currentOpen = 1
	p.mu.Unlock()

	return nil
}

// configureSessions sets MySQL session variables for optimal performance and consistency.
func (p *ConnectionPool) configureSessions(db *sqlx.DB) error {
	sessionVars := []string{
		// Use UTF8MB4 for full Unicode support
		"SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci",
		// Use UTC for consistent timestamps
		"SET time_zone = '+00:00'",
		// Set SQL mode for strict behavior
		"SET sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO'",
	}

	for _, query := range sessionVars {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute session config '%s': %w", query, err)
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
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database connection is closed")
	}

	if err := db.PingContext(ctx); err != nil {
		p.recordError(err)
		return fmt.Errorf("database ping failed: %w", err)
	}

	p.recordHealthCheck()
	return nil
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

// Version returns the MySQL server version.
func (p *ConnectionPool) Version(ctx context.Context) (string, error) {
	var version string
	err := p.GetContext(ctx, &version, "SELECT VERSION()")
	return version, err
}

// DatabaseSize returns the size of the current database in bytes.
func (p *ConnectionPool) DatabaseSize(ctx context.Context) (int64, error) {
	query := `SELECT SUM(data_length + index_length) as size 
		FROM information_schema.TABLES 
		WHERE table_schema = DATABASE()`

	var size sql.NullInt64
	if err := p.GetContext(ctx, &size, query); err != nil {
		return 0, err
	}

	if size.Valid {
		return size.Int64, nil
	}
	return 0, nil
}

// TableExists checks if a table exists in the database.
func (p *ConnectionPool) TableExists(ctx context.Context, tableName string) (bool, error) {
	query := `SELECT COUNT(*) FROM information_schema.TABLES 
		WHERE table_schema = DATABASE() AND table_name = ?`

	var count int
	if err := p.GetContext(ctx, &count, query, tableName); err != nil {
		return false, err
	}

	return count > 0, nil
}
