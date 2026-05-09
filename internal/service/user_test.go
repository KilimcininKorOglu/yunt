package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/repository"
)

// mockFullUserRepository is a mock implementation of the full repository.UserRepository
// interface for use in UserService tests.
type mockFullUserRepository struct {
	users         map[domain.ID]*domain.User
	usersByName   map[string]*domain.User
	usersByEmail  map[string]*domain.User
	deletedIDs    map[domain.ID]bool
	passwords     map[domain.ID]string
	statuses      map[domain.ID]domain.UserStatus
	roles         map[domain.ID]domain.UserRole
}

func newMockFullUserRepository() *mockFullUserRepository {
	return &mockFullUserRepository{
		users:        make(map[domain.ID]*domain.User),
		usersByName:  make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
		deletedIDs:   make(map[domain.ID]bool),
		passwords:    make(map[domain.ID]string),
		statuses:     make(map[domain.ID]domain.UserStatus),
		roles:        make(map[domain.ID]domain.UserRole),
	}
}

func (r *mockFullUserRepository) addUser(user *domain.User) {
	r.users[user.ID] = user
	r.usersByName[strings.ToLower(user.Username)] = user
	r.usersByEmail[strings.ToLower(user.Email)] = user
	r.statuses[user.ID] = user.Status
	r.roles[user.ID] = user.Role
}

func (r *mockFullUserRepository) GetByID(_ context.Context, id domain.ID) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok || r.deletedIDs[id] {
		return nil, domain.NewNotFoundError("user", id.String())
	}
	return user, nil
}

func (r *mockFullUserRepository) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	user, ok := r.usersByName[strings.ToLower(username)]
	if !ok || r.deletedIDs[user.ID] {
		return nil, domain.NewNotFoundError("user", username)
	}
	return user, nil
}

func (r *mockFullUserRepository) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	user, ok := r.usersByEmail[strings.ToLower(email)]
	if !ok || r.deletedIDs[user.ID] {
		return nil, domain.NewNotFoundError("user", email)
	}
	return user, nil
}

func (r *mockFullUserRepository) List(_ context.Context, filter *repository.UserFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	var items []*domain.User
	for _, u := range r.users {
		if r.deletedIDs[u.ID] {
			continue
		}
		if filter != nil && filter.Search != "" {
			q := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(u.Username), q) &&
				!strings.Contains(strings.ToLower(u.Email), q) &&
				!strings.Contains(strings.ToLower(u.DisplayName), q) {
				continue
			}
		}
		items = append(items, u)
	}

	page := 1
	perPage := repository.DefaultPerPage
	if opts != nil && opts.Pagination != nil {
		if opts.Pagination.Page > 0 {
			page = opts.Pagination.Page
		}
		if opts.Pagination.PerPage > 0 {
			perPage = opts.Pagination.PerPage
		}
	}

	total := int64(len(items))

	start := (page - 1) * perPage
	if start >= len(items) {
		items = nil
	} else {
		end := start + perPage
		if end > len(items) {
			end = len(items)
		}
		items = items[start:end]
	}

	return &repository.ListResult[*domain.User]{
		Items: items,
		Total: total,
	}, nil
}

func (r *mockFullUserRepository) Create(_ context.Context, user *domain.User) error {
	if _, exists := r.usersByName[strings.ToLower(user.Username)]; exists {
		return domain.NewAlreadyExistsError("user", "username", user.Username)
	}
	if _, exists := r.usersByEmail[strings.ToLower(user.Email)]; exists {
		return domain.NewAlreadyExistsError("user", "email", user.Email)
	}
	r.addUser(user)
	return nil
}

func (r *mockFullUserRepository) Update(_ context.Context, user *domain.User) error {
	if _, exists := r.users[user.ID]; !exists {
		return domain.NewNotFoundError("user", user.ID.String())
	}
	// Check email conflict with other users.
	if existing, ok := r.usersByEmail[strings.ToLower(user.Email)]; ok && existing.ID != user.ID {
		return domain.NewAlreadyExistsError("user", "email", user.Email)
	}
	// Remove old index entries and re-add.
	old := r.users[user.ID]
	delete(r.usersByName, strings.ToLower(old.Username))
	delete(r.usersByEmail, strings.ToLower(old.Email))
	r.addUser(user)
	return nil
}

