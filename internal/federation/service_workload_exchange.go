package federation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	oidclib "github.com/coreos/go-oidc/v3/oidc"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	// tokenTypeAccessToken is the RFC 8693 issued_token_type URI for access tokens.
	wifTokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"
	// subjectTokenTypeJWT is the RFC 8693 subject_token_type for a generic JWT.
	wifSubjectTokenTypeJWT = "urn:ietf:params:oauth:token-type:jwt"
	// providerCacheTTL bounds how long a discovered OIDC provider (and its
	// cached JWKS) is reused before re-discovery. RFC 9449 §8 and general key
	// rotation hygiene favour a short window.
	providerCacheTTL = 5 * time.Minute
	// defaultWorkloadTokenTTLSeconds is the fallback access-token lifetime when
	// the mapped client does not set its own access_token_ttl.
	defaultWorkloadTokenTTLSeconds = 900
	// wifClockSkewLeeway is the allowed clock skew when verifying a subject
	// token's expiry. go-oidc has no built-in expiry tolerance, so a small skew
	// between us and the issuer would spuriously reject valid tokens. Bounded —
	// it never disables the expiry check.
	wifClockSkewLeeway = 1 * time.Minute
)

// wifGenerateAccessToken is a package var so tests can stub token minting.
var wifGenerateAccessToken = jwt.GenerateAccessTokenWithOptionsContext

// WorkloadExchangeInput is the federation-native input for a workload token
// exchange, mapped from the oauth token endpoint at the composition root.
type WorkloadExchangeInput struct {
	SubjectToken string
	Scope        string // requested scope (space-delimited, OAuth form)
	Audience     string
	Resource     string
	IPAddress    string
}

// WorkloadExchangeResult is the federation-native result mapped back to an
// oauth token response by the composition root.
type WorkloadExchangeResult struct {
	AccessToken     string
	IssuedTokenType string
	TokenType       string
	ExpiresIn       int
	Scope           string
}

// ---------------------------------------------------------------------------
// OIDC provider cache (JWKS caching, keyed by issuer_url, 5-minute TTL)
// ---------------------------------------------------------------------------

type cachedProvider struct {
	provider *oidclib.Provider
	expires  time.Time
}

type providerCache struct {
	mu      sync.Mutex
	entries map[string]cachedProvider
}

func newProviderCache() *providerCache {
	return &providerCache{entries: map[string]cachedProvider{}}
}

// wifNewProvider is a package var so tests can stub OIDC discovery.
var wifNewProvider = func(ctx context.Context, issuer string) (*oidclib.Provider, error) {
	octx := oidclib.ClientContext(ctx, federationHTTPClientFactory())
	return oidclib.NewProvider(octx, issuer)
}

// getOrDiscoverProvider returns a cached OIDC provider for the issuer, running
// discovery (and thereby caching the JWKS) at most once per TTL window.
func (s *workloadIdentityFederationService) getOrDiscoverProvider(ctx context.Context, issuer string) (*oidclib.Provider, error) {
	s.provider.mu.Lock()
	if cached, ok := s.provider.entries[issuer]; ok && time.Now().Before(cached.expires) {
		s.provider.mu.Unlock()
		return cached.provider, nil
	}
	s.provider.mu.Unlock()

	provider, err := wifNewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	s.provider.mu.Lock()
	s.provider.entries[issuer] = cachedProvider{provider: provider, expires: time.Now().Add(providerCacheTTL)}
	s.provider.mu.Unlock()
	return provider, nil
}

// probeIssuer validates that issuer is a reachable OIDC issuer by running
// discovery. Called on create/update so a misconfigured issuer is rejected at
// write time rather than silently failing every future exchange.
func (s *workloadIdentityFederationService) probeIssuer(ctx context.Context, issuer string) error {
	_, err := s.getOrDiscoverProvider(ctx, issuer)
	return err
}

// ---------------------------------------------------------------------------
// Exchange flow
// ---------------------------------------------------------------------------

