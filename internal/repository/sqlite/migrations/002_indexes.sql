-- +migrate Up
-- Create indexes for improved query performance

-- User indexes
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users(last_login_at);

-- Mailbox indexes
CREATE INDEX IF NOT EXISTS idx_mailboxes_user_id ON mailboxes(user_id);
CREATE INDEX IF NOT EXISTS idx_mailboxes_address ON mailboxes(address);
CREATE INDEX IF NOT EXISTS idx_mailboxes_is_catch_all ON mailboxes(is_catch_all);
CREATE INDEX IF NOT EXISTS idx_mailboxes_is_default ON mailboxes(is_default);
CREATE INDEX IF NOT EXISTS idx_mailboxes_created_at ON mailboxes(created_at);

-- Message indexes
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_id ON messages(mailbox_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_from_address ON messages(from_address);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_is_starred ON messages(is_starred);
CREATE INDEX IF NOT EXISTS idx_messages_is_spam ON messages(is_spam);
CREATE INDEX IF NOT EXISTS idx_messages_received_at ON messages(received_at);
CREATE INDEX IF NOT EXISTS idx_messages_subject ON messages(subject);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_status ON messages(mailbox_id, status);
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_received ON messages(mailbox_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_mailbox_starred ON messages(mailbox_id, is_starred);

-- Message recipient indexes
CREATE INDEX IF NOT EXISTS idx_message_recipients_message_id ON message_recipients(message_id);
CREATE INDEX IF NOT EXISTS idx_message_recipients_address ON message_recipients(address);
CREATE INDEX IF NOT EXISTS idx_message_recipients_type ON message_recipients(recipient_type);

-- Attachment indexes
CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);
CREATE INDEX IF NOT EXISTS idx_attachments_content_id ON attachments(content_id);
CREATE INDEX IF NOT EXISTS idx_attachments_content_type ON attachments(content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_checksum ON attachments(checksum);

-- Webhook indexes
CREATE INDEX IF NOT EXISTS idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_status ON webhooks(status);
CREATE INDEX IF NOT EXISTS idx_webhooks_url ON webhooks(url);
CREATE INDEX IF NOT EXISTS idx_webhooks_created_at ON webhooks(created_at);

-- Webhook delivery indexes
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event ON webhook_deliveries(event);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_success ON webhook_deliveries(success);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created_at ON webhook_deliveries(created_at);

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
DROP INDEX IF EXISTS idx_webhook_deliveries_created_at;
DROP INDEX IF EXISTS idx_webhook_deliveries_success;
DROP INDEX IF EXISTS idx_webhook_deliveries_event;
DROP INDEX IF EXISTS idx_webhook_deliveries_webhook_id;

-- Webhook indexes
DROP INDEX IF EXISTS idx_webhooks_created_at;
DROP INDEX IF EXISTS idx_webhooks_url;
DROP INDEX IF EXISTS idx_webhooks_status;
DROP INDEX IF EXISTS idx_webhooks_user_id;

-- Attachment indexes
DROP INDEX IF EXISTS idx_attachments_checksum;
DROP INDEX IF EXISTS idx_attachments_content_type;
DROP INDEX IF EXISTS idx_attachments_content_id;
DROP INDEX IF EXISTS idx_attachments_message_id;

-- Message recipient indexes
DROP INDEX IF EXISTS idx_message_recipients_type;
DROP INDEX IF EXISTS idx_message_recipients_address;
DROP INDEX IF EXISTS idx_message_recipients_message_id;

-- Message composite indexes
DROP INDEX IF EXISTS idx_messages_mailbox_starred;
DROP INDEX IF EXISTS idx_messages_mailbox_received;
DROP INDEX IF EXISTS idx_messages_mailbox_status;

-- Message indexes
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_subject;
DROP INDEX IF EXISTS idx_messages_received_at;
DROP INDEX IF EXISTS idx_messages_is_spam;
DROP INDEX IF EXISTS idx_messages_is_starred;
DROP INDEX IF EXISTS idx_messages_status;
DROP INDEX IF EXISTS idx_messages_from_address;
DROP INDEX IF EXISTS idx_messages_message_id;
DROP INDEX IF EXISTS idx_messages_mailbox_id;

-- Mailbox indexes
DROP INDEX IF EXISTS idx_mailboxes_created_at;
DROP INDEX IF EXISTS idx_mailboxes_is_default;
DROP INDEX IF EXISTS idx_mailboxes_is_catch_all;
DROP INDEX IF EXISTS idx_mailboxes_address;
DROP INDEX IF EXISTS idx_mailboxes_user_id;

-- User indexes
DROP INDEX IF EXISTS idx_users_last_login_at;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;
