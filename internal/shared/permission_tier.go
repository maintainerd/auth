package shared

import "strings"

// Permission namespace prefixes. These mirror the three tiers documented in
// internal/setup/seeder/004_permission.go:
//
//	public:*            auto-assigned to every user (register, login, config…)
//	account:*:self      personal — the holder's own data only
//	everything else     "STRICT PERMISSIONS … elevated access" (tenant:*, user:*,
//	                    role:*, registration-flow:* …)
const (
	PermissionPrefixPublic  = "public:"
	PermissionPrefixAccount = "account:"
	PermissionSuffixSelf    = ":self"
)

// IsElevatedPermission reports whether a permission grants anything beyond
// public and own-account access — i.e. whether it belongs to the management
// plane.
//
// The classification is derived from the permission's namespace rather than a
// hand-maintained list, so newly seeded permissions are treated as elevated by
// default. That is the safe direction: a permission nobody has classified must
// not silently become self-service grantable.
func IsElevatedPermission(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	if strings.HasPrefix(n, PermissionPrefixPublic) {
		return false
	}
	// account:… is personal only when it is explicitly scoped to :self.
	// account:user:read (no :self) reads ANY user and is management-plane.
	if strings.HasPrefix(n, PermissionPrefixAccount) && strings.HasSuffix(n, PermissionSuffixSelf) {
		return false
	}
	return true
}

// FirstElevatedPermission returns the first management-plane permission in the
// list, or "" when every permission is public/own-account. Callers use the
// returned name to tell an operator exactly which permission blocked them.
func FirstElevatedPermission(names []string) string {
	for _, name := range names {
		if IsElevatedPermission(name) {
			return name
		}
	}
	return ""
}
