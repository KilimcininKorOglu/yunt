// Package testhelpers provides utilities for testing repository implementations
// across different database backends (SQLite, PostgreSQL, MySQL, MongoDB).
package testhelpers

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/repository/mongodb"
	"yunt/internal/repository/mysql"
	"yunt/internal/repository/postgres"
	"yunt/internal/repository/sqlite"
)

// DatabaseType represents the type of database backend.
type DatabaseType string

const (
	// DatabaseSQLite represents SQLite database.
	DatabaseSQLite DatabaseType = "sqlite"
	// DatabasePostgres represents PostgreSQL database.
	DatabasePostgres DatabaseType = "postgres"
	// DatabaseMySQL represents MySQL database.
	DatabaseMySQL DatabaseType = "mysql"
	// DatabaseMongoDB represents MongoDB database.
	DatabaseMongoDB DatabaseType = "mongodb"
)

// Environment variable names for database connections.
const (
	EnvPostgresHost     = "TEST_POSTGRES_HOST"
	EnvPostgresPort     = "TEST_POSTGRES_PORT"
	EnvPostgresUser     = "TEST_POSTGRES_USER"
	EnvPostgresPassword = "TEST_POSTGRES_PASSWORD"
	EnvPostgresDB       = "TEST_POSTGRES_DB"

	EnvMySQLHost     = "TEST_MYSQL_HOST"
	EnvMySQLPort     = "TEST_MYSQL_PORT"
	EnvMySQLUser     = "TEST_MYSQL_USER"
	EnvMySQLPassword = "TEST_MYSQL_PASSWORD"
	EnvMySQLDB       = "TEST_MYSQL_DB"

	EnvMongoHost     = "TEST_MONGO_HOST"
	EnvMongoPort     = "TEST_MONGO_PORT"
	EnvMongoUser     = "TEST_MONGO_USER"
	EnvMongoPassword = "TEST_MONGO_PASSWORD"
	EnvMongoDB       = "TEST_MONGO_DB"
)

// Default test database configuration.
const (
	DefaultPostgresHost     = "localhost"
	DefaultPostgresPort     = 15432
	DefaultPostgresUser     = "yunt"
	DefaultPostgresPassword = "yunt_test_password"
	DefaultPostgresDB       = "yunt_test"

	DefaultMySQLHost     = "localhost"
	DefaultMySQLPort     = 13306
	DefaultMySQLUser     = "yunt"
	DefaultMySQLPassword = "yunt_test_password"
	DefaultMySQLDB       = "yunt_test"

	DefaultMongoHost     = "localhost"
	DefaultMongoPort     = 17017
	DefaultMongoUser     = "yunt"
	DefaultMongoPassword = "yunt_test_password"
	DefaultMongoDB       = "yunt_test"
)

// TestConfig holds test database configuration.
type TestConfig struct {
	// PostgreSQL configuration
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	// MySQL configuration
	MySQLHost     string
	MySQLPort     int
	MySQLUser     string
	MySQLPassword string
	MySQLDB       string

	// MongoDB configuration
	MongoHost     string
	MongoPort     int
	MongoUser     string
	MongoPassword string
	MongoDB       string

	// Test options
	Timeout time.Duration
}

// DefaultTestConfig returns the default test configuration.
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		PostgresHost:     getEnvOrDefault(EnvPostgresHost, DefaultPostgresHost),
		PostgresPort:     getEnvIntOrDefault(EnvPostgresPort, DefaultPostgresPort),
		PostgresUser:     getEnvOrDefault(EnvPostgresUser, DefaultPostgresUser),
		PostgresPassword: getEnvOrDefault(EnvPostgresPassword, DefaultPostgresPassword),
		PostgresDB:       getEnvOrDefault(EnvPostgresDB, DefaultPostgresDB),

		MySQLHost:     getEnvOrDefault(EnvMySQLHost, DefaultMySQLHost),
		MySQLPort:     getEnvIntOrDefault(EnvMySQLPort, DefaultMySQLPort),
		MySQLUser:     getEnvOrDefault(EnvMySQLUser, DefaultMySQLUser),
		MySQLPassword: getEnvOrDefault(EnvMySQLPassword, DefaultMySQLPassword),
		MySQLDB:       getEnvOrDefault(EnvMySQLDB, DefaultMySQLDB),

		MongoHost:     getEnvOrDefault(EnvMongoHost, DefaultMongoHost),
		MongoPort:     getEnvIntOrDefault(EnvMongoPort, DefaultMongoPort),
		MongoUser:     getEnvOrDefault(EnvMongoUser, DefaultMongoUser),
		MongoPassword: getEnvOrDefault(EnvMongoPassword, DefaultMongoPassword),
		MongoDB:       getEnvOrDefault(EnvMongoDB, DefaultMongoDB),

		Timeout: 30 * time.Second,
	}
}