// ExchangeWorkloadToken implements WorkloadIdentityFederationService.
func (s *workloadIdentityFederationService) ExchangeWorkloadToken(ctx context.Context, in WorkloadExchangeInput) (*WorkloadExchangeResult, *apperror.OAuthError) {
	ctx, span := otel.Tracer("service").Start(ctx, "workloadIdentityFederation.exchange")
	defer span.End()

	// Extract the unverified issuer to decide whether any federation applies.
	issuer, ok := unverifiedIssuer(in.SubjectToken)
	if !ok || issuer == "" {
		// Not a federation-shaped token — let the standard exchange handle it.
		return nil, nil
	}
	span.SetAttributes(attribute.String("wif.issuer", issuer))

	feds, err := s.repo.FindActiveByIssuer(issuer)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "federation lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if len(feds) == 0 {
		// No federation trusts this issuer — fall back to standard exchange.
		return nil, nil
	}

	// A federation trusts this issuer, so this IS a workload exchange attempt.
	// From here on, failures are hard errors (not fallbacks).
	provider, err := s.getOrDiscoverProvider(ctx, issuer)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "issuer discovery failed")
		return nil, apperror.NewOAuthInvalidGrant("the token issuer could not be verified")
	}

	// Allow a small clock skew when checking token expiry. go-oidc's expiry check
	// has no built-in tolerance, so shifting "now" slightly into the past via
	// oidc.Config.Now avoids spuriously rejecting freshly-issued tokens. This
	// tolerates a bounded skew only — it never disables the expiry check.
	verifier := provider.Verifier(&oidclib.Config{
		SkipClientIDCheck: true,
		Now:               func() time.Time { return time.Now().Add(-wifClockSkewLeeway) },
	})
	idToken, verr := verifier.Verify(ctx, in.SubjectToken)
	if verr != nil {
		span.SetStatus(codes.Error, "subject token verification failed")
		return nil, apperror.NewOAuthInvalidGrant("subject_token signature or claims are invalid")
	}

	var claims map[string]interface{}
	if cerr := idToken.Claims(&claims); cerr != nil {
		return nil, apperror.NewOAuthInvalidGrant("subject_token claims could not be parsed")
	}

	// Resolve the federation. The request carries NO tenant signal — no client
	// credentials, no subdomain — so (issuer, aud, subject) is the only thing
	// separating tenants, and an ambiguous match must never be settled by database
	// row order.
	fed, matchErr := s.matchFederation(feds, claims)
	if matchErr != nil {
		span.SetStatus(codes.Error, "federation match rejected")
		return nil, matchErr
	}
	if fed == nil {
		span.SetStatus(codes.Error, "no matching federation")
		return nil, apperror.NewOAuthInvalidGrant("no workload identity federation matches the presented token")
	}
	span.SetAttributes(
		attribute.String("wif.federation_uuid", fed.WorkloadIdentityFederationUUID.String()),
		attribute.Int64("wif.tenant_id", fed.TenantID),
	)

	client, err := s.resolveClientByID(fed.ClientID)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil {
		return nil, apperror.NewOAuthInvalidGrant("the mapped client no longer exists")
	}
	// Create refuses an inactive client, but nothing re-checked it here: deactivating
	// a client did not stop its federation from minting tokens for it. Deactivation
	// has to be an effective kill switch on a keyless credential path.
	if client.Status != shared.StatusActive {
		return nil, apperror.NewOAuthInvalidGrant("the mapped client is not active")
	}
	// Defence in depth against a federation that somehow references another tenant's
	// client: the issued token's tenant comes from the federation, so a mismatch
	// would cross the tenant boundary.
	if client.TenantID != fed.TenantID {
		return nil, apperror.NewOAuthInvalidGrant("the mapped client does not belong to this federation's tenant")
	}

	// The federation row is not the only allow-list that applies. Consulting it
	// alone let a WIF token carry scopes the SAME client is refused at
	// /oauth/token, so the keyless path was the looser door into the client's own
	// grant. The effective allow-list is the intersection of both.
	effectiveAllowed := intersectAllowedScopes([]string(fed.AllowedScopes), client.AllowedScopes)
	grantedScopes, scopeOK := intersectScopes(in.Scope, effectiveAllowed)
	if !scopeOK {
		span.SetStatus(codes.Error, "requested scope not allowed")
		return nil, apperror.NewOAuthInvalidScope(
			"requested scope exceeds the scopes allowed by this federation and its mapped client")
	}

	subject := clientSubject(client)

	audience, audErr := resolveWorkloadAudience(in.Audience, fed.Audience)
	if audErr != nil {
		span.SetStatus(codes.Error, "requested audience not allowed")
		return nil, audErr
	}
	scopeStr := strings.Join(grantedScopes, " ")

	ttlSeconds := defaultWorkloadTokenTTLSeconds
	if client.AccessTokenTTL != nil && *client.AccessTokenTTL > 0 {
		ttlSeconds = *client.AccessTokenTTL
	}

	extraClaims, cerr := workloadTokenClaims(ctx, claims, fed)
	if cerr != nil {
		span.SetStatus(codes.Error, "tenant could not be resolved for the token claim")
		return nil, cerr
	}

	opts := &jwt.AccessTokenOptions{
		AccessTokenTTL: time.Duration(ttlSeconds) * time.Second,
		SubjectType:    "service",
		AMR:            []string{"wif"},
		ExtraClaims:    extraClaims,
	}

	token, gerr := wifGenerateAccessToken(
		ctx,
		subject,
		scopeStr,
		strings.TrimRight(config.AppPublicHostname, "/"),
		audience,
		subject,
		// Realm label for the provider_id claim. Uses the federation's PUBLIC UUID,
		// never the internal tenant PK — only UUIDs may leave the service.
		fmt.Sprintf("wif:%s", fed.WorkloadIdentityFederationUUID),
		opts,
	)
	if gerr != nil {
		span.RecordError(gerr)
		span.SetStatus(codes.Error, "token generation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Best-effort audit: record the exchange in oauth_token_exchanges (3.20).
	//
	// The issued jti is recorded so a token found later can be traced back to the
	// workload that obtained it. Without it the audit row said only "some token was
	// issued to this client", which is not actionable on an endpoint that takes no
	// client credentials.
	if s.auditor != nil {
		issuedJTI := unverifiedClaim(token, "jti")
		if aerr := s.auditor.RecordExchange(ctx, ExchangeAuditEntry{
			TenantID:           fed.TenantID,
			ActorClientID:      fed.ClientID,
			SubjectTokenType:   wifSubjectTokenTypeJWT,
			RequestedTokenType: wifTokenTypeAccessToken,
			ExchangeType:       "delegation",
			Scopes:             grantedScopes,
			IssuedJTI:          ptr.PtrOrNil(issuedJTI),
			IPAddress:          ptr.PtrOrNil(in.IPAddress),
			CreatedAt:          time.Now(),
		}); aerr != nil {
			span.RecordError(aerr)
			// Never block the exchange on an audit failure.
		}
	}

	span.SetStatus(codes.Ok, "")
	return &WorkloadExchangeResult{
		AccessToken:     token,
		IssuedTokenType: wifTokenTypeAccessToken,
		TokenType:       "Bearer",
		ExpiresIn:       ttlSeconds,
		Scope:           scopeStr,
	}, nil
}

// matchFederation resolves the single federation whose audience is present in the
// token's aud claim and whose subject_pattern matches the configured subject claim
// value.
//
// It returns an error when MORE THAN ONE federation matches and they do not all
// belong to the same tenant. Taking the first match meant one tenant could
// register the same issuer and audience with a broad subject_pattern and shadow
// another tenant's exact-match federation: the victim's workload would then be
// issued a token for the shadowing tenant's client. Which one won depended on
// Postgres row order, so it could flip after a routine UPDATE with no config
// change. On an unauthenticated endpoint that has to fail closed.
func (s *workloadIdentityFederationService) matchFederation(
	feds []WorkloadIdentityFederation,
	claims map[string]interface{},
) (*WorkloadIdentityFederation, *apperror.OAuthError) {
	var matches []*WorkloadIdentityFederation
	tenants := map[int64]struct{}{}

	for i := range feds {
		fed := &feds[i]
		if !audienceMatches(claims, fed.Audience) {
			continue
		}
		subjectClaim := fed.SubjectClaim
		if subjectClaim == "" {
			subjectClaim = "sub"
		}
		subjectVal := stringifyClaim(lookupClaim(claims, subjectClaim))
		if subjectVal == "" {
			continue
		}
		// Fail closed on a stored pattern that is too broad to be a boundary, even
		// though the validator refuses to write one — see SubjectPatternTooBroad.
		if SubjectPatternTooBroad(fed.SubjectPattern) {
			slog.Error("refusing a workload identity federation whose subject_pattern is too broad to be a trust boundary; "+
				"edit it to anchor on your organisation or namespace",
				"federation_uuid", fed.WorkloadIdentityFederationUUID.String(),
				"tenant_id", fed.TenantID,
				"subject_pattern", fed.SubjectPattern)
			continue
		}
		if matchSubjectPattern(fed.SubjectPattern, subjectVal) {
			matches = append(matches, fed)
			tenants[fed.TenantID] = struct{}{}
		}
	}

	if len(matches) == 0 {
		return nil, nil
	}
	if len(tenants) > 1 {
		// Cross-tenant ambiguity: refuse rather than pick a tenant for the caller.
		return nil, apperror.NewOAuthInvalidGrant(
			"this token matches workload identity federations in more than one tenant; " +
				"the configurations are ambiguous and must be narrowed")
	}
	// Within one tenant the caller has already proven the same trust relationship,
	// so the deterministic first match (ordered by id in the repository) is safe.
	return matches[0], nil
}

// ---------------------------------------------------------------------------
// Claim / pattern helpers
// ---------------------------------------------------------------------------

// unverifiedIssuer decodes the JWT payload (without signature verification) and
// returns the iss claim. Used only to route the request to a federation; the
// signature is verified afterwards against the issuer's JWKS.
func unverifiedIssuer(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", false
	}
	iss, _ := claims["iss"].(string)
	return iss, iss != ""
}

