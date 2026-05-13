-- +migrate Up
ALTER TABLE users ADD COLUMN signature TEXT NOT NULL DEFAULT ('');
ALTER TABLE users ADD COLUMN signature_html TEXT NOT NULL DEFAULT ('');

-- +migrate Down
ALTER TABLE users DROP COLUMN signature;
ALTER TABLE users DROP COLUMN signature_html;
