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

	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// mockUsersRepo implements repository.UserRepository for handler tests.
type mockUsersRepo struct {
	users       map[domain.ID]*domain.User
	usersByName map[string]*domain.User
	usersByMail map[string]*domain.User
}

var _ repository.UserRepository = (*mockUsersRepo)(nil)

func newMockUsersRepo() *mockUsersRepo {
	return &mockUsersRepo{
		users:       make(map[domain.ID]*domain.User),
		usersByName: make(map[string]*domain.User),
		usersByMail: make(map[string]*domain.User),
	}
}

func (r *mockUsersRepo) addUser(u *domain.User) {
	r.users[u.ID] = u
	r.usersByName[u.Username] = u
	r.usersByMail[u.Email] = u
}

func (r *mockUsersRepo) GetByID(_ context.Context, id domain.ID) (*domain.User, error) {
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user", id.String())
}

func (r *mockUsersRepo) GetByUsername(_ context.Context, name string) (*domain.User, error) {
	if u, ok := r.usersByName[name]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user", name)
}

func (r *mockUsersRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	if u, ok := r.usersByMail[email]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user", email)
}

func (r *mockUsersRepo) List(_ context.Context, _ *repository.UserFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	items := make([]*domain.User, 0, len(r.users))
	for _, u := range r.users {
		items = append(items, u)
	}
	return &repository.ListResult[*domain.User]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockUsersRepo) Create(_ context.Context, u *domain.User) error {
	if _, ok := r.usersByName[u.Username]; ok {
		return domain.NewAlreadyExistsError("user", "username", u.Username)
	}
	r.addUser(u)
	return nil
}

func (r *mockUsersRepo) Update(_ context.Context, u *domain.User) error {
	r.addUser(u)
	return nil
}

func (r *mockUsersRepo) Delete(_ context.Context, id domain.ID) error {
	if _, ok := r.users[id]; !ok {
		return domain.NewNotFoundError("user", id.String())
	}
	delete(r.users, id)
	return nil
}

func (r *mockUsersRepo) SoftDelete(_ context.Context, id domain.ID) error {
	if _, ok := r.users[id]; !ok {
		return domain.NewNotFoundError("user", id.String())
	}
	return nil
}

func (r *mockUsersRepo) Restore(_ context.Context, _ domain.ID) error { return nil }

func (r *mockUsersRepo) Exists(_ context.Context, id domain.ID) (bool, error) {
	_, ok := r.users[id]
	return ok, nil
}

func (r *mockUsersRepo) ExistsByUsername(_ context.Context, name string) (bool, error) {
	_, ok := r.usersByName[name]
	return ok, nil
}

func (r *mockUsersRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, ok := r.usersByMail[email]
	return ok, nil
}

func (r *mockUsersRepo) Count(_ context.Context, _ *repository.UserFilter) (int64, error) {
	return int64(len(r.users)), nil
}

func (r *mockUsersRepo) CountByRole(_ context.Context) (map[domain.UserRole]int64, error) {
	result := make(map[domain.UserRole]int64)
	for _, u := range r.users {
		result[u.Role]++
	}
	return result, nil
}

func (r *mockUsersRepo) CountByStatus(_ context.Context) (map[domain.UserStatus]int64, error) {
	result := make(map[domain.UserStatus]int64)
	for _, u := range r.users {
		result[u.Status]++
	}
	return result, nil
}

func (r *mockUsersRepo) UpdatePassword(_ context.Context, id domain.ID, hash string) error {
	if u, ok := r.users[id]; ok {
		u.PasswordHash = hash
		return nil
	}
	return domain.NewNotFoundError("user", id.String())
}

func (r *mockUsersRepo) UpdateLastLogin(_ context.Context, _ domain.ID) error { return nil }

func (r *mockUsersRepo) UpdateStatus(_ context.Context, id domain.ID, s domain.UserStatus) error {
	if u, ok := r.users[id]; ok {
		u.Status = s
		return nil
	}
	return domain.NewNotFoundError("user", id.String())
}

func (r *mockUsersRepo) UpdateRole(_ context.Context, id domain.ID, role domain.UserRole) error {
	if u, ok := r.users[id]; ok {
		u.Role = role
		return nil
	}
	return domain.NewNotFoundError("user", id.String())
}

func (r *mockUsersRepo) GetActiveUsers(_ context.Context, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	var items []*domain.User
	for _, u := range r.users {
		if u.Status == domain.StatusActive {
			items = append(items, u)
		}
	}
	return &repository.ListResult[*domain.User]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockUsersRepo) GetAdmins(_ context.Context) ([]*domain.User, error) {
	var admins []*domain.User
	for _, u := range r.users {
		if u.Role == domain.RoleAdmin {
			admins = append(admins, u)
		}
	}
	return admins, nil
}

func (r *mockUsersRepo) Search(_ context.Context, _ string, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	items := make([]*domain.User, 0, len(r.users))
	for _, u := range r.users {
		items = append(items, u)
	}
	return &repository.ListResult[*domain.User]{Items: items, Total: int64(len(items))}, nil
}

func (r *mockUsersRepo) BulkUpdateStatus(_ context.Context, ids []domain.ID, _ domain.UserStatus) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockUsersRepo) BulkDelete(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	op.Succeeded = int64(len(ids))
	return op, nil
}

func (r *mockUsersRepo) GetUsersCreatedBetween(_ context.Context, _ *repository.DateRangeFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return &repository.ListResult[*domain.User]{}, nil
}

