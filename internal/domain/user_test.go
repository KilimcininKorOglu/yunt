package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewUser(t *testing.T) {
	user := NewUser(ID("123"), "johndoe", "john@example.com")

	if user.ID != ID("123") {
		t.Errorf("NewUser().ID = %v, want %v", user.ID, "123")
	}
	if user.Username != "johndoe" {
		t.Errorf("NewUser().Username = %v, want %v", user.Username, "johndoe")
	}
	if user.Email != "john@example.com" {
		t.Errorf("NewUser().Email = %v, want %v", user.Email, "john@example.com")
	}
	if user.Role != RoleUser {
		t.Errorf("NewUser().Role = %v, want %v", user.Role, RoleUser)
	}
	if user.Status != StatusPending {
		t.Errorf("NewUser().Status = %v, want %v", user.Status, StatusPending)
	}
}

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    *User
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid user",
			user: &User{
				ID:       ID("123"),
				Username: "johndoe",
				Email:    "john@example.com",
				Role:     RoleUser,
				Status:   StatusActive,
			},
			wantErr: false,
		},
		{
			name: "missing id",
			user: &User{
				Username: "johndoe",
				Email:    "john@example.com",
				Role:     RoleUser,
				Status:   StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"id"},
		},
		{
			name: "missing username",
			user: &User{
				ID:     ID("123"),
				Email:  "john@example.com",
				Role:   RoleUser,
				Status: StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"username"},
		},
		{
			name: "username too short",
			user: &User{
				ID:       ID("123"),
				Username: "ab",
				Email:    "john@example.com",
				Role:     RoleUser,
				Status:   StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"username"},
		},
		{
			name: "username too long",
			user: &User{
				ID:       ID("123"),
				Username: strings.Repeat("a", 51),
				Email:    "john@example.com",
				Role:     RoleUser,
				Status:   StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"username"},
		},
		{
			name: "username invalid chars",
			user: &User{
				ID:       ID("123"),
				Username: "john@doe",
				Email:    "john@example.com",
				Role:     RoleUser,
				Status:   StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"username"},
		},
		{
			name: "missing email",
			user: &User{
				ID:       ID("123"),
				Username: "johndoe",
				Role:     RoleUser,
				Status:   StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"email"},
		},
		{
			name: "invalid email",
			user: &User{
				ID:       ID("123"),
				Username: "johndoe",
				Email:    "not-an-email",
				Role:     RoleUser,
				Status:   StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"email"},
		},
		{
			name: "invalid role",
			user: &User{
				ID:       ID("123"),
				Username: "johndoe",
				Email:    "john@example.com",
				Role:     UserRole("invalid"),
				Status:   StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"role"},
		},
		{
			name: "invalid status",
			user: &User{
				ID:       ID("123"),
				Username: "johndoe",
				Email:    "john@example.com",
				Role:     RoleUser,
				Status:   UserStatus("invalid"),
			},
			wantErr: true,
			errMsgs: []string{"status"},
		},
		{
			name: "display name too long",
			user: &User{
				ID:          ID("123"),
				Username:    "johndoe",
				Email:       "john@example.com",
				DisplayName: strings.Repeat("a", 101),
				Role:        RoleUser,
				Status:      StatusActive,
			},
			wantErr: true,
			errMsgs: []string{"displayName"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				errStr := err.Error()
				for _, msg := range tt.errMsgs {
					if !strings.Contains(errStr, msg) {
						t.Errorf("User.Validate() error should contain '%s', got %v", msg, errStr)
					}
				}
			}
		})
	}
}

func TestUser_SetPassword(t *testing.T) {
	user := NewUser(ID("123"), "johndoe", "john@example.com")

	user.SetPassword("hashed_password")

	if user.PasswordHash != "hashed_password" {
		t.Errorf("User.SetPassword() PasswordHash = %v, want %v", user.PasswordHash, "hashed_password")
	}
	// UpdatedAt is set during SetPassword, we just verify it's not zero
	if user.UpdatedAt.IsZero() {
		t.Error("User.SetPassword() should set UpdatedAt")
	}
}

