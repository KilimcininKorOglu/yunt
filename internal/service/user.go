// Package service provides business logic services for the Yunt mail server.
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"yunt/internal/config"
	"yunt/internal/domain"
	"yunt/internal/repository"
)

// UserService provides user management business logic.
type UserService struct {
	config         config.AuthConfig
	userRepo       repository.UserRepository
	webhookService *WebhookService
}

// NewUserService creates a new UserService.
func NewUserService(cfg config.AuthConfig, userRepo repository.UserRepository) *UserService {
	return &UserService{
		config:   cfg,
		userRepo: userRepo,
	}
}

// WithWebhookService sets the webhook service for user event dispatch.
func (s *UserService) WithWebhookService(ws *WebhookService) {
	s.webhookService = ws
}

// UserListResponse represents a paginated list of users.
type UserListResponse struct {
	// Users contains the user list.
	Users []*domain.User `json:"users"`
	// Total is the total number of users matching the filter.
	Total int64 `json:"total"`
	// Page is the current page number.
	Page int `json:"page"`
	// PageSize is the number of items per page.
	PageSize int `json:"pageSize"`
	// TotalPages is the total number of pages.
	TotalPages int `json:"totalPages"`
}

// List retrieves a paginated list of users with optional filtering.
func (s *UserService) List(ctx context.Context, filter *repository.UserFilter, page, pageSize int) (*UserListResponse, error) {
	// Normalize pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = repository.DefaultPerPage
	}
	if pageSize > repository.MaxPerPage {
		pageSize = repository.MaxPerPage
	}

	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    page,
			PerPage: pageSize,
		},
		Sort: &repository.SortOptions{
			Field: string(repository.UserSortByCreatedAt),
			Order: domain.SortDesc,
		},
	}

	result, err := s.userRepo.List(ctx, filter, opts)
	if err != nil {
		return nil, domain.NewInternalError("failed to list users", err)
	}

	users := result.Items

	// Calculate total pages
	totalPages := int(result.Total) / pageSize
	if int(result.Total)%pageSize > 0 {
		totalPages++
	}

	return &UserListResponse{
		Users:      users,
		Total:      result.Total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByID retrieves a user by their ID.
// Returns domain.ErrNotFound if the user does not exist.
func (s *UserService) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.NewNotFoundError("user", id.String())
		}
		return nil, domain.NewInternalError("failed to retrieve user", err)
	}
	return user, nil
}

// GetUserInfo retrieves public user information by ID.
// This is useful for API responses where password hash should never be included.
func (s *UserService) GetUserInfo(ctx context.Context, id domain.ID) (*domain.UserInfo, error) {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return domain.UserInfoFromUser(user), nil
}

// GetUserProfile retrieves the full user profile for the user's own profile view.
// This includes all fields except the password hash.
type UserProfile struct {
	ID          domain.ID          `json:"id"`
	Username    string             `json:"username"`
	Email       string             `json:"email"`
	DisplayName string             `json:"displayName,omitempty"`
	Role        domain.UserRole    `json:"role"`
	Status      domain.UserStatus  `json:"status"`
	AvatarURL   string             `json:"avatarUrl,omitempty"`
	LastLoginAt *domain.Timestamp  `json:"lastLoginAt,omitempty"`
	CreatedAt   domain.Timestamp   `json:"createdAt"`
	UpdatedAt   domain.Timestamp   `json:"updatedAt"`
}

// GetUserProfile retrieves the full profile for a user.
func (s *UserService) GetUserProfile(ctx context.Context, id domain.ID) (*UserProfile, error) {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Status:      user.Status,
		AvatarURL:   user.AvatarURL,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}, nil
}