func (r *mockUsersRepo) GetUsersWithRecentLogin(_ context.Context, _ int, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return &repository.ListResult[*domain.User]{}, nil
}

func (r *mockUsersRepo) GetInactiveUsers(_ context.Context, _ int, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return &repository.ListResult[*domain.User]{}, nil
}

// --- Test helpers ---

func setupUsersTest() (*echo.Echo, *service.AuthService, *service.UserService, *mockUsersRepo) {
	cfg := config.AuthConfig{
		JWTSecret:         "test-secret-key-for-testing-purposes",
		JWTExpiration:     15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
		BCryptCost:        bcrypt.MinCost,
	}
	repo := newMockUsersRepo()
	sessionStore := service.NewInMemorySessionStore()
	authSvc := service.NewAuthService(cfg, repo, sessionStore)
	userSvc := service.NewUserService(cfg, repo)

	e := echo.New()
	v1 := e.Group("/api/v1")

	authHandler := NewAuthHandler(authSvc)
	authHandler.RegisterRoutes(v1)

	usersHandler := NewUsersHandler(userSvc, authSvc)
	usersHandler.RegisterRoutes(v1, authSvc)

	return e, authSvc, userSvc, repo
}

func loginForToken(t *testing.T, e *echo.Echo, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login failed with status %d: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &raw)
	data := raw["data"].(map[string]interface{})
	tokens := data["tokens"].(map[string]interface{})
	return tokens["accessToken"].(string)
}

func makeAuthReq(method, path, token string, body interface{}) *http.Request {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	return req
}

func makeAdmin(id, username, password string) *domain.User {
	u := createTestUser(id, username, password)
	u.Role = domain.RoleAdmin
	return u
}

// --- Tests ---

func TestUsersHandler_ListUsers(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	admin := makeAdmin("admin-1", "admin", "password123")
	repo.addUser(admin)

	token := loginForToken(t, e, "admin", "password123")
	req := makeAuthReq(http.MethodGet, "/api/v1/users", token, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_ListUsers_Unauthorized(t *testing.T) {
	e, _, _, _ := setupUsersTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestUsersHandler_CreateUser(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	admin := makeAdmin("admin-1", "admin", "password123")
	repo.addUser(admin)

	token := loginForToken(t, e, "admin", "password123")
	body := map[string]string{
		"username": "newuser",
		"email":    "new@example.com",
		"password": "securepass123",
	}

	req := makeAuthReq(http.MethodPost, "/api/v1/users", token, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Errorf("expected 201 or 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_GetUser(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	admin := makeAdmin("admin-1", "admin", "password123")
	repo.addUser(admin)

	target := createTestUser("user-2", "target", "pass123")
	repo.addUser(target)

	token := loginForToken(t, e, "admin", "password123")
	req := makeAuthReq(http.MethodGet, "/api/v1/users/user-2", token, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_GetUser_NotFound(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	admin := makeAdmin("admin-1", "admin", "password123")
	repo.addUser(admin)

	token := loginForToken(t, e, "admin", "password123")
	req := makeAuthReq(http.MethodGet, "/api/v1/users/nonexistent", token, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_DeleteUser(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	admin := makeAdmin("admin-1", "admin", "password123")
	repo.addUser(admin)

	target := createTestUser("user-2", "target", "pass123")
	repo.addUser(target)

	token := loginForToken(t, e, "admin", "password123")
	req := makeAuthReq(http.MethodDelete, "/api/v1/users/user-2", token, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Errorf("expected 200 or 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_GetMyProfile(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	user := createTestUser("user-1", "testuser", "password123")
	repo.addUser(user)

	token := loginForToken(t, e, "testuser", "password123")
	req := makeAuthReq(http.MethodGet, "/api/v1/users/me/profile", token, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_UpdateUser(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	admin := makeAdmin("admin-1", "admin", "password123")
	repo.addUser(admin)

	target := createTestUser("user-2", "target", "pass123")
	repo.addUser(target)

	token := loginForToken(t, e, "admin", "password123")
	body := map[string]interface{}{
		"displayName": "Updated Name",
	}

	req := makeAuthReq(http.MethodPut, "/api/v1/users/user-2", token, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_GetUserStats(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	admin := makeAdmin("admin-1", "admin", "password123")
	repo.addUser(admin)

	token := loginForToken(t, e, "admin", "password123")
	req := makeAuthReq(http.MethodGet, "/api/v1/users/stats", token, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_ChangeMyPassword(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	user := createTestUser("user-1", "testuser", "password123")
	repo.addUser(user)

	token := loginForToken(t, e, "testuser", "password123")
	body := map[string]string{
		"currentPassword": "password123",
		"newPassword":     "newpassword456",
	}

	req := makeAuthReq(http.MethodPut, "/api/v1/users/me/password", token, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Errorf("expected 200 or 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersHandler_ChangeMyPassword_WrongCurrent(t *testing.T) {
	e, _, _, repo := setupUsersTest()
	user := createTestUser("user-1", "testuser", "password123")
	repo.addUser(user)

	token := loginForToken(t, e, "testuser", "password123")
	body := map[string]string{
		"currentPassword": "wrongpassword",
		"newPassword":     "newpassword456",
	}

	req := makeAuthReq(http.MethodPut, "/api/v1/users/me/password", token, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("expected non-200 for wrong current password")
	}
}