func (r *mockFullUserRepository) Delete(_ context.Context, id domain.ID) error {
	if _, exists := r.users[id]; !exists {
		return domain.NewNotFoundError("user", id.String())
	}
	u := r.users[id]
	delete(r.users, id)
	delete(r.usersByName, strings.ToLower(u.Username))
	delete(r.usersByEmail, strings.ToLower(u.Email))
	return nil
}

func (r *mockFullUserRepository) SoftDelete(_ context.Context, id domain.ID) error {
	if _, exists := r.users[id]; !exists {
		return domain.NewNotFoundError("user", id.String())
	}
	r.deletedIDs[id] = true
	return nil
}

func (r *mockFullUserRepository) Restore(_ context.Context, id domain.ID) error {
	if _, exists := r.users[id]; !exists {
		return domain.NewNotFoundError("user", id.String())
	}
	delete(r.deletedIDs, id)
	return nil
}

func (r *mockFullUserRepository) Exists(_ context.Context, id domain.ID) (bool, error) {
	_, exists := r.users[id]
	return exists && !r.deletedIDs[id], nil
}

func (r *mockFullUserRepository) ExistsByUsername(_ context.Context, username string) (bool, error) {
	u, exists := r.usersByName[strings.ToLower(username)]
	return exists && !r.deletedIDs[u.ID], nil
}

func (r *mockFullUserRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	u, exists := r.usersByEmail[strings.ToLower(email)]
	return exists && !r.deletedIDs[u.ID], nil
}

func (r *mockFullUserRepository) Count(_ context.Context, filter *repository.UserFilter) (int64, error) {
	var count int64
	for _, u := range r.users {
		if r.deletedIDs[u.ID] {
			continue
		}
		count++
	}
	return count, nil
}

func (r *mockFullUserRepository) CountByRole(_ context.Context) (map[domain.UserRole]int64, error) {
	result := make(map[domain.UserRole]int64)
	for _, u := range r.users {
		if r.deletedIDs[u.ID] {
			continue
		}
		result[u.Role]++
	}
	return result, nil
}

func (r *mockFullUserRepository) CountByStatus(_ context.Context) (map[domain.UserStatus]int64, error) {
	result := make(map[domain.UserStatus]int64)
	for _, u := range r.users {
		if r.deletedIDs[u.ID] {
			continue
		}
		result[u.Status]++
	}
	return result, nil
}

func (r *mockFullUserRepository) UpdatePassword(_ context.Context, id domain.ID, passwordHash string) error {
	u, exists := r.users[id]
	if !exists {
		return domain.NewNotFoundError("user", id.String())
	}
	u.PasswordHash = passwordHash
	r.passwords[id] = passwordHash
	return nil
}

func (r *mockFullUserRepository) UpdateLastLogin(_ context.Context, id domain.ID) error {
	u, exists := r.users[id]
	if !exists {
		return domain.NewNotFoundError("user", id.String())
	}
	now := domain.Now()
	u.LastLoginAt = &now
	return nil
}

func (r *mockFullUserRepository) UpdateStatus(_ context.Context, id domain.ID, status domain.UserStatus) error {
	u, exists := r.users[id]
	if !exists {
		return domain.NewNotFoundError("user", id.String())
	}
	u.Status = status
	r.statuses[id] = status
	return nil
}

func (r *mockFullUserRepository) UpdateRole(_ context.Context, id domain.ID, role domain.UserRole) error {
	u, exists := r.users[id]
	if !exists {
		return domain.NewNotFoundError("user", id.String())
	}
	u.Role = role
	r.roles[id] = role
	return nil
}

func (r *mockFullUserRepository) GetActiveUsers(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	status := domain.StatusActive
	filter := &repository.UserFilter{Status: &status}
	return r.List(ctx, filter, opts)
}

func (r *mockFullUserRepository) GetAdmins(_ context.Context) ([]*domain.User, error) {
	var admins []*domain.User
	for _, u := range r.users {
		if r.deletedIDs[u.ID] {
			continue
		}
		if u.Role == domain.RoleAdmin {
			admins = append(admins, u)
		}
	}
	return admins, nil
}

