// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

// UsersHandler handles user management HTTP requests.
type UsersHandler struct {
	userService *service.UserService
	authService *service.AuthService
}

// NewUsersHandler creates a new UsersHandler.
func NewUsersHandler(userService *service.UserService, authService *service.AuthService) *UsersHandler {
	return &UsersHandler{
		userService: userService,
		authService: authService,
	}
}

// RegisterRoutes registers the user management routes on the given group.
func (h *UsersHandler) RegisterRoutes(g *echo.Group, authService *service.AuthService) {
	users := g.Group("/users")

	// Apply authentication middleware to all user routes
	users.Use(middleware.Auth(authService))

	// Admin-only routes
	users.GET("", h.ListUsers, middleware.AdminOnly())
	users.POST("", h.CreateUser, middleware.AdminOnly())
	users.DELETE("/:id", h.DeleteUser, middleware.AdminOnly())
	users.PUT("/:id/role", h.UpdateUserRole, middleware.AdminOnly())
	users.PUT("/:id/status", h.UpdateUserStatus, middleware.AdminOnly())
	users.GET("/stats", h.GetUserStats, middleware.AdminOnly())

	// Owner or admin routes (user can access their own, admin can access all)
	users.GET("/:id", h.GetUser, middleware.OwnerOrAdmin("id"))
	users.PUT("/:id", h.UpdateUser, middleware.OwnerOrAdmin("id"))
	users.PUT("/:id/password", h.UpdatePassword, middleware.OwnerOrAdmin("id"))

	// Profile routes (current user)
	users.GET("/me/profile", h.GetMyProfile)
	users.PUT("/me/profile", h.UpdateMyProfile)
	users.PUT("/me/password", h.ChangeMyPassword)
}

// ListUsers handles listing all users (admin only).
// @Summary List Users
// @Description Get a paginated list of all users (admin only)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Items per page (default: 20, max: 100)"
// @Param status query string false "Filter by status (active, inactive, pending)"
// @Param role query string false "Filter by role (admin, user, viewer)"
// @Param search query string false "Search in username, email, display name"
// @Success 200 {object} api.Response{data=service.UserListResponse}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users [get]
func (h *UsersHandler) ListUsers(c echo.Context) error {
	// Parse pagination params
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))

	// Build filter
	filter := &repository.UserFilter{}

	// Status filter
	if statusStr := c.QueryParam("status"); statusStr != "" {
		status := domain.UserStatus(statusStr)
		if status.IsValid() {
			filter.Status = &status
		}
	}

	// Role filter
	if roleStr := c.QueryParam("role"); roleStr != "" {
		role := domain.UserRole(roleStr)
		if role.IsValid() {
			filter.Role = &role
		}
	}

	// Search filter
	if search := c.QueryParam("search"); search != "" {
		filter.Search = search
	}

	result, err := h.userService.List(c.Request().Context(), filter, page, pageSize)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, result)
}

// GetUser handles getting a single user by ID.
// @Summary Get User
// @Description Get a user by their ID. Users can view their own profile, admins can view all.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} api.Response{data=service.UserProfile}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/{id} [get]
func (h *UsersHandler) GetUser(c echo.Context) error {
	userID := domain.ID(c.Param("id"))

	profile, err := h.userService.GetUserProfile(c.Request().Context(), userID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, profile)
}

// CreateUser handles creating a new user (admin only).
// @Summary Create User
// @Description Create a new user (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.UserCreateInput true "User creation data"
// @Success 201 {object} api.Response{data=domain.UserInfo}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 409 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users [post]
func (h *UsersHandler) CreateUser(c echo.Context) error {
	var input domain.UserCreateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	user, err := h.userService.Create(c.Request().Context(), &input)
	if err != nil {
		return api.FromError(c, err)
	}

	// Return user info without password hash
	return api.Created(c, domain.UserInfoFromUser(user))
}

// UpdateUser handles updating a user.
// @Summary Update User
// @Description Update a user's information. Users can update their own profile, admins can update all.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param input body domain.UserUpdateInput true "User update data"
// @Success 200 {object} api.Response{data=service.UserProfile}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 409 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/{id} [put]
func (h *UsersHandler) UpdateUser(c echo.Context) error {
	userID := domain.ID(c.Param("id"))

	var input domain.UserUpdateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	// Check if request is from admin
	isAdmin := middleware.IsAdmin(c)

	user, err := h.userService.Update(c.Request().Context(), userID, &input, isAdmin)
	if err != nil {
		return api.FromError(c, err)
	}

	// Return full profile
	profile := &service.UserProfile{
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
	}

	return api.OK(c, profile)
}

