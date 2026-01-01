package service

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"yunt/internal/config"
	"yunt/internal/domain"
)

// mockUserRepository is a mock implementation of UserRepository for testing.
type mockUserRepository struct {
	users         map[domain.ID]*domain.User
	usersByName   map[string]*domain.User
	lastLoginTime time.Time
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:       make(map[domain.ID]*domain.User),
		usersByName: make(map[string]*domain.User),
	}
}

func (r *mockUserRepository) addUser(user *domain.User) {
	r.users[user.ID] = user
	r.usersByName[user.Username] = user
}

func (r *mockUserRepository) GetByID(_ context.Context, id domain.ID) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domain.NewNotFoundError("user", id.String())
	}
	return user, nil
}

func (r *mockUserRepository) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	user, ok := r.usersByName[username]
	if !ok {
		return nil, domain.NewNotFoundError("user", username)
	}
	return user, nil
}

func (r *mockUserRepository) UpdateLastLogin(_ context.Context, _ domain.ID) error {
	r.lastLoginTime = time.Now()
	return nil
}

// createTestUser creates a user with a hashed password for testing.
func createTestUser(id, username, password string) *domain.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return &domain.User{
		ID:           domain.ID(id),
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: string(hash),
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
}

// newTestAuthService creates an AuthService for testing.
func newTestAuthService() (*AuthService, *mockUserRepository, *InMemorySessionStore) {
	cfg := config.AuthConfig{
		JWTSecret:         "test-secret-key-for-testing-purposes",
		JWTExpiration:     15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
		BCryptCost:        bcrypt.MinCost,
	}
	userRepo := newMockUserRepository()
	sessionStore := NewInMemorySessionStore()
	authService := NewAuthService(cfg, userRepo, sessionStore)
	return authService, userRepo, sessionStore
}