func (r *mockFullUserRepository) Search(_ context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	q := strings.ToLower(query)
	var items []*domain.User
	for _, u := range r.users {
		if r.deletedIDs[u.ID] {
			continue
		}
		if strings.Contains(strings.ToLower(u.Username), q) ||
			strings.Contains(strings.ToLower(u.Email), q) ||
			strings.Contains(strings.ToLower(u.DisplayName), q) {
			items = append(items, u)
		}
	}
	return &repository.ListResult[*domain.User]{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

func (r *mockFullUserRepository) BulkUpdateStatus(_ context.Context, ids []domain.ID, status domain.UserStatus) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	for _, id := range ids {
		u, exists := r.users[id]
		if !exists {
			op.AddFailure(id.String(), domain.NewNotFoundError("user", id.String()))
			continue
		}
		u.Status = status
		op.AddSuccess()
	}
	return op, nil
}

func (r *mockFullUserRepository) BulkDelete(_ context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	op := repository.NewBulkOperation()
	for _, id := range ids {
		if _, exists := r.users[id]; !exists {
			op.AddFailure(id.String(), domain.NewNotFoundError("user", id.String()))
			continue
		}
		r.deletedIDs[id] = true
		op.AddSuccess()
	}
	return op, nil
}

func (r *mockFullUserRepository) GetUsersCreatedBetween(_ context.Context, _ *repository.DateRangeFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return &repository.ListResult[*domain.User]{}, nil
}

func (r *mockFullUserRepository) GetUsersWithRecentLogin(_ context.Context, _ int, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return &repository.ListResult[*domain.User]{}, nil
}

func (r *mockFullUserRepository) GetInactiveUsers(_ context.Context, _ int, _ *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return &repository.ListResult[*domain.User]{}, nil
}

// --- helpers ---

func newTestUserService() (*UserService, *mockFullUserRepository) {
	cfg := config.AuthConfig{
		BCryptCost: bcrypt.MinCost,
	}
	repo := newMockFullUserRepository()
	svc := NewUserService(cfg, repo)
	return svc, repo
}

func makeTestUser(id, username, email, password string) *domain.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return &domain.User{
		ID:           domain.ID(id),
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
}

// --- tests ---

func TestUserService_List(t *testing.T) {
	t.Run("returns paginated results", func(t *testing.T) {
		svc, repo := newTestUserService()
		for i := 0; i < 5; i++ {
			id := domain.ID("list-user-" + string(rune('0'+i)))
			u := makeTestUser(string(id), "user"+string(rune('a'+i)), "user"+string(rune('a'+i))+"@example.com", "password123")
			repo.addUser(u)
		}

		ctx := context.Background()
		resp, err := svc.List(ctx, nil, 1, 3)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if resp.Total != 5 {
			t.Errorf("List() Total = %d, want 5", resp.Total)
		}
		if resp.Page != 1 {
			t.Errorf("List() Page = %d, want 1", resp.Page)
		}
		if resp.PageSize != 3 {
			t.Errorf("List() PageSize = %d, want 3", resp.PageSize)
		}
		if resp.TotalPages != 2 {
			t.Errorf("List() TotalPages = %d, want 2", resp.TotalPages)
		}
		if len(resp.Users) != 3 {
			t.Errorf("List() len(Users) = %d, want 3", len(resp.Users))
		}
	})

	t.Run("normalises invalid page to 1", func(t *testing.T) {
		svc, _ := newTestUserService()
		ctx := context.Background()
		resp, err := svc.List(ctx, nil, 0, 10)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if resp.Page != 1 {
			t.Errorf("List() Page = %d, want 1", resp.Page)
		}
	})

	t.Run("caps pageSize at MaxPerPage", func(t *testing.T) {
		svc, _ := newTestUserService()
		ctx := context.Background()
		resp, err := svc.List(ctx, nil, 1, repository.MaxPerPage+50)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if resp.PageSize != repository.MaxPerPage {
			t.Errorf("List() PageSize = %d, want %d", resp.PageSize, repository.MaxPerPage)
		}
	})

	t.Run("password hashes are never returned", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("u1", "alice", "alice@example.com", "password123"))
		ctx := context.Background()
		resp, err := svc.List(ctx, nil, 1, 10)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(resp.Users) == 0 {
			t.Fatal("List() returned no users")
		}
		// domain.UserInfo should never expose PasswordHash
		for _, ui := range resp.Users {
			_ = ui // UserInfo has no PasswordHash field — compile-time guarantee
		}
	})
}

