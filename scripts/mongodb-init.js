// =============================================================================
// MongoDB Initialization Script for Yunt Mail Server
// =============================================================================
// This script runs automatically when the MongoDB container starts for the
// first time. It creates the database, user, and initial indexes.
//
// Note: The full index creation is handled by Yunt's auto-migrate feature.
// This script only creates essential indexes for faster startup.
// =============================================================================

// Switch to the yunt database
db = db.getSiblingDB('yunt');

// Create application user (if not using root credentials)
// The user inherits from MONGO_INITDB_ROOT_USERNAME/PASSWORD via env_file
// This is optional if using the root user for the application

print('Initializing Yunt database...');

// Create collections with validation
db.createCollection('users', {
    validator: {
        $jsonSchema: {
            bsonType: 'object',
            required: ['username', 'email', 'passwordHash', 'role', 'status', 'createdAt'],
            properties: {
                username: { bsonType: 'string', description: 'Username is required' },
                email: { bsonType: 'string', description: 'Email is required' },
                passwordHash: { bsonType: 'string', description: 'Password hash is required' },
                role: { bsonType: 'string', enum: ['admin', 'user'], description: 'Role must be admin or user' },
                status: { bsonType: 'string', enum: ['active', 'inactive', 'suspended'], description: 'Status is required' }
            }
        }
    }
});

db.createCollection('mailboxes');
db.createCollection('messages');
db.createCollection('message_recipients');
db.createCollection('attachments');
db.createCollection('attachment_content');
db.createCollection('webhooks');
db.createCollection('webhook_deliveries');
db.createCollection('settings');
db.createCollection('settings_history');

print('Collections created successfully');

// Create essential indexes for basic functionality
// Full index creation is handled by Yunt's IndexManager on startup

// Users collection - essential indexes
db.users.createIndex(
    { username: 1 },
    { unique: true, collation: { locale: 'en', strength: 2 }, name: 'idx_users_username_unique' }
);
db.users.createIndex(
    { email: 1 },
    { unique: true, collation: { locale: 'en', strength: 2 }, name: 'idx_users_email_unique' }
);
db.users.createIndex({ status: 1 }, { name: 'idx_users_status' });

// Mailboxes collection - essential indexes
db.mailboxes.createIndex({ userId: 1 }, { name: 'idx_mailboxes_user_id' });
db.mailboxes.createIndex(
    { address: 1 },
    { unique: true, collation: { locale: 'en', strength: 2 }, name: 'idx_mailboxes_address_unique' }
);

// Messages collection - essential indexes
db.messages.createIndex({ mailboxId: 1 }, { name: 'idx_messages_mailbox_id' });
db.messages.createIndex({ receivedAt: -1 }, { name: 'idx_messages_received_at' });
db.messages.createIndex(
    { mailboxId: 1, receivedAt: -1 },
    { name: 'idx_messages_mailbox_received' }
);

// Message recipients collection - essential indexes
db.message_recipients.createIndex({ messageId: 1 }, { name: 'idx_recipients_message_id' });

// Attachments collection - essential indexes
db.attachments.createIndex({ messageId: 1 }, { name: 'idx_attachments_message_id' });

// Webhooks collection - essential indexes
db.webhooks.createIndex({ userId: 1 }, { name: 'idx_webhooks_user_id' });
db.webhooks.createIndex({ status: 1 }, { name: 'idx_webhooks_status' });

// Webhook deliveries collection - essential indexes
db.webhook_deliveries.createIndex({ webhookId: 1 }, { name: 'idx_deliveries_webhook_id' });
db.webhook_deliveries.createIndex({ createdAt: -1 }, { name: 'idx_deliveries_created_at' });

print('Essential indexes created successfully');
print('Yunt database initialization complete');
print('Additional indexes will be created by Yunt on startup (autoMigrate)');
