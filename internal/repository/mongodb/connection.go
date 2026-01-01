// Package mongodb provides MongoDB-specific implementation of the repository interfaces.
// It implements connection management, schema creation, and CRUD operations for all entities.
package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"yunt/internal/config"
)

// Collection names for all entities.
const (
	CollectionUsers             = "users"
	CollectionMailboxes         = "mailboxes"
	CollectionMessages          = "messages"
	CollectionMessageRecipients = "message_recipients"
	CollectionAttachments       = "attachments"
	CollectionAttachmentContent = "attachment_content"
	CollectionWebhooks          = "webhooks"
	CollectionWebhookDeliveries = "webhook_deliveries"
	CollectionSettings          = "settings"
	CollectionSettingsHistory   = "settings_history"
)

// ConnectionPool manages MongoDB database connections with pooling support.
type ConnectionPool struct {
	client  *mongo.Client
	db      *mongo.Database
	mu      sync.RWMutex
	config  *ConnectionConfig
	metrics *ConnectionMetrics
}

// ConnectionConfig holds the configuration for the MongoDB connection pool.
type ConnectionConfig struct {
	// URI is the MongoDB connection string.
	// Format: "mongodb://[username:password@]host1[:port1][,...hostN[:portN]][/[database][?options]]"
	URI string

	// Database is the name of the database to use.
	Database string

	// MaxPoolSize is the maximum number of connections in the pool.
	MaxPoolSize uint64

	// MinPoolSize is the minimum number of connections in the pool.
	MinPoolSize uint64

	// MaxConnIdleTime is the maximum amount of time a connection may be idle.
	MaxConnIdleTime time.Duration

	// ConnectTimeout is the timeout for establishing a connection.
	ConnectTimeout time.Duration

	// ServerSelectionTimeout is the timeout for server selection.
	ServerSelectionTimeout time.Duration

	// HeartbeatInterval is the interval between heartbeat checks.
	HeartbeatInterval time.Duration

	// RetryWrites enables retryable writes.
	RetryWrites bool

	// RetryReads enables retryable reads.
	RetryReads bool

	// ReadPreference sets the read preference mode.
	ReadPreference string

	// WriteConcern sets the write concern level.
	WriteConcern string
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

// DefaultConnectionConfig returns a sensible default configuration for MongoDB.
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		URI:                    "mongodb://localhost:27017",
		Database:               "yunt",
		MaxPoolSize:            100,
		MinPoolSize:            5,
		MaxConnIdleTime:        30 * time.Minute,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 30 * time.Second,
		HeartbeatInterval:      10 * time.Second,
		RetryWrites:            true,
		RetryReads:             true,
		ReadPreference:         "primary",
		WriteConcern:           "majority",
	}
}

// NewConnectionConfig creates a ConnectionConfig from the application config.
func NewConnectionConfig(cfg *config.DatabaseConfig) *ConnectionConfig {
	uri := cfg.DSN
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	database := cfg.Name
	if database == "" {
		database = "yunt"
	}

	maxPoolSize := uint64(cfg.MaxOpenConns)
	if maxPoolSize == 0 {
		maxPoolSize = 100
	}

	minPoolSize := uint64(cfg.MaxIdleConns)
	if minPoolSize == 0 {
		minPoolSize = 5
	}

	maxConnIdleTime := cfg.ConnMaxIdleTime
	if maxConnIdleTime == 0 {
		maxConnIdleTime = 30 * time.Minute
	}

	return &ConnectionConfig{
		URI:                    uri,
		Database:               database,
		MaxPoolSize:            maxPoolSize,
		MinPoolSize:            minPoolSize,
		MaxConnIdleTime:        maxConnIdleTime,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 30 * time.Second,
		HeartbeatInterval:      10 * time.Second,
		RetryWrites:            true,
		RetryReads:             true,
		ReadPreference:         "primary",
		WriteConcern:           "majority",
	}
}

