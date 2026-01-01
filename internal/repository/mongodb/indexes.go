package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// IndexManager handles MongoDB index creation and management.
type IndexManager struct {
	pool   *ConnectionPool
	logger *slog.Logger
}

// IndexDefinition represents a MongoDB index definition.
type IndexDefinition struct {
	Collection string
	Index      mongo.IndexModel
}

// NewIndexManager creates a new IndexManager with the given connection pool.
func NewIndexManager(pool *ConnectionPool, logger *slog.Logger) *IndexManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &IndexManager{
		pool:   pool,
		logger: logger,
	}
}

// EnsureAllIndexes creates all required indexes for all collections.
// This method is idempotent - existing indexes will not be recreated.
func (im *IndexManager) EnsureAllIndexes(ctx context.Context) error {
	im.logger.Info("starting index creation for all collections")

	collections := []struct {
		name    string
		indexes []mongo.IndexModel
	}{
		{CollectionUsers, im.getUserIndexes()},
		{CollectionMailboxes, im.getMailboxIndexes()},
		{CollectionMessages, im.getMessageIndexes()},
		{CollectionMessageRecipients, im.getMessageRecipientIndexes()},
		{CollectionAttachments, im.getAttachmentIndexes()},
		{CollectionWebhooks, im.getWebhookIndexes()},
		{CollectionWebhookDeliveries, im.getWebhookDeliveryIndexes()},
		{CollectionSettings, im.getSettingsIndexes()},
		{CollectionSettingsHistory, im.getSettingsHistoryIndexes()},
	}

	var totalCreated int
	for _, col := range collections {
		created, err := im.createIndexes(ctx, col.name, col.indexes)
		if err != nil {
			return fmt.Errorf("failed to create indexes for %s: %w", col.name, err)
		}
		totalCreated += created
	}

	im.logger.Info("index creation completed",
		"totalIndexes", totalCreated)

	return nil
}

// createIndexes creates indexes for a single collection and returns the count of created indexes.
func (im *IndexManager) createIndexes(ctx context.Context, collectionName string, indexes []mongo.IndexModel) (int, error) {
	collection := im.pool.Collection(collectionName)
	if collection == nil {
		return 0, fmt.Errorf("collection not found: %s", collectionName)
	}

	if len(indexes) == 0 {
		return 0, nil
	}

	im.logger.Debug("creating indexes",
		"collection", collectionName,
		"indexCount", len(indexes))

	names, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		im.logger.Error("failed to create indexes",
			"collection", collectionName,
			"error", err)
		return 0, err
	}

	im.logger.Info("indexes created successfully",
		"collection", collectionName,
		"indexes", names)

	return len(names), nil
}

