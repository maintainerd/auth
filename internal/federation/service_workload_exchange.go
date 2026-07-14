package federation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	oidclib "github.com/coreos/go-oidc/v3/oidc"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
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

	// Find the first federation whose audience and subject_pattern match.
	fed := s.matchFederation(feds, claims)
	if fed == nil {
		span.SetStatus(codes.Error, "no matching federation")
		return nil, apperror.NewOAuthInvalidGrant("no workload identity federation matches the presented token")
	}

	client, err := s.resolveClientByID(fed.ClientID)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil {
		return nil, apperror.NewOAuthInvalidGrant("the mapped client no longer exists")
	}

	grantedScopes, scopeOK := intersectScopes(in.Scope, []string(fed.AllowedScopes))
	if !scopeOK {
		span.SetStatus(codes.Error, "requested scope not allowed")
		return nil, apperror.NewOAuthInvalidScope("requested scope exceeds the scopes allowed by this federation")
	}

	subject := clientSubject(client)
	audience := in.Audience
	if audience == "" {
		audience = fed.Audience
	}
	scopeStr := strings.Join(grantedScopes, " ")

	ttlSeconds := defaultWorkloadTokenTTLSeconds
	if client.AccessTokenTTL != nil && *client.AccessTokenTTL > 0 {
		ttlSeconds = *client.AccessTokenTTL
	}

	extraClaims := buildExtraClaims(claims, decodeAttributeMapping(fed.AttributeMapping), fed.SubjectClaim)

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
		fmt.Sprintf("tenant:%d", fed.TenantID),
		opts,
	)
	if gerr != nil {
		span.RecordError(gerr)
		span.SetStatus(codes.Error, "token generation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Best-effort audit: record the exchange in oauth_token_exchanges (3.20).
	if s.auditor != nil {
		if aerr := s.auditor.RecordExchange(ctx, ExchangeAuditEntry{
			TenantID:           fed.TenantID,
			ActorClientID:      fed.ClientID,
			SubjectTokenType:   wifSubjectTokenTypeJWT,
			RequestedTokenType: wifTokenTypeAccessToken,
			ExchangeType:       "delegation",
			Scopes:             grantedScopes,
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

// matchFederation returns the first federation whose audience is present in the
// token's aud claim and whose subject_pattern matches the configured subject
// claim value.
func (s *workloadIdentityFederationService) matchFederation(feds []WorkloadIdentityFederation, claims map[string]interface{}) *WorkloadIdentityFederation {
	for i := range feds {
		fed := feds[i]
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
		if matchSubjectPattern(fed.SubjectPattern, subjectVal) {
			return &fed
		}
	}
	return nil
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

// buildExtraClaims maps external token claims to internal claim names per the
// federation's attribute_mapping and records the external subject for audit.
func buildExtraClaims(claims map[string]interface{}, mapping map[string]string, subjectClaim string) map[string]any {
	extra := map[string]any{}
	for externalKey, internalKey := range mapping {
		if internalKey == "" {
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
		extra["act"] = map[string]any{"sub": ext}
	}
	return extra
}
