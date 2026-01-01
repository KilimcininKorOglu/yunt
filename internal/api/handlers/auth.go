// Package handlers provides HTTP request handlers for the Yunt API.
package handlers

import (
	"github.com/labstack/echo/v4"

	"yunt/internal/api"
	"yunt/internal/api/middleware"
	"yunt/internal/domain"
	"yunt/internal/service"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterRoutes registers the authentication routes on the given group.
func (h *AuthHandler) RegisterRoutes(g *echo.Group) {
	auth := g.Group("/auth")
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.RefreshToken)
	auth.POST("/logout", h.Logout, middleware.Auth(h.authService))
	auth.POST("/logout-all", h.LogoutAll, middleware.Auth(h.authService))
	auth.GET("/me", h.GetCurrentUser, middleware.Auth(h.authService))
}

// Login handles user login requests.
// @Summary Login
// @Description Authenticate a user and return JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body domain.LoginInput true "Login credentials"
// @Success 200 {object} api.Response{data=domain.AuthResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var input domain.LoginInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	// Get client information
	userAgent := c.Request().UserAgent()
	ipAddress := c.RealIP()

	response, err := h.authService.Login(c.Request().Context(), &input, userAgent, ipAddress)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, response)
}

// RefreshToken handles token refresh requests.
// @Summary Refresh Token
// @Description Generate new access and refresh tokens using a valid refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param input body domain.RefreshTokenInput true "Refresh token"
// @Success 200 {object} api.Response{data=domain.AuthResponse}
// @Failure 400 {object} api.Response{error=api.ErrorDetail}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 422 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var input domain.RefreshTokenInput
	if err := c.Bind(&input); err != nil {
		return api.BadRequest(c, "invalid request body")
	}

	response, err := h.authService.RefreshToken(c.Request().Context(), &input)
	if err != nil {
		return api.FromError(c, err)
	}

	return api.OK(c, response)
}

// Logout handles user logout requests.
// @Summary Logout
// @Description Invalidate the current session
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	sessionID := middleware.GetSessionID(c)
	if sessionID == "" {
		return api.Unauthorized(c, "session not found")
	}

	if err := h.authService.Logout(c.Request().Context(), sessionID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// LogoutAll handles logout from all devices.
// @Summary Logout All
// @Description Invalidate all sessions for the current user
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsEmpty() {
		return api.Unauthorized(c, "user not found")
	}

	if err := h.authService.LogoutAll(c.Request().Context(), userID); err != nil {
		return api.FromError(c, err)
	}

	return api.NoContent(c)
}

// GetCurrentUser returns the currently authenticated user.
// @Summary Get Current User
// @Description Get the profile of the currently authenticated user
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response{data=domain.UserInfo}
// @Failure 401 {object} api.Response{error=api.ErrorDetail}
// @Failure 500 {object} api.Response{error=api.ErrorDetail}
// @Router /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return api.Unauthorized(c, "authentication required")
	}

	userInfo := &domain.UserInfo{
		ID:       claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Role:     claims.Role,
	}

	return api.OK(c, userInfo)
}