// unverifiedClaim decodes a JWT payload without verifying the signature and returns
// a string claim. Used only for tokens this server just minted, to recover the jti
// for the audit trail — the generator does not return it.
func unverifiedClaim(token, name string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	value, _ := claims[name].(string)
	return value
}

// audienceMatches reports whether audience is present in the token's aud claim
// (string or array form).
func audienceMatches(claims map[string]interface{}, audience string) bool {
	if audience == "" {
		return false
	}
	switch a := claims["aud"].(type) {
	case string:
		return a == audience
	case []interface{}:
		for _, v := range a {
			if s, ok := v.(string); ok && s == audience {
				return true
			}
		}
	}
	return false
}

// lookupClaim resolves a possibly dotted claim path against nested maps, e.g.
// "sub" or "github.repository". Returns nil when absent.
func lookupClaim(claims map[string]interface{}, path string) interface{} {
	if v, ok := claims[path]; ok {
		return v
	}
	segments := strings.Split(path, ".")
	var current interface{} = claims
	for _, seg := range segments {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[seg]
		if !ok {
			return nil
		}
	}
	return current
}

// stringifyClaim renders a claim value as a string for pattern matching.
func stringifyClaim(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", val)
	}
}

// minAnchoredSubjectSegments is how many whole, wildcard-free segments must
// precede the first wildcard: one for the issuer's generic namespace keyword
// ("repo", "system", "project_path", "spiffe") and at least one more that names
// the organisation, group or namespace the trust is anchored to.
const minAnchoredSubjectSegments = 2

