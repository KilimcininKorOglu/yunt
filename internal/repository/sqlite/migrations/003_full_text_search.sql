-- +migrate Up
-- SQLite FTS5 full-text search for email messages

-- Create FTS5 virtual table for message search
-- Uses content-sync (external content) to avoid data duplication
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    subject,
    text_body,
    from_address,
    from_name,
    content=messages,
    content_rowid=rowid
);

-- Populate FTS index from existing messages
INSERT INTO messages_fts(rowid, subject, text_body, from_address, from_name)
SELECT rowid, COALESCE(subject, ''), COALESCE(text_body, ''), COALESCE(from_address, ''), COALESCE(from_name, '')
FROM messages;

-- Trigger to keep FTS index in sync on INSERT
CREATE TRIGGER IF NOT EXISTS messages_fts_insert AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, subject, text_body, from_address, from_name)
    VALUES (NEW.rowid, COALESCE(NEW.subject, ''), COALESCE(NEW.text_body, ''), COALESCE(NEW.from_address, ''), COALESCE(NEW.from_name, ''));
END;

-- Trigger to keep FTS index in sync on UPDATE
CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, subject, text_body, from_address, from_name)
    VALUES ('delete', OLD.rowid, COALESCE(OLD.subject, ''), COALESCE(OLD.text_body, ''), COALESCE(OLD.from_address, ''), COALESCE(OLD.from_name, ''));
    INSERT INTO messages_fts(rowid, subject, text_body, from_address, from_name)
    VALUES (NEW.rowid, COALESCE(NEW.subject, ''), COALESCE(NEW.text_body, ''), COALESCE(NEW.from_address, ''), COALESCE(NEW.from_name, ''));
END;

-- Trigger to keep FTS index in sync on DELETE
CREATE TRIGGER IF NOT EXISTS messages_fts_delete AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, subject, text_body, from_address, from_name)
    VALUES ('delete', OLD.rowid, COALESCE(OLD.subject, ''), COALESCE(OLD.text_body, ''), COALESCE(OLD.from_address, ''), COALESCE(OLD.from_name, ''));
END;

-- +migrate Down
-- Rollback: Remove FTS5 full-text search

DROP TRIGGER IF EXISTS messages_fts_delete;
DROP TRIGGER IF EXISTS messages_fts_update;
DROP TRIGGER IF EXISTS messages_fts_insert;
DROP TABLE IF EXISTS messages_fts;
