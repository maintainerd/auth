package federation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
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

// The rule used to be a raw count of non-wildcard characters across the WHOLE
// pattern, which the issuer's generic prefix satisfied on its own: "repo:" is
// already 5 of the 6 characters it wanted, so "repo:a*" passed and matched every
// repository of every organisation whose name starts with "a". An attacker creates
// a matching repo, its GitHub-issued token matches the victim's federation, and the
// exchange hands back the victim tenant's access token.
func TestSubjectPatternTooBroad_RejectsUnanchoredOrgSegment(t *testing.T) {
	unanchored := []string{
		"repo:a*",              // the original report: every org starting with "a"
		"repo:*",               // every org
		"repo:my-org*",         // "my-org-evil" matches too
		"system:*",             // every Kubernetes namespace
		"system:service*",      // still inside the generic prefix
		"project_path:group?*", // GitLab, same shape
		"my-workload-*",        // flat subject: no segment to anchor on
	}
	for _, pattern := range unanchored {
		assert.True(t, SubjectPatternTooBroad(pattern),
			"%q leaves the organisation segment unanchored and must be refused", pattern)
	}

	// Anchoring on a WHOLE org/namespace segment is the legitimate use, and must
	// keep working however short the real org name is — "a" is a valid GitHub org,
	// and "repo:a/*" pins exactly that one.
	anchored := []string{
		"repo:a/*",
		"repo:my-org/*",
		"repo:my-org/my-repo*",
		"system:serviceaccount:prod:*",
		"spiffe://my-domain/ns/*",
	}
	for _, pattern := range anchored {
		assert.False(t, SubjectPatternTooBroad(pattern),
			"%q pins a whole organisation or namespace segment and must be allowed", pattern)
	}
}

