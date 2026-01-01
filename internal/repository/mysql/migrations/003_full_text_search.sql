-- +migrate Up
-- Create FULLTEXT indexes for search functionality (MySQL)
-- InnoDB supports FULLTEXT indexes since MySQL 5.6

-- FULLTEXT index for message search (subject, from_address, text_body)
-- This allows natural language search across message content
ALTER TABLE messages ADD FULLTEXT INDEX ft_messages_search (subject, from_address, text_body);

-- FULLTEXT index for subject-only search
ALTER TABLE messages ADD FULLTEXT INDEX ft_messages_subject (subject);

-- FULLTEXT index for body-only search
ALTER TABLE messages ADD FULLTEXT INDEX ft_messages_body (text_body);

-- FULLTEXT index for user search (username, email, display_name)
ALTER TABLE users ADD FULLTEXT INDEX ft_users_search (username, email, display_name);

-- FULLTEXT index for mailbox search (name, address, description)
ALTER TABLE mailboxes ADD FULLTEXT INDEX ft_mailboxes_search (name, address, description);

-- FULLTEXT index for webhook search (name)
ALTER TABLE webhooks ADD FULLTEXT INDEX ft_webhooks_name (name);

-- +migrate Down
-- Rollback: Drop all FULLTEXT indexes

-- Drop webhook FULLTEXT index
ALTER TABLE webhooks DROP INDEX ft_webhooks_name;

-- Drop mailbox FULLTEXT index
ALTER TABLE mailboxes DROP INDEX ft_mailboxes_search;

-- Drop user FULLTEXT index
ALTER TABLE users DROP INDEX ft_users_search;

-- Drop message FULLTEXT indexes
ALTER TABLE messages DROP INDEX ft_messages_body;
ALTER TABLE messages DROP INDEX ft_messages_subject;
ALTER TABLE messages DROP INDEX ft_messages_search;
