-- +migrate Up
ALTER TABLE messages ADD COLUMN imap_uid INTEGER NOT NULL DEFAULT 0;
ALTER TABLE mailboxes ADD COLUMN uid_next INTEGER NOT NULL DEFAULT 1;

-- Backfill existing messages with sequential UIDs per mailbox
UPDATE messages SET imap_uid = (
    SELECT COUNT(*) FROM messages AS m2
    WHERE m2.mailbox_id = messages.mailbox_id
    AND (m2.received_at < messages.received_at
         OR (m2.received_at = messages.received_at AND m2.id <= messages.id))
);

-- Set uid_next for each mailbox to max(imap_uid) + 1
UPDATE mailboxes SET uid_next = COALESCE(
    (SELECT MAX(imap_uid) + 1 FROM messages WHERE messages.mailbox_id = mailboxes.id),
    1
);

CREATE UNIQUE INDEX idx_messages_mailbox_uid ON messages(mailbox_id, imap_uid);

-- +migrate Down
DROP INDEX IF EXISTS idx_messages_mailbox_uid;
ALTER TABLE messages DROP COLUMN imap_uid;
ALTER TABLE mailboxes DROP COLUMN uid_next;