// TestRepository wraps a repository with test metadata.
type TestRepository struct {
	Repository repository.Repository
	Type       DatabaseType
	Name       string
	Cleanup    func()
}

// TestSuite manages test repositories for all database backends.
type TestSuite struct {
	config *TestConfig
	repos  map[DatabaseType]*TestRepository
	mu     sync.Mutex
}

// NewTestSuite creates a new test suite with the given configuration.
func NewTestSuite(config *TestConfig) *TestSuite {
	if config == nil {
		config = DefaultTestConfig()
	}
	return &TestSuite{
		config: config,
		repos:  make(map[DatabaseType]*TestRepository),
	}
}

// SetupSQLite creates an in-memory SQLite repository for testing.
func (s *TestSuite) SetupSQLite(t *testing.T) *TestRepository {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := &sqlite.ConnectionConfig{
		DSN:               ":memory:",
		MaxOpenConns:      1,
		MaxIdleConns:      1,
		EnableForeignKeys: true,
		JournalMode:       "MEMORY",
		SynchronousMode:   "OFF",
	}

	pool, err := sqlite.NewConnectionPool(cfg)
	if err != nil {
		t.Fatalf("failed to create SQLite connection pool: %v", err)
	}

	repo, err := sqlite.NewWithOptions(pool, true, false)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create SQLite repository: %v", err)
	}

	testRepo := &TestRepository{
		Repository: repo,
		Type:       DatabaseSQLite,
		Name:       "SQLite (in-memory)",
		Cleanup: func() {
			repo.Close()
		},
	}

	s.repos[DatabaseSQLite] = testRepo
	t.Cleanup(testRepo.Cleanup)

	return testRepo
}

// SetupPostgres creates a PostgreSQL repository for testing.
func (s *TestSuite) SetupPostgres(t *testing.T) *TestRepository {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := &postgres.ConnectionConfig{
		Host:            s.config.PostgresHost,
		Port:            s.config.PostgresPort,
		Username:        s.config.PostgresUser,
		Password:        s.config.PostgresPassword,
		Database:        s.config.PostgresDB,
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		ApplicationName: "yunt-test",
	}

	pool, err := postgres.NewConnectionPool(cfg)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
		return nil
	}

	// Clean up existing data
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	if err := cleanPostgres(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("failed to clean PostgreSQL: %v", err)
	}

	repo, err := postgres.NewWithOptions(pool, true, false)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create PostgreSQL repository: %v", err)
	}

	testRepo := &TestRepository{
		Repository: repo,
		Type:       DatabasePostgres,
		Name:       fmt.Sprintf("PostgreSQL (%s:%d)", s.config.PostgresHost, s.config.PostgresPort),
		Cleanup: func() {
			repo.Close()
		},
	}

	s.repos[DatabasePostgres] = testRepo
	t.Cleanup(testRepo.Cleanup)

	return testRepo
}

// SetupMySQL creates a MySQL repository for testing.
func (s *TestSuite) SetupMySQL(t *testing.T) *TestRepository {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := &mysql.ConnectionConfig{
		Host:              s.config.MySQLHost,
		Port:              s.config.MySQLPort,
		Username:          s.config.MySQLUser,
		Password:          s.config.MySQLPassword,
		Database:          s.config.MySQLDB,
		MaxOpenConns:      10,
		MaxIdleConns:      5,
		ConnMaxLifetime:   5 * time.Minute,
		ConnMaxIdleTime:   1 * time.Minute,
		ParseTime:         true,
		Charset:           "utf8mb4",
		Collation:         "utf8mb4_unicode_ci",
		Location:          "UTC",
		MultiStatements:   true,
		InterpolateParams: true,
	}

	pool, err := mysql.NewConnectionPool(cfg)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
		return nil
	}

	// Clean up existing data
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	if err := cleanMySQL(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("failed to clean MySQL: %v", err)
	}

	repo, err := mysql.New(pool)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create MySQL repository: %v", err)
	}

	testRepo := &TestRepository{
		Repository: repo,
		Type:       DatabaseMySQL,
		Name:       fmt.Sprintf("MySQL (%s:%d)", s.config.MySQLHost, s.config.MySQLPort),
		Cleanup: func() {
			repo.Close()
		},
	}

	s.repos[DatabaseMySQL] = testRepo
	t.Cleanup(testRepo.Cleanup)

	return testRepo
}

