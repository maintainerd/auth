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
	if client.IdentityProvider != nil && client.IdentityProvider.Tenant != nil && client.IdentityProvider.Tenant.Identifier != "" {
		return client.IdentityProvider.Tenant.Identifier
	}
	if client.IdentityProvider != nil && client.IdentityProvider.Identifier != "" {
		return client.IdentityProvider.Identifier
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
		// The policy layer requests the tenant_id claim with a 0 placeholder
		// (tokenAuthContextWithPolicy has no client/tenant context). Stamp the
		// authoritative tenant here so authn-issued access tokens never carry
		// tenant_id: 0 — mirroring the OAuth token path (oauth/service_token.go),
		// which already sets client.TenantID.
		if _, ok := accessOpts.ExtraClaims["tenant_id"]; ok {
			accessOpts.ExtraClaims["tenant_id"] = clientTenantID(client)
		}
	}

	accessToken, err = jwt.GenerateAccessTokenWithOptionsContext(
		ctx,
		sub,
		DefaultTokenScope,
		*client.Domain,
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
	}

	idToken, err = jwtGenIDToken(ctx, sub, *client.Domain, *client.Identifier, authnTokenRealm(client), profile, "", params)
	if err != nil {
		return "", "", "", err
	}

	rtOpts := &jwt.RefreshTokenOptions{
		FamilyID:         authCtx.RefreshTokenFamilyID,
		AMR:              authCtx.AMR,
		ACR:              authCtx.ACR,
		SigningAlgorithm: authCtx.SigningAlgorithm,
	}
	if authCtx.RefreshTokenTTLSeconds > 0 {
		rtOpts.RefreshTokenTTL = time.Duration(authCtx.RefreshTokenTTLSeconds) * time.Second
		refreshToken, err = jwtGenRefreshTokenWithOptions(ctx, sub, *client.Domain, *client.Identifier, authnTokenRealm(client), rtOpts)
	} else if authCtx.RefreshTokenFamilyID != "" {
		refreshToken, err = jwtGenRefreshTokenWithOptions(ctx, sub, *client.Domain, *client.Identifier, authnTokenRealm(client), rtOpts)
	} else {
		refreshToken, err = jwtGenRefreshToken(ctx, sub, *client.Domain, *client.Identifier, authnTokenRealm(client))
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
