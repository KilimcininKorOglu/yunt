-- +migrate Up
ALTER TABLE messages ADD COLUMN is_deleted TINYINT(1) NOT NULL DEFAULT 0;
CREATE INDEX idx_messages_is_deleted ON messages(is_deleted);

-- +migrate Down
DROP INDEX idx_messages_is_deleted ON messages;
ALTER TABLE messages DROP COLUMN is_deleted;
