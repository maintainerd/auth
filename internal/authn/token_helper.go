package authn

import (
	"strings"

	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/shared"
)

const (
	DefaultAccessTokenExpiresIn = shared.DefaultAccessTokenExpiresIn
	DefaultTokenScope           = shared.DefaultTokenScope
)

func hashUserBearerToken(token string) string {
	return crypto.HashAuthorizationCode(strings.TrimSpace(token))
}

func generateTokenSet(sub string, user *User, client *Client) (accessToken, idToken, refreshToken string, err error) {
	accessToken, err = jwt.GenerateAccessToken(
		sub,
		DefaultTokenScope,
		*client.Domain,
		*client.Identifier,
		*client.Identifier,
		client.IdentityProvider.Identifier,
	)
	if err != nil {
		return "", "", "", err
	}

	profile := buildAuthNUserProfile(user)

	params := &jwt.IDTokenParams{
		RequestedScopes: strings.Fields(DefaultTokenScope),
		AMR:             []string{jwt.AMRPassword},
		ACR:             jwt.ACRLevel1,
	}

	idToken, err = generateIDTokenFn(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, profile, "", params)
	if err != nil {
		return "", "", "", err
	}

	refreshToken, err = generateRefreshTokenFn(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier)
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
