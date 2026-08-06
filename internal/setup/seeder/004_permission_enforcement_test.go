package seeder

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unenforcedByDesign lists seeded permissions that deliberately have no guard.
//
// It is empty, and adding to it needs a reason written here. A permission the
// server never checks is not a harmless placeholder: it lets an administrator
// compose a role that reads like a grant, hand it out, and believe access was
// conferred when nothing changed. Reviewers and compliance evidence read the
// catalog, not the routes.
var unenforcedByDesign = map[string]string{}

// permissionMiddlewarePattern captures the literal permission lists passed to
// middleware.PermissionMiddleware. Every call site in internal/ uses this
// literal form; enforcedPermissionNames fails if the scan finds none, so a
// refactor to a non-literal form surfaces as a failure rather than as a silently
// empty enforced set.
var permissionMiddlewarePattern = regexp.MustCompile(`PermissionMiddleware\(\[\]string\{([^}]*)\}`)

// grpcPermissionValuePattern captures the map VALUES of grpcServicePermissions,
// which is the second (and only other) place a permission is enforced.
var grpcPermissionValuePattern = regexp.MustCompile(`\):\s*"([^"]*)"`)

var quotedStringPattern = regexp.MustCompile(`"([^"]*)"`)

func TestSeededPermissionsMatchEnforcedPermissions(t *testing.T) {
	seeded := make(map[string]struct{})
	for _, permission := range defaultPermissions(1, 2) {
		seeded[permission.Name] = struct{}{}
	}

	enforced := enforcedPermissionNames(t)

	t.Run("every enforced permission is seeded", func(t *testing.T) {
		// GET /users/membership-candidates shipped guarded by tenant:member:create
		// while the seeder created no such row, so the endpoint returned 403 to
		// every role including super-admin and the console's Add Member picker
		// could never load. A guard naming a permission that does not exist is
		// unreachable for everyone.
		var missing []string
		for name := range enforced {
			if _, ok := seeded[name]; !ok {
				missing = append(missing, name)
			}
		}
		sort.Strings(missing)
		assert.Empty(t, missing, "guarded but never seeded — the endpoints are unreachable for every role")
	})

	t.Run("every seeded permission is enforced", func(t *testing.T) {
		var orphaned []string
		for name := range seeded {
			if _, ok := enforced[name]; ok {
				continue
			}
			if _, exempt := unenforcedByDesign[name]; exempt {
				continue
			}
			orphaned = append(orphaned, name)
		}
		sort.Strings(orphaned)
		assert.Empty(t, orphaned, "seeded but guarded nowhere — granting these authorises nothing")
	})
}

func TestDefaultPermissionsDropsUnenforceableNames(t *testing.T) {
	// Inverted from the original catalog, which seeded all of these. They named
	// capabilities the server does not implement (root:impersonate, audit:export,
	// the notification:*/system:* families), duplicated a name that IS enforced
	// (security:policy:update vs security-setting:update), or described work that
	// is really guarded by a coarser permission (user:disable, user:role:assign
	// and security:session:terminate:any are all user:update). Seeding them let a
	// role advertise access it did not grant.
	seeded := make(map[string]struct{})
	for _, permission := range defaultPermissions(1, 2) {
		seeded[permission.Name] = struct{}{}
	}

	for _, name := range []string{
		"root:impersonate",
		"root:hard-delete-user",
		"user:disable",
		"user:enable",
		"user:role:assign",
		"user:role:remove",
		"role:assign",
		"security:session:terminate:any",
		"security:policy:update",
		"audit:export",
		"audit:read:any",
		"auth_event:delete",
		"system:access-db-console",
		"notification:send:custom",
		"settings:update:any",
		"settings:read",
		"email:update-config",
		"account:token:create:self",
		"public:login",
	} {
		t.Run(name, func(t *testing.T) {
			_, exists := seeded[name]
			assert.False(t, exists)
		})
	}
}

func TestSeededPermissionsIncludeSigningKeyGuard(t *testing.T) {
	// Inverted. This name used to be asserted ABSENT by
	// TestDefaultPermissionsDropsUnenforceableNames, on the correct reasoning at
	// the time that nothing enforced it — the key lifecycle was a repository with
	// no callers and rotation only ever fired on a 24h timer. The lifecycle
	// endpoints now exist and guard on it, so the old assertion inverts: absent,
	// the guard 403s every role and an operator whose signing key leaked still has
	// no way to disown it.
	seeded := make(map[string]struct{})
	for _, permission := range defaultPermissions(1, 2) {
		seeded[permission.Name] = struct{}{}
	}

	_, exists := seeded["security:rotate-keys"]
	assert.True(t, exists, "POST /oauth/signing-keys/rotate is guarded by this name")

	// Rotate mints the global (tenant_id IS NULL) key and retire/compromise take a
	// bare kid, so the blast radius of this permission is the whole deployment. It
	// must not reach a tenant admin.
	assert.Contains(t, systemOnlyPermissions, "security:rotate-keys")
}

func TestSeededPermissionsIncludeMembershipCandidateGuard(t *testing.T) {
	seeded := make(map[string]struct{})
	for _, permission := range defaultPermissions(1, 2) {
		seeded[permission.Name] = struct{}{}
	}

	_, exists := seeded["tenant:member:create"]
	assert.True(t, exists, "GET /users/membership-candidates is guarded by this name")

	// Adding members is a tenant-admin action, not a platform action, so the
	// permission must reach non-system tenants. systemOnlyPermissions is skipped
	// for them.
	assert.NotContains(t, systemOnlyPermissions, "tenant:member:create")
}