// subjectSegmentSeparators are the delimiters workload issuers structure their
// subjects with: GitHub "repo:org/name:ref:...", Kubernetes
// "system:serviceaccount:ns:sa", GitLab "project_path:group/project:...",
// SPIFFE "spiffe://domain/ns/workload".
const subjectSegmentSeparators = ":/"

// SubjectPatternTooBroad reports whether a pattern is too permissive to be a trust
// boundary: a bare wildcard, a leading wildcard, or a wildcard that is not anchored
// behind an identifying segment.
//
// Counting literal CHARACTERS across the whole pattern did not work, because the
// leading segment is issuer-generic and identifies nobody. On GitHub Actions the
// literal "repo:" is already 5 of the 6 characters the old rule wanted, so
// "repo:a*" passed and matched every repository of every organisation starting
// with "a" — an attacker creates a matching repo and exchanges its token for the
// victim tenant's. The rule is therefore structural: the wildcard must sit behind
// at least one whole literal segment BEYOND the generic first one, so the org /
// group / namespace is pinned. "repo:my-org/*" is fine (org anchored, any repo);
// "repo:a*" is not, and neither is a flat pattern like "my-workload-*" that has no
// segment to anchor on.
//
// Enforced BOTH at write time (validation) and here at match time. Write-time
// validation alone would leave any row that predates the rule — or one inserted by
// direct SQL, a restored backup, or a future import path — matching every workload
// on its issuer forever, because validation never re-runs on read.
func SubjectPatternTooBroad(pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return true
	}
	if !strings.ContainsAny(pattern, "*?") {
		return false // an exact pattern cannot over-match
	}
	if strings.HasPrefix(pattern, "*") || strings.HasPrefix(pattern, "?") {
		return true
	}
	// Empty segments are dropped so "spiffe://domain/ns/*" reads as four segments
	// rather than gaining two free "anchors" from the "//".
	segments := strings.FieldsFunc(pattern, func(r rune) bool {
		return strings.ContainsRune(subjectSegmentSeparators, r)
	})
	anchors := 0
	for _, seg := range segments {
		if strings.ContainsAny(seg, "*?") {
			break // the first wildcard ends the anchored prefix
		}
		anchors++
	}
	return anchors < minAnchoredSubjectSegments
}