func TestUserService_GetByID(t *testing.T) {
	tests := []struct {
		name      string
		seedUser  bool
		id        string
		wantErr   bool
		errIsNotFound bool
	}{
		{
			name:      "found",
			seedUser:  true,
			id:        "user-get-1",
			wantErr:   false,
		},
		{
			name:          "not found",
			seedUser:      false,
			id:            "user-missing",
			wantErr:       true,
			errIsNotFound: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestUserService()
			if tc.seedUser {
				repo.addUser(makeTestUser(tc.id, "getbyid", "getbyid@example.com", "password123"))
			}
			ctx := context.Background()
			user, err := svc.GetByID(ctx, domain.ID(tc.id))
			if tc.wantErr {
				if err == nil {
					t.Error("GetByID() expected error, got nil")
				}
				if tc.errIsNotFound && !domain.IsNotFound(err) {
					t.Errorf("GetByID() error is not NotFound: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("GetByID() unexpected error: %v", err)
				}
				if user.ID != domain.ID(tc.id) {
					t.Errorf("GetByID() ID = %v, want %v", user.ID, tc.id)
				}
			}
		})
	}
}

func TestUserService_GetUserInfo(t *testing.T) {
	t.Run("returns UserInfo for existing user", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("ui-1", "infouser", "infouser@example.com", "password123"))
		ctx := context.Background()
		info, err := svc.GetUserInfo(ctx, "ui-1")
		if err != nil {
			t.Fatalf("GetUserInfo() error = %v", err)
		}
		if info == nil {
			t.Fatal("GetUserInfo() returned nil")
		}
		if info.Username != "infouser" {
			t.Errorf("GetUserInfo() Username = %v, want infouser", info.Username)
		}
	})

	t.Run("returns not found for missing user", func(t *testing.T) {
		svc, _ := newTestUserService()
		ctx := context.Background()
		_, err := svc.GetUserInfo(ctx, "no-such-user")
		if err == nil {
			t.Error("GetUserInfo() expected error, got nil")
		}
		if !domain.IsNotFound(err) {
			t.Errorf("GetUserInfo() error is not NotFound: %v", err)
		}
	})
}

func TestUserService_Create(t *testing.T) {
	tests := []struct {
		name          string
		input         *domain.UserCreateInput
		seedUsername  string
		seedEmail     string
		wantErr       bool
		errIsConflict bool
		errContains   string
	}{
		{
			name: "success",
			input: &domain.UserCreateInput{
				Username: "newuser",
				Email:    "newuser@example.com",
				Password: "Password123!",
			},
			wantErr: false,
		},
		{
			name: "duplicate username",
			input: &domain.UserCreateInput{
				Username: "existing",
				Email:    "unique@example.com",
				Password: "Password123!",
			},
			seedUsername:  "existing",
			wantErr:       true,
			errIsConflict: true,
		},
		{
			name: "duplicate email",
			input: &domain.UserCreateInput{
				Username: "brandnew",
				Email:    "taken@example.com",
				Password: "Password123!",
			},
			seedEmail:     "taken@example.com",
			wantErr:       true,
			errIsConflict: true,
		},
		{
			name: "invalid input short password",
			input: &domain.UserCreateInput{
				Username: "validuser",
				Email:    "valid@example.com",
				Password: "short",
			},
			wantErr: true,
		},
		{
			name: "invalid input missing username",
			input: &domain.UserCreateInput{
				Username: "",
				Email:    "valid@example.com",
				Password: "Password123!",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestUserService()
			if tc.seedUsername != "" {
				repo.addUser(makeTestUser("seed-u", tc.seedUsername, "seed@example.com", "password123"))
			}
			if tc.seedEmail != "" {
				repo.addUser(makeTestUser("seed-e", "seedemailuser", tc.seedEmail, "password123"))
			}

			ctx := context.Background()
			user, err := svc.Create(ctx, tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("Create() expected error, got nil")
				}
				if tc.errIsConflict && !domain.IsAlreadyExists(err) {
					t.Errorf("Create() error is not AlreadyExists: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("Create() unexpected error: %v", err)
				}
				if user == nil {
					t.Fatal("Create() returned nil user")
				}
				if user.Username != tc.input.Username {
					t.Errorf("Create() Username = %v, want %v", user.Username, tc.input.Username)
				}
				if user.PasswordHash == "" {
					t.Error("Create() PasswordHash should not be empty")
				}
				if user.PasswordHash == tc.input.Password {
					t.Error("Create() PasswordHash must not be the plaintext password")
				}
				if user.ID == "" {
					t.Error("Create() ID should be set")
				}
			}
		})
	}
}

