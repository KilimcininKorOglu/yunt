package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// emailRegex is a simple regex pattern for basic email validation.
// Allows standard emails and also accepts domains without TLD like "localhost".
// For production use, consider more comprehensive validation.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+(\.[a-zA-Z]{2,})?$`)

// User represents a user account in the Yunt mail server.
// Users can have multiple mailboxes and can be assigned different roles
// to control their access level within the system.
type User struct {
	// ID is the unique identifier for the user.
	ID ID `json:"id"`

	// Username is the unique login name for the user.
	// Must be alphanumeric and between 3-50 characters.
	Username string `json:"username"`

	// Email is the user's email address for notifications and recovery.
	Email string `json:"email"`

	// PasswordHash is the bcrypt hash of the user's password.
	// This field is never serialized to JSON.
	PasswordHash string `json:"-"`

	// DisplayName is the user's preferred display name.
	// Optional, defaults to Username if not set.
	DisplayName string `json:"displayName,omitempty"`

	// Role determines the user's access level in the system.
	Role UserRole `json:"role"`

	// Status indicates whether the user account is active.
	Status UserStatus `json:"status"`

	// AvatarURL is an optional URL to the user's profile picture.
	AvatarURL string `json:"avatarUrl,omitempty"`

	// LastLoginAt is the timestamp of the user's last successful login.
	LastLoginAt *Timestamp `json:"lastLoginAt,omitempty"`

	// CreatedAt is the timestamp when the user was created.
	CreatedAt Timestamp `json:"createdAt"`

	// UpdatedAt is the timestamp when the user was last updated.
	UpdatedAt Timestamp `json:"updatedAt"`
}

