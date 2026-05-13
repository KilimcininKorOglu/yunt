-- +migrate Up
ALTER TABLE messages ADD COLUMN IF NOT EXISTS thread_id TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS blob_id TEXT NOT NULL DEFAULT '';
ALTER TABLE mailboxes ADD COLUMN IF NOT EXISTS jmap_role TEXT NOT NULL DEFAULT '';
ALTER TABLE mailboxes ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_messages_thread_id ON messages(thread_id);
CREATE INDEX IF NOT EXISTS idx_messages_blob_id ON messages(blob_id);

CREATE TABLE IF NOT EXISTS jmap_state (
    account_id  TEXT NOT NULL,
    type_name   TEXT NOT NULL,
    state_value BIGINT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_id, type_name)
);

CREATE TABLE IF NOT EXISTS jmap_changes (
    id          BIGSERIAL PRIMARY KEY,
    account_id  TEXT NOT NULL,
    type_name   TEXT NOT NULL,
    state_value BIGINT NOT NULL,
    entity_id   TEXT NOT NULL,
    change_type TEXT NOT NULL CHECK (change_type IN ('created','updated','destroyed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_jmap_changes_lookup ON jmap_changes(account_id, type_name, state_value);

CREATE TABLE IF NOT EXISTS identities (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id),
    name            TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL,
    reply_to        JSONB NOT NULL DEFAULT '[]'::jsonb,
    bcc             JSONB NOT NULL DEFAULT '[]'::jsonb,
    text_signature  TEXT NOT NULL DEFAULT '',
    html_signature  TEXT NOT NULL DEFAULT '',
    may_delete      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_identities_user ON identities(user_id);

CREATE TABLE IF NOT EXISTS email_submissions (
    id              TEXT PRIMARY KEY,
    identity_id     TEXT NOT NULL REFERENCES identities(id),
    email_id        TEXT NOT NULL,
    thread_id       TEXT NOT NULL DEFAULT '',
    envelope_from   TEXT NOT NULL DEFAULT '',
    envelope_to     JSONB NOT NULL DEFAULT '[]'::jsonb,
    send_at         TIMESTAMPTZ,
    undo_status     TEXT NOT NULL DEFAULT 'final',
    delivery_status JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_email_submissions_identity ON email_submissions(identity_id);
CREATE INDEX IF NOT EXISTS idx_email_submissions_pending ON email_submissions(send_at) WHERE undo_status = 'pending';

CREATE TABLE IF NOT EXISTS vacation_responses (
    id          TEXT PRIMARY KEY DEFAULT 'singleton',
    user_id     TEXT NOT NULL UNIQUE REFERENCES users(id),
    is_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    from_date   TIMESTAMPTZ,
    to_date     TIMESTAMPTZ,
    subject     TEXT NOT NULL DEFAULT '',
    text_body   TEXT NOT NULL DEFAULT '',
    html_body   TEXT NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id                TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id),
    device_client_id  TEXT NOT NULL,
    url               TEXT NOT NULL,
    keys_p256dh       TEXT NOT NULL DEFAULT '',
    keys_auth         TEXT NOT NULL DEFAULT '',
    verification_code TEXT NOT NULL DEFAULT '',
    expires           TIMESTAMPTZ,
    types             JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user ON push_subscriptions(user_id);

CREATE TABLE IF NOT EXISTS address_books (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_address_books_user ON address_books(user_id);

CREATE TABLE IF NOT EXISTS contact_cards (
    id               TEXT PRIMARY KEY,
    uid              TEXT NOT NULL,
    user_id          TEXT NOT NULL REFERENCES users(id),
    address_book_ids JSONB NOT NULL DEFAULT '{}'::jsonb,
    kind             TEXT NOT NULL DEFAULT 'individual',
    full_name        TEXT NOT NULL DEFAULT '',
    name_data        JSONB NOT NULL DEFAULT '{}'::jsonb,
    emails           JSONB NOT NULL DEFAULT '[]'::jsonb,
    phones           JSONB NOT NULL DEFAULT '[]'::jsonb,
    addresses        JSONB NOT NULL DEFAULT '[]'::jsonb,
    notes            TEXT NOT NULL DEFAULT '',
    photos           JSONB NOT NULL DEFAULT '[]'::jsonb,
    extra_data       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, uid)
);
CREATE INDEX IF NOT EXISTS idx_contact_cards_user ON contact_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_contact_cards_fullname ON contact_cards(full_name);

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
ALTER TABLE mailboxes DROP COLUMN IF EXISTS sort_order;
ALTER TABLE mailboxes DROP COLUMN IF EXISTS jmap_role;
ALTER TABLE messages DROP COLUMN IF EXISTS blob_id;
ALTER TABLE messages DROP COLUMN IF EXISTS thread_id;
