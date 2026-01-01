package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"yunt/internal/domain"
	"yunt/internal/service"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// userContextKey is the context key for the authenticated user.
	userContextKey contextKey = "user"
	// claimsContextKey is the context key for the token claims.
	claimsContextKey contextKey = "claims"
	// sessionIDContextKey is the context key for the session ID.
	sessionIDContextKey contextKey = "sessionId"
)

// AuthConfig holds configuration for the authentication middleware.
type AuthConfig struct {
	// AuthService is the authentication service for token validation.
	AuthService *service.AuthService
	// SkipPaths is a list of paths that don't require authentication.
	SkipPaths []string
	// Optional indicates whether authentication is optional.
	// If true, requests without tokens will proceed but without user context.
	Optional bool
}

// Auth returns an authentication middleware that validates JWT tokens.
func Auth(authService *service.AuthService) echo.MiddlewareFunc {
	return AuthWithConfig(AuthConfig{
		AuthService: authService,
	})
}

// AuthWithConfig returns an authentication middleware with custom configuration.
func AuthWithConfig(cfg AuthConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if path should be skipped
			path := c.Request().URL.Path
			for _, skipPath := range cfg.SkipPaths {
				if path == skipPath || strings.HasPrefix(path, skipPath) {
					return next(c)
				}
			}

			// Extract token from Authorization header
			authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
			if authHeader == "" {
				if cfg.Optional {
					return next(c)
				}
				return unauthorizedError(c, "missing authorization header")
			}

			// Check Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				if cfg.Optional {
					return next(c)
				}
				return unauthorizedError(c, "invalid authorization header format")
			}

			tokenString := parts[1]
			if tokenString == "" {
				if cfg.Optional {
					return next(c)
				}
				return unauthorizedError(c, "missing token")
			}

			// Validate token
			claims, err := cfg.AuthService.ValidateAccessToken(c.Request().Context(), tokenString)
			if err != nil {
				// Handle specific token errors
				switch e := err.(type) {
				case *domain.ExpiredTokenError:
					return unauthorizedError(c, "token expired")
				case *domain.InvalidTokenError:
					return unauthorizedError(c, e.Error())
				case *domain.SessionRevokedError:
					return unauthorizedError(c, "session revoked")
				default:
					if cfg.Optional {
						return next(c)
					}
					return unauthorizedError(c, "invalid token")
				}
			}

			// Store claims in context
			ctx := c.Request().Context()
			ctx = context.WithValue(ctx, claimsContextKey, claims)
			ctx = context.WithValue(ctx, sessionIDContextKey, claims.SessionID)
			c.SetRequest(c.Request().WithContext(ctx))

			// Store claims in Echo context for easy access
			c.Set("claims", claims)
			c.Set("userId", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("userRole", claims.Role)
			c.Set("sessionId", claims.SessionID)

			return next(c)
		}
	}
}

// RequireRole returns a middleware that requires a specific role.
func RequireRole(roles ...domain.UserRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return unauthorizedError(c, "authentication required")
			}

			// Check if user has one of the required roles
			for _, role := range roles {
				if claims.Role == role {
					return next(c)
				}
			}

			return forbiddenError(c, "insufficient permissions")
		}
	}
}

// RequireAdmin returns a middleware that requires admin role.
func RequireAdmin() echo.MiddlewareFunc {
	return RequireRole(domain.RoleAdmin)
}

// RequireUser returns a middleware that requires at least user role.
func RequireUser() echo.MiddlewareFunc {
	return RequireRole(domain.RoleAdmin, domain.RoleUser)
}

// RequireViewer returns a middleware that requires at least viewer role.
func RequireViewer() echo.MiddlewareFunc {
	return RequireRole(domain.RoleAdmin, domain.RoleUser, domain.RoleViewer)
}

// GetClaims retrieves the token claims from the Echo context.
func GetClaims(c echo.Context) *domain.TokenClaims {
	if claims, ok := c.Get("claims").(*domain.TokenClaims); ok {
		return claims
	}
	return nil
}

// GetClaimsFromContext retrieves the token claims from a standard context.
func GetClaimsFromContext(ctx context.Context) *domain.TokenClaims {
	if claims, ok := ctx.Value(claimsContextKey).(*domain.TokenClaims); ok {
		return claims
	}
	return nil
}

// GetUserID retrieves the user ID from the Echo context.
func GetUserID(c echo.Context) domain.ID {
	if userID, ok := c.Get("userId").(domain.ID); ok {
		return userID
	}
	return ""
}

// GetUsername retrieves the username from the Echo context.
func GetUsername(c echo.Context) string {
	if username, ok := c.Get("username").(string); ok {
		return username
	}
	return ""
}

// GetUserRole retrieves the user role from the Echo context.
func GetUserRole(c echo.Context) domain.UserRole {
	if role, ok := c.Get("userRole").(domain.UserRole); ok {
		return role
	}
	return ""
}

// GetSessionID retrieves the session ID from the Echo context.
func GetSessionID(c echo.Context) string {
	if sessionID, ok := c.Get("sessionId").(string); ok {
		return sessionID
	}
	return ""
}

// GetSessionIDFromContext retrieves the session ID from a standard context.
func GetSessionIDFromContext(ctx context.Context) string {
	if sessionID, ok := ctx.Value(sessionIDContextKey).(string); ok {
		return sessionID
	}
	return ""
}

// IsAuthenticated checks if the request has valid authentication.
func IsAuthenticated(c echo.Context) bool {
	return GetClaims(c) != nil
}

// IsAdmin checks if the authenticated user is an admin.
func IsAdmin(c echo.Context) bool {
	return GetUserRole(c) == domain.RoleAdmin
}

// HasRole checks if the authenticated user has one of the specified roles.
func HasRole(c echo.Context, roles ...domain.UserRole) bool {
	userRole := GetUserRole(c)
	for _, role := range roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// AuthErrorResponse represents an authentication error response.
type AuthErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// unauthorizedError sends a 401 Unauthorized response.
func unauthorizedError(c echo.Context, message string) error {
	return c.JSON(http.StatusUnauthorized, AuthErrorResponse{
		Success: false,
		Error:   "UNAUTHORIZED",
		Message: message,
	})
}

// forbiddenError sends a 403 Forbidden response.
func forbiddenError(c echo.Context, message string) error {
	return c.JSON(http.StatusForbidden, AuthErrorResponse{
		Success: false,
		Error:   "FORBIDDEN",
		Message: message,
	})
}
