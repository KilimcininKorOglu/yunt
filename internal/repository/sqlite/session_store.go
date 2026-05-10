package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"yunt/internal/domain"
)

// DBSessionStore implements service.SessionStore using SQLite.
type DBSessionStore struct {
	db *sqlx.DB
}

// NewDBSessionStore creates a new database-backed session store.
func NewDBSessionStore(db *sqlx.DB) *DBSessionStore {
	return &DBSessionStore{db: db}
}

type sessionRow struct {
	ID               string `db:"id"`
	UserID           string `db:"user_id"`
	RefreshTokenHash string `db:"refresh_token_hash"`
	UserAgent        string `db:"user_agent"`
	IPAddress        string `db:"ip_address"`
	IsRevoked        bool   `db:"is_revoked"`
	CreatedAt        string `db:"created_at"`
	ExpiresAt        string `db:"expires_at"`
	LastUsedAt       string `db:"last_used_at"`
}

func (s *DBSessionStore) Create(_ context.Context, session *domain.Session) error {
	query := `INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, is_revoked, created_at, expires_at, last_used_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		session.ID,
		string(session.UserID),
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.IsRevoked,
		session.CreatedAt.Time.Format(time.RFC3339),
		session.ExpiresAt.Time.Format(time.RFC3339),
		session.LastUsedAt.Time.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (s *DBSessionStore) Get(_ context.Context, id string) (*domain.Session, error) {
	query := `SELECT id, user_id, refresh_token_hash, user_agent, ip_address, is_revoked, created_at, expires_at, last_used_at
		FROM sessions WHERE id = ?`

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
	query := `UPDATE sessions SET refresh_token_hash = ?, user_agent = ?, ip_address = ?, is_revoked = ?, last_used_at = ? WHERE id = ?`

	_, err := s.db.Exec(query,
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.IsRevoked,
		session.LastUsedAt.Time.Format(time.RFC3339),
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (s *DBSessionStore) Delete(_ context.Context, id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (s *DBSessionStore) DeleteByUserID(_ context.Context, userID domain.ID) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, string(userID))
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (s *DBSessionStore) Touch(_ context.Context, id string) error {
	query := `UPDATE sessions SET last_used_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}
	return nil
}

func rowToSession(row *sessionRow) *domain.Session {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	expiresAt, _ := time.Parse(time.RFC3339, row.ExpiresAt)
	lastUsedAt, _ := time.Parse(time.RFC3339, row.LastUsedAt)

	return &domain.Session{
		ID:               row.ID,
		UserID:           domain.ID(row.UserID),
		RefreshTokenHash: row.RefreshTokenHash,
		UserAgent:        row.UserAgent,
		IPAddress:        row.IPAddress,
		IsRevoked:        row.IsRevoked,
		CreatedAt:        domain.Timestamp{Time: createdAt},
		ExpiresAt:        domain.Timestamp{Time: expiresAt},
		LastUsedAt:       domain.Timestamp{Time: lastUsedAt},
	}
}