// NewConnectionPool creates a new MongoDB connection pool.
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
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectTimeout)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(p.config.URI).
		SetMaxPoolSize(p.config.MaxPoolSize).
		SetMinPoolSize(p.config.MinPoolSize).
		SetMaxConnIdleTime(p.config.MaxConnIdleTime).
		SetConnectTimeout(p.config.ConnectTimeout).
		SetServerSelectionTimeout(p.config.ServerSelectionTimeout).
		SetHeartbeatInterval(p.config.HeartbeatInterval).
		SetRetryWrites(p.config.RetryWrites).
		SetRetryReads(p.config.RetryReads)

	// Set read preference
	switch p.config.ReadPreference {
	case "primary":
		clientOpts.SetReadPreference(readpref.Primary())
	case "primaryPreferred":
		clientOpts.SetReadPreference(readpref.PrimaryPreferred())
	case "secondary":
		clientOpts.SetReadPreference(readpref.Secondary())
	case "secondaryPreferred":
		clientOpts.SetReadPreference(readpref.SecondaryPreferred())
	case "nearest":
		clientOpts.SetReadPreference(readpref.Nearest())
	default:
		clientOpts.SetReadPreference(readpref.Primary())
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		client.Disconnect(ctx)
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	p.mu.Lock()
	p.client = client
	p.db = client.Database(p.config.Database)
	p.metrics.totalOpened++
	p.metrics.currentOpen = 1
	p.mu.Unlock()

	return nil
}

// Client returns the underlying MongoDB client.
func (p *ConnectionPool) Client() *mongo.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.client
}

// Database returns the configured database.
func (p *ConnectionPool) Database() *mongo.Database {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db
}

// Collection returns a collection by name.
func (p *ConnectionPool) Collection(name string) *mongo.Collection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.db == nil {
		return nil
	}
	return p.db.Collection(name)
}

// Close closes all connections in the pool.
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := p.client.Disconnect(ctx)
	p.client = nil
	p.db = nil
	p.metrics.totalClosed++
	p.metrics.currentOpen = 0
	p.metrics.currentInUse = 0

	return err
}

// Health checks the health of the database connection.
func (p *ConnectionPool) Health(ctx context.Context) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("database connection is closed")
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		p.recordError(err)
		return fmt.Errorf("database ping failed: %w", err)
	}

	p.recordHealthCheck()
	return nil
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

// StartSession starts a new session for transaction support.
func (p *ConnectionPool) StartSession(opts ...*options.SessionOptions) (mongo.Session, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	return client.StartSession(opts...)
}

// Reconnect closes the existing connection and establishes a new one.
func (p *ConnectionPool) Reconnect() error {
	p.mu.Lock()
	if p.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		p.client.Disconnect(ctx)
		cancel()
		p.client = nil
		p.db = nil
		p.metrics.totalClosed++
	}
	p.mu.Unlock()

	return p.connect()
}

// Version returns the MongoDB server version.
func (p *ConnectionPool) Version(ctx context.Context) (string, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return "", fmt.Errorf("database connection is closed")
	}

	var result bson.M
	err := db.RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("failed to get MongoDB version: %w", err)
	}

	if version, ok := result["version"].(string); ok {
		return version, nil
	}

	return "unknown", nil
}

