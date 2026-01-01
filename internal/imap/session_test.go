package imap

import (
	"testing"
)

func TestSessionState_String(t *testing.T) {
	tests := []struct {
		state SessionState
		want  string
	}{
		{SessionStateNotAuthenticated, "not_authenticated"},
		{SessionStateAuthenticated, "authenticated"},
		{SessionStateSelected, "selected"},
		{SessionStateLogout, "logout"},
		{SessionState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_StateTransitions(t *testing.T) {
	// Test that session state constants have correct ordering
	if SessionStateNotAuthenticated >= SessionStateAuthenticated {
		t.Error("SessionStateNotAuthenticated should be less than SessionStateAuthenticated")
	}
	if SessionStateAuthenticated >= SessionStateSelected {
		t.Error("SessionStateAuthenticated should be less than SessionStateSelected")
	}
	if SessionStateSelected >= SessionStateLogout {
		t.Error("SessionStateSelected should be less than SessionStateLogout")
	}
}

func TestSession_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name        string
		state       SessionState
		userSession *UserSession
		want        bool
	}{
		{
			name:        "not authenticated state, no session",
			state:       SessionStateNotAuthenticated,
			userSession: nil,
			want:        false,
		},
		{
			name:        "authenticated state, with session",
			state:       SessionStateAuthenticated,
			userSession: &UserSession{},
			want:        true,
		},
		{
			name:        "selected state, with session",
			state:       SessionStateSelected,
			userSession: &UserSession{},
			want:        true,
		},
		{
			name:        "authenticated state, no session (edge case)",
			state:       SessionStateAuthenticated,
			userSession: nil,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{
				state:       tt.state,
				userSession: tt.userSession,
			}
			if got := s.IsAuthenticated(); got != tt.want {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_GetUserSession(t *testing.T) {
	t.Run("with user session", func(t *testing.T) {
		expected := &UserSession{ID: "test-session"}
		s := &Session{userSession: expected}
		if got := s.GetUserSession(); got != expected {
			t.Errorf("GetUserSession() = %v, want %v", got, expected)
		}
	})

	t.Run("without user session", func(t *testing.T) {
		s := &Session{}
		if got := s.GetUserSession(); got != nil {
			t.Errorf("GetUserSession() = %v, want nil", got)
		}
	})
}

func TestSession_GetState(t *testing.T) {
	tests := []struct {
		state SessionState
	}{
		{SessionStateNotAuthenticated},
		{SessionStateAuthenticated},
		{SessionStateSelected},
		{SessionStateLogout},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			s := &Session{state: tt.state}
			if got := s.GetState(); got != tt.state {
				t.Errorf("GetState() = %v, want %v", got, tt.state)
			}
		})
	}
}
