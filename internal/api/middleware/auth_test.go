package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

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
func createTestUser(id, username, password string, role domain.UserRole) *domain.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return &domain.User{
		ID:           domain.ID(id),
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: string(hash),
		Role:         role,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
}

// newTestAuthService creates an AuthService for testing.
func newTestAuthService() (*service.AuthService, *mockUserRepository) {
	cfg := config.AuthConfig{
		JWTSecret:         "test-secret-key-for-testing-purposes",
		JWTExpiration:     15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
		BCryptCost:        bcrypt.MinCost,
	}
	userRepo := newMockUserRepository()
	sessionStore := service.NewInMemorySessionStore()
	authService := service.NewAuthService(cfg, userRepo, sessionStore)
	return authService, userRepo
}

func TestAuth_ValidToken(t *testing.T) {
	authService, userRepo := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123", domain.RoleUser)
	userRepo.addUser(testUser)

	// Login to get token
	ctx := context.Background()
	input := &domain.LoginInput{Username: "testuser", Password: "password123"}
	response, err := authService.Login(ctx, input, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	// Create Echo instance with middleware
	e := echo.New()
	e.Use(Auth(authService))

	// Handler that checks for user context
	e.GET("/protected", func(c echo.Context) error {
		claims := GetClaims(c)
		if claims == nil {
			return c.String(http.StatusUnauthorized, "no claims")
		}
		return c.String(http.StatusOK, claims.Username)
	})

	// Make request with valid token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+response.Tokens.AccessToken)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "testuser" {
		t.Errorf("Expected body 'testuser', got '%s'", rec.Body.String())
	}
}

