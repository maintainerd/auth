package idp

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"golang.org/x/oauth2"
)

// providerProfile centralizes an external provider's UNIQUE broker quirks in one
// place. Adding or adjusting a provider is a single edit to profileFor() rather
// than scattered conditionals across the federation pipeline — the same
// per-provider strategy model Keycloak and Auth0 use for their social/enterprise
// connections.
//
// Scope of this registry: the OAuth2 / userinfo-shaped differences (no-OIDC
// providers, provider-specific scopes, claim normalization, and second-fetch
// augmentation like GitHub's verified-email lookup). Quirks that must live in
// the OIDC token-verification path are deliberately kept as focused functions in
// service_federation.go and referenced here for discoverability:
//   - Azure AD multi-tenant "{tenantid}" issuer → isMicrosoftMultiTenantIssuer /
//     validateMicrosoftMultiTenantIssuer + newOIDCProviderTolerant.
//   - Auth0 trailing-slash issuer / Cognito no-slash → newOIDCProviderTolerant.
//   - LinkedIn omits the id_token nonce → tolerated in exchangeUpstreamCode.
//   - email_verified trust for enterprise/SAML when the claim is omitted →
//     resolveFederatedEmailVerified (keyed by provider_type, not name).
//   - LinkedIn token endpoint requires client_secret_post (and omits the
//     discovery auth-methods hint, so auto-detect fails) → tokenAuthStyle below.
//   - LinkedIn does NOT support PKCE (rejects a code_challenge/verifier sent with
//     the client_secret as invalid_client) → upstreamSupportsPKCE in
//     internal/oauth/service_broker.go skips PKCE for its broker leg.
//   - Facebook returns a confirmed email but no email_verified claim →
//     normalizeFacebookClaims marks a present email verified.
type providerProfile struct {
	// oauth2Only marks a provider with NO OIDC discovery document / id_token
	// (GitHub, Facebook, Twitter). Identity comes from the userinfo endpoint, and
	// these providers must ship explicit authorization/token/userinfo endpoints.
	oauth2Only bool

	// brokerScopes are the scopes sent to the upstream authorize endpoint for an
	// oauth2Only provider. OIDC scopes (openid/profile/email) are meaningless to
	// them, so these provider-specific scopes are used verbatim. Empty for OIDC
	// providers (they use their configured scopes or the openid default).
	brokerScopes []string

	// normalizeClaims adapts a provider's raw userinfo into the OIDC claim shape
	// the pipeline reads (sub, email, picture, …). nil = no adaptation needed.
	normalizeClaims func(claims map[string]any)

	// augmentClaims performs a provider-specific SECOND fetch to complete the
	// profile the first userinfo call could not (e.g. GitHub /user/emails for a
	// verified email). Best-effort; nil = none.
	augmentClaims func(ctx context.Context, oauth2Cfg *oauth2.Config, tok *oauth2.Token, userinfoURL string, claims map[string]any)

	// tokenAuthStyle pins how client credentials are presented at the token
	// endpoint. The zero value (AuthStyleAutoDetect) lets x/oauth2 probe, which is
	// right for providers that accept HTTP Basic. LinkedIn requires
	// client_secret_post AND omits token_endpoint_auth_methods_supported from its
	// discovery doc, so auto-detect returns invalid_client — pin it to in-params.
	tokenAuthStyle oauth2.AuthStyle
}

// profileFor returns the quirks registry entry for a provider. OIDC providers
// (cognito, auth0, google, microsoft, gitlab, linkedin, maintainerd) return the
// zero profile — the shared OIDC pipeline handles them.
func profileFor(provider string) providerProfile {
	switch provider {
	case shared.IDPProviderGitHub:
		return providerProfile{
			oauth2Only: true,
			// read:user → GET /user; user:email → GET /user/emails (verified email).
			brokerScopes:    []string{"read:user", "user:email"},
			normalizeClaims: normalizeGitHubClaims,
			augmentClaims:   augmentGitHubEmails,
		}
	case shared.IDPProviderFacebook:
		return providerProfile{
			oauth2Only:      true,
			brokerScopes:    []string{"email", "public_profile"},
			normalizeClaims: normalizeFacebookClaims,
		}
	case shared.IDPProviderTwitter:
		return providerProfile{
			oauth2Only: true,
			// users.email is REQUIRED for X to return the address (plus the
			// confirmed_email field on /2/users/me and the app's "Request email
			// from users" permission). Without it X returns no email, so a login
			// can't match/link to an existing account by email.
			brokerScopes:    []string{"users.read", "tweet.read", "users.email"},
			normalizeClaims: normalizeTwitterClaims,
		}
	case shared.IDPProviderLinkedIn:
		// LinkedIn is standard OIDC (zero profile otherwise), but its token
		// endpoint only accepts client_secret_post and its discovery omits the
		// auth-methods hint, so pin the token auth style.
		return providerProfile{
			tokenAuthStyle: oauth2.AuthStyleInParams,
		}
	default:
		return providerProfile{}
	}
}

