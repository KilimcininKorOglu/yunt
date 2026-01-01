// Package service provides business logic services for the Yunt mail server.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"yunt/internal/config"
	"yunt/internal/domain"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	// GetByID retrieves a user by their ID.
	GetByID(ctx context.Context, id domain.ID) (*domain.User, error)
	// GetByUsername retrieves a user by their username.
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	// UpdateLastLogin updates the user's last login timestamp.
	UpdateLastLogin(ctx context.Context, id domain.ID) error
}

// SessionStore defines the interface for session storage.
type SessionStore interface {
	// Create stores a new session.
	Create(ctx context.Context, session *domain.Session) error
	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (*domain.Session, error)
	// Update updates an existing session.
	Update(ctx context.Context, session *domain.Session) error
	// Delete removes a session.
	Delete(ctx context.Context, id string) error
	// DeleteByUserID removes all sessions for a user.
	DeleteByUserID(ctx context.Context, userID domain.ID) error
	// Touch updates the last used timestamp.
	Touch(ctx context.Context, id string) error
}

// InMemorySessionStore is an in-memory implementation of SessionStore.
// This should be replaced with a persistent store in production.
type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*domain.Session
}

// NewInMemorySessionStore creates a new in-memory session store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*domain.Session),
	}
}

// Create stores a new session.
func (s *InMemorySessionStore) Create(_ context.Context, session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

// Get retrieves a session by ID.
func (s *InMemorySessionStore) Get(_ context.Context, id string) (*domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, domain.NewNotFoundError("session", id)
	}
	return session, nil
}

// Update updates an existing session.
func (s *InMemorySessionStore) Update(_ context.Context, session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[session.ID]; !ok {
		return domain.NewNotFoundError("session", session.ID)
	}
	s.sessions[session.ID] = session
	return nil
}

// Delete removes a session.
func (s *InMemorySessionStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

// DeleteByUserID removes all sessions for a user.
func (s *InMemorySessionStore) DeleteByUserID(_ context.Context, userID domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, id)
		}
	}
	return nil
}

// Touch updates the last used timestamp.
func (s *InMemorySessionStore) Touch(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return domain.NewNotFoundError("session", id)
	}
	session.Touch()
	return nil
}

// AuthService provides authentication-related business logic.
type AuthService struct {
	config       config.AuthConfig
	userRepo     UserRepository
	sessionStore SessionStore
}

// NewAuthService creates a new AuthService.
func NewAuthService(cfg config.AuthConfig, userRepo UserRepository, sessionStore SessionStore) *AuthService {
	return &AuthService{
		config:       cfg,
		userRepo:     userRepo,
		sessionStore: sessionStore,
	}
}

// jwtClaims represents the JWT claims structure.
type jwtClaims struct {
	jwt.RegisteredClaims
	domain.TokenClaims
}

// Login authenticates a user and returns a token pair.
func (s *AuthService) Login(ctx context.Context, input *domain.LoginInput, userAgent, ipAddress string) (*domain.AuthResponse, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, input.Username)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.NewUnauthorizedError("invalid credentials")
		}
		return nil, domain.NewInternalError("failed to retrieve user", err)
	}

	// Check if user is active
	if !user.IsActive() {
		return nil, domain.NewUnauthorizedError("account is not active")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.NewUnauthorizedError("invalid credentials")
	}

	// Generate token pair
	tokenPair, session, err := s.generateTokenPair(user)
	if err != nil {
		return nil, domain.NewInternalError("failed to generate tokens", err)
	}

	// Set session metadata
	session.UserAgent = userAgent
	session.IPAddress = ipAddress

	// Store session
	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, domain.NewInternalError("failed to create session", err)
	}

	// Update last login
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	return &domain.AuthResponse{
		User:   domain.UserInfoFromUser(user),
		Tokens: tokenPair,
	}, nil
}

// RefreshToken generates a new token pair using a valid refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, input *domain.RefreshTokenInput) (*domain.AuthResponse, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Parse and validate refresh token
	claims, err := s.ValidateToken(input.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Verify this is a refresh token
	if claims.TokenType != domain.TokenTypeRefresh {
		return nil, domain.NewInvalidTokenError("not a refresh token")
	}

	// Get session
	session, err := s.sessionStore.Get(ctx, claims.SessionID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.NewInvalidTokenError("session not found")
		}
		return nil, domain.NewInternalError("failed to retrieve session", err)
	}

	// Verify session is valid
	if !session.IsValid() {
		if session.IsRevoked {
			return nil, domain.NewSessionRevokedError(session.ID)
		}
		return nil, domain.NewExpiredTokenError(session.ExpiresAt.Time)
	}

	// Verify token hash matches session
	tokenHash := hashToken(input.RefreshToken)
	if session.RefreshTokenHash != tokenHash {
		// Token reuse detected - revoke session
		session.Revoke()
		_ = s.sessionStore.Update(ctx, session)
		return nil, domain.NewInvalidTokenError("token mismatch")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.NewUnauthorizedError("user not found")
		}
		return nil, domain.NewInternalError("failed to retrieve user", err)
	}

	// Check if user is still active
	if !user.IsActive() {
		return nil, domain.NewUnauthorizedError("account is not active")
	}

	// Delete old session
	_ = s.sessionStore.Delete(ctx, session.ID)

	// Generate new token pair
	tokenPair, newSession, err := s.generateTokenPair(user)
	if err != nil {
		return nil, domain.NewInternalError("failed to generate tokens", err)
	}

	// Preserve session metadata
	newSession.UserAgent = session.UserAgent
	newSession.IPAddress = session.IPAddress

	// Store new session
	if err := s.sessionStore.Create(ctx, newSession); err != nil {
		return nil, domain.NewInternalError("failed to create session", err)
	}

	return &domain.AuthResponse{
		User:   domain.UserInfoFromUser(user),
		Tokens: tokenPair,
	}, nil
}

