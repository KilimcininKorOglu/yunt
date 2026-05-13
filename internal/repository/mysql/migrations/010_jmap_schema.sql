-- +migrate Up
ALTER TABLE messages ADD COLUMN thread_id VARCHAR(255) NOT NULL DEFAULT ('');
ALTER TABLE messages ADD COLUMN blob_id VARCHAR(255) NOT NULL DEFAULT ('');
ALTER TABLE mailboxes ADD COLUMN jmap_role VARCHAR(50) NOT NULL DEFAULT ('');
ALTER TABLE mailboxes ADD COLUMN sort_order INT NOT NULL DEFAULT (0);

CREATE INDEX idx_messages_thread_id ON messages(thread_id);
CREATE INDEX idx_messages_blob_id ON messages(blob_id);

CREATE TABLE jmap_state (
    account_id  VARCHAR(255) NOT NULL,
    type_name   VARCHAR(100) NOT NULL,
    state_value BIGINT NOT NULL DEFAULT 0,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, type_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE jmap_changes (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    account_id  VARCHAR(255) NOT NULL,
    type_name   VARCHAR(100) NOT NULL,
    state_value BIGINT NOT NULL,
    entity_id   VARCHAR(255) NOT NULL,
    change_type ENUM('created','updated','destroyed') NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_jmap_changes_lookup (account_id, type_name, state_value)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE identities (
    id              VARCHAR(255) PRIMARY KEY,
    user_id         VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL DEFAULT '',
    email           VARCHAR(255) NOT NULL,
    reply_to        JSON NOT NULL,
    bcc             JSON NOT NULL,
    text_signature  TEXT NOT NULL,
    html_signature  TEXT NOT NULL,
    may_delete      TINYINT(1) NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_identities_user (user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE email_submissions (
    id              VARCHAR(255) PRIMARY KEY,
    identity_id     VARCHAR(255) NOT NULL,
    email_id        VARCHAR(255) NOT NULL,
    thread_id       VARCHAR(255) NOT NULL DEFAULT '',
    envelope_from   VARCHAR(255) NOT NULL DEFAULT '',
    envelope_to     JSON NOT NULL,
    send_at         DATETIME,
    undo_status     VARCHAR(50) NOT NULL DEFAULT 'final',
    delivery_status JSON NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_email_submissions_identity (identity_id),
    INDEX idx_email_submissions_pending (send_at),
    FOREIGN KEY (identity_id) REFERENCES identities(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE vacation_responses (
    id          VARCHAR(255) PRIMARY KEY DEFAULT ('singleton'),
    user_id     VARCHAR(255) NOT NULL,
    is_enabled  TINYINT(1) NOT NULL DEFAULT 0,
    from_date   DATETIME,
    to_date     DATETIME,
    subject     VARCHAR(255) NOT NULL DEFAULT '',
    text_body   TEXT NOT NULL,
    html_body   TEXT NOT NULL,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_vacation_user (user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE push_subscriptions (
    id                VARCHAR(255) PRIMARY KEY,
    user_id           VARCHAR(255) NOT NULL,
    device_client_id  VARCHAR(255) NOT NULL,
    url               TEXT NOT NULL,
    keys_p256dh       TEXT NOT NULL,
    keys_auth         TEXT NOT NULL,
    verification_code VARCHAR(255) NOT NULL DEFAULT '',
    expires           DATETIME,
    types             JSON NOT NULL,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_push_subscriptions_user (user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE address_books (
    id          VARCHAR(255) PRIMARY KEY,
    user_id     VARCHAR(255) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0,
    is_default  TINYINT(1) NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_address_books_user (user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE contact_cards (
    id               VARCHAR(255) PRIMARY KEY,
    uid              VARCHAR(255) NOT NULL,
    user_id          VARCHAR(255) NOT NULL,
    address_book_ids JSON NOT NULL,
    kind             VARCHAR(50) NOT NULL DEFAULT 'individual',
    full_name        VARCHAR(500) NOT NULL DEFAULT '',
    name_data        JSON NOT NULL,
    emails           JSON NOT NULL,
    phones           JSON NOT NULL,
    addresses        JSON NOT NULL,
    notes            TEXT NOT NULL,
    photos           JSON NOT NULL,
    extra_data       JSON NOT NULL,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_contact_user_uid (user_id, uid),
    INDEX idx_contact_cards_user (user_id),
    INDEX idx_contact_cards_fullname (full_name),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

UPDATE mailboxes SET jmap_role = 'inbox' WHERE name = 'Inbox' AND mailbox_type = 'system';
UPDATE mailboxes SET jmap_role = 'sent' WHERE name = 'Sent' AND mailbox_type = 'system';
UPDATE mailboxes SET jmap_role = 'drafts' WHERE name = 'Drafts' AND mailbox_type = 'system';
UPDATE mailboxes SET jmap_role = 'trash' WHERE name = 'Trash' AND mailbox_type = 'system';
UPDATE mailboxes SET jmap_role = 'junk' WHERE name = 'Spam' AND mailbox_type = 'system';

-- +migrate Down
DROP TABLE IF EXISTS contact_cards;
DROP TABLE IF EXISTS address_books;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS vacation_responses;
DROP TABLE IF EXISTS email_submissions;
DROP TABLE IF EXISTS identities;
DROP TABLE IF EXISTS jmap_changes;
DROP TABLE IF EXISTS jmap_state;
ALTER TABLE mailboxes DROP COLUMN sort_order;
ALTER TABLE mailboxes DROP COLUMN jmap_role;
ALTER TABLE messages DROP COLUMN blob_id;
ALTER TABLE messages DROP COLUMN thread_id;