// Create creates a new user with the given input.
// Returns domain.ErrAlreadyExists if username or email is taken.
func (s *UserService) Create(ctx context.Context, input *domain.UserCreateInput) (*domain.User, error) {
	// Normalize input
	input.Normalize()

	// Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check if username is taken
	exists, err := s.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return nil, domain.NewInternalError("failed to check username", err)
	}
	if exists {
		return nil, domain.NewAlreadyExistsError("user", "username", input.Username)
	}

	// Check if email is taken
	exists, err = s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, domain.NewInternalError("failed to check email", err)
	}
	if exists {
		return nil, domain.NewAlreadyExistsError("user", "email", input.Email)
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), s.config.BCryptCost)
	if err != nil {
		return nil, domain.NewInternalError("failed to hash password", err)
	}

	// Create user
	userID := domain.ID(uuid.New().String())
	user := domain.NewUser(userID, input.Username, input.Email)
	user.PasswordHash = string(passwordHash)
	user.DisplayName = input.DisplayName

	// Set role (default to user if not specified)
	if input.Role != "" {
		user.Role = input.Role
	} else {
		user.Role = domain.RoleUser
	}

	// Set status to active for new users created by admin
	user.Status = domain.StatusActive

	// Save user
	if err := s.userRepo.Create(ctx, user); err != nil {
		if domain.IsAlreadyExists(err) {
			return nil, err
		}
		return nil, domain.NewInternalError("failed to create user", err)
	}

	if s.webhookService != nil {
		go s.webhookService.TriggerUserCreated(context.Background(), user)
	}

	return user, nil
}

// Update updates an existing user with the given input.
// Returns domain.ErrNotFound if the user does not exist.
// Returns domain.ErrAlreadyExists if the new email conflicts with another user.
func (s *UserService) Update(ctx context.Context, id domain.ID, input *domain.UserUpdateInput, isAdmin bool) (*domain.User, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Get existing user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.NewNotFoundError("user", id.String())
		}
		return nil, domain.NewInternalError("failed to retrieve user", err)
	}

	// Check if new email is taken by another user
	if input.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*input.Email))
		if email != user.Email {
			existingUser, err := s.userRepo.GetByEmail(ctx, email)
			if err != nil && !domain.IsNotFound(err) {
				return nil, domain.NewInternalError("failed to check email", err)
			}
			if existingUser != nil && existingUser.ID != id {
				return nil, domain.NewAlreadyExistsError("user", "email", email)
			}
		}
	}

	// Non-admin users cannot change their own role or status
	if !isAdmin {
		input.Role = nil
		input.Status = nil
	}

	// Apply updates
	input.Apply(user)

	// Save user
	if err := s.userRepo.Update(ctx, user); err != nil {
		if domain.IsAlreadyExists(err) {
			return nil, err
		}
		return nil, domain.NewInternalError("failed to update user", err)
	}

	return user, nil
}

// Delete removes a user by their ID.
// Returns domain.ErrNotFound if the user does not exist.
func (s *UserService) Delete(ctx context.Context, id domain.ID) error {
	// Check if user exists
	exists, err := s.userRepo.Exists(ctx, id)
	if err != nil {
		return domain.NewInternalError("failed to check user", err)
	}
	if !exists {
		return domain.NewNotFoundError("user", id.String())
	}

	// Delete user (soft delete)
	if err := s.userRepo.SoftDelete(ctx, id); err != nil {
		return domain.NewInternalError("failed to delete user", err)
	}

	return nil
}

