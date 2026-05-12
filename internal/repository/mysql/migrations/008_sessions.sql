-- +migrate Up
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    refresh_token_hash VARCHAR(255) NOT NULL,
    user_agent TEXT DEFAULT '',
    ip_address VARCHAR(45) DEFAULT '',
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    last_used_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- +migrate Down
DROP TABLE IF EXISTS sessions;
