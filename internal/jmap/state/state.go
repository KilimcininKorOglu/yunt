package state

import (
	"context"
	"fmt"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// Manager wraps StateRepository to provide JMAP state management.
type Manager struct {
	repo repository.StateRepository
}

// NewManager creates a new JMAP state manager.
func NewManager(repo repository.StateRepository) *Manager {
	return &Manager{repo: repo}
}

// CurrentState returns the current state string for the given type.
func (m *Manager) CurrentState(ctx context.Context, accountID domain.ID, typeName string) (string, error) {
	val, err := m.repo.CurrentState(ctx, accountID, typeName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", val), nil
}

// BumpState increments the state and records a change.
func (m *Manager) BumpState(ctx context.Context, accountID domain.ID, typeName string, entityID domain.ID, changeType string) (string, error) {
	val, err := m.repo.BumpState(ctx, accountID, typeName, entityID, changeType)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", val), nil
}

// GetChanges returns changes since the given state.
func (m *Manager) GetChanges(ctx context.Context, accountID domain.ID, typeName string, sinceState string, maxChanges int64) (*repository.ChangesResult, error) {
	var since int64
	if sinceState != "" && sinceState != "0" {
		if _, err := fmt.Sscanf(sinceState, "%d", &since); err != nil {
			return nil, fmt.Errorf("invalid state string: %s", sinceState)
		}
	}
	return m.repo.GetChanges(ctx, accountID, typeName, since, maxChanges)
}
