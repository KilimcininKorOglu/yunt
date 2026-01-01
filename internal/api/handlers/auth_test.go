package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/service"
)

// mockUserRepository is a mock implementation of UserRepository for testing.
type mockUserRepository struct {
	users       map[domain.ID]*domain.User
	usersByName map[string]*domain.User
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

// testSetup creates a test environment with Echo, AuthService, and handlers.
func testSetup() (*echo.Echo, *service.AuthService, *mockUserRepository) {
	cfg := config.AuthConfig{
		JWTSecret:         "test-secret-key-for-testing-purposes",
		JWTExpiration:     15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
		BCryptCost:        bcrypt.MinCost,
	}
	userRepo := newMockUserRepository()
	sessionStore := service.NewInMemorySessionStore()
	authService := service.NewAuthService(cfg, userRepo, sessionStore)

	e := echo.New()
	handler := NewAuthHandler(authService)
	v1 := e.Group("/api/v1")
	handler.RegisterRoutes(v1)

	return e, authService, userRepo
}

func TestAuthHandler_Login(t *testing.T) {
	e, _, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	tests := []struct {
		name           string
		body           map[string]string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:           "successful login",
			body:           map[string]string{"username": "testuser", "password": "password123"},
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name:           "wrong password",
			body:           map[string]string{"username": "testuser", "password": "wrongpassword"},
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  false,
		},
		{
			name:           "unknown user",
			body:           map[string]string{"username": "unknownuser", "password": "password123"},
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  false,
		},
		{
			name:           "missing username",
			body:           map[string]string{"password": "password123"},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse:  false,
		},
		{
			name:           "missing password",
			body:           map[string]string{"username": "testuser"},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if tc.checkResponse {
				var resp api.Response
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				if !resp.Success {
					t.Error("Expected success to be true")
				}
				if resp.Data == nil {
					t.Error("Expected data to be present")
				}
			}
		})
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	e, authService, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// First login to get tokens
	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}
	loginResponse, _ := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")

	tests := []struct {
		name           string
		body           map[string]string
		expectedStatus int
	}{
		{
			name:           "successful refresh",
			body:           map[string]string{"refreshToken": loginResponse.Tokens.RefreshToken},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid refresh token",
			body:           map[string]string{"refreshToken": "invalid-token"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing refresh token",
			body:           map[string]string{},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "using access token as refresh token",
			body:           map[string]string{"refreshToken": loginResponse.Tokens.AccessToken},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	e, authService, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// First login to get tokens
	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}
	loginResponse, _ := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "successful logout",
			token:          loginResponse.Tokens.AccessToken,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
			if tc.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAuthHandler_LogoutAll(t *testing.T) {
	e, authService, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get tokens
	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}
	loginResponse, _ := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "successful logout all",
			token:          loginResponse.Tokens.AccessToken,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)
			if tc.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAuthHandler_GetCurrentUser(t *testing.T) {
	e, authService, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get tokens
	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}
	loginResponse, _ := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		checkUser      bool
	}{
		{
			name:           "get current user",
			token:          loginResponse.Tokens.AccessToken,
			expectedStatus: http.StatusOK,
			checkUser:      true,
		},
		{
			name:           "missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			checkUser:      false,
		},
		{
			name:           "invalid token",
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			checkUser:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			if tc.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if tc.checkUser {
				var resp api.Response
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				if !resp.Success {
					t.Error("Expected success to be true")
				}
				// Check that user info is present
				data, ok := resp.Data.(map[string]interface{})
				if !ok {
					t.Fatal("Expected data to be a map")
				}
				if data["username"] != "testuser" {
					t.Errorf("Expected username 'testuser', got '%v'", data["username"])
				}
			}
		})
	}
}

