package federation

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The exchange endpoint takes NO client credentials, and ExtraClaims is merged LAST
// over the standard claim set — so a mapping onto a reserved claim name is a
// token-forgery primitive. Mapping onto `sub` + `client_id` is the worst case:
// UserContextMiddleware resolves the user, tenant and roles from exactly those two
// claims, so a forged pair is an unauthenticated cross-tenant account takeover.
func TestBuildExtraClaims_DropsReservedDestinations(t *testing.T) {
	external := map[string]interface{}{
		"sub":           "repo:attacker/repo:ref:refs/heads/main",
		"victim_sub":    "11111111-2222-3333-4444-555555555555",
		"victim_client": "console-client",
		"team":          "platform",
	}

	mapping := map[string]string{
		"victim_sub":    "sub",       // identity takeover
		"victim_client": "client_id", // tenant + client takeover
		"team":          "svc",       // gRPC service principal
	}

	extra := buildExtraClaims(external, mapping, "sub")

	assert.NotContains(t, extra, "sub")
	assert.NotContains(t, extra, "client_id")
	assert.NotContains(t, extra, "svc")
}

func TestBuildExtraClaims_KeepsNonReservedDestinations(t *testing.T) {
	external := map[string]interface{}{
		"sub":        "repo:my-org/my-repo:ref:refs/heads/main",
		"repository": "my-org/my-repo",
		"ref":        "refs/heads/main",
	}
	mapping := map[string]string{
		"repository": "repository",
		"ref":        "git_ref",
	}

	extra := buildExtraClaims(external, mapping, "sub")

	assert.Equal(t, "my-org/my-repo", extra["repository"])
	assert.Equal(t, "refs/heads/main", extra["git_ref"])
}

// act.sub records the delegated workload per RFC 8693 §4.1. The SYSTEM sets it, so
// it is reserved against operator mapping but must still be stamped — and it must be
// set after the mapping so a mapping onto "act" cannot forge the delegation chain.
func TestBuildExtraClaims_SystemStampsActAfterMapping(t *testing.T) {
	external := map[string]interface{}{
		"sub":   "repo:my-org/my-repo:ref:refs/heads/main",
		"forge": map[string]any{"sub": "someone-else"},
	}

	extra := buildExtraClaims(external, map[string]string{"forge": "act"}, "sub")

	act, ok := extra["act"].(map[string]any)
	require.True(t, ok, "act must be stamped by the system")
	assert.Equal(t, "repo:my-org/my-repo:ref:refs/heads/main", act["sub"])
}

// Every reserved claim must be refused, not just the ones named above — the point of
// routing through jwt.IsReservedClaim is that the list is maintained in one place.
func TestBuildExtraClaims_DropsEveryReservedClaim(t *testing.T) {
	for _, reserved := range []string{
		"iss", "sub", "aud", "exp", "nbf", "iat", "jti", "scope", "client_id",
		"tenant_id", "permissions", "roles", "sid", "cnf", "acr", "amr",
		"svc", "provider_id", "token_type", "sub_type",
	} {
		require.True(t, jwt.IsReservedClaim(reserved), "%s should be reserved", reserved)

		extra := buildExtraClaims(
			map[string]interface{}{"sub": "workload", "evil": "forged"},
			map[string]string{"evil": reserved},
			"sub",
		)
		assert.NotEqual(t, "forged", extra[reserved], "reserved claim %q was forgeable", reserved)
	}
}

// A case variant must not slip past: JSON keys are case-sensitive, but a consumer
// that lowercases would treat "Sub" as the real subject.
func TestBuildExtraClaims_ReservedCheckIsCaseInsensitive(t *testing.T) {
	extra := buildExtraClaims(
		map[string]interface{}{"sub": "workload", "evil": "forged"},
		map[string]string{"evil": "SUB"},
		"sub",
	)
	assert.NotContains(t, extra, "SUB")
	assert.NotContains(t, extra, "sub")
}

// matchSubjectPattern stays deliberately permissive — it is a glob matcher, not a
// gate. Breadth is decided by SubjectPatternTooBroad, which runs BOTH at write time
// (validation) and at match time (matchFederation). This documents the split so
// nobody "fixes" the matcher instead of the gate.
func TestMatchSubjectPattern_IsIntentionallyPermissive(t *testing.T) {
	assert.True(t, matchSubjectPattern("*", "literally-anything"),
		"the glob matcher is permissive by design; SubjectPatternTooBroad is the gate")
	assert.True(t, matchSubjectPattern("repo:my-org/my-repo:*", "repo:my-org/my-repo:ref:refs/heads/main"))
	assert.False(t, matchSubjectPattern("repo:my-org/my-repo:*", "repo:other-org/repo:ref:refs/heads/main"))
	// A wildcard must span '/' and ':' — that is why path.Match is not used.
	assert.True(t, matchSubjectPattern("system:serviceaccount:prod:*", "system:serviceaccount:prod:api"))
}

