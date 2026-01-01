// Package smtp provides the SMTP server implementation for Yunt mail server.
package smtp

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"

	"github.com/emersion/go-sasl"
	"golang.org/x/crypto/bcrypt"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// AuthenticationError represents an authentication failure with optional details.
type AuthenticationError struct {
	// Reason is a machine-readable reason for the failure.
	Reason string
	// Message is a human-readable description of the failure.
	Message string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("authentication failed: %s - %s", e.Reason, e.Message)
}

// Common authentication error reasons.
const (
	AuthReasonInvalidCredentials = "invalid_credentials"
	AuthReasonAccountDisabled    = "account_disabled"
	AuthReasonAccountPending     = "account_pending"
	AuthReasonInternalError      = "internal_error"
)

// NewInvalidCredentialsError creates an error for invalid username or password.
func NewInvalidCredentialsError() *AuthenticationError {
	return &AuthenticationError{
		Reason:  AuthReasonInvalidCredentials,
		Message: "Invalid username or password",
	}
}

// NewAccountDisabledError creates an error for disabled accounts.
func NewAccountDisabledError() *AuthenticationError {
	return &AuthenticationError{
		Reason:  AuthReasonAccountDisabled,
		Message: "Account is disabled",
	}
}

// NewAccountPendingError creates an error for pending accounts.
func NewAccountPendingError() *AuthenticationError {
	return &AuthenticationError{
		Reason:  AuthReasonAccountPending,
		Message: "Account is pending activation",
	}
}

// NewInternalAuthError creates an error for internal failures.
func NewInternalAuthError(err error) *AuthenticationError {
	return &AuthenticationError{
		Reason:  AuthReasonInternalError,
		Message: fmt.Sprintf("Internal error: %v", err),
	}
}

// Authenticator handles user authentication for SMTP.
type Authenticator struct {
	repo repository.Repository
}

// NewAuthenticator creates a new Authenticator instance.
func NewAuthenticator(repo repository.Repository) *Authenticator {
	return &Authenticator{
		repo: repo,
	}
}

// AuthResult contains the result of a successful authentication.
type AuthResult struct {
	// User is the authenticated user.
	User *domain.User
}

// Authenticate validates user credentials and returns the authenticated user.
// It supports authentication by username or email address.
// Returns an AuthenticationError if authentication fails.
// Uses constant-time comparison to prevent timing attacks.
func (a *Authenticator) Authenticate(ctx context.Context, username, password string) (*AuthResult, error) {
	if username == "" || password == "" {
		return nil, NewInvalidCredentialsError()
	}

	// Try to find user by username first, then by email
	user, err := a.findUser(ctx, username)
	if err != nil {
		return nil, err
	}

	// Verify the password using constant-time comparison
	if err := a.verifyPassword(user.PasswordHash, password); err != nil {
		return nil, NewInvalidCredentialsError()
	}

	// Check user status
	if err := a.validateUserStatus(user); err != nil {
		return nil, err
	}

	// Update last login timestamp
	if err := a.repo.Users().UpdateLastLogin(ctx, user.ID); err != nil {
		// Log the error but don't fail authentication
		// The user is already authenticated at this point
	}

	return &AuthResult{
		User: user,
	}, nil
}

// findUser attempts to find a user by username or email.
func (a *Authenticator) findUser(ctx context.Context, identifier string) (*domain.User, error) {
	// First, try to find by username
	user, err := a.repo.Users().GetByUsername(ctx, identifier)
	if err == nil {
		return user, nil
	}

	// If not found by username, check if not found error
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, NewInternalAuthError(err)
	}

	// Try to find by email
	user, err = a.repo.Users().GetByEmail(ctx, identifier)
	if err == nil {
		return user, nil
	}

	// Check error type
	if errors.Is(err, domain.ErrNotFound) {
		return nil, NewInvalidCredentialsError()
	}

	return nil, NewInternalAuthError(err)
}

