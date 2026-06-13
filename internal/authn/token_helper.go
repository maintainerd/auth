package authn

import (
	"context"
	"strings"
	"time"

	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
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
		AMR:       authCtx.AMR,
		ACR:       authCtx.ACR,
		SessionID: authCtx.SessionID,
	}
	if authCtx.AccessTokenTTLSeconds > 0 {
		accessOpts.AccessTokenTTL = time.Duration(authCtx.AccessTokenTTLSeconds) * time.Second
	}

	accessToken, err = jwt.GenerateAccessTokenWithOptionsContext(
		ctx,
		sub,
		DefaultTokenScope,
		*client.Domain,
		*client.Identifier,
		*client.Identifier,
		client.IdentityProvider.Identifier,
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
	}

	idToken, err = jwtGenIDToken(ctx, sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, profile, "", params)
	if err != nil {
		return "", "", "", err
	}

	if authCtx.RefreshTokenTTLSeconds > 0 {
		refreshToken, err = jwtGenRefreshTokenWithOptions(ctx, sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, &jwt.RefreshTokenOptions{
			RefreshTokenTTL: time.Duration(authCtx.RefreshTokenTTLSeconds) * time.Second,
			FamilyID:        authCtx.RefreshTokenFamilyID,
		})
	} else if authCtx.RefreshTokenFamilyID != "" {
		refreshToken, err = jwtGenRefreshTokenWithOptions(ctx, sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, &jwt.RefreshTokenOptions{
			FamilyID: authCtx.RefreshTokenFamilyID,
		})
	} else {
		refreshToken, err = jwtGenRefreshToken(ctx, sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier)
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