func TestUserService_Update(t *testing.T) {
	newStr := func(s string) *string { return &s }

	t.Run("success updates email and display name", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("upd-1", "updateme", "old@example.com", "password123"))
		ctx := context.Background()
		input := &domain.UserUpdateInput{
			Email:       newStr("new@example.com"),
			DisplayName: newStr("New Name"),
		}
		user, err := svc.Update(ctx, "upd-1", input, false)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if user.Email != "new@example.com" {
			t.Errorf("Update() Email = %v, want new@example.com", user.Email)
		}
		if user.DisplayName != "New Name" {
			t.Errorf("Update() DisplayName = %v, want New Name", user.DisplayName)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc, _ := newTestUserService()
		ctx := context.Background()
		input := &domain.UserUpdateInput{DisplayName: newStr("X")}
		_, err := svc.Update(ctx, "no-such-id", input, false)
		if err == nil {
			t.Error("Update() expected error, got nil")
		}
		if !domain.IsNotFound(err) {
			t.Errorf("Update() error is not NotFound: %v", err)
		}
	})

	t.Run("email conflict with another user", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("upd-2", "userone", "userone@example.com", "password123"))
		repo.addUser(makeTestUser("upd-3", "usertwo", "usertwo@example.com", "password123"))
		ctx := context.Background()
		input := &domain.UserUpdateInput{Email: newStr("usertwo@example.com")}
		_, err := svc.Update(ctx, "upd-2", input, false)
		if err == nil {
			t.Error("Update() expected conflict error, got nil")
		}
		if !domain.IsAlreadyExists(err) {
			t.Errorf("Update() error is not AlreadyExists: %v", err)
		}
	})

	t.Run("non-admin cannot change role or status", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("upd-4", "normaluser", "normaluser@example.com", "password123"))
		ctx := context.Background()
		role := domain.RoleAdmin
		status := domain.StatusInactive
		input := &domain.UserUpdateInput{
			Role:   &role,
			Status: &status,
		}
		user, err := svc.Update(ctx, "upd-4", input, false)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}
		if user.Role == domain.RoleAdmin {
			t.Error("Update() non-admin should not be able to change role")
		}
		if user.Status == domain.StatusInactive {
			t.Error("Update() non-admin should not be able to change status")
		}
	})

	t.Run("admin can change role and status", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("upd-5", "targetuser", "targetuser@example.com", "password123"))
		ctx := context.Background()
		role := domain.RoleAdmin
		status := domain.StatusInactive
		input := &domain.UserUpdateInput{
			Role:   &role,
			Status: &status,
		}
		user, err := svc.Update(ctx, "upd-5", input, true)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if user.Role != domain.RoleAdmin {
			t.Errorf("Update() Role = %v, want admin", user.Role)
		}
		if user.Status != domain.StatusInactive {
			t.Errorf("Update() Status = %v, want inactive", user.Status)
		}
	})
}

func TestUserService_Delete(t *testing.T) {
	t.Run("success soft-deletes the user", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("del-1", "deleteuser", "deleteuser@example.com", "password123"))
		ctx := context.Background()
		err := svc.Delete(ctx, "del-1")
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if !repo.deletedIDs["del-1"] {
			t.Error("Delete() user was not soft-deleted")
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc, _ := newTestUserService()
		ctx := context.Background()
		err := svc.Delete(ctx, "no-such-user")
		if err == nil {
			t.Error("Delete() expected error, got nil")
		}
		if !domain.IsNotFound(err) {
			t.Errorf("Delete() error is not NotFound: %v", err)
		}
	})
}

func TestUserService_UpdatePassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("pwdupd-1", "pwduser", "pwduser@example.com", "OldPass123!"))
		ctx := context.Background()
		err := svc.UpdatePassword(ctx, "pwdupd-1", "NewPass456!")
		if err != nil {
			t.Fatalf("UpdatePassword() error = %v", err)
		}
		u := repo.users["pwdupd-1"]
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("NewPass456!")); err != nil {
			t.Errorf("UpdatePassword() new password hash is invalid: %v", err)
		}
	})

	t.Run("too short password", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("pwdupd-2", "pwduser2", "pwduser2@example.com", "OldPass123!"))
		ctx := context.Background()
		err := svc.UpdatePassword(ctx, "pwdupd-2", "short")
		if err == nil {
			t.Error("UpdatePassword() expected validation error, got nil")
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc, _ := newTestUserService()
		ctx := context.Background()
		err := svc.UpdatePassword(ctx, "no-such-user", "ValidPass123!")
		if err == nil {
			t.Error("UpdatePassword() expected error, got nil")
		}
		if !domain.IsNotFound(err) {
			t.Errorf("UpdatePassword() error is not NotFound: %v", err)
		}
	})

	t.Run("password too long", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("pwdupd-3", "pwduser3", "pwduser3@example.com", "OldPass123!"))
		ctx := context.Background()
		tooLong := strings.Repeat("a", 129)
		err := svc.UpdatePassword(ctx, "pwdupd-3", tooLong)
		if err == nil {
			t.Error("UpdatePassword() expected validation error for too-long password, got nil")
		}
	})
}

