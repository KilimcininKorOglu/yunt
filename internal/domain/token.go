package domain

import (
	"time"
)

// TokenType represents the type of JWT token.
type TokenType string

const (
	// TokenTypeAccess represents an access token for API authentication.
	TokenTypeAccess TokenType = "access"
	// TokenTypeRefresh represents a refresh token for obtaining new access tokens.
	TokenTypeRefresh TokenType = "refresh"
)

// IsValid returns true if the token type is a recognized value.
func (t TokenType) IsValid() bool {
	switch t {
	case TokenTypeAccess, TokenTypeRefresh:
		return true
	default:
		return false
	}
}

// String returns the string representation of the token type.
func (t TokenType) String() string {
	return string(t)
}

// TokenClaims represents the custom claims included in JWT tokens.
type TokenClaims struct {
	// UserID is the unique identifier of the authenticated user.
	UserID ID `json:"userId"`
	// Username is the login name of the authenticated user.
	Username string `json:"username"`
	// Email is the email address of the authenticated user.
	Email string `json:"email"`
	// Role is the user's role in the system.
	Role UserRole `json:"role"`
	// TokenType indicates whether this is an access or refresh token.
	TokenType TokenType `json:"tokenType"`
	// SessionID is a unique identifier for this authentication session.
	SessionID string `json:"sessionId"`
}

// TokenPair represents a pair of access and refresh tokens.
type TokenPair struct {
	// AccessToken is the JWT access token for API authentication.
	AccessToken string `json:"accessToken"`
	// RefreshToken is the JWT refresh token for obtaining new access tokens.
	RefreshToken string `json:"refreshToken"`
	// AccessTokenExpiresAt is when the access token expires.
	AccessTokenExpiresAt time.Time `json:"accessTokenExpiresAt"`
	// RefreshTokenExpiresAt is when the refresh token expires.
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
	// TokenType is the type of token (always "Bearer" for JWT).
	TokenType string `json:"tokenType"`
}

// NewTokenPair creates a new TokenPair with the given tokens and expiration times.
func NewTokenPair(accessToken, refreshToken string, accessExp, refreshExp time.Time) *TokenPair {
	return &TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: refreshExp,
		TokenType:             "Bearer",
	}
}

// Session represents an active user session.
type Session struct {
	// ID is the unique identifier for the session.
	ID string `json:"id"`
	// UserID is the ID of the user who owns this session.
	UserID ID `json:"userId"`
	// RefreshTokenHash is the hash of the refresh token for validation.
	RefreshTokenHash string `json:"-"`
	// UserAgent is the client's user agent string.
	UserAgent string `json:"userAgent,omitempty"`
	// IPAddress is the client's IP address.
	IPAddress string `json:"ipAddress,omitempty"`
	// IsRevoked indicates whether the session has been revoked.
	IsRevoked bool `json:"isRevoked"`
	// CreatedAt is when the session was created.
	CreatedAt Timestamp `json:"createdAt"`
	// ExpiresAt is when the session expires.
	ExpiresAt Timestamp `json:"expiresAt"`
	// LastUsedAt is when the session was last used.
	LastUsedAt Timestamp `json:"lastUsedAt"`
}

// NewSession creates a new Session with default values.
func NewSession(id string, userID ID, tokenHash string, expiresAt time.Time) *Session {
	now := Now()
	return &Session{
		ID:               id,
		UserID:           userID,
		RefreshTokenHash: tokenHash,
		IsRevoked:        false,
		CreatedAt:        now,
		ExpiresAt:        Timestamp{Time: expiresAt},
		LastUsedAt:       now,
	}
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt.Time)
}

// IsValid returns true if the session is valid (not expired and not revoked).
func (s *Session) IsValid() bool {
	return !s.IsRevoked && !s.IsExpired()
}

// Revoke marks the session as revoked.
func (s *Session) Revoke() {
	s.IsRevoked = true
}