// DeleteUser handles deleting a user (admin only).
// @Summary Delete User
// @Description Delete a user by ID (admin only). This performs a soft delete.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/{id} [delete]
func (h *UsersHandler) DeleteUser(c echo.Context) error {
	userID := domain.ID(c.Param("id"))

	// Prevent admin from deleting themselves
	claims := middleware.GetClaims(c)
	if claims != nil && claims.UserID == userID {
		return api.BadRequest(c, "cannot delete your own account")
	}

	if err := h.userService.Delete(c.Request().Context(), userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// UpdatePassword handles updating a user's password.
// @Summary Update Password
// @Description Update a user's password. Users updating their own password must provide current password.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param input body service.PasswordUpdateInput true "Password update data"
// @Success 204
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/{id}/password [put]
func (h *UsersHandler) UpdatePassword(c echo.Context) error {
	userID := domain.ID(c.Param("id"))

	var input service.PasswordUpdateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if err := input.Validate(); err != nil {
		return api.FromError(c, err)
	}

	claims := middleware.GetClaims(c)
	isAdmin := middleware.IsAdmin(c)
	isSelf := claims != nil && claims.UserID == userID

	// If user is changing their own password, require current password
	if isSelf && !isAdmin {
		if input.CurrentPassword == "" {
			return api.ValidationFailed(c, []*domain.ValidationError{
				{Field: "currentPassword", Message: "current password is required"},
			})
		}
		if err := h.userService.ChangePassword(c.Request().Context(), userID, input.CurrentPassword, input.NewPassword); err != nil {
			return api.FromError(c, err)
		}
	} else {
		// Admin can set password without current password
		if err := h.userService.UpdatePassword(c.Request().Context(), userID, input.NewPassword); err != nil {
			return api.FromError(c, err)
		}
	}

	return api.NoContent(c)
}

// UpdateUserRole handles updating a user's role (admin only).
// @Summary Update User Role
// @Description Update a user's role (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param input body roleUpdateInput true "Role update data"
// @Success 204
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/{id}/role [put]
func (h *UsersHandler) UpdateUserRole(c echo.Context) error {
	userID := domain.ID(c.Param("id"))

	var input roleUpdateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if input.Role == "" {
		return api.ValidationFailed(c, []*domain.ValidationError{
			{Field: "role", Message: "role is required"},
		})
	}

	role := domain.UserRole(input.Role)
	if !role.IsValid() {
		return api.ValidationFailed(c, []*domain.ValidationError{
			{Field: "role", Message: "invalid role, must be one of: admin, user, viewer"},
		})
	}

	// Prevent admin from demoting themselves
	claims := middleware.GetClaims(c)
	if claims != nil && claims.UserID == userID && role != domain.RoleAdmin {
		return api.BadRequest(c, "cannot change your own role")
	}

	if err := h.userService.UpdateRole(c.Request().Context(), userID, role); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// roleUpdateInput represents the input for updating a user's role.
type roleUpdateInput struct {
	Role string `json:"role"`
}

// UpdateUserStatus handles updating a user's status (admin only).
// @Summary Update User Status
// @Description Update a user's status (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param input body statusUpdateInput true "Status update data"
// @Success 204
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 404 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/{id}/status [put]
func (h *UsersHandler) UpdateUserStatus(c echo.Context) error {
	userID := domain.ID(c.Param("id"))

	var input statusUpdateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	if input.Status == "" {
		return api.ValidationFailed(c, []*domain.ValidationError{
			{Field: "status", Message: "status is required"},
		})
	}

	status := domain.UserStatus(input.Status)
	if !status.IsValid() {
		return api.ValidationFailed(c, []*domain.ValidationError{
			{Field: "status", Message: "invalid status, must be one of: active, inactive, pending"},
		})
	}

	// Prevent admin from deactivating themselves
	claims := middleware.GetClaims(c)
	if claims != nil && claims.UserID == userID && status != domain.StatusActive {
		return api.BadRequest(c, "cannot deactivate your own account")
	}

	if err := h.userService.UpdateStatus(c.Request().Context(), userID, status); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// statusUpdateInput represents the input for updating a user's status.
type statusUpdateInput struct {
	Status string `json:"status"`
}

// GetUserStats handles getting user statistics (admin only).
// @Summary Get User Statistics
// @Description Get statistics about users (admin only)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=service.UserStats}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 403 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/stats [get]
func (h *UsersHandler) GetUserStats(c echo.Context) error {
	stats, err := h.userService.GetStats(c.Request().Context())
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, stats)
}

// GetMyProfile handles getting the current user's profile.
// @Summary Get My Profile
// @Description Get the currently authenticated user's profile
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=service.UserProfile}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/me/profile [get]
func (h *UsersHandler) GetMyProfile(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return api.Unauthorized(c, "authentication required")
	}

	profile, err := h.userService.GetUserProfile(c.Request().Context(), claims.UserID)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, profile)
}

// UpdateMyProfile handles updating the current user's profile.
// @Summary Update My Profile
// @Description Update the currently authenticated user's profile
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body domain.UserUpdateInput true "Profile update data"
// @Success 200 {object} api.Response{data=service.UserProfile}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 409 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/me/profile [put]
func (h *UsersHandler) UpdateMyProfile(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return api.Unauthorized(c, "authentication required")
	}

	var input domain.UserUpdateInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	// Users cannot change their own role or status
	input.Role = nil
	input.Status = nil

	user, err := h.userService.Update(c.Request().Context(), claims.UserID, &input, false)
	if err != nil {
		return api.FromError(c, err)
	}

	profile := &service.UserProfile{
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
	}

	return api.OK(c, profile)
}

// ChangeMyPassword handles changing the current user's password.
// @Summary Change My Password
// @Description Change the currently authenticated user's password
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body changePasswordInput true "Password change data"
// @Success 204
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /users/me/password [put]
func (h *UsersHandler) ChangeMyPassword(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return api.Unauthorized(c, "authentication required")
	}

	var input changePasswordInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	// Validate input
	errs := domain.NewValidationErrors()
	if input.CurrentPassword == "" {
		errs.Add("currentPassword", "current password is required")
	}
	if input.NewPassword == "" {
		errs.Add("newPassword", "new password is required")
	} else if len(input.NewPassword) < 8 {
		errs.Add("newPassword", "password must be at least 8 characters")
	} else if len(input.NewPassword) > 128 {
		errs.Add("newPassword", "password must be at most 128 characters")
	}
	if errs.HasErrors() {
		return api.FromError(c, errs)
	}

	if err := h.userService.ChangePassword(c.Request().Context(), claims.UserID, input.CurrentPassword, input.NewPassword); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// changePasswordInput represents the input for changing the current user's password.
type changePasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
