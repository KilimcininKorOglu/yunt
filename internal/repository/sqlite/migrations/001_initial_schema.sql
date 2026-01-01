-- +migrate Up
-- Initial database schema for Yunt Mail Server

-- Schema migrations tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL COLLATE NOCASE,
    email TEXT UNIQUE NOT NULL COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    display_name TEXT,
    role TEXT NOT NULL DEFAULT 'user',
    status TEXT NOT NULL DEFAULT 'pending',
    avatar_url TEXT,
    last_login_at DATETIME,
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Mailboxes table
CREATE TABLE IF NOT EXISTS mailboxes (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    address TEXT UNIQUE NOT NULL COLLATE NOCASE,
    description TEXT,
    is_catch_all INTEGER NOT NULL DEFAULT 0,
    is_default INTEGER NOT NULL DEFAULT 0,
    message_count INTEGER NOT NULL DEFAULT 0,
    unread_count INTEGER NOT NULL DEFAULT 0,
    total_size INTEGER NOT NULL DEFAULT 0,
    retention_days INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    mailbox_id TEXT NOT NULL,
    message_id TEXT,
    from_name TEXT,
    from_address TEXT NOT NULL,
    subject TEXT,
    text_body TEXT,
    html_body TEXT,
    raw_body BLOB,
    headers TEXT,
    content_type TEXT NOT NULL DEFAULT 'text/plain',
    size INTEGER NOT NULL DEFAULT 0,
    attachment_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'unread',
    is_starred INTEGER NOT NULL DEFAULT 0,
    is_spam INTEGER NOT NULL DEFAULT 0,
    in_reply_to TEXT,
    references_list TEXT,
    received_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
);

-- Message recipients table (for To, Cc, Bcc)
CREATE TABLE IF NOT EXISTS message_recipients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    recipient_type TEXT NOT NULL,
    name TEXT,
    address TEXT NOT NULL,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

-- Attachments table
CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    content_id TEXT,
    disposition TEXT NOT NULL DEFAULT 'attachment',
    storage_path TEXT,
    checksum TEXT,
    is_inline INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

-- Attachment content table (for storing binary data)
CREATE TABLE IF NOT EXISTS attachment_content (
    attachment_id TEXT PRIMARY KEY,
    content BLOB,
    FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE
);

-- Webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    secret TEXT,
    events TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    headers TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    last_triggered_at DATETIME,
    last_success_at DATETIME,
    last_failure_at DATETIME,
    last_error TEXT,
    success_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Webhook deliveries table
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL,
    event TEXT NOT NULL,
    payload TEXT NOT NULL,
    status_code INTEGER,
    response TEXT,
    error TEXT,
    success INTEGER NOT NULL DEFAULT 0,
    duration INTEGER NOT NULL DEFAULT 0,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
);

-- Settings table
CREATE TABLE IF NOT EXISTS settings (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Settings change history table
CREATE TABLE IF NOT EXISTS settings_history (
    id TEXT PRIMARY KEY,
    field_path TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by TEXT,
    reason TEXT,
    changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE SET NULL
);

-- +migrate Down
-- Rollback: Drop all tables in reverse order of dependencies

DROP TABLE IF EXISTS settings_history;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS attachment_content;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS message_recipients;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS mailboxes;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS schema_migrations;
