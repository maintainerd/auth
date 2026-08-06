package authn

import (
	"context"
	"strings"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

const (
	DefaultAccessTokenExpiresIn = shared.DefaultAccessTokenExpiresIn
	DefaultTokenScope           = shared.DefaultTokenScope
)

func hashUserBearerToken(token string) string {
	return crypto.HashAuthorizationCode(strings.TrimSpace(token))
}

type tokenAuthContext struct {
	AMR                    []string
	ACR                    string
	SessionID              string
	AccessTokenTTLSeconds  int
	RefreshTokenTTLSeconds int
	RefreshTokenFamilyID   string
	SigningAlgorithm       string
	ExtraAccessClaims      map[string]any
	CookieSecure           bool
	CookieHTTPOnly         bool
	CookieSameSite         string
	CookieRefreshMaxAge    int
	HasCookiePolicy        bool
}

func passwordAuthContext() tokenAuthContext {
	return tokenAuthContext{
		AMR: []string{jwt.AMRPassword},
		ACR: jwt.ACRLevel1,
	}
}

func generateTokenSet(sub string, user *User, client *Client) (accessToken, idToken, refreshToken string, err error) {
	return generateTokenSetWithContext(context.Background(), sub, user, client)
}

func generateTokenSetWithContext(ctx context.Context, sub string, user *User, client *Client) (accessToken, idToken, refreshToken string, err error) {
	return generateTokenSetWithAuthContext(ctx, sub, user, client, passwordAuthContext())
}

var jwtGenIDToken = jwt.GenerateIDTokenWithContext
var jwtGenRefreshToken = jwt.GenerateRefreshTokenWithContext
var jwtGenRefreshTokenWithOptions = jwt.GenerateRefreshTokenWithOptionsContext

// authnTokenRealm resolves the token realm (provider_id claim) for login/register
// issued tokens. It is anchored to the tenant identifier so that tokens minted by
// password login carry the SAME realm as tokens minted by the OAuth code flow and
// federation (see oauth.tokenRealm / idp.federationTokenRealm). A client can now
// connect many identity providers, so the realm must NOT depend on any single IdP.
func authnTokenRealm(client *Client) string {
	if client == nil {
		return ""
	}
	// The realm is the TENANT, never an identity provider — matching
	// oauth.tokenRealm and the federation path. It previously preferred
	// client.IdentityProvider (a gorm:"-" phantom, populated on some read paths
	// and not others), so the same client produced a different provider_id
	// depending on how it had been loaded: the IdP identifier at login, the
	// tenant name after a refresh. Because the refresh path resolves the client
	// by joining on identity_providers.identifier, that drift made the SECOND
	// refresh fail with "client not found" — refresh worked exactly once.
	if client.Tenant != nil && client.Tenant.Name != "" {
		return client.Tenant.Name
	}
	if client.IdentityProvider != nil && client.IdentityProvider.Tenant != nil && client.IdentityProvider.Tenant.Name != "" {
		return client.IdentityProvider.Tenant.Name
	}
	if client.Identifier != nil {
		return *client.Identifier
	}
	return ""
}

func generateTokenSetWithAuthContext(ctx context.Context, sub string, user *User, client *Client, authCtx tokenAuthContext) (accessToken, idToken, refreshToken string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(authCtx.AMR) == 0 {
		authCtx.AMR = []string{jwt.AMRPassword}
	}
	if authCtx.ACR == "" {
		authCtx.ACR = jwt.ACRLevel1
	}

	accessOpts := &jwt.AccessTokenOptions{
		AMR:              authCtx.AMR,
		ACR:              authCtx.ACR,
		SessionID:        authCtx.SessionID,
		SigningAlgorithm: authCtx.SigningAlgorithm,
	}
	if authCtx.AccessTokenTTLSeconds > 0 {
		accessOpts.AccessTokenTTL = time.Duration(authCtx.AccessTokenTTLSeconds) * time.Second
	}
	if len(authCtx.ExtraAccessClaims) > 0 {
		accessOpts.ExtraClaims = authCtx.ExtraAccessClaims
		// The policy layer requests the tenant_id claim with a placeholder
		// (tokenAuthContextWithPolicy has no client/tenant context). Stamp the
		// tenant's opaque UUID here — NEVER the internal PK (least-disclosure per
		// RFC 9068, consistent with the dual-key design). The JWT parse layer
		// resolves it back to the internal id. Mirrors the OAuth token path.
		if _, ok := accessOpts.ExtraClaims["tenant_id"]; ok {
			if s := shared.TenantUUIDStringByID(ctx, clientTenantID(client)); s != "" {
				accessOpts.ExtraClaims["tenant_id"] = s
			} else {
				delete(accessOpts.ExtraClaims, "tenant_id")
			}
		}
	}

	accessToken, err = jwt.GenerateAccessTokenWithOptionsContext(
		ctx,
		sub,
		DefaultTokenScope,
		jwt.TokenIssuerPtr(client.Domain),
		*client.Identifier,
		*client.Identifier,
		authnTokenRealm(client),
		accessOpts,
	)
	if err != nil {
		return "", "", "", err
	}

	profile := buildAuthNUserProfile(user)

	params := &jwt.IDTokenParams{
		RequestedScopes: strings.Fields(DefaultTokenScope),
		AMR:             authCtx.AMR,
		ACR:             authCtx.ACR,
		// Honor the tenant's signing_algorithm here too. The access and refresh
		// tokens above already pass it; without this line the first-party ID
		// token was always RS256, so a tenant on PS256 got an inconsistently
		// signed token set.
		SigningAlgorithm: authCtx.SigningAlgorithm,
		// Same session the access and refresh tokens are bound to, so a
		// back-channel logout token's sid resolves to a session the RP holds.
		SessionID: authCtx.SessionID,
	}

	// The ID token is the one a relying party actually validates `iss` on (OIDC
	// Core §3.1.3.7 step 2), so it must carry the AUTHORIZATION SERVER's issuer
	// exactly as discovery advertises it — same as the access token above. This
	// passed the raw client domain while the access token had already moved to
	// jwt.TokenIssuer, so a single login handed out a token set whose access and
	// ID tokens disagreed about who issued them, and every compliant RP rejected
	// the ID token.
	idToken, err = jwtGenIDToken(ctx, sub, jwt.TokenIssuerPtr(client.Domain), *client.Identifier, authnTokenRealm(client), profile, "", params)
	if err != nil {
		return "", "", "", err
	}

	rtOpts := &jwt.RefreshTokenOptions{
		FamilyID: authCtx.RefreshTokenFamilyID,
		AMR:      authCtx.AMR,
		ACR:      authCtx.ACR,
		// Bind the refresh token to this session, so revoking the session
		// revokes the refresh token with it.
		SessionID:        authCtx.SessionID,
		SigningAlgorithm: authCtx.SigningAlgorithm,
	}
	// Same issuer rule as the access and ID tokens: the refresh token is re-parsed
	// by this server on every rotation, and validateIssuerClaim matches `iss`
	// against the allowlist. Minting it under the client domain while the rest of
	// the set carries the authorization server's issuer left one token in the set
	// depending on the legacy client-domain entries staying in that allowlist.
	rtIssuer := jwt.TokenIssuerPtr(client.Domain)
	if authCtx.RefreshTokenTTLSeconds > 0 {
		rtOpts.RefreshTokenTTL = time.Duration(authCtx.RefreshTokenTTLSeconds) * time.Second
		refreshToken, err = jwtGenRefreshTokenWithOptions(ctx, sub, rtIssuer, *client.Identifier, authnTokenRealm(client), rtOpts)
	} else if authCtx.RefreshTokenFamilyID != "" {
		refreshToken, err = jwtGenRefreshTokenWithOptions(ctx, sub, rtIssuer, *client.Identifier, authnTokenRealm(client), rtOpts)
	} else {
		refreshToken, err = jwtGenRefreshToken(ctx, sub, rtIssuer, *client.Identifier, authnTokenRealm(client))
	}
	if err != nil {
		return "", "", "", err
	}

	return accessToken, idToken, refreshToken, nil
}

func buildAuthNUserProfile(user *User) *jwt.UserProfile {
	return &jwt.UserProfile{
		Name:          user.Fullname,
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}
}

func buildLoginTokenResponse(accessToken, idToken, refreshToken string, issuedAt int64) *LoginResponseDTO {
	return &LoginResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    DefaultAccessTokenExpiresIn,
		TokenType:    "Bearer",
		IssuedAt:     issuedAt,
	}
}