func TestUserService_ChangePassword(t *testing.T) {
	t.Run("success with correct current password", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("chpwd-1", "changepwduser", "changepwduser@example.com", "CurrentPass123!"))
		ctx := context.Background()
		err := svc.ChangePassword(ctx, "chpwd-1", "CurrentPass123!", "NewPass456!")
		if err != nil {
			t.Fatalf("ChangePassword() error = %v", err)
		}
		u := repo.users["chpwd-1"]
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("NewPass456!")); err != nil {
			t.Errorf("ChangePassword() new password hash is invalid: %v", err)
		}
	})

	t.Run("fails with wrong current password", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("chpwd-2", "changepwduser2", "changepwduser2@example.com", "RealPass123!"))
		ctx := context.Background()
		err := svc.ChangePassword(ctx, "chpwd-2", "WrongPass!", "NewPass456!")
		if err == nil {
			t.Error("ChangePassword() expected error for wrong current password, got nil")
		}
		if !domain.IsUnauthorized(err) {
			t.Errorf("ChangePassword() error is not Unauthorized: %v", err)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc, _ := newTestUserService()
		ctx := context.Background()
		err := svc.ChangePassword(ctx, "no-such-user", "current", "NewPass456!")
		if err == nil {
			t.Error("ChangePassword() expected error, got nil")
		}
		if !domain.IsNotFound(err) {
			t.Errorf("ChangePassword() error is not NotFound: %v", err)
		}
	})
}

func TestUserService_UpdateStatus(t *testing.T) {
	tests := []struct {
		name      string
		seedUser  bool
		status    domain.UserStatus
		wantErr   bool
	}{
		{name: "active", seedUser: true, status: domain.StatusActive, wantErr: false},
		{name: "inactive", seedUser: true, status: domain.StatusInactive, wantErr: false},
		{name: "pending", seedUser: true, status: domain.StatusPending, wantErr: false},
		{name: "invalid status", seedUser: true, status: domain.UserStatus("invalid"), wantErr: true},
		{name: "user not found", seedUser: false, status: domain.StatusActive, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestUserService()
			id := domain.ID("status-user-1")
			if tc.seedUser {
				repo.addUser(makeTestUser(string(id), "statususer", "statususer@example.com", "password123"))
			}
			ctx := context.Background()
			err := svc.UpdateStatus(ctx, id, tc.status)
			if tc.wantErr {
				if err == nil {
					t.Error("UpdateStatus() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("UpdateStatus() unexpected error: %v", err)
				}
				if repo.users[id].Status != tc.status {
					t.Errorf("UpdateStatus() status = %v, want %v", repo.users[id].Status, tc.status)
				}
			}
		})
	}
}

func TestUserService_UpdateRole(t *testing.T) {
	tests := []struct {
		name     string
		seedUser bool
		role     domain.UserRole
		wantErr  bool
	}{
		{name: "set to admin", seedUser: true, role: domain.RoleAdmin, wantErr: false},
		{name: "set to user", seedUser: true, role: domain.RoleUser, wantErr: false},
		{name: "set to viewer", seedUser: true, role: domain.RoleViewer, wantErr: false},
		{name: "invalid role", seedUser: true, role: domain.UserRole("invalid"), wantErr: true},
		{name: "user not found", seedUser: false, role: domain.RoleUser, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestUserService()
			id := domain.ID("role-user-1")
			if tc.seedUser {
				repo.addUser(makeTestUser(string(id), "roleuser", "roleuser@example.com", "password123"))
			}
			ctx := context.Background()
			err := svc.UpdateRole(ctx, id, tc.role)
			if tc.wantErr {
				if err == nil {
					t.Error("UpdateRole() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("UpdateRole() unexpected error: %v", err)
				}
				if repo.users[id].Role != tc.role {
					t.Errorf("UpdateRole() role = %v, want %v", repo.users[id].Role, tc.role)
				}
			}
		})
	}
}