// SetupMongoDB creates a MongoDB repository for testing.
func (s *TestSuite) SetupMongoDB(t *testing.T) *TestRepository {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()

	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=admin",
		s.config.MongoUser,
		s.config.MongoPassword,
		s.config.MongoHost,
		s.config.MongoPort,
	)

	cfg := &mongodb.ConnectionConfig{
		URI:                    uri,
		Database:               s.config.MongoDB,
		MaxPoolSize:            10,
		MinPoolSize:            2,
		MaxConnIdleTime:        5 * time.Minute,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 10 * time.Second,
		HeartbeatInterval:      10 * time.Second,
		RetryWrites:            true,
		RetryReads:             true,
		ReadPreference:         "primary",
		WriteConcern:           "majority",
	}

	pool, err := mongodb.NewConnectionPool(cfg)
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
		return nil
	}

	// Clean up existing data
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	if err := cleanMongoDB(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("failed to clean MongoDB: %v", err)
	}

	repo, err := mongodb.New(pool)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create MongoDB repository: %v", err)
	}

	testRepo := &TestRepository{
		Repository: repo,
		Type:       DatabaseMongoDB,
		Name:       fmt.Sprintf("MongoDB (%s:%d)", s.config.MongoHost, s.config.MongoPort),
		Cleanup: func() {
			repo.Close()
		},
	}

	s.repos[DatabaseMongoDB] = testRepo
	t.Cleanup(testRepo.Cleanup)

	return testRepo
}

// SetupAllDatabases sets up all available database repositories.
func (s *TestSuite) SetupAllDatabases(t *testing.T) []*TestRepository {
	t.Helper()

	var repos []*TestRepository

	// SQLite is always available (in-memory)
	repos = append(repos, s.SetupSQLite(t))

	// Try to set up external databases (may skip if not available)
	if pg := s.SetupPostgres(t); pg != nil {
		repos = append(repos, pg)
	}

	if my := s.SetupMySQL(t); my != nil {
		repos = append(repos, my)
	}

	if mongo := s.SetupMongoDB(t); mongo != nil {
		repos = append(repos, mongo)
	}

	return repos
}

// cleanPostgres removes all data from PostgreSQL tables.
func cleanPostgres(ctx context.Context, pool *postgres.ConnectionPool) error {
	tables := []string{
		"webhook_deliveries",
		"webhooks",
		"attachment_content",
		"attachments",
		"message_recipients",
		"messages",
		"mailboxes",
		"settings_history",
		"settings",
		"users",
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := pool.ExecContext(ctx, query); err != nil {
			// Table might not exist yet, ignore
			continue
		}
	}

	return nil
}

// cleanMySQL removes all data from MySQL tables.
func cleanMySQL(ctx context.Context, pool *mysql.ConnectionPool) error {
	// Disable foreign key checks temporarily
	if _, err := pool.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}

	tables := []string{
		"webhook_deliveries",
		"webhooks",
		"attachment_content",
		"attachments",
		"message_recipients",
		"messages",
		"mailboxes",
		"settings_history",
		"settings",
		"users",
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s", table)
		if _, err := pool.ExecContext(ctx, query); err != nil {
			// Table might not exist yet, ignore
			continue
		}
	}

	// Re-enable foreign key checks
	_, err := pool.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1")
	return err
}

// cleanMongoDB removes all data from MongoDB collections.
func cleanMongoDB(ctx context.Context, pool *mongodb.ConnectionPool) error {
	collections := []string{
		mongodb.CollectionUsers,
		mongodb.CollectionMailboxes,
		mongodb.CollectionMessages,
		mongodb.CollectionMessageRecipients,
		mongodb.CollectionAttachments,
		mongodb.CollectionAttachmentContent,
		mongodb.CollectionWebhooks,
		mongodb.CollectionWebhookDeliveries,
		mongodb.CollectionSettings,
		mongodb.CollectionSettingsHistory,
	}

	for _, name := range collections {
		coll := pool.Collection(name)
		if coll != nil {
			if err := coll.Drop(ctx); err != nil {
				// Collection might not exist yet, ignore
				continue
			}
		}
	}

	return nil
}