func TestAuth_MissingToken(t *testing.T) {
	authService, _ := newTestAuthService()

	e := echo.New()
	e.Use(Auth(authService))
	e.GET("/protected", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Make request without token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	authService, _ := newTestAuthService()

	e := echo.New()
	e.Use(Auth(authService))
	e.GET("/protected", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Make request with invalid token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer invalid-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuth_InvalidFormat(t *testing.T) {
	authService, _ := newTestAuthService()

	e := echo.New()
	e.Use(Auth(authService))
	e.GET("/protected", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Make request with wrong format
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(echo.HeaderAuthorization, "Basic token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthWithConfig_SkipPaths(t *testing.T) {
	authService, _ := newTestAuthService()

	e := echo.New()
	e.Use(AuthWithConfig(AuthConfig{
		AuthService: authService,
		SkipPaths:   []string{"/public", "/health"},
	}))
	e.GET("/public", func(c echo.Context) error {
		return c.String(http.StatusOK, "public")
	})
	e.GET("/protected", func(c echo.Context) error {
		return c.String(http.StatusOK, "protected")
	})

	// Request to skip path should succeed without token
	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d for skip path, got %d", http.StatusOK, rec.Code)
	}

	// Request to protected path should fail without token
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for protected path, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthWithConfig_Optional(t *testing.T) {
	authService, userRepo := newTestAuthService()
	testUser := createTestUser("user-1", "testuser", "password123", domain.RoleUser)
	userRepo.addUser(testUser)

	e := echo.New()
	e.Use(AuthWithConfig(AuthConfig{
		AuthService: authService,
		Optional:    true,
	}))
	e.GET("/optional", func(c echo.Context) error {
		claims := GetClaims(c)
		if claims == nil {
			return c.String(http.StatusOK, "anonymous")
		}
		return c.String(http.StatusOK, claims.Username)
	})

	// Request without token should succeed with anonymous
	req := httptest.NewRequest(http.MethodGet, "/optional", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "anonymous" {
		t.Errorf("Expected body 'anonymous', got '%s'", rec.Body.String())
	}

	// Request with valid token should succeed with username
	ctx := context.Background()
	input := &domain.LoginInput{Username: "testuser", Password: "password123"}
	response, _ := authService.Login(ctx, input, "test-agent", "127.0.0.1")

	req = httptest.NewRequest(http.MethodGet, "/optional", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+response.Tokens.AccessToken)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "testuser" {
		t.Errorf("Expected body 'testuser', got '%s'", rec.Body.String())
	}
}

func TestRequireRole(t *testing.T) {
	authService, userRepo := newTestAuthService()
	adminUser := createTestUser("admin-1", "adminuser", "password123", domain.RoleAdmin)
	regularUser := createTestUser("user-1", "regularuser", "password123", domain.RoleUser)
	viewerUser := createTestUser("viewer-1", "vieweruser", "password123", domain.RoleViewer)
	userRepo.addUser(adminUser)
	userRepo.addUser(regularUser)
	userRepo.addUser(viewerUser)

	e := echo.New()
	e.Use(Auth(authService))

	// Admin-only endpoint
	e.GET("/admin", func(c echo.Context) error {
		return c.String(http.StatusOK, "admin")
	}, RequireAdmin())

	// User or admin endpoint
	e.GET("/user", func(c echo.Context) error {
		return c.String(http.StatusOK, "user")
	}, RequireUser())

	// Any authenticated user
	e.GET("/viewer", func(c echo.Context) error {
		return c.String(http.StatusOK, "viewer")
	}, RequireViewer())

	ctx := context.Background()

	// Test admin user
	adminLogin := &domain.LoginInput{Username: "adminuser", Password: "password123"}
	adminResponse, _ := authService.Login(ctx, adminLogin, "test-agent", "127.0.0.1")

	// Admin should access all
	for _, path := range []string{"/admin", "/user", "/viewer"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+adminResponse.Tokens.AccessToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Admin should access %s, got status %d", path, rec.Code)
		}
	}

	// Test regular user
	userLogin := &domain.LoginInput{Username: "regularuser", Password: "password123"}
	userResponse, _ := authService.Login(ctx, userLogin, "test-agent", "127.0.0.1")

	// Regular user should not access admin
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+userResponse.Tokens.AccessToken)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Regular user should not access /admin, got status %d", rec.Code)
	}

	// Regular user should access /user and /viewer
	for _, path := range []string{"/user", "/viewer"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+userResponse.Tokens.AccessToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Regular user should access %s, got status %d", path, rec.Code)
		}
	}

	// Test viewer user
	viewerLogin := &domain.LoginInput{Username: "vieweruser", Password: "password123"}
	viewerResponse, _ := authService.Login(ctx, viewerLogin, "test-agent", "127.0.0.1")

	// Viewer should not access /admin or /user
	for _, path := range []string{"/admin", "/user"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+viewerResponse.Tokens.AccessToken)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Viewer should not access %s, got status %d", path, rec.Code)
		}
	}

	// Viewer should access /viewer
	req = httptest.NewRequest(http.MethodGet, "/viewer", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+viewerResponse.Tokens.AccessToken)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Viewer should access /viewer, got status %d", rec.Code)
	}
}

func TestGetClaims(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No claims set
	claims := GetClaims(c)
	if claims != nil {
		t.Error("GetClaims() should return nil when no claims set")
	}

	// Set claims
	testClaims := &domain.TokenClaims{
		UserID:   domain.ID("user-1"),
		Username: "testuser",
	}
	c.Set("claims", testClaims)

	claims = GetClaims(c)
	if claims == nil {
		t.Fatal("GetClaims() should return claims when set")
	}
	if claims.Username != "testuser" {
		t.Errorf("GetClaims() returned wrong username: %s", claims.Username)
	}
}

func TestGetUserID(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No user ID set
	userID := GetUserID(c)
	if userID != "" {
		t.Error("GetUserID() should return empty when no user ID set")
	}

	// Set user ID
	c.Set("userId", domain.ID("user-1"))
	userID = GetUserID(c)
	if userID != domain.ID("user-1") {
		t.Errorf("GetUserID() returned wrong ID: %s", userID)
	}
}

