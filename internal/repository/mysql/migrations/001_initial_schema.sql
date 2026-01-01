-- +migrate Up
-- Initial database schema for Yunt Mail Server (MySQL)
-- Uses InnoDB engine and UTF8MB4 encoding for full Unicode support

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    avatar_url TEXT,
    last_login_at DATETIME(6),
    deleted_at DATETIME(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    UNIQUE KEY uk_users_username (username),
    UNIQUE KEY uk_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Mailboxes table
CREATE TABLE IF NOT EXISTS mailboxes (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    description TEXT,
    is_catch_all TINYINT(1) NOT NULL DEFAULT 0,
    is_default TINYINT(1) NOT NULL DEFAULT 0,
    message_count BIGINT NOT NULL DEFAULT 0,
    unread_count BIGINT NOT NULL DEFAULT 0,
    total_size BIGINT NOT NULL DEFAULT 0,
    retention_days INT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    UNIQUE KEY uk_mailboxes_address (address),
    CONSTRAINT fk_mailboxes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(36) PRIMARY KEY,
    mailbox_id VARCHAR(36) NOT NULL,
    message_id VARCHAR(255),
    from_name VARCHAR(255),
    from_address VARCHAR(255) NOT NULL,
    subject TEXT,
    text_body MEDIUMTEXT,
    html_body MEDIUMTEXT,
    raw_body MEDIUMBLOB,
    headers JSON,
    content_type VARCHAR(100) NOT NULL DEFAULT 'text/plain',
    size BIGINT NOT NULL DEFAULT 0,
    attachment_count INT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'unread',
    is_starred TINYINT(1) NOT NULL DEFAULT 0,
    is_spam TINYINT(1) NOT NULL DEFAULT 0,
    in_reply_to VARCHAR(255),
    references_list JSON,
    received_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    sent_at DATETIME(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_messages_mailbox FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Message recipients table (for To, Cc, Bcc)
CREATE TABLE IF NOT EXISTS message_recipients (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    message_id VARCHAR(36) NOT NULL,
    recipient_type VARCHAR(20) NOT NULL,
    name VARCHAR(255),
    address VARCHAR(255) NOT NULL,
    CONSTRAINT fk_recipients_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Attachments table
CREATE TABLE IF NOT EXISTS attachments (
    id VARCHAR(36) PRIMARY KEY,
    message_id VARCHAR(36) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    content_id VARCHAR(255),
    disposition VARCHAR(50) NOT NULL DEFAULT 'attachment',
    storage_path TEXT,
    checksum VARCHAR(64),
    is_inline TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_attachments_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Attachment content table (for storing binary data)
CREATE TABLE IF NOT EXISTS attachment_content (
    attachment_id VARCHAR(36) PRIMARY KEY,
    content MEDIUMBLOB,
    CONSTRAINT fk_content_attachment FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    secret VARCHAR(255),
    events JSON NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    headers JSON,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    timeout_seconds INT NOT NULL DEFAULT 30,
    last_triggered_at DATETIME(6),
    last_success_at DATETIME(6),
    last_failure_at DATETIME(6),
    last_error TEXT,
    success_count BIGINT NOT NULL DEFAULT 0,
    failure_count BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_webhooks_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Webhook deliveries table
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR(36) PRIMARY KEY,
    webhook_id VARCHAR(36) NOT NULL,
    event VARCHAR(100) NOT NULL,
    payload MEDIUMTEXT NOT NULL,
    status_code INT,
    response MEDIUMTEXT,
    error TEXT,
    success TINYINT(1) NOT NULL DEFAULT 0,
    duration BIGINT NOT NULL DEFAULT 0,
    attempt_number INT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_deliveries_webhook FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Settings table
CREATE TABLE IF NOT EXISTS settings (
    id VARCHAR(36) PRIMARY KEY,
    data JSON NOT NULL,
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Settings change history table
CREATE TABLE IF NOT EXISTS settings_history (
    id VARCHAR(36) PRIMARY KEY,
    field_path VARCHAR(255) NOT NULL,
    old_value MEDIUMTEXT,
    new_value MEDIUMTEXT,
    changed_by VARCHAR(36),
    reason TEXT,
    changed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_settings_changed_by FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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
