package smtp

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// mockUserRepository implements a minimal mock for testing authentication.
type mockUserRepository struct {
	users map[string]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) AddUser(user *domain.User) {
	m.users[user.Username] = user
}

func (m *mockUserRepository) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if user, ok := m.users[username]; ok {
		return user, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepository) UpdateLastLogin(ctx context.Context, id domain.ID) error {
	for _, u := range m.users {
		if u.ID == id {
			now := domain.Now()
			u.LastLoginAt = &now
			return nil
		}
	}
	return domain.ErrNotFound
}

// Stubs for other UserRepository methods
func (m *mockUserRepository) List(ctx context.Context, filter *repository.UserFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return nil, nil
}
func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error { return nil }
func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error { return nil }
func (m *mockUserRepository) Delete(ctx context.Context, id domain.ID) error      { return nil }
func (m *mockUserRepository) SoftDelete(ctx context.Context, id domain.ID) error  { return nil }
func (m *mockUserRepository) Restore(ctx context.Context, id domain.ID) error     { return nil }
func (m *mockUserRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	return false, nil
}
func (m *mockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return false, nil
}
func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (m *mockUserRepository) Count(ctx context.Context, filter *repository.UserFilter) (int64, error) {
	return 0, nil
}
func (m *mockUserRepository) CountByRole(ctx context.Context) (map[domain.UserRole]int64, error) {
	return nil, nil
}
func (m *mockUserRepository) CountByStatus(ctx context.Context) (map[domain.UserStatus]int64, error) {
	return nil, nil
}
func (m *mockUserRepository) UpdatePassword(ctx context.Context, id domain.ID, passwordHash string) error {
	return nil
}
func (m *mockUserRepository) UpdateStatus(ctx context.Context, id domain.ID, status domain.UserStatus) error {
	return nil
}
func (m *mockUserRepository) UpdateRole(ctx context.Context, id domain.ID, role domain.UserRole) error {
	return nil
}
func (m *mockUserRepository) GetActiveUsers(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return nil, nil
}
func (m *mockUserRepository) GetAdmins(ctx context.Context) ([]*domain.User, error) { return nil, nil }
func (m *mockUserRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return nil, nil
}
func (m *mockUserRepository) BulkUpdateStatus(ctx context.Context, ids []domain.ID, status domain.UserStatus) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockUserRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	return nil, nil
}
func (m *mockUserRepository) GetUsersCreatedBetween(ctx context.Context, dateRange *repository.DateRangeFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return nil, nil
}
func (m *mockUserRepository) GetUsersWithRecentLogin(ctx context.Context, days int, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return nil, nil
}
func (m *mockUserRepository) GetInactiveUsers(ctx context.Context, days int, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	return nil, nil
}

// mockRepository wraps the mock user repository.
type mockRepository struct {
	userRepo *mockUserRepository
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		userRepo: newMockUserRepository(),
	}
}

func (m *mockRepository) Users() repository.UserRepository {
	return m.userRepo
}

// Stubs for other repository methods
func (m *mockRepository) Mailboxes() repository.MailboxRepository      { return nil }
func (m *mockRepository) Messages() repository.MessageRepository       { return nil }
func (m *mockRepository) Attachments() repository.AttachmentRepository { return nil }
func (m *mockRepository) Webhooks() repository.WebhookRepository       { return nil }
func (m *mockRepository) Settings() repository.SettingsRepository      { return nil }
func (m *mockRepository) JMAP() repository.JMAPRepository              { return nil }
func (m *mockRepository) Transaction(ctx context.Context, fn func(tx repository.Repository) error) error {
	return fn(m)
}
func (m *mockRepository) TransactionWithOptions(ctx context.Context, opts repository.TransactionOptions, fn func(tx repository.Repository) error) error {
	return fn(m)
}
func (m *mockRepository) Health(ctx context.Context) error { return nil }
func (m *mockRepository) Close() error                     { return nil }

