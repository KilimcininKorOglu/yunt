-- +migrate Up
ALTER TABLE messages ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_messages_is_deleted ON messages(is_deleted);

-- +migrate Down
DROP INDEX IF EXISTS idx_messages_is_deleted;
ALTER TABLE messages DROP COLUMN is_deleted;
