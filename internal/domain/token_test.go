package domain

import (
	"testing"
	"time"
)

func TestTokenType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		tt       TokenType
		expected bool
	}{
		{"access token is valid", TokenTypeAccess, true},
		{"refresh token is valid", TokenTypeRefresh, true},
		{"empty string is invalid", TokenType(""), false},
		{"unknown type is invalid", TokenType("unknown"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tt.IsValid(); got != tc.expected {
				t.Errorf("TokenType.IsValid() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		name     string
		tt       TokenType
		expected string
	}{
		{"access token string", TokenTypeAccess, "access"},
		{"refresh token string", TokenTypeRefresh, "refresh"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tt.String(); got != tc.expected {
				t.Errorf("TokenType.String() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestNewTokenPair(t *testing.T) {
	accessToken := "access-token"
	refreshToken := "refresh-token"
	accessExp := time.Now().Add(15 * time.Minute)
	refreshExp := time.Now().Add(7 * 24 * time.Hour)

	pair := NewTokenPair(accessToken, refreshToken, accessExp, refreshExp)

	if pair.AccessToken != accessToken {
		t.Errorf("AccessToken = %v, want %v", pair.AccessToken, accessToken)
	}
	if pair.RefreshToken != refreshToken {
		t.Errorf("RefreshToken = %v, want %v", pair.RefreshToken, refreshToken)
	}
	if !pair.AccessTokenExpiresAt.Equal(accessExp) {
		t.Errorf("AccessTokenExpiresAt = %v, want %v", pair.AccessTokenExpiresAt, accessExp)
	}
	if !pair.RefreshTokenExpiresAt.Equal(refreshExp) {
		t.Errorf("RefreshTokenExpiresAt = %v, want %v", pair.RefreshTokenExpiresAt, refreshExp)
	}
	if pair.TokenType != "Bearer" {
		t.Errorf("TokenType = %v, want %v", pair.TokenType, "Bearer")
	}
}

func TestNewSession(t *testing.T) {
	id := "session-123"
	userID := ID("user-456")
	tokenHash := "hash-789"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	session := NewSession(id, userID, tokenHash, expiresAt)

	if session.ID != id {
		t.Errorf("ID = %v, want %v", session.ID, id)
	}
	if session.UserID != userID {
		t.Errorf("UserID = %v, want %v", session.UserID, userID)
	}
	if session.RefreshTokenHash != tokenHash {
		t.Errorf("RefreshTokenHash = %v, want %v", session.RefreshTokenHash, tokenHash)
	}
	if session.IsRevoked {
		t.Error("IsRevoked should be false for new session")
	}
	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{"future expiry is not expired", time.Now().Add(time.Hour), false},
		{"past expiry is expired", time.Now().Add(-time.Hour), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := &Session{
				ExpiresAt: Timestamp{Time: tc.expiresAt},
			}
			if got := session.IsExpired(); got != tc.expected {
				t.Errorf("Session.IsExpired() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestSession_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		isRevoked bool
		expiresAt time.Time
		expected  bool
	}{
		{"valid session", false, time.Now().Add(time.Hour), true},
		{"revoked session is invalid", true, time.Now().Add(time.Hour), false},
		{"expired session is invalid", false, time.Now().Add(-time.Hour), false},
		{"revoked and expired is invalid", true, time.Now().Add(-time.Hour), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := &Session{
				IsRevoked: tc.isRevoked,
				ExpiresAt: Timestamp{Time: tc.expiresAt},
			}
			if got := session.IsValid(); got != tc.expected {
				t.Errorf("Session.IsValid() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestSession_Revoke(t *testing.T) {
	session := &Session{IsRevoked: false}
	session.Revoke()
	if !session.IsRevoked {
		t.Error("Session.Revoke() should set IsRevoked to true")
	}
}

func TestSession_Touch(t *testing.T) {
	session := &Session{
		LastUsedAt: Timestamp{Time: time.Now().Add(-time.Hour)},
	}
	oldTime := session.LastUsedAt.Time
	time.Sleep(time.Millisecond) // Ensure time difference
	session.Touch()
	if !session.LastUsedAt.Time.After(oldTime) {
		t.Error("Session.Touch() should update LastUsedAt")
	}
}

func TestLoginInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   LoginInput
		wantErr bool
	}{
		{
			name:    "valid input",
			input:   LoginInput{Username: "testuser", Password: "password123"},
			wantErr: false,
		},
		{
			name:    "missing username",
			input:   LoginInput{Username: "", Password: "password123"},
			wantErr: true,
		},
		{
			name:    "missing password",
			input:   LoginInput{Username: "testuser", Password: ""},
			wantErr: true,
		},
		{
			name:    "missing both",
			input:   LoginInput{Username: "", Password: ""},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("LoginInput.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestRefreshTokenInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   RefreshTokenInput
		wantErr bool
	}{
		{
			name:    "valid input",
			input:   RefreshTokenInput{RefreshToken: "valid-refresh-token"},
			wantErr: false,
		},
		{
			name:    "missing refresh token",
			input:   RefreshTokenInput{RefreshToken: ""},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("RefreshTokenInput.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestUserInfoFromUser(t *testing.T) {
	user := &User{
		ID:          ID("user-123"),
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Role:        RoleUser,
	}

	info := UserInfoFromUser(user)

	if info.ID != user.ID {
		t.Errorf("ID = %v, want %v", info.ID, user.ID)
	}
	if info.Username != user.Username {
		t.Errorf("Username = %v, want %v", info.Username, user.Username)
	}
	if info.Email != user.Email {
		t.Errorf("Email = %v, want %v", info.Email, user.Email)
	}
	if info.DisplayName != user.DisplayName {
		t.Errorf("DisplayName = %v, want %v", info.DisplayName, user.DisplayName)
	}
	if info.Role != user.Role {
		t.Errorf("Role = %v, want %v", info.Role, user.Role)
	}
}

func TestInvalidTokenError(t *testing.T) {
	err := NewInvalidTokenError("token has expired")

	if err.Reason != "token has expired" {
		t.Errorf("Reason = %v, want %v", err.Reason, "token has expired")
	}

	expectedMsg := "invalid token: token has expired"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
	}

	// Test without reason
	err2 := NewInvalidTokenError("")
	if err2.Error() != "invalid token" {
		t.Errorf("Error() = %v, want %v", err2.Error(), "invalid token")
	}

	// Test Is
	if !err.Is(&InvalidTokenError{}) {
		t.Error("InvalidTokenError.Is() should return true for InvalidTokenError")
	}
}

func TestExpiredTokenError(t *testing.T) {
	expiredAt := time.Now().Add(-time.Hour)
	err := NewExpiredTokenError(expiredAt)

	if !err.ExpiredAt.Equal(expiredAt) {
		t.Errorf("ExpiredAt = %v, want %v", err.ExpiredAt, expiredAt)
	}

	if err.Error() != "token expired" {
		t.Errorf("Error() = %v, want %v", err.Error(), "token expired")
	}

	// Test Is
	if !err.Is(&ExpiredTokenError{}) {
		t.Error("ExpiredTokenError.Is() should return true for ExpiredTokenError")
	}
}

func TestSessionRevokedError(t *testing.T) {
	err := NewSessionRevokedError("session-123")

	if err.SessionID != "session-123" {
		t.Errorf("SessionID = %v, want %v", err.SessionID, "session-123")
	}

	if err.Error() != "session has been revoked" {
		t.Errorf("Error() = %v, want %v", err.Error(), "session has been revoked")
	}

	// Test Is
	if !err.Is(&SessionRevokedError{}) {
		t.Error("SessionRevokedError.Is() should return true for SessionRevokedError")
	}
}
