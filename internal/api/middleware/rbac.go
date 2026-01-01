// Package middleware provides HTTP middleware for the Yunt API.
package middleware

import (
	"github.com/labstack/echo/v4"

	"yunt/internal/domain"
)

// RBACConfig holds configuration for role-based access control middleware.
type RBACConfig struct {
	// AllowedRoles specifies which roles can access the endpoint.
	AllowedRoles []domain.UserRole
	// RequireOwnership requires the user to be the resource owner.
	RequireOwnership bool
	// OwnerIDParam is the name of the URL parameter containing the owner ID.
	OwnerIDParam string
}

// RBAC returns a middleware that enforces role-based access control.
// It checks if the authenticated user has one of the allowed roles.
// If RequireOwnership is true, it also checks if the user is the resource owner.
func RBAC(cfg RBACConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return unauthorizedError(c, "authentication required")
			}

			// Check if user has one of the allowed roles
			hasRequiredRole := false
			for _, role := range cfg.AllowedRoles {
				if claims.Role == role {
					hasRequiredRole = true
					break
				}
			}

			// Admin always has access regardless of ownership
			if claims.Role == domain.RoleAdmin {
				return next(c)
			}

			// If ownership is required and user doesn't have admin role
			if cfg.RequireOwnership && cfg.OwnerIDParam != "" {
				ownerID := c.Param(cfg.OwnerIDParam)
				if ownerID != "" && ownerID == claims.UserID.String() {
					// User is the owner, allow access
					return next(c)
				}
				// User is not the owner and not admin
				if !hasRequiredRole {
					return forbiddenError(c, "access denied: you can only access your own resources")
				}
			}

			// Check role if no ownership bypass
			if !hasRequiredRole {
				return forbiddenError(c, "insufficient permissions")
			}

			return next(c)
		}
	}
}

// AdminOnly returns a middleware that only allows admin users.
func AdminOnly() echo.MiddlewareFunc {
	return RBAC(RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleAdmin},
	})
}

// UserOrAdmin returns a middleware that allows user or admin roles.
func UserOrAdmin() echo.MiddlewareFunc {
	return RBAC(RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleAdmin, domain.RoleUser},
	})
}

// ViewerOrHigher returns a middleware that allows viewer, user, or admin roles.
func ViewerOrHigher() echo.MiddlewareFunc {
	return RBAC(RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleAdmin, domain.RoleUser, domain.RoleViewer},
	})
}

// OwnerOrAdmin returns a middleware that allows access if:
// - The user is an admin, OR
// - The user is the resource owner (identified by the URL parameter).
func OwnerOrAdmin(ownerIDParam string) echo.MiddlewareFunc {
	return RBAC(RBACConfig{
		AllowedRoles:     []domain.UserRole{domain.RoleAdmin, domain.RoleUser, domain.RoleViewer},
		RequireOwnership: true,
		OwnerIDParam:     ownerIDParam,
	})
}

// RequirePermission returns a middleware that checks if the user has specific permissions.
// The permission check is based on the user's role and the permission type.
func RequirePermission(permission Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return unauthorizedError(c, "authentication required")
			}

			if !HasPermission(claims.Role, permission) {
				return forbiddenError(c, "insufficient permissions")
			}

			return next(c)
		}
	}
}

// Permission represents a specific permission in the system.
type Permission string

const (
	// PermissionUserRead allows reading user information.
	PermissionUserRead Permission = "user:read"
	// PermissionUserWrite allows creating and updating users.
	PermissionUserWrite Permission = "user:write"
	// PermissionUserDelete allows deleting users.
	PermissionUserDelete Permission = "user:delete"
	// PermissionUserManage allows managing all user operations (admin only).
	PermissionUserManage Permission = "user:manage"

	// PermissionMailboxRead allows reading mailbox information.
	PermissionMailboxRead Permission = "mailbox:read"
	// PermissionMailboxWrite allows creating and updating mailboxes.
	PermissionMailboxWrite Permission = "mailbox:write"
	// PermissionMailboxDelete allows deleting mailboxes.
	PermissionMailboxDelete Permission = "mailbox:delete"

	// PermissionMessageRead allows reading messages.
	PermissionMessageRead Permission = "message:read"
	// PermissionMessageWrite allows creating and updating messages.
	PermissionMessageWrite Permission = "message:write"
	// PermissionMessageDelete allows deleting messages.
	PermissionMessageDelete Permission = "message:delete"

	// PermissionSettingsRead allows reading system settings.
	PermissionSettingsRead Permission = "settings:read"
	// PermissionSettingsWrite allows modifying system settings.
	PermissionSettingsWrite Permission = "settings:write"

	// PermissionWebhookRead allows reading webhook configurations.
	PermissionWebhookRead Permission = "webhook:read"
	// PermissionWebhookWrite allows creating and updating webhooks.
	PermissionWebhookWrite Permission = "webhook:write"
	// PermissionWebhookDelete allows deleting webhooks.
	PermissionWebhookDelete Permission = "webhook:delete"
)

// rolePermissions defines which permissions each role has.
var rolePermissions = map[domain.UserRole][]Permission{
	domain.RoleAdmin: {
		PermissionUserRead,
		PermissionUserWrite,
		PermissionUserDelete,
		PermissionUserManage,
		PermissionMailboxRead,
		PermissionMailboxWrite,
		PermissionMailboxDelete,
		PermissionMessageRead,
		PermissionMessageWrite,
		PermissionMessageDelete,
		PermissionSettingsRead,
		PermissionSettingsWrite,
		PermissionWebhookRead,
		PermissionWebhookWrite,
		PermissionWebhookDelete,
	},
	domain.RoleUser: {
		PermissionUserRead,
		PermissionMailboxRead,
		PermissionMailboxWrite,
		PermissionMailboxDelete,
		PermissionMessageRead,
		PermissionMessageWrite,
		PermissionMessageDelete,
		PermissionWebhookRead,
		PermissionWebhookWrite,
	},
	domain.RoleViewer: {
		PermissionUserRead,
		PermissionMailboxRead,
		PermissionMessageRead,
	},
}

// HasPermission checks if a role has the specified permission.
func HasPermission(role domain.UserRole, permission Permission) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	return false
}

// GetPermissions returns all permissions for a given role.
func GetPermissions(role domain.UserRole) []Permission {
	permissions, ok := rolePermissions[role]
	if !ok {
		return nil
	}

	// Return a copy to prevent modification
	result := make([]Permission, len(permissions))
	copy(result, permissions)
	return result
}

// CanUserManageOtherUser checks if the current user can manage another user.
// Admins can manage all users, users can only manage themselves.
func CanUserManageOtherUser(c echo.Context, targetUserID domain.ID) bool {
	claims := GetClaims(c)
	if claims == nil {
		return false
	}

	// Admins can manage everyone
	if claims.Role == domain.RoleAdmin {
		return true
	}

	// Users can only manage themselves
	return claims.UserID == targetUserID
}

// IsSelfOrAdmin checks if the current user is accessing their own resource or is an admin.
func IsSelfOrAdmin(c echo.Context, targetUserID domain.ID) bool {
	return CanUserManageOtherUser(c, targetUserID)
}
