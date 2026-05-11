-- +migrate Up
ALTER TABLE mailboxes ADD COLUMN mailbox_type VARCHAR(20) NOT NULL DEFAULT 'custom';
UPDATE mailboxes SET mailbox_type = 'system' WHERE is_default = true;

-- +migrate Down
ALTER TABLE mailboxes DROP COLUMN IF EXISTS mailbox_type;
