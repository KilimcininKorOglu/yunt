-- +migrate Up
-- Initial database schema for Yunt Mail Server (PostgreSQL)

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Schema migrations tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    avatar_url TEXT,
    last_login_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_role_check CHECK (role IN ('admin', 'user', 'viewer')),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'inactive', 'pending'))
);

-- Mailboxes table
CREATE TABLE IF NOT EXISTS mailboxes (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    is_catch_all BOOLEAN NOT NULL DEFAULT FALSE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    message_count BIGINT NOT NULL DEFAULT 0,
    unread_count BIGINT NOT NULL DEFAULT 0,
    total_size BIGINT NOT NULL DEFAULT 0,
    retention_days INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT mailboxes_counts_check CHECK (
        message_count >= 0 AND 
        unread_count >= 0 AND 
        total_size >= 0 AND
        retention_days >= 0
    )
);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(36) PRIMARY KEY,
    mailbox_id VARCHAR(36) NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    message_id VARCHAR(255),
    from_name VARCHAR(255),
    from_address VARCHAR(255) NOT NULL,
    subject TEXT,
    text_body TEXT,
    html_body TEXT,
    raw_body BYTEA,
    headers JSONB,
    content_type VARCHAR(100) NOT NULL DEFAULT 'text/plain',
    size BIGINT NOT NULL DEFAULT 0,
    attachment_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'unread',
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    is_spam BOOLEAN NOT NULL DEFAULT FALSE,
    in_reply_to VARCHAR(255),
    references_list JSONB,
    received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    search_vector TSVECTOR,
    CONSTRAINT messages_status_check CHECK (status IN ('unread', 'read')),
    CONSTRAINT messages_size_check CHECK (size >= 0 AND attachment_count >= 0)
);

-- Message recipients table (for To, Cc, Bcc)
CREATE TABLE IF NOT EXISTS message_recipients (
    id SERIAL PRIMARY KEY,
    message_id VARCHAR(36) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    recipient_type VARCHAR(20) NOT NULL,
    name VARCHAR(255),
    address VARCHAR(255) NOT NULL,
    CONSTRAINT message_recipients_type_check CHECK (recipient_type IN ('to', 'cc', 'bcc'))
);

-- Attachments table
CREATE TABLE IF NOT EXISTS attachments (
    id VARCHAR(36) PRIMARY KEY,
    message_id VARCHAR(36) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    content_id VARCHAR(255),
    disposition VARCHAR(50) NOT NULL DEFAULT 'attachment',
    storage_path TEXT,
    checksum VARCHAR(64),
    is_inline BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT attachments_size_check CHECK (size >= 0),
    CONSTRAINT attachments_disposition_check CHECK (disposition IN ('attachment', 'inline'))
);

-- Attachment content table (for storing binary data)
CREATE TABLE IF NOT EXISTS attachment_content (
    attachment_id VARCHAR(36) PRIMARY KEY REFERENCES attachments(id) ON DELETE CASCADE,
    content BYTEA
);

-- Webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    secret TEXT,
    events JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    headers JSONB,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    success_count BIGINT NOT NULL DEFAULT 0,
    failure_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT webhooks_status_check CHECK (status IN ('active', 'inactive', 'failed')),
    CONSTRAINT webhooks_counts_check CHECK (
        retry_count >= 0 AND 
        max_retries >= 0 AND 
        timeout_seconds > 0 AND
        success_count >= 0 AND
        failure_count >= 0
    )
);

-- Webhook deliveries table
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR(36) PRIMARY KEY,
    webhook_id VARCHAR(36) NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event VARCHAR(100) NOT NULL,
    payload TEXT NOT NULL,
    status_code INTEGER,
    response TEXT,
    error TEXT,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    duration BIGINT NOT NULL DEFAULT 0,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT webhook_deliveries_attempt_check CHECK (attempt_number > 0 AND duration >= 0)
);

-- Settings table
CREATE TABLE IF NOT EXISTS settings (
    id VARCHAR(36) PRIMARY KEY,
    data JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Settings change history table
CREATE TABLE IF NOT EXISTS settings_history (
    id VARCHAR(36) PRIMARY KEY,
    field_path VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by VARCHAR(36) REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT,
    changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
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

-- Drop extensions (optional, may be used by other databases)
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS "uuid-ossp";
