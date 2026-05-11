-- +migrate Up
ALTER TABLE messages ADD COLUMN imap_uid INT UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE mailboxes ADD COLUMN uid_next INT UNSIGNED NOT NULL DEFAULT 1;

-- Backfill existing messages with sequential UIDs per mailbox
UPDATE messages m
INNER JOIN (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY mailbox_id ORDER BY received_at, id) AS rn
    FROM messages
) sub ON m.id = sub.id
SET m.imap_uid = sub.rn;

-- Set uid_next for each mailbox
UPDATE mailboxes SET uid_next = COALESCE(
    (SELECT MAX(imap_uid) + 1 FROM messages WHERE messages.mailbox_id = mailboxes.id),
    1
);

CREATE UNIQUE INDEX idx_messages_mailbox_uid ON messages(mailbox_id, imap_uid);

-- +migrate Down
DROP INDEX idx_messages_mailbox_uid ON messages;
ALTER TABLE messages DROP COLUMN imap_uid;
ALTER TABLE mailboxes DROP COLUMN uid_next;