func TestRegisteredRoleCoversEverySelfPermission(t *testing.T) {
	seeded := make(map[string]struct{})
	for _, permission := range defaultPermissions(1, 2) {
		seeded[permission.Name] = struct{}{}
	}

	registered := make(map[string]struct{}, len(registeredAccountPermissions))
	for _, name := range registeredAccountPermissions {
		registered[name] = struct{}{}
	}

	t.Run("no self permission is left to super-admin only", func(t *testing.T) {
		// SeedRolePermissions gives super-admin everything NOT in this list, so a
		// self permission omitted here is granted to admins instead of to the
		// registered role every user holds — ordinary users lose access to their
		// own account.
		var missing []string
		for name := range seeded {
			if !strings.HasSuffix(name, shared.PermissionSuffixSelf) {
				continue
			}
			if _, ok := registered[name]; !ok {
				missing = append(missing, name)
			}
		}
		sort.Strings(missing)
		assert.Empty(t, missing)
	})

	t.Run("no entry refers to a permission that is not seeded", func(t *testing.T) {
		var stale []string
		for _, name := range registeredAccountPermissions {
			if _, ok := seeded[name]; !ok {
				stale = append(stale, name)
			}
		}
		sort.Strings(stale)
		assert.Empty(t, stale)
	})

	t.Run("no administrative permission is granted to every user", func(t *testing.T) {
		// The registered role is held by every account, so a management-plane name
		// slipping into this list would hand the whole tenant that access. The
		// :self suffix is the marker; note that shared.IsElevatedPermission is
		// deliberately stricter (it only exempts public: and account:…:self), so it
		// cannot be used here — settings:read:self is personal but classified
		// elevated on purpose.
		for _, name := range registeredAccountPermissions {
			assert.True(t, strings.HasSuffix(name, shared.PermissionSuffixSelf), name)
		}
	})
}

func TestControlPolicyAllowsOnlyExistingPermissionFamilies(t *testing.T) {
	// An allow pattern that matches no seeded permission is a standing
	// pre-authorisation for whatever is later created under that name — the
	// policy would widen before anyone reviewed the new permission.
	seeded := defaultPermissions(1, 2)

	for _, action := range DefaultControlActions {
		{
			t.Run(action, func(t *testing.T) {
				matched := false
				for _, permission := range seeded {
					if wildcardActionMatches(action, permission.Name) {
						matched = true
						break
					}
				}
				assert.True(t, matched, "control policy allows a permission family that does not exist")
			})
		}
	}
}

// wildcardActionMatches mirrors the "*" semantics of iam.wildcardMatch closely
// enough for the catalog cross-check (prefix/suffix around a single wildcard).
func wildcardActionMatches(pattern, name string) bool {
	if pattern == name {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return false
	}
	parts := strings.Split(pattern, "*")
	rest := name
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(rest, part)
		if idx < 0 {
			return false
		}
		if i == 0 && idx != 0 {
			return false
		}
		rest = rest[idx+len(part):]
	}
	if strings.HasSuffix(pattern, "*") {
		return true
	}
	return rest == ""
}

// enforcedPermissionNames scans the source tree for the two places a permission
// name is actually checked: PermissionMiddleware on HTTP routes and the
// grpcServicePermissions map. Tests are excluded — a name only a test mentions is
// not enforced.
func enforcedPermissionNames(t *testing.T) map[string]struct{} {
	t.Helper()

	root := repoRoot(t)
	names := make(map[string]struct{})

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// Generated protobuf output holds no hand-written guards and is large.
			if entry.Name() == "gen" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		source := string(content)

		for _, match := range permissionMiddlewarePattern.FindAllStringSubmatch(source, -1) {
			for _, quoted := range quotedStringPattern.FindAllStringSubmatch(match[1], -1) {
				if name := strings.TrimSpace(quoted[1]); name != "" {
					names[name] = struct{}{}
				}
			}
		}

		if block, ok := grpcPermissionMapBlock(source); ok {
			for _, match := range grpcPermissionValuePattern.FindAllStringSubmatch(block, -1) {
				// "" marks a method that is authenticated but not PDP-gated.
				if name := strings.TrimSpace(match[1]); name != "" {
					names[name] = struct{}{}
				}
			}
		}

		return nil
	})
	require.NoError(t, err)

	// A scan that finds nothing would make "every seeded permission is enforced"
	// pass by vacuum and "every enforced permission is seeded" pass trivially, so
	// an unreadable or restructured tree must fail loudly instead.
	require.NotEmpty(t, names, "found no permission guards in internal/ — the scanner is out of date")

	return names
}

func grpcPermissionMapBlock(source string) (string, bool) {
	const marker = "grpcServicePermissions = map[string]string{"
	start := strings.Index(source, marker)
	if start < 0 {
		return "", false
	}
	rest := source[start:]
	end := strings.Index(rest, "\n}")
	if end < 0 {
		return rest, true
	}
	return rest[:end], true
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "could not locate go.mod above %s", dir)
		dir = parent
	}
}