// Write-time validation never re-runs on read, so a row that predates the rule — or
// one inserted by direct SQL, a restored backup, or a future import path — would
// otherwise keep matching every workload on its issuer forever.
func TestSubjectPatternTooBroad(t *testing.T) {
	tooBroad := []string{"*", "**", "*:ref:refs/heads/main", "?x", "rep*", "", "   "}
	for _, pattern := range tooBroad {
		assert.True(t, SubjectPatternTooBroad(pattern), "%q should be refused", pattern)
	}

	anchored := []string{
		"repo:my-org/my-repo:*",
		"system:serviceaccount:production:*",
		"repo:my-org/*",
		// An exact pattern cannot over-match, however short.
		"ci",
		"repo:my-org/my-repo:ref:refs/heads/main",
	}
	for _, pattern := range anchored {
		assert.False(t, SubjectPatternTooBroad(pattern), "%q should be allowed", pattern)
	}
}

// The matcher must skip an over-broad stored pattern rather than honour it.
func TestMatchFederation_SkipsOverBroadStoredPatterns(t *testing.T) {
	svc := &workloadIdentityFederationService{}
	claims := map[string]interface{}{
		"aud": "https://auth.example.com",
		"sub": "repo:someone-else/their-repo:ref:refs/heads/main",
	}

	// A row that should never have been storable.
	feds := []WorkloadIdentityFederation{
		{
			WorkloadIdentityFederationID: 1,
			TenantID:                     7,
			Audience:                     "https://auth.example.com",
			SubjectClaim:                 "sub",
			SubjectPattern:               "*",
		},
	}

	fed, oerr := svc.matchFederation(feds, claims)
	assert.Nil(t, oerr)
	assert.Nil(t, fed, "an over-broad stored pattern must not match, even though it was persisted")
}

// Ambiguity across tenants must fail closed: the request carries no tenant signal,
// so picking one would hand a workload to whichever row sorted first.
func TestMatchFederation_RejectsCrossTenantAmbiguity(t *testing.T) {
	svc := &workloadIdentityFederationService{}
	claims := map[string]interface{}{
		"aud": "https://auth.example.com",
		"sub": "repo:my-org/my-repo:ref:refs/heads/main",
	}
	feds := []WorkloadIdentityFederation{
		{WorkloadIdentityFederationID: 1, TenantID: 1, Audience: "https://auth.example.com",
			SubjectClaim: "sub", SubjectPattern: "repo:my-org/my-repo:*"},
		{WorkloadIdentityFederationID: 2, TenantID: 2, Audience: "https://auth.example.com",
			SubjectClaim: "sub", SubjectPattern: "repo:my-org/*"},
	}

	fed, oerr := svc.matchFederation(feds, claims)
	assert.Nil(t, fed)
	require.NotNil(t, oerr)
	assert.Contains(t, oerr.Error(), "more than one tenant")
}

// Two rules in the SAME tenant are not ambiguous in the dangerous sense — the caller
// has already proven that one trust relationship — so the deterministic first match
// (ordered by id in the repository) is used.
func TestMatchFederation_AllowsMultipleMatchesWithinOneTenant(t *testing.T) {
	svc := &workloadIdentityFederationService{}
	claims := map[string]interface{}{
		"aud": "https://auth.example.com",
		"sub": "repo:my-org/my-repo:ref:refs/heads/main",
	}
	feds := []WorkloadIdentityFederation{
		{WorkloadIdentityFederationID: 1, TenantID: 5, ClientID: 11, Audience: "https://auth.example.com",
			SubjectClaim: "sub", SubjectPattern: "repo:my-org/my-repo:*"},
		{WorkloadIdentityFederationID: 2, TenantID: 5, ClientID: 22, Audience: "https://auth.example.com",
			SubjectClaim: "sub", SubjectPattern: "repo:my-org/*"},
	}

	fed, oerr := svc.matchFederation(feds, claims)
	assert.Nil(t, oerr)
	require.NotNil(t, fed)
	assert.Equal(t, int64(11), fed.ClientID, "the first row by id must win, deterministically")
}