// verifyPassword compares a bcrypt hashed password with a plaintext password.
// Uses bcrypt's constant-time comparison to prevent timing attacks.
func (a *Authenticator) verifyPassword(hashedPassword, plainPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
}

// validateUserStatus checks if the user account is in a state that allows login.
func (a *Authenticator) validateUserStatus(user *domain.User) error {
	switch user.Status {
	case domain.StatusActive:
		return nil
	case domain.StatusInactive:
		return NewAccountDisabledError()
	case domain.StatusPending:
		return NewAccountPendingError()
	default:
		return NewAccountDisabledError()
	}
}

// SupportedAuthMechanisms returns the list of supported SASL authentication mechanisms.
func SupportedAuthMechanisms() []string {
	return []string{"PLAIN", "LOGIN"}
}

// AuthMechanism represents an authentication mechanism type.
type AuthMechanism string

const (
	// AuthMechanismPlain represents the PLAIN SASL mechanism (RFC 4616).
	AuthMechanismPlain AuthMechanism = "PLAIN"
	// AuthMechanismLogin represents the LOGIN SASL mechanism.
	AuthMechanismLogin AuthMechanism = "LOGIN"
)

// IsSupported returns true if the mechanism is supported.
func (m AuthMechanism) IsSupported() bool {
	switch m {
	case AuthMechanismPlain, AuthMechanismLogin:
		return true
	default:
		return false
	}
}

// plainServer implements the sasl.Server interface for PLAIN authentication.
type plainServer struct {
	authenticator func(identity, username, password string) error
}

// Next implements sasl.Server.Next for PLAIN authentication.
func (s *plainServer) Next(response []byte) (challenge []byte, done bool, err error) {
	// PLAIN format: [authzid] NUL authcid NUL passwd
	// authzid is optional and can be empty
	parts := splitNull(response)
	if len(parts) != 3 {
		return nil, false, fmt.Errorf("invalid PLAIN response format")
	}

	identity := string(parts[0])
	username := string(parts[1])
	password := string(parts[2])

	if err := s.authenticator(identity, username, password); err != nil {
		return nil, false, err
	}

	return nil, true, nil
}

// loginServer implements the sasl.Server interface for LOGIN authentication.
type loginServer struct {
	authenticator func(username, password string) error
	state         int
	username      string
}

const (
	loginStateUsername = iota
	loginStatePassword
)

// Next implements sasl.Server.Next for LOGIN authentication.
func (s *loginServer) Next(response []byte) (challenge []byte, done bool, err error) {
	switch s.state {
	case loginStateUsername:
		// First response contains username (may be sent with initial response)
		if len(response) == 0 {
			// Send "Username:" challenge
			return []byte("Username:"), false, nil
		}
		s.username = string(response)
		s.state = loginStatePassword
		// Send "Password:" challenge
		return []byte("Password:"), false, nil

	case loginStatePassword:
		// Second response contains password
		password := string(response)

		if err := s.authenticator(s.username, password); err != nil {
			return nil, false, err
		}

		return nil, true, nil
	}

	return nil, false, fmt.Errorf("unexpected LOGIN state")
}

// splitNull splits a byte slice by null bytes.
func splitNull(data []byte) [][]byte {
	var parts [][]byte
	start := 0
	for i, b := range data {
		if b == 0 {
			parts = append(parts, data[start:i])
			start = i + 1
		}
	}
	parts = append(parts, data[start:])
	return parts
}

// NewPlainServer creates a new PLAIN SASL server.
func NewPlainServer(authenticator func(identity, username, password string) error) sasl.Server {
	return &plainServer{authenticator: authenticator}
}

// NewLoginServer creates a new LOGIN SASL server.
func NewLoginServer(authenticator func(username, password string) error) sasl.Server {
	return &loginServer{
		authenticator: authenticator,
		state:         loginStateUsername,
	}
}

// constantTimeCompare performs a constant-time comparison of two strings.
// This is used for comparing usernames to prevent timing attacks.
func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