func TestAuthService_Login(t *testing.T) {
	authService, userRepo, _ := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	tests := []struct {
		name      string
		input     *domain.LoginInput
		wantErr   bool
		errReason string
	}{
		{
			name:    "successful login",
			input:   &domain.LoginInput{Username: "testuser", Password: "password123"},
			wantErr: false,
		},
		{
			name:      "wrong password",
			input:     &domain.LoginInput{Username: "testuser", Password: "wrongpassword"},
			wantErr:   true,
			errReason: "invalid credentials",
		},
		{
			name:      "unknown user",
			input:     &domain.LoginInput{Username: "unknownuser", Password: "password123"},
			wantErr:   true,
			errReason: "invalid credentials",
		},
		{
			name:      "empty username",
			input:     &domain.LoginInput{Username: "", Password: "password123"},
			wantErr:   true,
			errReason: "validation",
		},
		{
			name:      "empty password",
			input:     &domain.LoginInput{Username: "testuser", Password: ""},
			wantErr:   true,
			errReason: "validation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := authService.Login(ctx, tc.input, "test-agent", "127.0.0.1")

			if (err != nil) != tc.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if response == nil {
					t.Error("Login() returned nil response for successful login")
					return
				}
				if response.User == nil {
					t.Error("Login() response.User is nil")
				}
				if response.Tokens == nil {
					t.Error("Login() response.Tokens is nil")
				}
				if response.Tokens.AccessToken == "" {
					t.Error("Login() response.Tokens.AccessToken is empty")
				}
				if response.Tokens.RefreshToken == "" {
					t.Error("Login() response.Tokens.RefreshToken is empty")
				}
				if response.User.Username != tc.input.Username {
					t.Errorf("Login() response.User.Username = %v, want %v", response.User.Username, tc.input.Username)
				}
			}
		})
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	authService, userRepo, _ := newTestAuthService()
	inactiveUser := createTestUser("user-inactive", "inactiveuser", "password123")
	inactiveUser.Status = domain.StatusInactive
	userRepo.addUser(inactiveUser)

	ctx := context.Background()
	input := &domain.LoginInput{Username: "inactiveuser", Password: "password123"}
	_, err := authService.Login(ctx, input, "test-agent", "127.0.0.1")

	if err == nil {
		t.Error("Login() should fail for inactive user")
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	authService, userRepo, _ := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get tokens
	ctx := context.Background()
	input := &domain.LoginInput{Username: "testuser", Password: "password123"}
	response, err := authService.Login(ctx, input, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	// Validate access token
	claims, err := authService.ValidateToken(response.Tokens.AccessToken)
	if err != nil {
		t.Errorf("ValidateToken() error = %v", err)
	}
	if claims == nil {
		t.Fatal("ValidateToken() returned nil claims")
	}
	if claims.UserID != testUser.ID {
		t.Errorf("ValidateToken() claims.UserID = %v, want %v", claims.UserID, testUser.ID)
	}
	if claims.TokenType != domain.TokenTypeAccess {
		t.Errorf("ValidateToken() claims.TokenType = %v, want %v", claims.TokenType, domain.TokenTypeAccess)
	}

	// Validate refresh token
	refreshClaims, err := authService.ValidateToken(response.Tokens.RefreshToken)
	if err != nil {
		t.Errorf("ValidateToken() error = %v", err)
	}
	if refreshClaims.TokenType != domain.TokenTypeRefresh {
		t.Errorf("ValidateToken() refreshClaims.TokenType = %v, want %v", refreshClaims.TokenType, domain.TokenTypeRefresh)
	}

	// Validate invalid token
	_, err = authService.ValidateToken("invalid-token")
	if err == nil {
		t.Error("ValidateToken() should fail for invalid token")
	}
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	authService, userRepo, _ := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get tokens
	ctx := context.Background()
	input := &domain.LoginInput{Username: "testuser", Password: "password123"}
	response, err := authService.Login(ctx, input, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	// Validate access token
	claims, err := authService.ValidateAccessToken(ctx, response.Tokens.AccessToken)
	if err != nil {
		t.Errorf("ValidateAccessToken() error = %v", err)
	}
	if claims == nil {
		t.Fatal("ValidateAccessToken() returned nil claims")
	}

	// Try to validate refresh token as access token (should fail)
	_, err = authService.ValidateAccessToken(ctx, response.Tokens.RefreshToken)
	if err == nil {
		t.Error("ValidateAccessToken() should fail for refresh token")
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	authService, userRepo, _ := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get tokens
	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}
	response, err := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	// Refresh token
	refreshInput := &domain.RefreshTokenInput{RefreshToken: response.Tokens.RefreshToken}
	newResponse, err := authService.RefreshToken(ctx, refreshInput)
	if err != nil {
		t.Errorf("RefreshToken() error = %v", err)
	}
	if newResponse == nil {
		t.Fatal("RefreshToken() returned nil response")
	}
	if newResponse.Tokens.AccessToken == "" {
		t.Error("RefreshToken() returned empty access token")
	}
	if newResponse.Tokens.RefreshToken == "" {
		t.Error("RefreshToken() returned empty refresh token")
	}
	// New tokens should be different
	if newResponse.Tokens.AccessToken == response.Tokens.AccessToken {
		t.Error("RefreshToken() should return a new access token")
	}
	if newResponse.Tokens.RefreshToken == response.Tokens.RefreshToken {
		t.Error("RefreshToken() should return a new refresh token")
	}

	// Old refresh token should be invalid now (token rotation)
	_, err = authService.RefreshToken(ctx, refreshInput)
	if err == nil {
		t.Error("RefreshToken() should fail with old refresh token after rotation")
	}
}

func TestAuthService_RefreshToken_WithAccessToken(t *testing.T) {
	authService, userRepo, _ := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get tokens
	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}
	response, err := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	// Try to refresh with access token (should fail)
	refreshInput := &domain.RefreshTokenInput{RefreshToken: response.Tokens.AccessToken}
	_, err = authService.RefreshToken(ctx, refreshInput)
	if err == nil {
		t.Error("RefreshToken() should fail when using access token")
	}
}

func TestAuthService_Logout(t *testing.T) {
	authService, userRepo, sessionStore := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get tokens
	ctx := context.Background()
	input := &domain.LoginInput{Username: "testuser", Password: "password123"}
	response, err := authService.Login(ctx, input, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	// Get session ID from token
	claims, err := authService.ValidateToken(response.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	// Logout
	err = authService.Logout(ctx, claims.SessionID)
	if err != nil {
		t.Errorf("Logout() error = %v", err)
	}

	// Session should be deleted
	_, err = sessionStore.Get(ctx, claims.SessionID)
	if err == nil {
		t.Error("Logout() should delete the session")
	}

	// Token should be invalid now
	_, err = authService.ValidateAccessToken(ctx, response.Tokens.AccessToken)
	if err == nil {
		t.Error("ValidateAccessToken() should fail after logout")
	}
}

func TestAuthService_LogoutAll(t *testing.T) {
	authService, userRepo, sessionStore := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	ctx := context.Background()
	input := &domain.LoginInput{Username: "testuser", Password: "password123"}

	// Login multiple times to create multiple sessions
	var sessions []string
	for i := 0; i < 3; i++ {
		response, err := authService.Login(ctx, input, "test-agent", "127.0.0.1")
		if err != nil {
			t.Fatalf("Login() failed: %v", err)
		}
		claims, _ := authService.ValidateToken(response.Tokens.AccessToken)
		sessions = append(sessions, claims.SessionID)
	}

	// Logout all
	err := authService.LogoutAll(ctx, testUser.ID)
	if err != nil {
		t.Errorf("LogoutAll() error = %v", err)
	}

	// All sessions should be deleted
	for _, sessionID := range sessions {
		_, err := sessionStore.Get(ctx, sessionID)
		if err == nil {
			t.Errorf("LogoutAll() should delete session %s", sessionID)
		}
	}
}

func TestAuthService_HashPassword(t *testing.T) {
	authService, _, _ := newTestAuthService()

	password := "securePassword123!"
	hash, err := authService.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Hash should be different from password
	if hash == password {
		t.Error("HashPassword() should return a different value from password")
	}

	// Hash should be valid bcrypt hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		t.Errorf("HashPassword() produced invalid hash: %v", err)
	}

	// Different passwords should produce different hashes
	hash2, _ := authService.HashPassword("differentPassword")
	if hash == hash2 {
		t.Error("HashPassword() should produce different hashes for different passwords")
	}
}

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()
	ctx := context.Background()

	// Create session
	session := domain.NewSession("session-1", domain.ID("user-1"), "hash-1", time.Now().Add(time.Hour))
	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get session
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.ID != session.ID {
		t.Errorf("Get() returned session with ID = %v, want %v", retrieved.ID, session.ID)
	}

	// Update session
	session.UserAgent = "Updated Agent"
	err = store.Update(ctx, session)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, _ := store.Get(ctx, session.ID)
	if updated.UserAgent != "Updated Agent" {
		t.Errorf("Update() did not update UserAgent")
	}

	// Touch session
	oldLastUsed := updated.LastUsedAt.Time
	time.Sleep(time.Millisecond)
	err = store.Touch(ctx, session.ID)
	if err != nil {
		t.Fatalf("Touch() error = %v", err)
	}
	touched, _ := store.Get(ctx, session.ID)
	if !touched.LastUsedAt.Time.After(oldLastUsed) {
		t.Error("Touch() should update LastUsedAt")
	}

	// Delete session
	err = store.Delete(ctx, session.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err = store.Get(ctx, session.ID)
	if err == nil {
		t.Error("Get() should fail after Delete()")
	}

	// Create multiple sessions for same user
	session1 := domain.NewSession("session-a", domain.ID("user-2"), "hash-a", time.Now().Add(time.Hour))
	session2 := domain.NewSession("session-b", domain.ID("user-2"), "hash-b", time.Now().Add(time.Hour))
	_ = store.Create(ctx, session1)
	_ = store.Create(ctx, session2)

	// Delete by user ID
	err = store.DeleteByUserID(ctx, domain.ID("user-2"))
	if err != nil {
		t.Fatalf("DeleteByUserID() error = %v", err)
	}
	_, err = store.Get(ctx, session1.ID)
	if err == nil {
		t.Error("DeleteByUserID() should delete session-a")
	}
	_, err = store.Get(ctx, session2.ID)
	if err == nil {
		t.Error("DeleteByUserID() should delete session-b")
	}
}