func applyLoginCookiePolicy(resp *LoginResponseDTO, policy secpolicy.EffectiveSessionPolicy) {
	if resp == nil {
		return
	}
	resp.CookieSecure = &policy.CookieSecure
	resp.CookieHTTPOnly = &policy.CookieHTTPOnly
	resp.CookieSameSite = policy.CookieSameSite
	if policy.RefreshTokenTTLSeconds > 0 {
		resp.RefreshTokenMaxAge = policy.RefreshTokenTTLSeconds
	}
	if policy.IdleTimeoutSeconds > 0 {
		resp.AccessTokenCookieMaxAge = int64(policy.IdleTimeoutSeconds)
	}
}

func applyRegisterCookiePolicy(resp *RegisterResponseDTO, policy secpolicy.EffectiveSessionPolicy) {
	if resp == nil {
		return
	}
	resp.CookieSecure = &policy.CookieSecure
	resp.CookieHTTPOnly = &policy.CookieHTTPOnly
	resp.CookieSameSite = policy.CookieSameSite
	if policy.RefreshTokenTTLSeconds > 0 {
		resp.RefreshTokenMaxAge = policy.RefreshTokenTTLSeconds
	}
	if policy.IdleTimeoutSeconds > 0 {
		resp.AccessTokenCookieMaxAge = int64(policy.IdleTimeoutSeconds)
	}
}

func buildRegisterTokenResponse(accessToken, idToken, refreshToken string, issuedAt int64) *RegisterResponseDTO {
	return &RegisterResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    DefaultAccessTokenExpiresIn,
		TokenType:    "Bearer",
		IssuedAt:     issuedAt,
	}
}
