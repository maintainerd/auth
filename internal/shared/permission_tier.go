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

// FirstUnheldElevatedPermission returns the first management-plane permission in
// granting that does not appear in held, or "" when the actor holds all of them.
//
// This is the containment rule behind every "you cannot grant what you do not
// hold" guard in the codebase — role→permission, client→permission and
// user→role each apply it. It lived as three separate copies of the same loop,
// which is a bad shape for a privilege check: three places to keep the
// elevated-only filter, the set construction and the comparison in agreement,
// and a single divergence in any of them is a privilege-escalation hole that
// still passes its own domain's tests.
//
// Only elevated permissions are compared, because public:… and account:…:self
// confer nothing beyond the holder's own account and gating them would block
// ordinary self-service. A super-admin is seeded with every administrative
// permission, so the rule does not restrict them and needs no special case.
//
// The callers keep their own message and their own way of reading held, because
// those legitimately differ; what must not differ is this comparison.
//
// Note the direction of failure: the caller must fail CLOSED when it cannot read
// held. Passing an empty held slice means "the actor holds nothing", which
// refuses every elevated grant — the safe answer — so an error path that
// mistakenly reaches here still denies rather than allows.
func FirstUnheldElevatedPermission(granting []string, held []string) string {
	var elevated []string
	for _, name := range granting {
		if IsElevatedPermission(name) {
			elevated = append(elevated, name)
		}
	}
	if len(elevated) == 0 {
		return ""
	}

	heldSet := make(map[string]struct{}, len(held))
	for _, name := range held {
		heldSet[name] = struct{}{}
	}
	for _, name := range elevated {
		if _, ok := heldSet[name]; !ok {
			return name
		}
	}
	return ""
}