// Helper to create a user with hashed password
func createTestUser(username, email, password string, status domain.UserStatus) *domain.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	user := domain.NewUser(domain.ID("user-"+username), username, email)
	user.PasswordHash = string(hash)
	user.Status = status
	return user
}

func TestAuthenticator_Authenticate(t *testing.T) {
	repo := newMockRepository()

	// Add test users
	activeUser := createTestUser("testuser", "test@example.com", "password123", domain.StatusActive)
	repo.userRepo.AddUser(activeUser)

	inactiveUser := createTestUser("inactive", "inactive@example.com", "password123", domain.StatusInactive)
	repo.userRepo.AddUser(inactiveUser)

	pendingUser := createTestUser("pending", "pending@example.com", "password123", domain.StatusPending)
	repo.userRepo.AddUser(pendingUser)

	auth := NewAuthenticator(repo)
	ctx := context.Background()

	tests := []struct {
		name      string
		username  string
		password  string
		wantErr   bool
		errReason string
	}{
		{
			name:     "valid credentials by username",
			username: "testuser",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "valid credentials by email",
			username: "test@example.com",
			password: "password123",
			wantErr:  false,
		},
		{
			name:      "wrong password",
			username:  "testuser",
			password:  "wrongpassword",
			wantErr:   true,
			errReason: AuthReasonInvalidCredentials,
		},
		{
			name:      "non-existent user",
			username:  "nobody",
			password:  "password123",
			wantErr:   true,
			errReason: AuthReasonInvalidCredentials,
		},
		{
			name:      "empty username",
			username:  "",
			password:  "password123",
			wantErr:   true,
			errReason: AuthReasonInvalidCredentials,
		},
		{
			name:      "empty password",
			username:  "testuser",
			password:  "",
			wantErr:   true,
			errReason: AuthReasonInvalidCredentials,
		},
		{
			name:      "inactive account",
			username:  "inactive",
			password:  "password123",
			wantErr:   true,
			errReason: AuthReasonAccountDisabled,
		},
		{
			name:      "pending account",
			username:  "pending",
			password:  "password123",
			wantErr:   true,
			errReason: AuthReasonAccountPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := auth.Authenticate(ctx, tt.username, tt.password)

			if tt.wantErr {
				if err == nil {
					t.Error("Authenticate() expected error, got nil")
					return
				}
				var authErr *AuthenticationError
				if errors.As(err, &authErr) {
					if authErr.Reason != tt.errReason {
						t.Errorf("Authenticate() error reason = %v, want %v", authErr.Reason, tt.errReason)
					}
				} else {
					t.Errorf("Authenticate() expected AuthenticationError, got %T", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Authenticate() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Error("Authenticate() returned nil result")
				return
			}

			if result.User == nil {
				t.Error("Authenticate() result.User is nil")
				return
			}
		})
	}
}

func TestAuthenticationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AuthenticationError
		contains string
	}{
		{
			name:     "invalid credentials",
			err:      NewInvalidCredentialsError(),
			contains: "invalid_credentials",
		},
		{
			name:     "account disabled",
			err:      NewAccountDisabledError(),
			contains: "account_disabled",
		},
		{
			name:     "account pending",
			err:      NewAccountPendingError(),
			contains: "account_pending",
		},
		{
			name:     "internal error",
			err:      NewInternalAuthError(errors.New("test error")),
			contains: "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			if errStr == "" {
				t.Error("Error() returned empty string")
			}
			if !containsString(errStr, tt.contains) {
				t.Errorf("Error() = %v, should contain %v", errStr, tt.contains)
			}
		})
	}
}

func TestSupportedAuthMechanisms(t *testing.T) {
	mechanisms := SupportedAuthMechanisms()

	if len(mechanisms) == 0 {
		t.Error("SupportedAuthMechanisms() returned empty slice")
	}

	// Check for PLAIN
	hasPlain := false
	for _, m := range mechanisms {
		if m == "PLAIN" {
			hasPlain = true
			break
		}
	}
	if !hasPlain {
		t.Error("SupportedAuthMechanisms() should include PLAIN")
	}

	// Check for LOGIN
	hasLogin := false
	for _, m := range mechanisms {
		if m == "LOGIN" {
			hasLogin = true
			break
		}
	}
	if !hasLogin {
		t.Error("SupportedAuthMechanisms() should include LOGIN")
	}
}