// The gate runs at match time too, so an over-broad row that reached the table by
// any route is skipped rather than honoured.
func TestMatchFederation_SkipsUnanchoredOrgPattern(t *testing.T) {
	svc := &workloadIdentityFederationService{}
	claims := map[string]interface{}{
		"aud": "https://auth.example.com",
		// An attacker-controlled repo under an org starting with "a".
		"sub": "repo:attacker-org/pwn:ref:refs/heads/main",
	}
	feds := []WorkloadIdentityFederation{
		{
			WorkloadIdentityFederationID: 1,
			TenantID:                     7,
			Audience:                     "https://auth.example.com",
			SubjectClaim:                 "sub",
			SubjectPattern:               "repo:a*",
		},
	}

	fed, oerr := svc.matchFederation(feds, claims)
	assert.Nil(t, oerr)
	assert.Nil(t, fed, "an unanchored org segment must not match another org's repo")
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

// ---------------------------------------------------------------------------
// Tenant claim
// ---------------------------------------------------------------------------

// stubTenantRefResolver installs a tenant id<->uuid mapping for the duration of a
// test and restores the previous (nil) one.
type stubTenantRefResolver struct {
	byID map[int64]uuid.UUID
}

func (s stubTenantRefResolver) TenantUUIDByID(_ context.Context, id int64) (uuid.UUID, bool) {
	u, ok := s.byID[id]
	return u, ok
}

func (s stubTenantRefResolver) TenantIDByUUID(_ context.Context, u uuid.UUID) (int64, bool) {
	for id, candidate := range s.byID {
		if candidate == u {
			return id, true
		}
	}
	return 0, false
}

func withTenantRefResolver(t *testing.T, byID map[int64]uuid.UUID) {
	t.Helper()
	shared.SetTenantRefResolver(stubTenantRefResolver{byID: byID})
	t.Cleanup(func() { shared.SetTenantRefResolver(nil) })
}

// The issued token carried NO tenant_id claim: "tenant:N" was being passed as the
// 7th positional argument to the generator, which is providerID. Every consumer
// reads the tenant from the tenant_id claim, so a WIF token resolved to
// TenantID = 0 — no tenant scoping at all on a keyless credential path.
func TestWorkloadTokenClaims_StampsTheFederationsTenant(t *testing.T) {
	tenantUUID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000012")
	withTenantRefResolver(t, map[int64]uuid.UUID{12: tenantUUID})

	fed := &WorkloadIdentityFederation{TenantID: 12, SubjectClaim: "sub"}
	extra, oerr := workloadTokenClaims(
		context.Background(),
		map[string]interface{}{"sub": "repo:my-org/my-repo:ref:refs/heads/main"},
		fed,
	)
	require.Nil(t, oerr)

	// The claim VALUE is the tenant's opaque UUID, not the internal PK — that is
	// what buildJWTClaims and the gRPC interceptor parse.
	assert.Equal(t, tenantUUID.String(), extra["tenant_id"])
	assert.NotEqual(t, "tenant:12", extra["tenant_id"])
}

// An operator's attribute_mapping must not be able to reach the claim the system
// stamps — the exchange endpoint takes no client credentials, so a forged tenant
// is an unauthenticated cross-tenant escalation.
func TestWorkloadTokenClaims_TenantIsNotForgeableByMapping(t *testing.T) {
	realTenant := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000012")
	withTenantRefResolver(t, map[int64]uuid.UUID{12: realTenant})

	fed := &WorkloadIdentityFederation{
		TenantID:         12,
		SubjectClaim:     "sub",
		AttributeMapping: datatypes.JSON([]byte(`{"evil":"tenant_id"}`)),
	}
	extra, oerr := workloadTokenClaims(
		context.Background(),
		map[string]interface{}{
			"sub":  "repo:my-org/my-repo:ref:refs/heads/main",
			"evil": "bbbbbbbb-0000-0000-0000-000000000099",
		},
		fed,
	)
	require.Nil(t, oerr)
	assert.Equal(t, realTenant.String(), extra["tenant_id"])
}

// Fail closed: this endpoint takes no client credentials, so a token whose tenant
// reads as 0 downstream is worse than a denial.
func TestWorkloadTokenClaims_RefusesWhenTenantCannotBeResolved(t *testing.T) {
	withTenantRefResolver(t, map[int64]uuid.UUID{})

	_, oerr := workloadTokenClaims(
		context.Background(),
		map[string]interface{}{"sub": "repo:my-org/my-repo:ref:refs/heads/main"},
		&WorkloadIdentityFederation{TenantID: 12, SubjectClaim: "sub"},
	)
	require.NotNil(t, oerr)
}

// ---------------------------------------------------------------------------
// Scope allow-lists
// ---------------------------------------------------------------------------

// The exchange consulted the federation row only, so a WIF token could carry
// scopes the SAME client is refused at /oauth/token — the keyless path was the
// looser door into the client's own grant.
func TestIntersectAllowedScopes(t *testing.T) {
	t.Run("a federation cannot widen its client's grant", func(t *testing.T) {
		got := intersectAllowedScopes(
			[]string{"deploy:write", "admin:all"},
			[]string{"deploy:write"},
		)
		assert.Equal(t, []string{"deploy:write"}, got)
	})

	t.Run("a client cannot widen its federation's grant either", func(t *testing.T) {
		got := intersectAllowedScopes(
			[]string{"deploy:write"},
			[]string{"deploy:write", "admin:all"},
		)
		assert.Equal(t, []string{"deploy:write"}, got)
	})

	// An empty list means "no scopes", never "all scopes" — the whole point is
	// that neither side may widen the other.
	t.Run("an empty client allow-list grants nothing", func(t *testing.T) {
		assert.Empty(t, intersectAllowedScopes([]string{"deploy:write"}, nil))
		assert.Empty(t, intersectAllowedScopes([]string{"deploy:write"}, []string{}))
	})

	t.Run("federation ordering is preserved so the scope string is deterministic", func(t *testing.T) {
		got := intersectAllowedScopes(
			[]string{"c", "a", "b"},
			[]string{"b", "a", "c"},
		)
		assert.Equal(t, []string{"c", "a", "b"}, got)
	})
}

// The requested-scope check runs against the intersected list, so a scope the
// federation allows but the client does not is refused outright rather than
// silently granted.
func TestIntersectScopes_AgainstIntersectedAllowList(t *testing.T) {
	allowed := intersectAllowedScopes([]string{"deploy:write", "admin:all"}, []string{"deploy:write"})

	granted, ok := intersectScopes("deploy:write", allowed)
	require.True(t, ok)
	assert.Equal(t, []string{"deploy:write"}, granted)

	_, ok = intersectScopes("admin:all", allowed)
	assert.False(t, ok, "a scope the mapped client is not allowed must be refused")

	// An omitted scope parameter grants the intersection, not the federation list.
	granted, ok = intersectScopes("", allowed)
	require.True(t, ok)
	assert.Equal(t, []string{"deploy:write"}, granted)
}
