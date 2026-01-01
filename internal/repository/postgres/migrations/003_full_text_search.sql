-- +migrate Up
-- PostgreSQL Full-Text Search configuration for email messages

-- Create a custom text search configuration for email content
-- This configuration is optimized for searching email messages

-- Create English text search configuration (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_ts_config WHERE cfgname = 'yunt_email'
    ) THEN
        CREATE TEXT SEARCH CONFIGURATION yunt_email (COPY = english);
    END IF;
END $$;

-- Add trigram-based similarity search index on subject
CREATE INDEX IF NOT EXISTS idx_messages_subject_trgm ON messages USING gin(subject gin_trgm_ops);

-- Add trigram-based similarity search index on from_name
CREATE INDEX IF NOT EXISTS idx_messages_from_name_trgm ON messages USING gin(from_name gin_trgm_ops);

-- Add trigram-based similarity search index on from_address
CREATE INDEX IF NOT EXISTS idx_messages_from_address_trgm ON messages USING gin(from_address gin_trgm_ops);

-- Create GIN index on the search_vector column for full-text search
CREATE INDEX IF NOT EXISTS idx_messages_search_vector ON messages USING gin(search_vector);

-- Create function to update search_vector on message insert/update
CREATE OR REPLACE FUNCTION messages_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.subject, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.from_address, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.from_name, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.text_body, '')), 'C');
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update search_vector
DROP TRIGGER IF EXISTS messages_search_vector_trigger ON messages;
CREATE TRIGGER messages_search_vector_trigger
    BEFORE INSERT OR UPDATE OF subject, from_address, from_name, text_body
    ON messages
    FOR EACH ROW
    EXECUTE FUNCTION messages_search_vector_update();

-- Update existing messages to populate search_vector
UPDATE messages SET search_vector =
    setweight(to_tsvector('english', COALESCE(subject, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(from_address, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(from_name, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(text_body, '')), 'C')
WHERE search_vector IS NULL;

-- Create function to search messages with ranking
CREATE OR REPLACE FUNCTION search_messages(
    p_mailbox_id VARCHAR(36),
    p_query TEXT,
    p_limit INTEGER DEFAULT 50,
    p_offset INTEGER DEFAULT 0
) RETURNS TABLE (
    message_id VARCHAR(36),
    rank REAL,
    headline_subject TEXT,
    headline_body TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id AS message_id,
        ts_rank_cd(m.search_vector, plainto_tsquery('english', p_query)) AS rank,
        ts_headline('english', COALESCE(m.subject, ''), plainto_tsquery('english', p_query), 
            'MaxWords=30, MinWords=15, ShortWord=3, HighlightAll=FALSE, MaxFragments=1') AS headline_subject,
        ts_headline('english', COALESCE(m.text_body, ''), plainto_tsquery('english', p_query), 
            'MaxWords=50, MinWords=25, ShortWord=3, HighlightAll=FALSE, MaxFragments=2') AS headline_body
    FROM messages m
    WHERE m.mailbox_id = p_mailbox_id
      AND m.search_vector @@ plainto_tsquery('english', p_query)
    ORDER BY rank DESC, m.received_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql STABLE;

-- Create function to search messages with fuzzy matching (trigram similarity)
CREATE OR REPLACE FUNCTION search_messages_fuzzy(
    p_mailbox_id VARCHAR(36),
    p_query TEXT,
    p_similarity_threshold REAL DEFAULT 0.3,
    p_limit INTEGER DEFAULT 50,
    p_offset INTEGER DEFAULT 0
) RETURNS TABLE (
    message_id VARCHAR(36),
    similarity_score REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id AS message_id,
        GREATEST(
            similarity(COALESCE(m.subject, ''), p_query),
            similarity(COALESCE(m.from_address, ''), p_query),
            similarity(COALESCE(m.from_name, ''), p_query)
        ) AS similarity_score
    FROM messages m
    WHERE m.mailbox_id = p_mailbox_id
      AND (
          similarity(COALESCE(m.subject, ''), p_query) > p_similarity_threshold
          OR similarity(COALESCE(m.from_address, ''), p_query) > p_similarity_threshold
          OR similarity(COALESCE(m.from_name, ''), p_query) > p_similarity_threshold
      )
    ORDER BY similarity_score DESC, m.received_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql STABLE;

-- Create a combined search function that uses both FTS and fuzzy matching
CREATE OR REPLACE FUNCTION search_messages_combined(
    p_mailbox_id VARCHAR(36),
    p_query TEXT,
    p_limit INTEGER DEFAULT 50,
    p_offset INTEGER DEFAULT 0
) RETURNS TABLE (
    message_id VARCHAR(36),
    combined_score REAL,
    match_type TEXT
) AS $$
BEGIN
    RETURN QUERY
    WITH fts_results AS (
        SELECT 
            m.id,
            ts_rank_cd(m.search_vector, plainto_tsquery('english', p_query)) AS score,
            'fts'::TEXT AS match_type
        FROM messages m
        WHERE m.mailbox_id = p_mailbox_id
          AND m.search_vector @@ plainto_tsquery('english', p_query)
    ),
    fuzzy_results AS (
        SELECT 
            m.id,
            GREATEST(
                similarity(COALESCE(m.subject, ''), p_query),
                similarity(COALESCE(m.from_address, ''), p_query),
                similarity(COALESCE(m.from_name, ''), p_query)
            ) AS score,
            'fuzzy'::TEXT AS match_type
        FROM messages m
        WHERE m.mailbox_id = p_mailbox_id
          AND m.id NOT IN (SELECT id FROM fts_results)
          AND (
              similarity(COALESCE(m.subject, ''), p_query) > 0.3
              OR similarity(COALESCE(m.from_address, ''), p_query) > 0.3
              OR similarity(COALESCE(m.from_name, ''), p_query) > 0.3
          )
    ),
    combined AS (
        SELECT * FROM fts_results
        UNION ALL
        SELECT * FROM fuzzy_results
    )
    SELECT 
        c.id AS message_id,
        c.score AS combined_score,
        c.match_type
    FROM combined c
    JOIN messages m ON m.id = c.id
    ORDER BY c.score DESC, m.received_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql STABLE;

-- +migrate Down
-- Rollback: Remove full-text search configuration

-- Drop functions
DROP FUNCTION IF EXISTS search_messages_combined(VARCHAR(36), TEXT, INTEGER, INTEGER);
DROP FUNCTION IF EXISTS search_messages_fuzzy(VARCHAR(36), TEXT, REAL, INTEGER, INTEGER);
DROP FUNCTION IF EXISTS search_messages(VARCHAR(36), TEXT, INTEGER, INTEGER);

-- Drop trigger
DROP TRIGGER IF EXISTS messages_search_vector_trigger ON messages;

-- Drop function
DROP FUNCTION IF EXISTS messages_search_vector_update();

-- Drop indexes
DROP INDEX IF EXISTS idx_messages_search_vector;
DROP INDEX IF EXISTS idx_messages_from_address_trgm;
DROP INDEX IF EXISTS idx_messages_from_name_trgm;
DROP INDEX IF EXISTS idx_messages_subject_trgm;

-- Drop custom text search configuration
DROP TEXT SEARCH CONFIGURATION IF EXISTS yunt_email;