// normalizeNumericSub maps a numeric `id` onto `sub` when `sub` is absent.
// GitHub, Facebook and Twitter identify a user by a numeric id, not an OIDC
// subject; stringClaim formats the number so it is a stable string sub.
func normalizeNumericSub(claims map[string]any) {
	if stringClaim(claims, "sub") == "" {
		if id := stringClaim(claims, "id"); id != "" {
			claims["sub"] = id
		}
	}
}

// normalizeGitHubClaims: numeric id → sub, and avatar_url → picture (GitHub uses
// avatar_url, not the OIDC `picture`).
func normalizeGitHubClaims(claims map[string]any) {
	normalizeNumericSub(claims)
	if stringClaim(claims, "picture") == "" {
		if avatar := stringClaim(claims, "avatar_url"); avatar != "" {
			claims["picture"] = avatar
		}
	}
}

// normalizeFacebookClaims maps Facebook's numeric id → sub and marks a present
// email as verified. Facebook's Graph API returns only the account's CONFIRMED
// primary email and exposes no email_verified field of its own, so a present
// email IS a verified email. Without this, federated Facebook users are pushed
// through maintainerd's own email-verification flow, because social providers
// are not trusted on an omitted email_verified (see resolveFederatedEmailVerified).
// Mirrors the verified-email guarantee GitHub gets from its /user/emails augment.
func normalizeFacebookClaims(claims map[string]any) {
	normalizeNumericSub(claims)
	if stringClaim(claims, "email") != "" {
		if _, present := claims["email_verified"]; !present {
			claims["email_verified"] = true
		}
	}
}

// normalizeTwitterClaims: Twitter v2 /users/me nests the user under "data";
// flatten it, then map the numeric id → sub and profile_image_url → picture.
func normalizeTwitterClaims(claims map[string]any) {
	if data, ok := claims["data"].(map[string]any); ok {
		for k, v := range data {
			claims[k] = v
		}
	}
	normalizeNumericSub(claims)
	if stringClaim(claims, "picture") == "" {
		if img := stringClaim(claims, "profile_image_url"); img != "" {
			claims["picture"] = img
		}
	}
	// X returns the address as confirmed_email (only when users.email + the
	// confirmed_email field are requested). Map it to the standard email claim and
	// mark it verified — X only ever returns a CONFIRMED address.
	if stringClaim(claims, "email") == "" {
		if e := stringClaim(claims, "confirmed_email"); e != "" {
			claims["email"] = e
			if _, ok := claims["email_verified"]; !ok {
				claims["email_verified"] = true
			}
		}
	}
}

// augmentGitHubEmails fills the email (+ its verified flag) from GET
// /user/emails and is a no-op on failure. GitHub's /user omits a private email
// and never reports verification, so without this a GitHub login yields an
// emailless, unverified account. The endpoint is derived from the configured
// userinfo host so GitHub Enterprise (custom API host) works too.
func augmentGitHubEmails(ctx context.Context, oauth2Cfg *oauth2.Config, tok *oauth2.Token, userinfoURL string, claims map[string]any) {
	emailsURL := strings.TrimRight(userinfoURL, "/") + "/emails"
	resp, err := idpOAuth2GetUserinfo(ctx, oauth2Cfg, tok, emailsURL)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamResponseBytes))
	if err != nil {
		return
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if json.Unmarshal(body, &emails) != nil {
		return
	}
	chosen, verified := "", false
	for _, e := range emails { // prefer the primary verified address
		if e.Primary && e.Verified {
			chosen, verified = e.Email, true
			break
		}
	}
	if chosen == "" { // else the first verified address
		for _, e := range emails {
			if e.Verified {
				chosen, verified = e.Email, true
				break
			}
		}
	}
	if chosen != "" {
		claims["email"] = chosen
		claims["email_verified"] = verified
	}
}