func TestGetUsername(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No username set
	username := GetUsername(c)
	if username != "" {
		t.Error("GetUsername() should return empty when no username set")
	}

	// Set username
	c.Set("username", "testuser")
	username = GetUsername(c)
	if username != "testuser" {
		t.Errorf("GetUsername() returned wrong username: %s", username)
	}
}

func TestGetUserRole(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No role set
	role := GetUserRole(c)
	if role != "" {
		t.Error("GetUserRole() should return empty when no role set")
	}

	// Set role
	c.Set("userRole", domain.RoleAdmin)
	role = GetUserRole(c)
	if role != domain.RoleAdmin {
		t.Errorf("GetUserRole() returned wrong role: %s", role)
	}
}

func TestGetSessionID(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No session ID set
	sessionID := GetSessionID(c)
	if sessionID != "" {
		t.Error("GetSessionID() should return empty when no session ID set")
	}

	// Set session ID
	c.Set("sessionId", "session-123")
	sessionID = GetSessionID(c)
	if sessionID != "session-123" {
		t.Errorf("GetSessionID() returned wrong session ID: %s", sessionID)
	}
}

func TestIsAuthenticated(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No claims set
	if IsAuthenticated(c) {
		t.Error("IsAuthenticated() should return false when no claims set")
	}

	// Set claims
	c.Set("claims", &domain.TokenClaims{})
	if !IsAuthenticated(c) {
		t.Error("IsAuthenticated() should return true when claims set")
	}
}

func TestIsAdmin(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No role set
	if IsAdmin(c) {
		t.Error("IsAdmin() should return false when no role set")
	}

	// Set non-admin role
	c.Set("userRole", domain.RoleUser)
	if IsAdmin(c) {
		t.Error("IsAdmin() should return false for non-admin role")
	}

	// Set admin role
	c.Set("userRole", domain.RoleAdmin)
	if !IsAdmin(c) {
		t.Error("IsAdmin() should return true for admin role")
	}
}

func TestHasRole(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// No role set
	if HasRole(c, domain.RoleAdmin) {
		t.Error("HasRole() should return false when no role set")
	}

	// Set user role
	c.Set("userRole", domain.RoleUser)

	if !HasRole(c, domain.RoleUser) {
		t.Error("HasRole() should return true for matching role")
	}
	if !HasRole(c, domain.RoleAdmin, domain.RoleUser) {
		t.Error("HasRole() should return true when role is in list")
	}
	if HasRole(c, domain.RoleAdmin) {
		t.Error("HasRole() should return false for non-matching role")
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	ctx := context.Background()

	// No claims in context
	claims := GetClaimsFromContext(ctx)
	if claims != nil {
		t.Error("GetClaimsFromContext() should return nil when no claims in context")
	}

	// Add claims to context
	testClaims := &domain.TokenClaims{
		UserID:   domain.ID("user-1"),
		Username: "testuser",
	}
	ctx = context.WithValue(ctx, claimsContextKey, testClaims)

	claims = GetClaimsFromContext(ctx)
	if claims == nil {
		t.Fatal("GetClaimsFromContext() should return claims when set")
	}
	if claims.Username != "testuser" {
		t.Errorf("GetClaimsFromContext() returned wrong username: %s", claims.Username)
	}
}

func TestGetSessionIDFromContext(t *testing.T) {
	ctx := context.Background()

	// No session ID in context
	sessionID := GetSessionIDFromContext(ctx)
	if sessionID != "" {
		t.Error("GetSessionIDFromContext() should return empty when no session ID in context")
	}

	// Add session ID to context
	ctx = context.WithValue(ctx, sessionIDContextKey, "session-123")

	sessionID = GetSessionIDFromContext(ctx)
	if sessionID != "session-123" {
		t.Errorf("GetSessionIDFromContext() returned wrong session ID: %s", sessionID)
	}
}