// matchSubjectPattern matches a glob pattern (with '*' and '?' wildcards)
// against subject. Unlike path.Match, '*' spans '/' and ':' so patterns like
// "repo:org/repo:*" match "repo:org/repo:ref:refs/heads/main".
func matchSubjectPattern(pattern, subject string) bool {
	if pattern == "" {
		return false
	}
	if pattern == subject {
		return true
	}
	var b strings.Builder
	b.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(subject)
}

// intersectAllowedScopes narrows the federation's allow-list to what the mapped
// client is itself allowed, preserving the federation's ordering so the granted
// scope string is deterministic.
//
// An empty result means "no scopes", not "all scopes" — the whole point is that
// neither list may widen the other, so a client with an empty allow-list grants
// nothing here regardless of what its federation lists.
func intersectAllowedScopes(federationScopes, clientScopes []string) []string {
	clientSet := make(map[string]struct{}, len(clientScopes))
	for _, s := range clientScopes {
		clientSet[s] = struct{}{}
	}
	allowed := make([]string, 0, len(federationScopes))
	for _, s := range federationScopes {
		if _, ok := clientSet[s]; ok {
			allowed = append(allowed, s)
		}
	}
	return allowed
}

// intersectScopes returns the scopes to grant. When requested is empty, all
// allowed scopes are granted. Otherwise every requested scope must be allowed;
// a requested scope outside the allow-list returns ok=false.
func intersectScopes(requested string, allowed []string) ([]string, bool) {
	if strings.TrimSpace(requested) == "" {
		return allowed, true
	}
	allowedSet := map[string]struct{}{}
	for _, s := range allowed {
		allowedSet[s] = struct{}{}
	}
	var granted []string
	for _, s := range strings.Fields(requested) {
		if _, ok := allowedSet[s]; !ok {
			return nil, false
		}
		granted = append(granted, s)
	}
	return granted, true
}

// clientSubject returns the identifier used as the token subject/azp for the
// mapped client, falling back to a synthesized value when the client has no
// identifier configured.
func clientSubject(client *Client) string {
	if client.Identifier != nil && *client.Identifier != "" {
		return *client.Identifier
	}
	return fmt.Sprintf("client:%d", client.ClientID)
}

