package seeder

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The seeders write straight through GORM, so nothing forces what they create to
// satisfy the DTO validators the API enforces on the same rows afterwards. A
// seeded row the validator rejects is not a caught error — it is a fresh install
// whose console cannot re-save its own baseline. These tests pin that contract.
//
// The rules below are transcribed from the validators rather than imported: the
// patterns and reserved sets are unexported. internal/iam/validation_permission.go
// is the source of truth; when it moves, this moves with it.
var (
	seededPermissionNameFormat = regexp.MustCompile(`^[a-z0-9]+([_-][a-z0-9]+)*(:[a-z0-9]+([_-][a-z0-9]+)*){1,3}$`)

	// reservedPermissionNamespaces in internal/iam. Every namespace the seeder
	// allocates has to be in it, or a tenant admin holding `permission:create` can
	// mint a permission with the same name as one of these route guards.
	seededReservedPermissionNamespaces = map[string]struct{}{
		"account": {}, "api": {}, "audit": {}, "auth_event": {}, "branding": {},
		"client": {}, "email": {}, "email-config": {}, "email-template": {},
		"idp": {}, "ip-restriction-rule": {}, "notification": {}, "permission": {},
		"policy": {}, "public": {}, "registration-flow": {}, "role": {}, "root": {},
		"security": {}, "security-setting": {}, "service": {}, "settings": {},
		"sms-config": {}, "sms-template": {}, "system": {}, "tenant": {},
		"tenant-setting": {}, "user": {}, "webhook-endpoint": {},
		"workload-identity-federation": {},
	}
)

func TestSeededPermissionsSatisfyTheValidator(t *testing.T) {
	for _, permission := range defaultPermissions(1, 2) {
		t.Run(permission.Name, func(t *testing.T) {
			assert.Regexp(t, seededPermissionNameFormat, permission.Name,
				"name must be 2 to 4 lowercase colon-separated segments")
			assert.GreaterOrEqual(t, len(permission.Name), 3)
			assert.LessOrEqual(t, len(permission.Name), 50)
			assert.LessOrEqual(t, len(permission.Description), 200)

			namespace, _, _ := strings.Cut(permission.Name, ":")
			assert.Contains(t, seededReservedPermissionNamespaces, namespace,
				"namespace is seeded but not reserved, so a tenant admin could mint the same name")
		})
	}
}

// The control policy is no longer seeded — it is built during setup — so the
// validator contract for its name and document moved to
// internal/setup: TestControlPolicySatisfiesTheValidator. What stays here is
// TestControlPolicyAllowsOnlyExistingPermissionFamilies, which has to live
// beside the seeded permission catalog it cross-checks against.

func TestSystemClientScopes(t *testing.T) {
	// offline_access is load-bearing: both seeded clients declare the
	// refresh_token grant and no refresh token is minted without it.
	assert.Contains(t, systemClientScopes, "offline_access")
	assert.Contains(t, systemClientScopes, "openid")

	// A scope is not a permission. If a permission name ever leaks into the
	// allow-list it grants nothing and only widens the scope claim.
	for _, scope := range systemClientScopes {
		assert.NotContains(t, scope, ":", "%q looks like a permission name, not an OIDC scope", scope)
	}
}

func TestSeededEmailTemplatesCoverTheEmailChangedNotice(t *testing.T) {
	templates := defaultEmailTemplates(1)

	names := make(map[string]int, len(templates))
	for i, template := range templates {
		require.NotContains(t, names, template.Name, "duplicate email template")
		names[template.Name] = i
	}

	index, ok := names["user:email:changed"]
	require.True(t, ok, "the email-change security notice must be seeded")

	notice := templates[index]
	require.NotNil(t, notice.BodyPlain)
	require.NotNil(t, notice.ParametersDoc)
	assert.Equal(t, "active", notice.Status)

	// Every parameter must appear in both bodies AND be documented: an
	// undocumented placeholder renders as "<no value>" for whoever edits the
	// template in the console and drops it from the notice.
	for _, param := range []string{"{{.PreviousEmail}}", "{{.NewEmail}}", "{{.LogoURL}}"} {
		assert.Contains(t, notice.BodyHTML, param)
		assert.Contains(t, *notice.ParametersDoc, param)
	}
	for _, param := range []string{"{{.PreviousEmail}}", "{{.NewEmail}}"} {
		assert.Contains(t, *notice.BodyPlain, param)
	}

	// Distinct from user:email:change, the OTP sent to the NEW address.
	assert.Contains(t, names, "user:email:change")
}
