-- +migrate Up
-- Create indexes for improved query performance (MySQL)

-- User indexes
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_last_login_at ON users(last_login_at);

-- Mailbox indexes
CREATE INDEX idx_mailboxes_user_id ON mailboxes(user_id);
CREATE INDEX idx_mailboxes_is_catch_all ON mailboxes(is_catch_all);
CREATE INDEX idx_mailboxes_is_default ON mailboxes(is_default);
CREATE INDEX idx_mailboxes_created_at ON mailboxes(created_at);

-- Message indexes
CREATE INDEX idx_messages_mailbox_id ON messages(mailbox_id);
CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_from_address ON messages(from_address);
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_is_starred ON messages(is_starred);
CREATE INDEX idx_messages_is_spam ON messages(is_spam);
CREATE INDEX idx_messages_received_at ON messages(received_at);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- Composite indexes for common queries
CREATE INDEX idx_messages_mailbox_status ON messages(mailbox_id, status);
CREATE INDEX idx_messages_mailbox_received ON messages(mailbox_id, received_at DESC);
CREATE INDEX idx_messages_mailbox_starred ON messages(mailbox_id, is_starred);

-- Message recipient indexes
CREATE INDEX idx_message_recipients_message_id ON message_recipients(message_id);
CREATE INDEX idx_message_recipients_address ON message_recipients(address);
CREATE INDEX idx_message_recipients_type ON message_recipients(recipient_type);

-- Attachment indexes
CREATE INDEX idx_attachments_message_id ON attachments(message_id);
CREATE INDEX idx_attachments_content_id ON attachments(content_id);
CREATE INDEX idx_attachments_content_type ON attachments(content_type);
CREATE INDEX idx_attachments_checksum ON attachments(checksum);

-- Webhook indexes
CREATE INDEX idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX idx_webhooks_status ON webhooks(status);
CREATE INDEX idx_webhooks_created_at ON webhooks(created_at);

-- Webhook delivery indexes
CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX idx_webhook_deliveries_event ON webhook_deliveries(event);
CREATE INDEX idx_webhook_deliveries_success ON webhook_deliveries(success);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at);

-- Settings history indexes
CREATE INDEX idx_settings_history_field_path ON settings_history(field_path);
CREATE INDEX idx_settings_history_changed_at ON settings_history(changed_at);
CREATE INDEX idx_settings_history_changed_by ON settings_history(changed_by);

-- +migrate Down
-- Rollback: Drop all indexes

-- Settings history indexes
DROP INDEX idx_settings_history_changed_by ON settings_history;
DROP INDEX idx_settings_history_changed_at ON settings_history;
DROP INDEX idx_settings_history_field_path ON settings_history;

-- Webhook delivery indexes
DROP INDEX idx_webhook_deliveries_created_at ON webhook_deliveries;
DROP INDEX idx_webhook_deliveries_success ON webhook_deliveries;
DROP INDEX idx_webhook_deliveries_event ON webhook_deliveries;
DROP INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries;

-- Webhook indexes
DROP INDEX idx_webhooks_created_at ON webhooks;
DROP INDEX idx_webhooks_status ON webhooks;
DROP INDEX idx_webhooks_user_id ON webhooks;

-- Attachment indexes
DROP INDEX idx_attachments_checksum ON attachments;
DROP INDEX idx_attachments_content_type ON attachments;
DROP INDEX idx_attachments_content_id ON attachments;
DROP INDEX idx_attachments_message_id ON attachments;

-- Message recipient indexes
DROP INDEX idx_message_recipients_type ON message_recipients;
DROP INDEX idx_message_recipients_address ON message_recipients;
DROP INDEX idx_message_recipients_message_id ON message_recipients;

-- Message composite indexes
DROP INDEX idx_messages_mailbox_starred ON messages;
DROP INDEX idx_messages_mailbox_received ON messages;
DROP INDEX idx_messages_mailbox_status ON messages;

-- Message indexes
DROP INDEX idx_messages_created_at ON messages;
DROP INDEX idx_messages_received_at ON messages;
DROP INDEX idx_messages_is_spam ON messages;
DROP INDEX idx_messages_is_starred ON messages;
DROP INDEX idx_messages_status ON messages;
DROP INDEX idx_messages_from_address ON messages;
DROP INDEX idx_messages_message_id ON messages;
DROP INDEX idx_messages_mailbox_id ON messages;

-- Mailbox indexes
DROP INDEX idx_mailboxes_created_at ON mailboxes;
DROP INDEX idx_mailboxes_is_default ON mailboxes;
DROP INDEX idx_mailboxes_is_catch_all ON mailboxes;
DROP INDEX idx_mailboxes_user_id ON mailboxes;

-- User indexes
DROP INDEX idx_users_last_login_at ON users;
DROP INDEX idx_users_created_at ON users;
DROP INDEX idx_users_deleted_at ON users;
DROP INDEX idx_users_role ON users;
DROP INDEX idx_users_status ON users;