// EnsureIndexes creates all required indexes for the collections.
func (p *ConnectionPool) EnsureIndexes(ctx context.Context) error {
	// Users collection indexes
	if err := p.createIndexes(ctx, CollectionUsers, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true).SetCollation(&options.Collation{Locale: "en", Strength: 2}),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetCollation(&options.Collation{Locale: "en", Strength: 2}),
		},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "role", Value: 1}}},
		{Keys: bson.D{{Key: "deletedAt", Value: 1}}},
		{
			Keys: bson.D{
				{Key: "username", Value: "text"},
				{Key: "email", Value: "text"},
				{Key: "displayName", Value: "text"},
			},
			Options: options.Index().SetName("users_text_search"),
		},
	}); err != nil {
		return fmt.Errorf("failed to create users indexes: %w", err)
	}

	// Mailboxes collection indexes
	if err := p.createIndexes(ctx, CollectionMailboxes, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		{
			Keys:    bson.D{{Key: "address", Value: 1}},
			Options: options.Index().SetUnique(true).SetCollation(&options.Collation{Locale: "en", Strength: 2}),
		},
		{Keys: bson.D{{Key: "isCatchAll", Value: 1}}},
		{Keys: bson.D{{Key: "isDefault", Value: 1}}},
		{
			Keys: bson.D{
				{Key: "name", Value: "text"},
				{Key: "address", Value: "text"},
				{Key: "description", Value: "text"},
			},
			Options: options.Index().SetName("mailboxes_text_search"),
		},
	}); err != nil {
		return fmt.Errorf("failed to create mailboxes indexes: %w", err)
	}

	// Messages collection indexes
	if err := p.createIndexes(ctx, CollectionMessages, []mongo.IndexModel{
		{Keys: bson.D{{Key: "mailboxId", Value: 1}}},
		{Keys: bson.D{{Key: "messageId", Value: 1}}},
		{Keys: bson.D{{Key: "from.address", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "isStarred", Value: 1}}},
		{Keys: bson.D{{Key: "isSpam", Value: 1}}},
		{Keys: bson.D{{Key: "receivedAt", Value: -1}}},
		{Keys: bson.D{{Key: "mailboxId", Value: 1}, {Key: "receivedAt", Value: -1}}},
		{
			Keys: bson.D{
				{Key: "subject", Value: "text"},
				{Key: "textBody", Value: "text"},
				{Key: "htmlBody", Value: "text"},
				{Key: "from.address", Value: "text"},
				{Key: "from.name", Value: "text"},
			},
			Options: options.Index().SetName("messages_text_search"),
		},
	}); err != nil {
		return fmt.Errorf("failed to create messages indexes: %w", err)
	}

	// Message recipients collection indexes
	if err := p.createIndexes(ctx, CollectionMessageRecipients, []mongo.IndexModel{
		{Keys: bson.D{{Key: "messageId", Value: 1}}},
		{Keys: bson.D{{Key: "address", Value: 1}}},
	}); err != nil {
		return fmt.Errorf("failed to create message_recipients indexes: %w", err)
	}

	// Attachments collection indexes
	if err := p.createIndexes(ctx, CollectionAttachments, []mongo.IndexModel{
		{Keys: bson.D{{Key: "messageId", Value: 1}}},
		{Keys: bson.D{{Key: "contentId", Value: 1}}},
		{Keys: bson.D{{Key: "contentType", Value: 1}}},
		{Keys: bson.D{{Key: "checksum", Value: 1}}},
		{
			Keys:    bson.D{{Key: "filename", Value: "text"}},
			Options: options.Index().SetName("attachments_text_search"),
		},
	}); err != nil {
		return fmt.Errorf("failed to create attachments indexes: %w", err)
	}

	// Webhooks collection indexes
	if err := p.createIndexes(ctx, CollectionWebhooks, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "events", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "url", Value: 1}}},
		{
			Keys:    bson.D{{Key: "name", Value: "text"}, {Key: "url", Value: "text"}},
			Options: options.Index().SetName("webhooks_text_search"),
		},
	}); err != nil {
		return fmt.Errorf("failed to create webhooks indexes: %w", err)
	}

	// Webhook deliveries collection indexes
	if err := p.createIndexes(ctx, CollectionWebhookDeliveries, []mongo.IndexModel{
		{Keys: bson.D{{Key: "webhookId", Value: 1}}},
		{Keys: bson.D{{Key: "event", Value: 1}}},
		{Keys: bson.D{{Key: "success", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "webhookId", Value: 1}, {Key: "createdAt", Value: -1}}},
	}); err != nil {
		return fmt.Errorf("failed to create webhook_deliveries indexes: %w", err)
	}

	// Settings history collection indexes
	if err := p.createIndexes(ctx, CollectionSettingsHistory, []mongo.IndexModel{
		{Keys: bson.D{{Key: "fieldPath", Value: 1}}},
		{Keys: bson.D{{Key: "changedAt", Value: -1}}},
	}); err != nil {
		return fmt.Errorf("failed to create settings_history indexes: %w", err)
	}

	return nil
}

// createIndexes creates indexes for a collection.
func (p *ConnectionPool) createIndexes(ctx context.Context, collectionName string, indexes []mongo.IndexModel) error {
	collection := p.Collection(collectionName)
	if collection == nil {
		return fmt.Errorf("collection not found: %s", collectionName)
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// DropCollection drops a collection (use with caution).
func (p *ConnectionPool) DropCollection(ctx context.Context, name string) error {
	collection := p.Collection(name)
	if collection == nil {
		return fmt.Errorf("collection not found: %s", name)
	}
	return collection.Drop(ctx)
}

// ListCollections returns a list of all collection names.
func (p *ConnectionPool) ListCollections(ctx context.Context) ([]string, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	names, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	return names, nil
}

// CollectionStats returns statistics for a collection.
func (p *ConnectionPool) CollectionStats(ctx context.Context, name string) (map[string]interface{}, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	var result bson.M
	err := db.RunCommand(ctx, bson.D{{Key: "collStats", Value: name}}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	return result, nil
}