// getUserIndexes returns index definitions for the users collection.
func (im *IndexManager) getUserIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		// Unique index on username (case-insensitive)
		{
			Keys: bson.D{{Key: "username", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetCollation(&options.Collation{Locale: "en", Strength: 2}).
				SetName("idx_users_username_unique"),
		},
		// Unique index on email (case-insensitive)
		{
			Keys: bson.D{{Key: "email", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetCollation(&options.Collation{Locale: "en", Strength: 2}).
				SetName("idx_users_email_unique"),
		},
		// Index on status for filtering active/inactive users
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_users_status"),
		},
		// Index on role for filtering by user role
		{
			Keys:    bson.D{{Key: "role", Value: 1}},
			Options: options.Index().SetName("idx_users_role"),
		},
		// Index on deletedAt for soft delete queries
		{
			Keys:    bson.D{{Key: "deletedAt", Value: 1}},
			Options: options.Index().SetName("idx_users_deleted_at"),
		},
		// Compound index for role and status filtering
		{
			Keys:    bson.D{{Key: "role", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_users_role_status"),
		},
		// Text index for search on username, email, and displayName
		{
			Keys: bson.D{
				{Key: "username", Value: "text"},
				{Key: "email", Value: "text"},
				{Key: "displayName", Value: "text"},
			},
			Options: options.Index().SetName("idx_users_text_search"),
		},
	}
}

// getMailboxIndexes returns index definitions for the mailboxes collection.
func (im *IndexManager) getMailboxIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		// Index on userId for user's mailboxes lookup
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("idx_mailboxes_user_id"),
		},
		// Unique index on address (case-insensitive)
		{
			Keys: bson.D{{Key: "address", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetCollation(&options.Collation{Locale: "en", Strength: 2}).
				SetName("idx_mailboxes_address_unique"),
		},
		// Index on isCatchAll for finding catch-all mailboxes
		{
			Keys:    bson.D{{Key: "isCatchAll", Value: 1}},
			Options: options.Index().SetName("idx_mailboxes_catch_all"),
		},
		// Index on isDefault for finding default mailboxes
		{
			Keys:    bson.D{{Key: "isDefault", Value: 1}},
			Options: options.Index().SetName("idx_mailboxes_default"),
		},
		// Compound index for user's default mailbox lookup
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "isDefault", Value: 1}},
			Options: options.Index().SetName("idx_mailboxes_user_default"),
		},
		// Compound index for mailboxes with unread messages
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "unreadCount", Value: 1}},
			Options: options.Index().SetName("idx_mailboxes_user_unread"),
		},
		// Text index for search on name, address, and description
		{
			Keys: bson.D{
				{Key: "name", Value: "text"},
				{Key: "address", Value: "text"},
				{Key: "description", Value: "text"},
			},
			Options: options.Index().SetName("idx_mailboxes_text_search"),
		},
	}
}

// getMessageIndexes returns index definitions for the messages collection.
func (im *IndexManager) getMessageIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		// Index on mailboxId for listing messages in a mailbox
		{
			Keys:    bson.D{{Key: "mailboxId", Value: 1}},
			Options: options.Index().SetName("idx_messages_mailbox_id"),
		},
		// Index on messageId for looking up by SMTP message ID
		{
			Keys:    bson.D{{Key: "messageId", Value: 1}},
			Options: options.Index().SetName("idx_messages_message_id"),
		},
		// Index on from.address for sender lookups
		{
			Keys:    bson.D{{Key: "from.address", Value: 1}},
			Options: options.Index().SetName("idx_messages_from_address"),
		},
		// Index on status for filtering read/unread messages
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_messages_status"),
		},
		// Index on isStarred for filtering starred messages
		{
			Keys:    bson.D{{Key: "isStarred", Value: 1}},
			Options: options.Index().SetName("idx_messages_starred"),
		},
		// Index on isSpam for spam filtering
		{
			Keys:    bson.D{{Key: "isSpam", Value: 1}},
			Options: options.Index().SetName("idx_messages_spam"),
		},
		// Index on receivedAt for sorting by date (descending for newest first)
		{
			Keys:    bson.D{{Key: "receivedAt", Value: -1}},
			Options: options.Index().SetName("idx_messages_received_at"),
		},
		// Compound index for listing messages in a mailbox sorted by date
		{
			Keys:    bson.D{{Key: "mailboxId", Value: 1}, {Key: "receivedAt", Value: -1}},
			Options: options.Index().SetName("idx_messages_mailbox_received"),
		},
		// Compound index for unread messages in a mailbox
		{
			Keys:    bson.D{{Key: "mailboxId", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_messages_mailbox_status"),
		},
		// Compound index for starred messages in a mailbox
		{
			Keys:    bson.D{{Key: "mailboxId", Value: 1}, {Key: "isStarred", Value: 1}},
			Options: options.Index().SetName("idx_messages_mailbox_starred"),
		},
		// Text index for full-text search on message content
		{
			Keys: bson.D{
				{Key: "subject", Value: "text"},
				{Key: "textBody", Value: "text"},
				{Key: "htmlBody", Value: "text"},
				{Key: "from.address", Value: "text"},
				{Key: "from.name", Value: "text"},
			},
			Options: options.Index().SetName("idx_messages_text_search"),
		},
	}
}