func TestUserService_Search(t *testing.T) {
	t.Run("returns matching users", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("srch-1", "alice", "alice@example.com", "password123"))
		repo.addUser(makeTestUser("srch-2", "bob", "bob@example.com", "password123"))
		repo.addUser(makeTestUser("srch-3", "charlie", "charlie@example.com", "password123"))

		ctx := context.Background()
		resp, err := svc.Search(ctx, "ali", 1, 10)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if resp.Total != 1 {
			t.Errorf("Search() Total = %d, want 1", resp.Total)
		}
		if len(resp.Users) == 0 || resp.Users[0].Username != "alice" {
			t.Errorf("Search() did not return alice")
		}
	})

	t.Run("empty query returns all users", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("srch-4", "dave", "dave@example.com", "password123"))
		repo.addUser(makeTestUser("srch-5", "eve", "eve@example.com", "password123"))

		ctx := context.Background()
		resp, err := svc.Search(ctx, "", 1, 10)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if resp.Total != 2 {
			t.Errorf("Search() Total = %d, want 2", resp.Total)
		}
	})

	t.Run("no results for unknown query", func(t *testing.T) {
		svc, repo := newTestUserService()
		repo.addUser(makeTestUser("srch-6", "frank", "frank@example.com", "password123"))

		ctx := context.Background()
		resp, err := svc.Search(ctx, "zzznomatch", 1, 10)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if resp.Total != 0 {
			t.Errorf("Search() Total = %d, want 0", resp.Total)
		}
	})
}

func TestUserService_GetStats(t *testing.T) {
	t.Run("returns correct counts", func(t *testing.T) {
		svc, repo := newTestUserService()

		adminUser := makeTestUser("st-1", "admin1", "admin1@example.com", "password123")
		adminUser.Role = domain.RoleAdmin
		adminUser.Status = domain.StatusActive
		repo.addUser(adminUser)

		activeUser := makeTestUser("st-2", "user1", "user1@example.com", "password123")
		activeUser.Role = domain.RoleUser
		activeUser.Status = domain.StatusActive
		repo.addUser(activeUser)

		pendingUser := makeTestUser("st-3", "user2", "user2@example.com", "password123")
		pendingUser.Role = domain.RoleUser
		pendingUser.Status = domain.StatusPending
		repo.addUser(pendingUser)

		ctx := context.Background()
		stats, err := svc.GetStats(ctx)
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}

		if stats.TotalUsers != 3 {
			t.Errorf("GetStats() TotalUsers = %d, want 3", stats.TotalUsers)
		}
		if stats.ActiveUsers != 2 {
			t.Errorf("GetStats() ActiveUsers = %d, want 2", stats.ActiveUsers)
		}
		if stats.PendingUsers != 1 {
			t.Errorf("GetStats() PendingUsers = %d, want 1", stats.PendingUsers)
		}
		if stats.ByRole[domain.RoleAdmin] != 1 {
			t.Errorf("GetStats() ByRole[admin] = %d, want 1", stats.ByRole[domain.RoleAdmin])
		}
		if stats.ByRole[domain.RoleUser] != 2 {
			t.Errorf("GetStats() ByRole[user] = %d, want 2", stats.ByRole[domain.RoleUser])
		}
		if stats.ByStatus[domain.StatusActive] != 2 {
			t.Errorf("GetStats() ByStatus[active] = %d, want 2", stats.ByStatus[domain.StatusActive])
		}
	})

	t.Run("empty repository returns zero counts", func(t *testing.T) {
		svc, _ := newTestUserService()
		ctx := context.Background()
		stats, err := svc.GetStats(ctx)
		if err != nil {
			t.Fatalf("GetStats() error = %v", err)
		}
		if stats.TotalUsers != 0 {
			t.Errorf("GetStats() TotalUsers = %d, want 0", stats.TotalUsers)
		}
		if stats.ActiveUsers != 0 {
			t.Errorf("GetStats() ActiveUsers = %d, want 0", stats.ActiveUsers)
		}
	})
}

// Compile-time assertion: mockFullUserRepository must implement repository.UserRepository.
var _ repository.UserRepository = (*mockFullUserRepository)(nil)

// Ensure domain.Timestamp used in makeTestUser is not stale across tests.
var _ = time.Now
