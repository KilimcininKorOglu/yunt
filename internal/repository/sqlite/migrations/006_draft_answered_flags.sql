-- +migrate Up
ALTER TABLE messages ADD COLUMN is_draft INTEGER NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN is_answered INTEGER NOT NULL DEFAULT 0;

-- +migrate Down
ALTER TABLE messages DROP COLUMN is_answered;
ALTER TABLE messages DROP COLUMN is_draft;
