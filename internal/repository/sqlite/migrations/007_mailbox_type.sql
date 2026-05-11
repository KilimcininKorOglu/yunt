-- +migrate Up
ALTER TABLE mailboxes ADD COLUMN mailbox_type TEXT NOT NULL DEFAULT 'custom';
UPDATE mailboxes SET mailbox_type = 'system' WHERE is_default = 1;

-- +migrate Down
ALTER TABLE mailboxes DROP COLUMN mailbox_type;