// TestUser creates a test user with default values.
func TestUser(id string) *domain.User {
	return &domain.User{
		ID:           domain.ID(id),
		Username:     fmt.Sprintf("user_%s", id),
		Email:        fmt.Sprintf("%s@test.local", id),
		PasswordHash: "hashed_password_" + id,
		DisplayName:  fmt.Sprintf("Test User %s", id),
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
}

// TestMailbox creates a test mailbox with default values.
func TestMailbox(id string, userID domain.ID) *domain.Mailbox {
	return &domain.Mailbox{
		ID:        domain.ID(id),
		UserID:    userID,
		Name:      fmt.Sprintf("Mailbox %s", id),
		Address:   fmt.Sprintf("mailbox_%s@test.local", id),
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
}

// TestMessage creates a test message with default values.
func TestMessage(id string, mailboxID domain.ID) *domain.Message {
	return &domain.Message{
		ID:        domain.ID(id),
		MailboxID: mailboxID,
		MessageID: fmt.Sprintf("<%s@test.local>", id),
		From: domain.EmailAddress{
			Name:    "Sender",
			Address: "sender@test.local",
		},
		To: []domain.EmailAddress{
			{Address: "recipient@test.local"},
		},
		Subject:     fmt.Sprintf("Test Subject %s", id),
		TextBody:    fmt.Sprintf("Test body content for message %s", id),
		ContentType: domain.ContentTypePlain,
		Size:        1024,
		Status:      domain.MessageUnread,
		ReceivedAt:  domain.Now(),
		CreatedAt:   domain.Now(),
		UpdatedAt:   domain.Now(),
	}
}

// TestAttachment creates a test attachment with default values.
func TestAttachment(id string, messageID domain.ID) *domain.Attachment {
	return &domain.Attachment{
		ID:          domain.ID(id),
		MessageID:   messageID,
		Filename:    fmt.Sprintf("file_%s.txt", id),
		ContentType: "text/plain",
		Size:        256,
		Disposition: domain.DispositionAttachment,
		CreatedAt:   domain.Now(),
	}
}

// TestWebhook creates a test webhook with default values.
func TestWebhook(id string, userID domain.ID) *domain.Webhook {
	return &domain.Webhook{
		ID:             domain.ID(id),
		UserID:         userID,
		Name:           fmt.Sprintf("Webhook %s", id),
		URL:            fmt.Sprintf("https://example.com/webhook/%s", id),
		Events:         []domain.WebhookEvent{domain.WebhookEventMessageReceived},
		Status:         domain.WebhookStatusActive,
		MaxRetries:     3,
		TimeoutSeconds: 30,
		CreatedAt:      domain.Now(),
		UpdatedAt:      domain.Now(),
	}
}

// RunForAllDatabases runs a test function for all available databases.
func RunForAllDatabases(t *testing.T, name string, fn func(t *testing.T, repo repository.Repository)) {
	t.Helper()

	suite := NewTestSuite(nil)
	repos := suite.SetupAllDatabases(t)

	for _, testRepo := range repos {
		testRepo := testRepo // capture range variable
		t.Run(fmt.Sprintf("%s/%s", name, testRepo.Name), func(t *testing.T) {
			fn(t, testRepo.Repository)
		})
	}
}

// RunForAllDatabasesParallel runs a test function for all available databases in parallel.
func RunForAllDatabasesParallel(t *testing.T, name string, fn func(t *testing.T, repo repository.Repository)) {
	t.Helper()

	suite := NewTestSuite(nil)
	repos := suite.SetupAllDatabases(t)

	for _, testRepo := range repos {
		testRepo := testRepo // capture range variable
		t.Run(fmt.Sprintf("%s/%s", name, testRepo.Name), func(t *testing.T) {
			t.Parallel()
			fn(t, testRepo.Repository)
		})
	}
}

// PerformanceResult holds the results of a performance test.
type PerformanceResult struct {
	DatabaseType DatabaseType
	Operation    string
	Count        int
	Duration     time.Duration
	OpsPerSecond float64
}

// MeasurePerformance measures the performance of an operation.
func MeasurePerformance(dbType DatabaseType, operation string, count int, fn func() error) (*PerformanceResult, error) {
	start := time.Now()

	for i := 0; i < count; i++ {
		if err := fn(); err != nil {
			return nil, err
		}
	}

	duration := time.Since(start)
	opsPerSecond := float64(count) / duration.Seconds()

	return &PerformanceResult{
		DatabaseType: dbType,
		Operation:    operation,
		Count:        count,
		Duration:     duration,
		OpsPerSecond: opsPerSecond,
	}, nil
}

// AssertEqual compares two values and fails the test if they differ.
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotNil fails the test if the value is nil.
func AssertNotNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value == nil {
		t.Errorf("%s: expected non-nil value", msg)
	}
}

// AssertNil fails the test if the value is not nil.
func AssertNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value != nil {
		t.Errorf("%s: expected nil, got %v", msg, value)
	}
}

// AssertNoError fails the test if the error is not nil.
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: unexpected error: %v", msg, err)
	}
}

// AssertError fails the test if the error is nil.
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error, got nil", msg)
	}
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault returns the environment variable as int or a default.
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
