-- +migrate Up
-- Create indexes for improved query performance (PostgreSQL)

-- User indexes
CREATE INDEX IF NOT EXISTS idx_users_username ON users(LOWER(username));
CREATE INDEX IF NOT EXISTS idx_users_email ON users(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users(last_login_at);

-- Composite index for active users lookup
CREATE INDEX IF NOT EXISTS idx_users_status_role ON users(status, role) WHERE deleted_at IS NULL;

-- Mailbox indexes
CREATE INDEX IF NOT EXISTS idx_mailboxes_user_id ON mailboxes(user_id);
CREATE INDEX IF NOT EXISTS idx_mailboxes_address ON mailboxes(LOWER(address));
CREATE INDEX IF NOT EXISTS idx_mailboxes_is_catch_all ON mailboxes(is_catch_all);
CREATE INDEX IF NOT EXISTS idx_mailboxes_is_default ON mailboxes(is_default);
CREATE INDEX IF NOT EXISTS idx_mailboxes_created_at ON mailboxes(created_at);

-- Composite index for user's mailboxes
CREATE INDEX IF NOT EXISTS idx_mailboxes_user_default ON mailboxes(user_id, is_default);
CREATE INDEX IF NOT EXISTS idx_mailboxes_user_catch_all ON mailboxes(user_id, is_catch_all);

-- Message indexes
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_id ON messages(mailbox_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_from_address ON messages(LOWER(from_address));
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_is_starred ON messages(is_starred);
CREATE INDEX IF NOT EXISTS idx_messages_is_spam ON messages(is_spam);
CREATE INDEX IF NOT EXISTS idx_messages_received_at ON messages(received_at);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- Composite indexes for common message queries
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_status ON messages(mailbox_id, status);
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_received ON messages(mailbox_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_starred ON messages(mailbox_id, is_starred) WHERE is_starred = TRUE;
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_spam ON messages(mailbox_id, is_spam);

-- Partial index for unread messages (common query pattern)
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_unread ON messages(mailbox_id, received_at DESC) WHERE status = 'unread';

-- Message recipient indexes
CREATE INDEX IF NOT EXISTS idx_message_recipients_message_id ON message_recipients(message_id);
CREATE INDEX IF NOT EXISTS idx_message_recipients_address ON message_recipients(LOWER(address));
CREATE INDEX IF NOT EXISTS idx_message_recipients_type ON message_recipients(recipient_type);

-- Composite index for finding messages by recipient
CREATE INDEX IF NOT EXISTS idx_message_recipients_address_type ON message_recipients(LOWER(address), recipient_type);

-- Attachment indexes
CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);
CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
CREATE INDEX IF NOT EXISTS idx_attachments_content_type ON attachments(content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_checksum ON attachments(checksum);

-- Partial index for inline attachments
CREATE INDEX IF NOT EXISTS idx_attachments_inline ON attachments(message_id) WHERE is_inline = TRUE;

-- Webhook indexes
CREATE INDEX IF NOT EXISTS idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_status ON webhooks(status);
CREATE INDEX IF NOT EXISTS idx_webhooks_url ON webhooks(url);
CREATE INDEX IF NOT EXISTS idx_webhooks_created_at ON webhooks(created_at);

-- Composite index for active webhooks
CREATE INDEX IF NOT EXISTS idx_webhooks_user_active ON webhooks(user_id, status) WHERE status = 'active';

-- Webhook delivery indexes
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event ON webhook_deliveries(event);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_success ON webhook_deliveries(success);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created_at ON webhook_deliveries(created_at);

-- Composite index for webhook delivery history
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_created ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_success ON webhook_deliveries(webhook_id, success);

-- Settings history indexes
CREATE INDEX IF NOT EXISTS idx_settings_history_field_path ON settings_history(field_path);
CREATE INDEX IF NOT EXISTS idx_settings_history_changed_at ON settings_history(changed_at);
CREATE INDEX IF NOT EXISTS idx_settings_history_changed_by ON settings_history(changed_by);

-- Schema migrations index
CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON schema_migrations(applied_at);

-- +migrate Down
-- Rollback: Drop all indexes

-- Schema migrations index
DROP INDEX IF EXISTS idx_schema_migrations_applied_at;

-- Settings history indexes
DROP INDEX IF EXISTS idx_settings_history_changed_by;
DROP INDEX IF EXISTS idx_settings_history_changed_at;
DROP INDEX IF EXISTS idx_settings_history_field_path;

-- Webhook delivery indexes
DROP INDEX IF EXISTS idx_webhook_deliveries_webhook_success;
DROP INDEX IF EXISTS idx_webhook_deliveries_webhook_created;
DROP INDEX IF EXISTS idx_webhook_deliveries_created_at;
DROP INDEX IF EXISTS idx_webhook_deliveries_success;
DROP INDEX IF EXISTS idx_webhook_deliveries_event;
DROP INDEX IF EXISTS idx_webhook_deliveries_webhook_id;

-- Webhook indexes
DROP INDEX IF EXISTS idx_webhooks_user_active;
DROP INDEX IF EXISTS idx_webhooks_created_at;
DROP INDEX IF EXISTS idx_webhooks_url;
DROP INDEX IF EXISTS idx_webhooks_status;
DROP INDEX IF EXISTS idx_webhooks_user_id;

-- Attachment indexes
DROP INDEX IF EXISTS idx_attachments_inline;
DROP INDEX IF EXISTS idx_attachments_checksum;
DROP INDEX IF EXISTS idx_attachments_content_type;
DROP INDEX IF EXISTS idx_attachments_content_id;
DROP INDEX IF EXISTS idx_attachments_message_id;

-- Message recipient indexes
DROP INDEX IF EXISTS idx_message_recipients_address_type;
DROP INDEX IF EXISTS idx_message_recipients_type;
DROP INDEX IF EXISTS idx_message_recipients_address;
DROP INDEX IF EXISTS idx_message_recipients_message_id;

-- Message indexes
DROP INDEX IF EXISTS idx_messages_mailbox_unread;
DROP INDEX IF EXISTS idx_messages_mailbox_spam;
DROP INDEX IF EXISTS idx_messages_mailbox_starred;
DROP INDEX IF EXISTS idx_messages_mailbox_received;
DROP INDEX IF EXISTS idx_messages_mailbox_status;
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_received_at;
DROP INDEX IF EXISTS idx_messages_is_spam;
DROP INDEX IF EXISTS idx_messages_is_starred;
DROP INDEX IF EXISTS idx_messages_status;
DROP INDEX IF EXISTS idx_messages_from_address;
DROP INDEX IF EXISTS idx_messages_message_id;
DROP INDEX IF EXISTS idx_messages_mailbox_id;

-- Mailbox indexes
DROP INDEX IF EXISTS idx_mailboxes_user_catch_all;
DROP INDEX IF EXISTS idx_mailboxes_user_default;
DROP INDEX IF EXISTS idx_mailboxes_created_at;
DROP INDEX IF EXISTS idx_mailboxes_is_default;
DROP INDEX IF EXISTS idx_mailboxes_is_catch_all;
DROP INDEX IF EXISTS idx_mailboxes_address;
DROP INDEX IF EXISTS idx_mailboxes_user_id;

-- User indexes
DROP INDEX IF EXISTS idx_users_status_role;
DROP INDEX IF EXISTS idx_users_last_login_at;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;
