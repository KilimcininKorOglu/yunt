-- +migrate Up
ALTER TABLE messages ADD COLUMN imap_uid INTEGER NOT NULL DEFAULT 0;
ALTER TABLE mailboxes ADD COLUMN uid_next INTEGER NOT NULL DEFAULT 1;

-- Backfill existing messages with sequential UIDs per mailbox
UPDATE messages SET imap_uid = sub.rn FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY mailbox_id ORDER BY received_at, id) AS rn
    FROM messages
) sub WHERE messages.id = sub.id;

-- Set uid_next for each mailbox
UPDATE mailboxes SET uid_next = COALESCE(
    (SELECT MAX(imap_uid) + 1 FROM messages WHERE messages.mailbox_id = mailboxes.id),
    1
);

CREATE UNIQUE INDEX idx_messages_mailbox_uid ON messages(mailbox_id, imap_uid);

-- +migrate Down
DROP INDEX IF EXISTS idx_messages_mailbox_uid;
ALTER TABLE messages DROP COLUMN imap_uid;
ALTER TABLE mailboxes DROP COLUMN uid_next;
