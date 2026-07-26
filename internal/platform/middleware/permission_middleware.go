package middleware

import (
	"context"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
	"github.com/maintainerd/maintainerd-auth/internal/platform/telemetry"
)

// OnAccessDenied, when set at startup, records a durable AUTHZ/failure auth event
// for an authenticated-but-unauthorized request. It is a package-level hook (like
// security.OnAccountLockout) so the boundary check does not have to thread the
// auth-event service through every route's PermissionMiddleware call. Nil is safe
// (no-op). Denials are also always metered via telemetry.RecordSecurityDenial so
// alerting works even when the durable hook is not wired.
var OnAccessDenied func(ctx context.Context, tenantID, actorUserID int64, ip string, requiredPermissions []string)

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
				// Failed authorization is a security-relevant event (NIST AU-2 /
				// PCI 10.2.4): meter it for alerting on privilege probing, and record
				// a durable AUTHZ/failure auth event with the actor + required perms.
				telemetry.RecordSecurityDenial(r.Context(), telemetry.DenialPermission)
				if OnAccessDenied != nil {
					var tenantID int64
					if auth.Tenant != nil {
						tenantID = auth.Tenant.TenantID
					}
					OnAccessDenied(r.Context(), tenantID, auth.User.UserID, ClientIPFromContext(r.Context()), requiredPermissions)
				}
				resp.Error(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hasAnyPermission checks if the user has at least one of the required permissions
func hasAnyPermission(user *authctx.AuthUser, required []string) bool {
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