// UpdatePassword updates a user's password.
// Returns domain.ErrNotFound if the user does not exist.
func (s *UserService) UpdatePassword(ctx context.Context, id domain.ID, newPassword string) error {
	// Validate password
	if len(newPassword) < 8 {
		return domain.NewValidationError("password", "password must be at least 8 characters")
	}
	if len(newPassword) > 128 {
		return domain.NewValidationError("password", "password must be at most 128 characters")
	}

	// Check if user exists
	exists, err := s.userRepo.Exists(ctx, id)
	if err != nil {
		return domain.NewInternalError("failed to check user", err)
	}
	if !exists {
		return domain.NewNotFoundError("user", id.String())
	}

	// Hash new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.config.BCryptCost)
	if err != nil {
		return domain.NewInternalError("failed to hash password", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, id, string(passwordHash)); err != nil {
		return domain.NewInternalError("failed to update password", err)
	}

	return nil
}

// ChangePassword changes a user's password after verifying the current password.
func (s *UserService) ChangePassword(ctx context.Context, id domain.ID, currentPassword, newPassword string) error {
	// Get user to verify current password
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if domain.IsNotFound(err) {
			return domain.NewNotFoundError("user", id.String())
		}
		return domain.NewInternalError("failed to retrieve user", err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return domain.NewUnauthorizedError("current password is incorrect")
	}

	// Update to new password
	return s.UpdatePassword(ctx, id, newPassword)
}

// UpdateStatus updates a user's status.
// Returns domain.ErrNotFound if the user does not exist.
func (s *UserService) UpdateStatus(ctx context.Context, id domain.ID, status domain.UserStatus) error {
	// Validate status
	if !status.IsValid() {
		return domain.NewValidationError("status", "invalid status")
	}

	// Check if user exists
	exists, err := s.userRepo.Exists(ctx, id)
	if err != nil {
		return domain.NewInternalError("failed to check user", err)
	}
	if !exists {
		return domain.NewNotFoundError("user", id.String())
	}

	// Update status
	if err := s.userRepo.UpdateStatus(ctx, id, status); err != nil {
		return domain.NewInternalError("failed to update status", err)
	}

	return nil
}

// UpdateRole updates a user's role.
// Returns domain.ErrNotFound if the user does not exist.
func (s *UserService) UpdateRole(ctx context.Context, id domain.ID, role domain.UserRole) error {
	// Validate role
	if !role.IsValid() {
		return domain.NewValidationError("role", "invalid role")
	}

	// Check if user exists
	exists, err := s.userRepo.Exists(ctx, id)
	if err != nil {
		return domain.NewInternalError("failed to check user", err)
	}
	if !exists {
		return domain.NewNotFoundError("user", id.String())
	}

	// Update role
	if err := s.userRepo.UpdateRole(ctx, id, role); err != nil {
		return domain.NewInternalError("failed to update role", err)
	}

	return nil
}

// PasswordUpdateInput represents the input for updating a user's password.
type PasswordUpdateInput struct {
	// CurrentPassword is required when the user is changing their own password.
	CurrentPassword string `json:"currentPassword,omitempty"`
	// NewPassword is the new password to set.
	NewPassword string `json:"newPassword"`
}

// Validate checks if the password update input is valid.
func (i *PasswordUpdateInput) Validate() error {
	errs := domain.NewValidationErrors()

	if i.NewPassword == "" {
		errs.Add("newPassword", "new password is required")
	} else if len(i.NewPassword) < 8 {
		errs.Add("newPassword", "password must be at least 8 characters")
	} else if len(i.NewPassword) > 128 {
		errs.Add("newPassword", "password must be at most 128 characters")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// Search searches for users by username, email, or display name.
func (s *UserService) Search(ctx context.Context, query string, page, pageSize int) (*UserListResponse, error) {
	filter := &repository.UserFilter{
		Search: query,
	}
	return s.List(ctx, filter, page, pageSize)
}

// GetUserStats returns statistics about users in the system.
type UserStats struct {
	TotalUsers   int64                       `json:"totalUsers"`
	ByRole       map[domain.UserRole]int64   `json:"byRole"`
	ByStatus     map[domain.UserStatus]int64 `json:"byStatus"`
	ActiveUsers  int64                       `json:"activeUsers"`
	PendingUsers int64                       `json:"pendingUsers"`
}

// GetStats returns user statistics.
func (s *UserService) GetStats(ctx context.Context) (*UserStats, error) {
	total, err := s.userRepo.Count(ctx, nil)
	if err != nil {
		return nil, domain.NewInternalError("failed to count users", err)
	}

	byRole, err := s.userRepo.CountByRole(ctx)
	if err != nil {
		return nil, domain.NewInternalError("failed to count by role", err)
	}

	byStatus, err := s.userRepo.CountByStatus(ctx)
	if err != nil {
		return nil, domain.NewInternalError("failed to count by status", err)
	}

	return &UserStats{
		TotalUsers:   total,
		ByRole:       byRole,
		ByStatus:     byStatus,
		ActiveUsers:  byStatus[domain.StatusActive],
		PendingUsers: byStatus[domain.StatusPending],
	}, nil
}