// getMessageRecipientIndexes returns index definitions for the message_recipients collection.
func (im *IndexManager) getMessageRecipientIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		// Index on messageId for looking up recipients by message
		{
			Keys:    bson.D{{Key: "messageId", Value: 1}},
			Options: options.Index().SetName("idx_recipients_message_id"),
		},
		// Index on address for looking up messages by recipient
		{
			Keys:    bson.D{{Key: "address", Value: 1}},
			Options: options.Index().SetName("idx_recipients_address"),
		},
		// Compound index for message and type (to, cc, bcc)
		{
			Keys:    bson.D{{Key: "messageId", Value: 1}, {Key: "type", Value: 1}},
			Options: options.Index().SetName("idx_recipients_message_type"),
		},
	}
}

// getAttachmentIndexes returns index definitions for the attachments collection.
func (im *IndexManager) getAttachmentIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		// Index on messageId for looking up attachments by message
		{
			Keys:    bson.D{{Key: "messageId", Value: 1}},
			Options: options.Index().SetName("idx_attachments_message_id"),
		},
		// Index on contentId for inline attachments
		{
			Keys:    bson.D{{Key: "contentId", Value: 1}},
			Options: options.Index().SetName("idx_attachments_content_id"),
		},
		// Index on contentType for filtering by type
		{
			Keys:    bson.D{{Key: "contentType", Value: 1}},
			Options: options.Index().SetName("idx_attachments_content_type"),
		},
		// Index on checksum for deduplication
		{
			Keys:    bson.D{{Key: "checksum", Value: 1}},
			Options: options.Index().SetName("idx_attachments_checksum"),
		},
		// Compound index for deduplication lookup
		{
			Keys:    bson.D{{Key: "checksum", Value: 1}, {Key: "size", Value: 1}},
			Options: options.Index().SetName("idx_attachments_checksum_size"),
		},
		// Text index for filename search
		{
			Keys:    bson.D{{Key: "filename", Value: "text"}},
			Options: options.Index().SetName("idx_attachments_text_search"),
		},
	}
}

// getWebhookIndexes returns index definitions for the webhooks collection.
func (im *IndexManager) getWebhookIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		// Index on userId for user's webhooks lookup
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("idx_webhooks_user_id"),
		},
		// Index on status for filtering active webhooks
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_webhooks_status"),
		},
		// Index on events for finding webhooks by event type
		{
			Keys:    bson.D{{Key: "events", Value: 1}},
			Options: options.Index().SetName("idx_webhooks_events"),
		},
		// Compound index for unique webhook URL per user
		{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "url", Value: 1}},
			Options: options.Index().SetName("idx_webhooks_user_url"),
		},
		// Compound index for active webhooks by event
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "events", Value: 1}},
			Options: options.Index().SetName("idx_webhooks_status_events"),
		},
		// Text index for search on name and url
		{
			Keys:    bson.D{{Key: "name", Value: "text"}, {Key: "url", Value: "text"}},
			Options: options.Index().SetName("idx_webhooks_text_search"),
		},
	}
}

// getWebhookDeliveryIndexes returns index definitions for the webhook_deliveries collection.
func (im *IndexManager) getWebhookDeliveryIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		// Index on webhookId for delivery history lookup
		{
			Keys:    bson.D{{Key: "webhookId", Value: 1}},
			Options: options.Index().SetName("idx_deliveries_webhook_id"),
		},
		// Index on event type
		{
			Keys:    bson.D{{Key: "event", Value: 1}},
			Options: options.Index().SetName("idx_deliveries_event"),
		},
		// Index on success for filtering failed deliveries
		{
			Keys:    bson.D{{Key: "success", Value: 1}},
			Options: options.Index().SetName("idx_deliveries_success"),
		},
		// Index on createdAt for time-based queries (descending for recent first)
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_deliveries_created_at"),
		},
		// Compound index for webhook delivery history sorted by date
		{
			Keys:    bson.D{{Key: "webhookId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_deliveries_webhook_created"),
		},
		// Compound index for failed deliveries by webhook
		{
			Keys:    bson.D{{Key: "webhookId", Value: 1}, {Key: "success", Value: 1}},
			Options: options.Index().SetName("idx_deliveries_webhook_success"),
		},
	}
}

