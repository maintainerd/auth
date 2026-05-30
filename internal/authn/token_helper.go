package authn

import (
	"github.com/maintainerd/auth/internal/platform/jwt"
)

func generateTokenSet(sub string, user *User, client *Client) (accessToken, idToken, refreshToken string, err error) {
	accessToken, err = jwt.GenerateAccessToken(
		sub,
		"openid profile email",
		*client.Domain,
		*client.Identifier,
		*client.Identifier,
		client.IdentityProvider.Identifier,
	)
	if err != nil {
		return "", "", "", err
	}

	profile := &jwt.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}

	idToken, err = generateIDTokenFn(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, profile, "", nil)
	if err != nil {
		return "", "", "", err
	}

	refreshToken, err = generateRefreshTokenFn(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier)
	if err != nil {
		return "", "", "", err
	}

	return accessToken, idToken, refreshToken, nil
}