// Touch updates the last used timestamp.
func (s *Session) Touch() {
	s.LastUsedAt = Now()
}

// LoginInput represents the input for user login.
type LoginInput struct {
	// Username is the user's login name.
	Username string `json:"username"`
	// Password is the user's password.
	Password string `json:"password"`
}

// Validate checks if the login input is valid.
func (i *LoginInput) Validate() error {
	errs := NewValidationErrors()

	if i.Username == "" {
		errs.Add("username", "username is required")
	}

	if i.Password == "" {
		errs.Add("password", "password is required")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// RefreshTokenInput represents the input for token refresh.
type RefreshTokenInput struct {
	// RefreshToken is the refresh token to use for obtaining new tokens.
	RefreshToken string `json:"refreshToken"`
}

// Validate checks if the refresh token input is valid.
func (i *RefreshTokenInput) Validate() error {
	errs := NewValidationErrors()

	if i.RefreshToken == "" {
		errs.Add("refreshToken", "refresh token is required")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// AuthResponse represents the response from authentication operations.
type AuthResponse struct {
	// User contains the authenticated user's information.
	User *UserInfo `json:"user"`
	// Tokens contains the JWT token pair.
	Tokens *TokenPair `json:"tokens"`
}

// UserInfo represents public user information for authentication responses.
type UserInfo struct {
	// ID is the user's unique identifier.
	ID ID `json:"id"`
	// Username is the user's login name.
	Username string `json:"username"`
	// Email is the user's email address.
	Email string `json:"email"`
	// DisplayName is the user's display name.
	DisplayName string `json:"displayName,omitempty"`
	// Role is the user's role in the system.
	Role UserRole `json:"role"`
}

// UserInfoFromUser creates a UserInfo from a User.
func UserInfoFromUser(user *User) *UserInfo {
	return &UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}
}

// InvalidTokenError represents an error for invalid JWT tokens.
type InvalidTokenError struct {
	// Reason provides additional context for why the token is invalid.
	Reason string
}

// Error implements the error interface.
func (e *InvalidTokenError) Error() string {
	if e.Reason != "" {
		return "invalid token: " + e.Reason
	}
	return "invalid token"
}

// Is implements errors.Is interface for error comparison.
func (e *InvalidTokenError) Is(target error) bool {
	_, ok := target.(*InvalidTokenError)
	return ok
}

// NewInvalidTokenError creates a new InvalidTokenError.
func NewInvalidTokenError(reason string) *InvalidTokenError {
	return &InvalidTokenError{
		Reason: reason,
	}
}

// ExpiredTokenError represents an error for expired JWT tokens.
type ExpiredTokenError struct {
	// ExpiredAt is when the token expired.
	ExpiredAt time.Time
}

// Error implements the error interface.
func (e *ExpiredTokenError) Error() string {
	return "token expired"
}

// Is implements errors.Is interface for error comparison.
func (e *ExpiredTokenError) Is(target error) bool {
	_, ok := target.(*ExpiredTokenError)
	return ok
}

// NewExpiredTokenError creates a new ExpiredTokenError.
func NewExpiredTokenError(expiredAt time.Time) *ExpiredTokenError {
	return &ExpiredTokenError{
		ExpiredAt: expiredAt,
	}
}

// SessionRevokedError represents an error when a session has been revoked.
type SessionRevokedError struct {
	// SessionID is the ID of the revoked session.
	SessionID string
}

// Error implements the error interface.
func (e *SessionRevokedError) Error() string {
	return "session has been revoked"
}

// Is implements errors.Is interface for error comparison.
func (e *SessionRevokedError) Is(target error) bool {
	_, ok := target.(*SessionRevokedError)
	return ok
}

// NewSessionRevokedError creates a new SessionRevokedError.
func NewSessionRevokedError(sessionID string) *SessionRevokedError {
	return &SessionRevokedError{
		SessionID: sessionID,
	}
}