func TestUser_ActivateDeactivate(t *testing.T) {
	user := NewUser(ID("123"), "johndoe", "john@example.com")

	user.Activate()
	if user.Status != StatusActive {
		t.Errorf("User.Activate() Status = %v, want %v", user.Status, StatusActive)
	}
	if !user.IsActive() {
		t.Error("User.IsActive() should return true after Activate()")
	}

	user.Deactivate()
	if user.Status != StatusInactive {
		t.Errorf("User.Deactivate() Status = %v, want %v", user.Status, StatusInactive)
	}
	if user.IsActive() {
		t.Error("User.IsActive() should return false after Deactivate()")
	}
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		role UserRole
		want bool
	}{
		{RoleAdmin, true},
		{RoleUser, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			user := &User{Role: tt.role}
			if got := user.IsAdmin(); got != tt.want {
				t.Errorf("User.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_Permissions(t *testing.T) {
	tests := []struct {
		name              string
		role              UserRole
		canManageUsers    bool
		canManageMailboxes bool
		canViewMessages   bool
	}{
		{"admin", RoleAdmin, true, true, true},
		{"user", RoleUser, false, true, true},
		{"viewer", RoleViewer, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role}
			if got := user.CanManageUsers(); got != tt.canManageUsers {
				t.Errorf("User.CanManageUsers() = %v, want %v", got, tt.canManageUsers)
			}
			if got := user.CanManageMailboxes(); got != tt.canManageMailboxes {
				t.Errorf("User.CanManageMailboxes() = %v, want %v", got, tt.canManageMailboxes)
			}
			if got := user.CanViewMessages(); got != tt.canViewMessages {
				t.Errorf("User.CanViewMessages() = %v, want %v", got, tt.canViewMessages)
			}
		})
	}
}

func TestUser_RecordLogin(t *testing.T) {
	user := NewUser(ID("123"), "johndoe", "john@example.com")

	if user.LastLoginAt != nil {
		t.Error("User.LastLoginAt should be nil initially")
	}

	user.RecordLogin()

	if user.LastLoginAt == nil {
		t.Error("User.RecordLogin() should set LastLoginAt")
	}
}

func TestUser_GetDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		displayName string
		want        string
	}{
		{"with display name", "johndoe", "John Doe", "John Doe"},
		{"without display name", "johndoe", "", "johndoe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Username: tt.username, DisplayName: tt.displayName}
			if got := user.GetDisplayName(); got != tt.want {
				t.Errorf("User.GetDisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_JSONMarshal(t *testing.T) {
	user := NewUser(ID("123"), "johndoe", "john@example.com")
	user.PasswordHash = "secret_hash"
	user.DisplayName = "John Doe"

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// PasswordHash should not be in JSON
	if strings.Contains(string(data), "secret_hash") {
		t.Error("json.Marshal() should not include PasswordHash")
	}

	// Other fields should be present
	if !strings.Contains(string(data), "johndoe") {
		t.Error("json.Marshal() should include username")
	}
}

func TestUserCreateInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *UserCreateInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: &UserCreateInput{
				Username: "johndoe",
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "missing username",
			input: &UserCreateInput{
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "missing email",
			input: &UserCreateInput{
				Username: "johndoe",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			input: &UserCreateInput{
				Username: "johndoe",
				Email:    "john@example.com",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			input: &UserCreateInput{
				Username: "johndoe",
				Email:    "john@example.com",
				Password: "short",
			},
			wantErr: true,
		},
		{
			name: "password too long",
			input: &UserCreateInput{
				Username: "johndoe",
				Email:    "john@example.com",
				Password: strings.Repeat("a", 129),
			},
			wantErr: true,
		},
		{
			name: "invalid role",
			input: &UserCreateInput{
				Username: "johndoe",
				Email:    "john@example.com",
				Password: "password123",
				Role:     UserRole("invalid"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UserCreateInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserCreateInput_Normalize(t *testing.T) {
	input := &UserCreateInput{
		Username:    "  johndoe  ",
		Email:       "  JOHN@EXAMPLE.COM  ",
		DisplayName: "  John Doe  ",
	}

	input.Normalize()

	if input.Username != "johndoe" {
		t.Errorf("Normalize() Username = %v, want %v", input.Username, "johndoe")
	}
	if input.Email != "john@example.com" {
		t.Errorf("Normalize() Email = %v, want %v", input.Email, "john@example.com")
	}
	if input.DisplayName != "John Doe" {
		t.Errorf("Normalize() DisplayName = %v, want %v", input.DisplayName, "John Doe")
	}
}

func TestUserUpdateInput_Validate(t *testing.T) {
	invalidEmail := "not-an-email"
	invalidRole := UserRole("invalid")
	invalidStatus := UserStatus("invalid")

	tests := []struct {
		name    string
		input   *UserUpdateInput
		wantErr bool
	}{
		{
			name:    "empty update (valid)",
			input:   &UserUpdateInput{},
			wantErr: false,
		},
		{
			name:    "invalid email",
			input:   &UserUpdateInput{Email: &invalidEmail},
			wantErr: true,
		},
		{
			name:    "invalid role",
			input:   &UserUpdateInput{Role: &invalidRole},
			wantErr: true,
		},
		{
			name:    "invalid status",
			input:   &UserUpdateInput{Status: &invalidStatus},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UserUpdateInput.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserUpdateInput_Apply(t *testing.T) {
	user := NewUser(ID("123"), "johndoe", "john@example.com")
	user.Status = StatusActive

	newDisplayName := "John Doe"
	newEmail := "newemail@example.com"
	newRole := RoleAdmin
	newStatus := StatusInactive

	input := &UserUpdateInput{
		DisplayName: &newDisplayName,
		Email:       &newEmail,
		Role:        &newRole,
		Status:      &newStatus,
	}

	input.Apply(user)

	if user.DisplayName != newDisplayName {
		t.Errorf("Apply() DisplayName = %v, want %v", user.DisplayName, newDisplayName)
	}
	if user.Email != newEmail {
		t.Errorf("Apply() Email = %v, want %v", user.Email, newEmail)
	}
	if user.Role != newRole {
		t.Errorf("Apply() Role = %v, want %v", user.Role, newRole)
	}
	if user.Status != newStatus {
		t.Errorf("Apply() Status = %v, want %v", user.Status, newStatus)
	}
}