// NewUser creates a new User with default values.
// The ID should be set by the caller using an ID generator.
func NewUser(id ID, username, email string) *User {
	now := Now()
	return &User{
		ID:        id,
		Username:  username,
		Email:     email,
		Role:      RoleUser,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate checks if the user has valid field values.
// Returns a ValidationErrors if any validation fails.
func (u *User) Validate() error {
	errs := NewValidationErrors()

	// Validate ID
	if u.ID.IsEmpty() {
		errs.Add("id", "id is required")
	}

	// Validate Username
	if u.Username == "" {
		errs.Add("username", "username is required")
	} else if len(u.Username) < 3 {
		errs.Add("username", "username must be at least 3 characters")
	} else if len(u.Username) > 50 {
		errs.Add("username", "username must be at most 50 characters")
	} else if !isValidUsername(u.Username) {
		errs.Add("username", "username must contain only alphanumeric characters, underscores, and hyphens")
	}

	// Validate Email
	if u.Email == "" {
		errs.Add("email", "email is required")
	} else if !isValidEmail(u.Email) {
		errs.Add("email", "email format is invalid")
	}

	// Validate Role
	if !u.Role.IsValid() {
		errs.Add("role", "invalid role")
	}

	// Validate Status
	if !u.Status.IsValid() {
		errs.Add("status", "invalid status")
	}

	// Validate DisplayName length if set
	if u.DisplayName != "" && utf8.RuneCountInString(u.DisplayName) > 100 {
		errs.Add("displayName", "display name must be at most 100 characters")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// SetPassword sets the password hash for the user.
// The caller is responsible for hashing the password before calling this method.
func (u *User) SetPassword(hash string) {
	u.PasswordHash = hash
	u.UpdatedAt = Now()
}

// Activate sets the user status to active.
func (u *User) Activate() {
	u.Status = StatusActive
	u.UpdatedAt = Now()
}

// Deactivate sets the user status to inactive.
func (u *User) Deactivate() {
	u.Status = StatusInactive
	u.UpdatedAt = Now()
}

// IsActive returns true if the user account is active.
func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

// IsAdmin returns true if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// CanManageUsers returns true if the user can manage other users.
func (u *User) CanManageUsers() bool {
	return u.Role == RoleAdmin
}

// CanManageMailboxes returns true if the user can manage mailboxes.
func (u *User) CanManageMailboxes() bool {
	return u.Role == RoleAdmin || u.Role == RoleUser
}

// CanViewMessages returns true if the user can view messages.
func (u *User) CanViewMessages() bool {
	return u.Role == RoleAdmin || u.Role == RoleUser || u.Role == RoleViewer
}

// RecordLogin updates the last login timestamp.
func (u *User) RecordLogin() {
	now := Now()
	u.LastLoginAt = &now
}

// GetDisplayName returns the display name or username as fallback.
func (u *User) GetDisplayName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}

// isValidUsername checks if a username contains only valid characters.
func isValidUsername(username string) bool {
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// isValidEmail performs basic email validation.
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// UserCreateInput represents the input for creating a new user.
type UserCreateInput struct {
	// Username is the unique login name for the user.
	Username string `json:"username"`

	// Email is the user's email address.
	Email string `json:"email"`

	// Password is the plaintext password (will be hashed).
	Password string `json:"password"`

	// DisplayName is the user's preferred display name.
	DisplayName string `json:"displayName,omitempty"`

	// Role is the user's role (defaults to RoleUser).
	Role UserRole `json:"role,omitempty"`
}

// Validate checks if the create input is valid.
func (i *UserCreateInput) Validate() error {
	errs := NewValidationErrors()

	// Validate Username
	username := strings.TrimSpace(i.Username)
	if username == "" {
		errs.Add("username", "username is required")
	} else if len(username) < 3 {
		errs.Add("username", "username must be at least 3 characters")
	} else if len(username) > 50 {
		errs.Add("username", "username must be at most 50 characters")
	} else if !isValidUsername(username) {
		errs.Add("username", "username must contain only alphanumeric characters, underscores, and hyphens")
	}

	// Validate Email
	email := strings.TrimSpace(i.Email)
	if email == "" {
		errs.Add("email", "email is required")
	} else if !isValidEmail(email) {
		errs.Add("email", "email format is invalid")
	}

	// Validate Password
	if i.Password == "" {
		errs.Add("password", "password is required")
	} else if len(i.Password) < 8 {
		errs.Add("password", "password must be at least 8 characters")
	} else if len(i.Password) > 128 {
		errs.Add("password", "password must be at most 128 characters")
	}

	// Validate Role if provided
	if i.Role != "" && !i.Role.IsValid() {
		errs.Add("role", "invalid role")
	}

	// Validate DisplayName length if set
	if i.DisplayName != "" && utf8.RuneCountInString(i.DisplayName) > 100 {
		errs.Add("displayName", "display name must be at most 100 characters")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Normalize trims and normalizes the input fields.
func (i *UserCreateInput) Normalize() {
	i.Username = strings.TrimSpace(i.Username)
	i.Email = strings.TrimSpace(strings.ToLower(i.Email))
	i.DisplayName = strings.TrimSpace(i.DisplayName)
}

// UserUpdateInput represents the input for updating a user.
type UserUpdateInput struct {
	// DisplayName is the new display name (optional).
	DisplayName *string `json:"displayName,omitempty"`

	// Email is the new email address (optional).
	Email *string `json:"email,omitempty"`

	// Role is the new role (optional, admin only).
	Role *UserRole `json:"role,omitempty"`

	// Status is the new status (optional, admin only).
	Status *UserStatus `json:"status,omitempty"`

	// AvatarURL is the new avatar URL (optional).
	AvatarURL *string `json:"avatarUrl,omitempty"`
}

// Validate checks if the update input is valid.
func (i *UserUpdateInput) Validate() error {
	errs := NewValidationErrors()

	// Validate DisplayName if provided
	if i.DisplayName != nil && utf8.RuneCountInString(*i.DisplayName) > 100 {
		errs.Add("displayName", "display name must be at most 100 characters")
	}

	// Validate Email if provided
	if i.Email != nil {
		email := strings.TrimSpace(*i.Email)
		if email == "" {
			errs.Add("email", "email cannot be empty")
		} else if !isValidEmail(email) {
			errs.Add("email", "email format is invalid")
		}
	}

	// Validate Role if provided
	if i.Role != nil && !i.Role.IsValid() {
		errs.Add("role", "invalid role")
	}

	// Validate Status if provided
	if i.Status != nil && !i.Status.IsValid() {
		errs.Add("status", "invalid status")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Apply applies the update to the given user.
func (i *UserUpdateInput) Apply(user *User) {
	if i.DisplayName != nil {
		user.DisplayName = strings.TrimSpace(*i.DisplayName)
	}
	if i.Email != nil {
		user.Email = strings.TrimSpace(strings.ToLower(*i.Email))
	}
	if i.Role != nil {
		user.Role = *i.Role
	}
	if i.Status != nil {
		user.Status = *i.Status
	}
	if i.AvatarURL != nil {
		user.AvatarURL = strings.TrimSpace(*i.AvatarURL)
	}
	user.UpdatedAt = Now()
}

// UserFilter represents filtering options for listing users.
type UserFilter struct {
	// Status filters by user status.
	Status *UserStatus `json:"status,omitempty"`

	// Role filters by user role.
	Role *UserRole `json:"role,omitempty"`

	// Search is a text search on username, email, and display name.
	Search string `json:"search,omitempty"`
}