func TestAuthHandler_InvalidJSON(t *testing.T) {
	e, _, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAuthHandler_TokenExpired(t *testing.T) {
	// This test would require a way to create an expired token
	// For now, we'll skip this as it would require mocking time
	t.Skip("Requires time mocking to test expired tokens")
}

func TestAuthHandler_SessionRevoked(t *testing.T) {
	e, authService, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get tokens
	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}
	loginResponse, _ := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")

	// Get session ID from token
	claims, _ := authService.ValidateToken(loginResponse.Tokens.AccessToken)

	// Logout to revoke the session
	_ = authService.Logout(ctx, claims.SessionID)

	// Try to use the token after logout
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+loginResponse.Tokens.AccessToken)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d after session revoked, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthHandler_MultipleLogins(t *testing.T) {
	e, authService, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}

	// Login multiple times
	var tokens []string
	for i := 0; i < 3; i++ {
		response, err := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")
		if err != nil {
			t.Fatalf("Login %d failed: %v", i, err)
		}
		tokens = append(tokens, response.Tokens.AccessToken)
	}

	// All tokens should be valid
	for i, token := range tokens {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Token %d should be valid, got status %d", i, rec.Code)
		}
	}

	// Logout all should invalidate all tokens
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tokens[0])
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Logout all failed with status %d", rec.Code)
	}

	// All tokens should now be invalid
	for i, token := range tokens {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Token %d should be invalid after logout all, got status %d", i, rec.Code)
		}
	}
}

func TestAuthHandler_RefreshTokenRotation(t *testing.T) {
	e, authService, userRepo := testSetup()
	testUser := createTestUser("user-1", "testuser", "password123")
	userRepo.addUser(testUser)

	// Login to get initial tokens
	ctx := context.Background()
	loginInput := &domain.LoginInput{Username: "testuser", Password: "password123"}
	loginResponse, _ := authService.Login(ctx, loginInput, "test-agent", "127.0.0.1")

	// Refresh the token
	refreshBody, _ := json.Marshal(map[string]string{"refreshToken": loginResponse.Tokens.RefreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("First refresh failed with status %d", rec.Code)
	}

	// Try to use the old refresh token again (should fail due to rotation)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Old refresh token should be invalid after rotation, got status %d", rec.Code)
	}
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
	cfg := config.AuthConfig{
		JWTSecret:         "test-secret",
		JWTExpiration:     15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
		BCryptCost:        bcrypt.MinCost,
	}
	sessionStore := service.NewInMemorySessionStore()
	authService := service.NewAuthService(cfg, newMockUserRepository(), sessionStore)
	handler := NewAuthHandler(authService)

	e := echo.New()
	v1 := e.Group("/api/v1")
	handler.RegisterRoutes(v1)

	// Verify routes are registered by making requests
	routes := e.Routes()
	expectedRoutes := map[string]string{
		"POST:/api/v1/auth/login":      "login",
		"POST:/api/v1/auth/refresh":    "refresh",
		"POST:/api/v1/auth/logout":     "logout",
		"POST:/api/v1/auth/logout-all": "logout-all",
		"GET:/api/v1/auth/me":          "me",
	}

	for _, route := range routes {
		key := route.Method + ":" + route.Path
		delete(expectedRoutes, key)
	}

	if len(expectedRoutes) > 0 {
		t.Errorf("Missing routes: %v", expectedRoutes)
	}
}

func TestGetClaimsFromEchoContext(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No claims set
	claims := middleware.GetClaims(c)
	if claims != nil {
		t.Error("GetClaims() should return nil when no claims set")
	}

	// Set claims
	testClaims := &domain.TokenClaims{
		UserID:   domain.ID("user-1"),
		Username: "testuser",
		Email:    "test@example.com",
		Role:     domain.RoleUser,
	}
	c.Set("claims", testClaims)

	claims = middleware.GetClaims(c)
	if claims == nil {
		t.Fatal("GetClaims() should return claims when set")
	}
	if claims.Username != "testuser" {
		t.Errorf("GetClaims() returned wrong username: %s", claims.Username)
	}
}
