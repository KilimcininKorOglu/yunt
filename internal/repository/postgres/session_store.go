package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"yunt/internal/domain"
)

// DBSessionStore implements service.SessionStore using PostgreSQL.
type DBSessionStore struct {
	db *sqlx.DB
}

// NewDBSessionStore creates a new database-backed session store.
func NewDBSessionStore(db *sqlx.DB) *DBSessionStore {
	return &DBSessionStore{db: db}
}

type sessionRow struct {
	ID               string    `db:"id"`
	UserID           string    `db:"user_id"`
	RefreshTokenHash string    `db:"refresh_token_hash"`
	UserAgent        string    `db:"user_agent"`
	IPAddress        string    `db:"ip_address"`
	IsRevoked        bool      `db:"is_revoked"`
	CreatedAt        time.Time `db:"created_at"`
	ExpiresAt        time.Time `db:"expires_at"`
	LastUsedAt       time.Time `db:"last_used_at"`
}

func (s *DBSessionStore) Create(_ context.Context, session *domain.Session) error {
	query := `INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, is_revoked, created_at, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := s.db.Exec(query,
		session.ID,
		string(session.UserID),
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.IsRevoked,
		session.CreatedAt.Time,
		session.ExpiresAt.Time,
		session.LastUsedAt.Time,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (s *DBSessionStore) Get(_ context.Context, id string) (*domain.Session, error) {
	query := `SELECT id, user_id, refresh_token_hash, user_agent, ip_address, is_revoked, created_at, expires_at, last_used_at
		FROM sessions WHERE id = $1`

	var row sessionRow
	if err := s.db.Get(&row, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return rowToSession(&row), nil
}

func (s *DBSessionStore) Update(_ context.Context, session *domain.Session) error {
	query := `UPDATE sessions SET refresh_token_hash = $1, user_agent = $2, ip_address = $3, is_revoked = $4, last_used_at = $5 WHERE id = $6`

	_, err := s.db.Exec(query,
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.IsRevoked,
		session.LastUsedAt.Time,
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (s *DBSessionStore) Delete(_ context.Context, id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (s *DBSessionStore) DeleteByUserID(_ context.Context, userID domain.ID) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = $1`, string(userID))
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (s *DBSessionStore) Touch(_ context.Context, id string) error {
	_, err := s.db.Exec(`UPDATE sessions SET last_used_at = $1 WHERE id = $2`, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}
	return nil
}

func rowToSession(row *sessionRow) *domain.Session {
	return &domain.Session{
		ID:               row.ID,
		UserID:           domain.ID(row.UserID),
		RefreshTokenHash: row.RefreshTokenHash,
		UserAgent:        row.UserAgent,
		IPAddress:        row.IPAddress,
		IsRevoked:        row.IsRevoked,
		CreatedAt:        domain.Timestamp{Time: row.CreatedAt},
		ExpiresAt:        domain.Timestamp{Time: row.ExpiresAt},
		LastUsedAt:       domain.Timestamp{Time: row.LastUsedAt},
	}
}