// Logout invalidates the user's current session.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	session, err := s.sessionStore.Get(ctx, sessionID)
	if err != nil {
		if domain.IsNotFound(err) {
			// Session already deleted, nothing to do
			return nil
		}
		return domain.NewInternalError("failed to retrieve session", err)
	}

	// Revoke and delete session
	session.Revoke()
	if err := s.sessionStore.Delete(ctx, sessionID); err != nil {
		return domain.NewInternalError("failed to delete session", err)
	}

	return nil
}

// LogoutAll invalidates all sessions for a user.
func (s *AuthService) LogoutAll(ctx context.Context, userID domain.ID) error {
	if err := s.sessionStore.DeleteByUserID(ctx, userID); err != nil {
		return domain.NewInternalError("failed to delete sessions", err)
	}
	return nil
}

// ValidateToken parses and validates a JWT token, returning the claims.
func (s *AuthService) ValidateToken(tokenString string) (*domain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.NewInvalidTokenError("invalid signing method")
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		// Check for specific JWT errors
		if err == jwt.ErrTokenExpired {
			return nil, domain.NewExpiredTokenError(time.Now())
		}
		return nil, domain.NewInvalidTokenError(err.Error())
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, domain.NewInvalidTokenError("invalid token claims")
	}

	return &claims.TokenClaims, nil
}

// ValidateAccessToken validates an access token and returns the claims.
func (s *AuthService) ValidateAccessToken(ctx context.Context, tokenString string) (*domain.TokenClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Verify this is an access token
	if claims.TokenType != domain.TokenTypeAccess {
		return nil, domain.NewInvalidTokenError("not an access token")
	}

	// Verify session is still valid
	session, err := s.sessionStore.Get(ctx, claims.SessionID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.NewInvalidTokenError("session not found")
		}
		return nil, domain.NewInternalError("failed to retrieve session", err)
	}

	if !session.IsValid() {
		if session.IsRevoked {
			return nil, domain.NewSessionRevokedError(session.ID)
		}
		return nil, domain.NewExpiredTokenError(session.ExpiresAt.Time)
	}

	// Touch session to update last used time
	_ = s.sessionStore.Touch(ctx, claims.SessionID)

	return claims, nil
}

// GetUserFromToken retrieves the full user from a valid access token.
func (s *AuthService) GetUserFromToken(ctx context.Context, tokenString string) (*domain.User, error) {
	claims, err := s.ValidateAccessToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.NewUnauthorizedError("user not found")
		}
		return nil, domain.NewInternalError("failed to retrieve user", err)
	}

	if !user.IsActive() {
		return nil, domain.NewUnauthorizedError("account is not active")
	}

	return user, nil
}

// HashPassword hashes a plaintext password using bcrypt.
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BCryptCost)
	if err != nil {
		return "", domain.NewInternalError("failed to hash password", err)
	}
	return string(hash), nil
}

// generateTokenPair creates a new access token, refresh token, and session.
func (s *AuthService) generateTokenPair(user *domain.User) (*domain.TokenPair, *domain.Session, error) {
	sessionID := uuid.New().String()
	now := time.Now()
	accessExp := now.Add(s.config.JWTExpiration)
	refreshExp := now.Add(s.config.RefreshExpiration)

	// Create token claims
	baseClaims := domain.TokenClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		SessionID: sessionID,
	}

	// Generate access token
	accessClaims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "yunt",
		},
		TokenClaims: baseClaims,
	}
	accessClaims.TokenType = domain.TokenTypeAccess

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, nil, err
	}

	// Generate refresh token
	refreshClaims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "yunt",
		},
		TokenClaims: baseClaims,
	}
	refreshClaims.TokenType = domain.TokenTypeRefresh

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, nil, err
	}

	// Create session
	tokenHash := hashToken(refreshTokenString)
	session := domain.NewSession(sessionID, user.ID, tokenHash, refreshExp)

	tokenPair := domain.NewTokenPair(accessTokenString, refreshTokenString, accessExp, refreshExp)

	return tokenPair, session, nil
}

// hashToken creates a SHA-256 hash of a token for storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