// workloadTokenClaims assembles the ExtraClaims for an issued workload token: the
// federation's attribute mapping, plus the tenant this server stamps itself.
//
// Nothing used to stamp the tenant. The 7th positional argument of
// jwt.GenerateAccessTokenWithOptionsContext is providerID, not tenant, so passing
// "tenant:N" there only labelled the realm — the token carried NO tenant_id claim
// at all and every consumer (middleware.buildJWTClaims, the gRPC interceptor)
// resolved TenantID = 0. attribute_mapping cannot supply it either: buildExtraClaims
// drops tenant_id as a reserved destination, which is precisely why the SYSTEM sets
// it here, AFTER the operator's mapping, so a mapping cannot forge it.
//
// The claim VALUE is the tenant's opaque UUID, never the internal PK
// (least disclosure, RFC 9068) — that is the form both consumers parse.
func workloadTokenClaims(
	ctx context.Context,
	claims map[string]interface{},
	fed *WorkloadIdentityFederation,
) (map[string]any, *apperror.OAuthError) {
	extra := buildExtraClaims(claims, decodeAttributeMapping(fed.AttributeMapping), fed.SubjectClaim)

	tenantUUID := shared.TenantUUIDStringByID(ctx, fed.TenantID)
	if tenantUUID == "" {
		// Fail closed. This endpoint takes no client credentials, so a token whose
		// tenant reads as 0 downstream is worse than a denial.
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if extra == nil {
		extra = map[string]any{}
	}
	extra["tenant_id"] = tenantUUID
	return extra, nil
}

// buildExtraClaims maps external token claims to internal claim names per the
// federation's attribute_mapping and records the external subject for audit.
//
// SECURITY: the values in attribute_mapping are DESTINATION claim names, and
// jwt.GenerateAccessTokenWithOptions merges ExtraClaims LAST, over the standard
// claim set. A mapping onto a reserved name is therefore a token-forgery
// primitive on an endpoint that requires no client authentication: mapping an
// attacker-chosen external claim onto `sub` and `client_id` yields a token signed
// by the real platform key whose identity is the victim's, and
// UserContextMiddleware resolves the user, tenant and roles from exactly those two
// claims. Mapping onto `svc` forges a service principal over gRPC.
//
// Reserved destinations are dropped rather than rejected because issuance must
// degrade to a correct token rather than deny service; the create/update
// validator refuses them up front so an operator learns at configuration time.
// Note the direction is inverted relative to clients.claim_mappers: here the
// reserved check applies to the map's VALUE, not its key.
func buildExtraClaims(claims map[string]interface{}, mapping map[string]string, subjectClaim string) map[string]any {
	extra := map[string]any{}
	for externalKey, internalKey := range mapping {
		if internalKey == "" || jwt.IsReservedClaim(internalKey) {
			continue
		}
		if v := lookupClaim(claims, externalKey); v != nil {
			extra[internalKey] = v
		}
	}
	if subjectClaim == "" {
		subjectClaim = "sub"
	}
	if ext := stringifyClaim(lookupClaim(claims, subjectClaim)); ext != "" {
		// act.sub records the delegated (workload) principal per RFC 8693 §4.1.
		// Set AFTER the operator's mapping so a mapping onto "act" cannot forge the
		// delegation chain — this is the system stamping its own claim, which is why
		// `act` is reserved above.
		extra["act"] = map[string]any{"sub": ext}
	}
	return extra
}

// resolveWorkloadAudience decides the `aud` of a workload-identity token.
//
// The audience names the resource server the token is addressed to, so honouring
// a caller-supplied one turns this keyless endpoint into a way to mint a token
// aimed at ANY service that trusts this issuer. Only the audience registered on
// the federation may be requested.
//
// A mismatch is refused rather than quietly downgraded to the registered value:
// silently redirecting a token to a different audience than the caller asked for
// is its own hazard, and RFC 8693 §2.2.2 gives invalid_target for precisely this.
func resolveWorkloadAudience(requested, registered string) (string, *apperror.OAuthError) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return registered, nil
	}
	if requested != registered {
		return "", apperror.NewOAuthInvalidTarget(
			"the requested audience is not registered for this federation")
	}
	return requested, nil
}
