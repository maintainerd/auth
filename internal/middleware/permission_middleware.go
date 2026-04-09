package middleware

import (
	"net/http"

	"github.com/maintainerd/auth/internal/model"
	resp "github.com/maintainerd/auth/internal/rest/response"
)

// PermissionMiddleware ensures the user has at least one of the required permissions
func PermissionMiddleware(requiredPermissions []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := AuthFromRequest(r)
			if auth.User == nil {
				resp.Error(w, http.StatusUnauthorized, "User not found in context")
				return
			}

			// Check user permission
			if !hasAnyPermission(auth.User, requiredPermissions) {
				resp.Error(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hasAnyPermission checks if the user has at least one of the required permissions
func hasAnyPermission(user *model.User, required []string) bool {
	userPerms := make(map[string]bool)

	// Collect user permissions
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			userPerms[perm.Name] = true
		}
	}

	// Check if any required permission is present
	for _, rp := range required {
		if userPerms[rp] {
			return true
		}
	}

	return false
}
