-- +migrate Up
ALTER TABLE messages ADD COLUMN thread_id TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN blob_id TEXT NOT NULL DEFAULT '';
ALTER TABLE mailboxes ADD COLUMN jmap_role TEXT NOT NULL DEFAULT '';
ALTER TABLE mailboxes ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX idx_messages_thread_id ON messages(thread_id);
CREATE INDEX idx_messages_blob_id ON messages(blob_id);

CREATE TABLE jmap_state (
    account_id  TEXT NOT NULL,
    type_name   TEXT NOT NULL,
    state_value INTEGER NOT NULL DEFAULT 0,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, type_name)
);

CREATE TABLE jmap_changes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id  TEXT NOT NULL,
    type_name   TEXT NOT NULL,
    state_value INTEGER NOT NULL,
    entity_id   TEXT NOT NULL,
    change_type TEXT NOT NULL CHECK (change_type IN ('created','updated','destroyed')),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_jmap_changes_lookup ON jmap_changes(account_id, type_name, state_value);

CREATE TABLE identities (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id),
    name            TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL,
    reply_to        TEXT NOT NULL DEFAULT '[]',
    bcc             TEXT NOT NULL DEFAULT '[]',
    text_signature  TEXT NOT NULL DEFAULT '',
    html_signature  TEXT NOT NULL DEFAULT '',
    may_delete      INTEGER NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_identities_user ON identities(user_id);

CREATE TABLE email_submissions (
    id              TEXT PRIMARY KEY,
    identity_id     TEXT NOT NULL REFERENCES identities(id),
    email_id        TEXT NOT NULL,
    thread_id       TEXT NOT NULL DEFAULT '',
    envelope_from   TEXT NOT NULL DEFAULT '',
    envelope_to     TEXT NOT NULL DEFAULT '[]',
    send_at         DATETIME,
    undo_status     TEXT NOT NULL DEFAULT 'final',
    delivery_status TEXT NOT NULL DEFAULT '{}',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_email_submissions_identity ON email_submissions(identity_id);
CREATE INDEX idx_email_submissions_pending ON email_submissions(send_at) WHERE undo_status = 'pending';

CREATE TABLE vacation_responses (
    id          TEXT PRIMARY KEY DEFAULT 'singleton',
    user_id     TEXT NOT NULL UNIQUE REFERENCES users(id),
    is_enabled  INTEGER NOT NULL DEFAULT 0,
    from_date   DATETIME,
    to_date     DATETIME,
    subject     TEXT NOT NULL DEFAULT '',
    text_body   TEXT NOT NULL DEFAULT '',
    html_body   TEXT NOT NULL DEFAULT '',
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE push_subscriptions (
    id                TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id),
    device_client_id  TEXT NOT NULL,
    url               TEXT NOT NULL,
    keys_p256dh       TEXT NOT NULL DEFAULT '',
    keys_auth         TEXT NOT NULL DEFAULT '',
    verification_code TEXT NOT NULL DEFAULT '',
    expires           DATETIME,
    types             TEXT NOT NULL DEFAULT '[]',
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_push_subscriptions_user ON push_subscriptions(user_id);

CREATE TABLE address_books (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_default  INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_address_books_user ON address_books(user_id);

CREATE TABLE contact_cards (
    id               TEXT PRIMARY KEY,
    uid              TEXT NOT NULL,
    user_id          TEXT NOT NULL REFERENCES users(id),
    address_book_ids TEXT NOT NULL DEFAULT '{}',
    kind             TEXT NOT NULL DEFAULT 'individual',
    full_name        TEXT NOT NULL DEFAULT '',
    name_data        TEXT NOT NULL DEFAULT '{}',
    emails           TEXT NOT NULL DEFAULT '[]',
    phones           TEXT NOT NULL DEFAULT '[]',
    addresses        TEXT NOT NULL DEFAULT '[]',
    notes            TEXT NOT NULL DEFAULT '',
    photos           TEXT NOT NULL DEFAULT '[]',
    extra_data       TEXT NOT NULL DEFAULT '{}',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, uid)
);
CREATE INDEX idx_contact_cards_user ON contact_cards(user_id);
CREATE INDEX idx_contact_cards_fullname ON contact_cards(full_name);

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
DROP INDEX IF EXISTS idx_messages_blob_id;
DROP INDEX IF EXISTS idx_messages_thread_id;
ALTER TABLE mailboxes DROP COLUMN sort_order;
ALTER TABLE mailboxes DROP COLUMN jmap_role;
ALTER TABLE messages DROP COLUMN blob_id;
ALTER TABLE messages DROP COLUMN thread_id;