func TestAuthMechanism_IsSupported(t *testing.T) {
	tests := []struct {
		mechanism AuthMechanism
		want      bool
	}{
		{AuthMechanismPlain, true},
		{AuthMechanismLogin, true},
		{AuthMechanism("UNSUPPORTED"), false},
		{AuthMechanism(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mechanism), func(t *testing.T) {
			if got := tt.mechanism.IsSupported(); got != tt.want {
				t.Errorf("IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlainServer(t *testing.T) {
	tests := []struct {
		name      string
		response  []byte
		wantError bool
	}{
		{
			name:      "valid PLAIN response",
			response:  []byte("\x00testuser\x00password123"),
			wantError: false,
		},
		{
			name:      "valid PLAIN with identity",
			response:  []byte("identity\x00testuser\x00password123"),
			wantError: false,
		},
		{
			name:      "invalid format - missing parts",
			response:  []byte("testuser\x00password123"),
			wantError: true,
		},
		{
			name:      "invalid format - too many parts",
			response:  []byte("a\x00b\x00c\x00d"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var authenticated bool
			server := NewPlainServer(func(identity, username, password string) error {
				if username == "testuser" && password == "password123" {
					authenticated = true
					return nil
				}
				return NewInvalidCredentialsError()
			})

			_, done, err := server.Next(tt.response)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if done {
					t.Error("expected done=false on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !done {
					t.Error("expected done=true on success")
				}
				if !authenticated {
					t.Error("expected authenticated=true")
				}
			}
		})
	}
}

func TestLoginServer(t *testing.T) {
	var authenticated bool
	server := NewLoginServer(func(username, password string) error {
		if username == "testuser" && password == "password123" {
			authenticated = true
			return nil
		}
		return NewInvalidCredentialsError()
	})

	// First call with empty response should return username challenge
	challenge, done, err := server.Next(nil)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if done {
		t.Error("expected done=false after first call")
	}
	if string(challenge) != "Username:" {
		t.Errorf("expected 'Username:' challenge, got %s", string(challenge))
	}

	// Second call with username should return password challenge
	challenge, done, err = server.Next([]byte("testuser"))
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if done {
		t.Error("expected done=false after second call")
	}
	if string(challenge) != "Password:" {
		t.Errorf("expected 'Password:' challenge, got %s", string(challenge))
	}

	// Third call with password should complete authentication
	_, done, err = server.Next([]byte("password123"))
	if err != nil {
		t.Fatalf("unexpected error on third call: %v", err)
	}
	if !done {
		t.Error("expected done=true after third call")
	}
	if !authenticated {
		t.Error("expected authenticated=true")
	}
}

func TestLoginServerWithInitialResponse(t *testing.T) {
	var authenticated bool
	server := NewLoginServer(func(username, password string) error {
		if username == "testuser" && password == "password123" {
			authenticated = true
			return nil
		}
		return NewInvalidCredentialsError()
	})

	// First call with username should return password challenge (some clients send username with initial response)
	challenge, done, err := server.Next([]byte("testuser"))
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if done {
		t.Error("expected done=false after first call")
	}
	if string(challenge) != "Password:" {
		t.Errorf("expected 'Password:' challenge, got %s", string(challenge))
	}

	// Second call with password should complete authentication
	_, done, err = server.Next([]byte("password123"))
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if !done {
		t.Error("expected done=true after second call")
	}
	if !authenticated {
		t.Error("expected authenticated=true")
	}
}

func TestLoginServerInvalidCredentials(t *testing.T) {
	server := NewLoginServer(func(username, password string) error {
		return NewInvalidCredentialsError()
	})

	// Skip to password prompt
	server.Next([]byte("testuser"))

	// Third call with wrong password should fail
	_, done, err := server.Next([]byte("wrongpassword"))
	if err == nil {
		t.Error("expected error for invalid credentials")
	}
	if done {
		t.Error("expected done=false on error")
	}
}

func TestSplitNull(t *testing.T) {
	tests := []struct {
		input    []byte
		expected [][]byte
	}{
		{
			input:    []byte("a\x00b\x00c"),
			expected: [][]byte{[]byte("a"), []byte("b"), []byte("c")},
		},
		{
			input:    []byte("\x00user\x00pass"),
			expected: [][]byte{[]byte(""), []byte("user"), []byte("pass")},
		},
		{
			input:    []byte("notnull"),
			expected: [][]byte{[]byte("notnull")},
		},
		{
			input:    []byte(""),
			expected: [][]byte{[]byte("")},
		},
	}

	for _, tt := range tests {
		result := splitNull(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitNull(%v) returned %d parts, expected %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i := range result {
			if string(result[i]) != string(tt.expected[i]) {
				t.Errorf("splitNull(%v)[%d] = %v, expected %v", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSessionAuthMechanisms(t *testing.T) {
	// Test without authenticator
	serverWithoutAuth := createTestServer(t)
	sessionWithoutAuth := NewSession(serverWithoutAuth.Backend(), nil, "127.0.0.1:12345")
	mechanisms := sessionWithoutAuth.AuthMechanisms()
	if len(mechanisms) != 0 {
		t.Errorf("expected empty mechanisms without authenticator, got %v", mechanisms)
	}

	// Test with authenticator
	repo := newMockRepository()
	serverWithAuth := createTestServer(t, WithRepo(repo))
	sessionWithAuth := NewSession(serverWithAuth.Backend(), nil, "127.0.0.1:12345")
	mechanisms = sessionWithAuth.AuthMechanisms()
	if len(mechanisms) != 2 {
		t.Errorf("expected 2 mechanisms with authenticator, got %d", len(mechanisms))
	}
}

func TestSessionAuth(t *testing.T) {
	repo := newMockRepository()
	activeUser := createTestUser("testuser", "test@example.com", "password123", domain.StatusActive)
	repo.userRepo.AddUser(activeUser)

	server := createTestServer(t, WithRepo(repo))
	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")

	// Test PLAIN auth
	saslServer, err := session.Auth("PLAIN")
	if err != nil {
		t.Fatalf("unexpected error for PLAIN auth: %v", err)
	}
	if saslServer == nil {
		t.Fatal("expected SASL server, got nil")
	}

	// Authenticate with PLAIN
	_, done, err := saslServer.Next([]byte("\x00testuser\x00password123"))
	if err != nil {
		t.Fatalf("PLAIN auth failed: %v", err)
	}
	if !done {
		t.Error("expected done=true after PLAIN auth")
	}
	if !session.IsAuthenticated() {
		t.Error("expected session to be authenticated")
	}
	if session.AuthenticatedUser() == nil {
		t.Error("expected authenticated user to be set")
	}
	if session.AuthenticatedUser().Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", session.AuthenticatedUser().Username)
	}

	// Test LOGIN auth
	session2 := NewSession(server.Backend(), nil, "127.0.0.1:12346")
	saslServer, err = session2.Auth("LOGIN")
	if err != nil {
		t.Fatalf("unexpected error for LOGIN auth: %v", err)
	}
	if saslServer == nil {
		t.Fatal("expected SASL server for LOGIN, got nil")
	}

	// Test unsupported mechanism
	session3 := NewSession(server.Backend(), nil, "127.0.0.1:12347")
	_, err = session3.Auth("UNSUPPORTED")
	if err == nil {
		t.Error("expected error for unsupported mechanism")
	}
}

func TestSessionAuthWithoutAuthenticator(t *testing.T) {
	server := createTestServer(t)
	session := NewSession(server.Backend(), nil, "127.0.0.1:12345")

	_, err := session.Auth("PLAIN")
	if err == nil {
		t.Error("expected error when no authenticator configured")
	}
}