// getSettingsIndexes returns index definitions for the settings collection.
func (im *IndexManager) getSettingsIndexes() []mongo.IndexModel {
	// Settings collection typically has a single document, minimal indexing needed
	return []mongo.IndexModel{
		// Index on updatedAt for change tracking
		{
			Keys:    bson.D{{Key: "updatedAt", Value: -1}},
			Options: options.Index().SetName("idx_settings_updated_at"),
		},
	}
}

// getSettingsHistoryIndexes returns index definitions for the settings_history collection.
func (im *IndexManager) getSettingsHistoryIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		// Index on fieldPath for looking up changes by field
		{
			Keys:    bson.D{{Key: "fieldPath", Value: 1}},
			Options: options.Index().SetName("idx_settings_history_field"),
		},
		// Index on changedAt for time-based queries (descending for recent first)
		{
			Keys:    bson.D{{Key: "changedAt", Value: -1}},
			Options: options.Index().SetName("idx_settings_history_changed_at"),
		},
		// Compound index for field history sorted by date
		{
			Keys:    bson.D{{Key: "fieldPath", Value: 1}, {Key: "changedAt", Value: -1}},
			Options: options.Index().SetName("idx_settings_history_field_changed"),
		},
	}
}

// DropAllIndexes drops all non-_id indexes from all collections.
// Use with caution - this should only be used in development/testing.
func (im *IndexManager) DropAllIndexes(ctx context.Context) error {
	im.logger.Warn("dropping all indexes from collections")

	collections := []string{
		CollectionUsers,
		CollectionMailboxes,
		CollectionMessages,
		CollectionMessageRecipients,
		CollectionAttachments,
		CollectionWebhooks,
		CollectionWebhookDeliveries,
		CollectionSettings,
		CollectionSettingsHistory,
	}

	for _, collName := range collections {
		collection := im.pool.Collection(collName)
		if collection == nil {
			continue
		}

		if _, err := collection.Indexes().DropAll(ctx); err != nil {
			im.logger.Error("failed to drop indexes",
				"collection", collName,
				"error", err)
			return fmt.Errorf("failed to drop indexes for %s: %w", collName, err)
		}

		im.logger.Info("dropped all indexes",
			"collection", collName)
	}

	return nil
}

// ListIndexes returns a list of all indexes for a collection.
func (im *IndexManager) ListIndexes(ctx context.Context, collectionName string) ([]bson.M, error) {
	collection := im.pool.Collection(collectionName)
	if collection == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(ctx)

	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		return nil, fmt.Errorf("failed to decode indexes: %w", err)
	}

	return indexes, nil
}

// GetIndexStats returns statistics about indexes for a collection.
func (im *IndexManager) GetIndexStats(ctx context.Context, collectionName string) ([]IndexStats, error) {
	indexes, err := im.ListIndexes(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	stats := make([]IndexStats, 0, len(indexes))
	for _, idx := range indexes {
		stat := IndexStats{
			Name:       idx["name"].(string),
			Collection: collectionName,
		}

		if keys, ok := idx["key"].(bson.M); ok {
			stat.Keys = keys
		}

		if unique, ok := idx["unique"].(bool); ok {
			stat.Unique = unique
		}

		if sparse, ok := idx["sparse"].(bool); ok {
			stat.Sparse = sparse
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

// IndexStats contains statistics about a MongoDB index.
type IndexStats struct {
	Name       string `json:"name"`
	Collection string `json:"collection"`
	Keys       bson.M `json:"keys"`
	Unique     bool   `json:"unique"`
	Sparse     bool   `json:"sparse"`
}
