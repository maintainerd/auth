package middleware

import (
	"net/http"

	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
)

// PermissionMiddleware ensures the user has at least one of the required permissions
func PermissionMiddleware(requiredPermissions []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			val := r.Context().Value(UserContextKey)
			if val == nil {
				util.Error(w, http.StatusUnauthorized, "User not found in context")
				return
			}

			user, ok := val.(*model.User)
			if !ok || user == nil {
				util.Error(w, http.StatusInternalServerError, "Invalid user in context")
				return
			}

			if !hasAnyPermission(user, requiredPermissions) {
				util.Error(w, http.StatusForbidden, "Insufficient permissions")
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
