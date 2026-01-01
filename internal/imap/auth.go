package imap

import (
	"context"
	"errors"
	"fmt"

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

// NewInternalError creates an error for internal failures.
func NewInternalError(err error) *AuthenticationError {
	return &AuthenticationError{
		Reason:  AuthReasonInternalError,
		Message: fmt.Sprintf("Internal error: %v", err),
	}
}

// Authenticator handles user authentication for IMAP.
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
func (a *Authenticator) Authenticate(ctx context.Context, username, password string) (*AuthResult, error) {
	if username == "" || password == "" {
		return nil, NewInvalidCredentialsError()
	}

	// Try to find user by username first, then by email
	user, err := a.findUser(ctx, username)
	if err != nil {
		return nil, err
	}

	// Verify the password
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
		return nil, NewInternalError(err)
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

	return nil, NewInternalError(err)
}

// verifyPassword compares a bcrypt hashed password with a plaintext password.
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
