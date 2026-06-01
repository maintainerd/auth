package oauth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"gorm.io/gorm"
)

func authenticateOAuthClient(db *gorm.DB, creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
	if creds.ClientID == "" {
		return nil, apperror.NewOAuthInvalidClient("client_id is required")
	}
	client, err := findActiveClientByIdentifier(db, creds.ClientID)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil {
		return nil, apperror.NewOAuthInvalidClient("client authentication failed")
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodNone {
		return client, nil
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodSecretBasic || client.TokenEndpointAuthMethod == TokenAuthMethodSecretPost {
		if !clientSecretMatches(client, creds.ClientSecret) {
			return nil, apperror.NewOAuthInvalidClient("client authentication failed")
		}
		return client, nil
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodPrivateKeyJWT {
		return authenticatePrivateKeyJWT(client, creds)
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodClientSecretJWT {
		return authenticateClientSecretJWT(client, creds)
	}
	return nil, apperror.NewOAuthInvalidClient("unsupported token_endpoint_auth_method")
}

func clientHasGrant(client *Client, grantType string) bool {
	if client.GrantTypes == nil {
		return false
	}
	for _, g := range client.GrantTypes {
		if g == grantType {
			return true
		}
	}
	return false
}

const (
	assertionTypeJWTBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	assertionMaxAge        = 5 * time.Minute
)

func authenticatePrivateKeyJWT(client *Client, creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
	if creds.ClientAssertionType != assertionTypeJWTBearer {
		return nil, apperror.NewOAuthInvalidClient("client_assertion_type must be " + assertionTypeJWTBearer)
	}
	if creds.ClientAssertion == "" {
		return nil, apperror.NewOAuthInvalidClient("client_assertion is required")
	}

	if client.JWKS == nil && client.JWKSUri == nil {
		return nil, apperror.NewOAuthInvalidClient("client has no JWKS or jwks_uri configured")
	}

	token, err := jwtlib.Parse(creds.ClientAssertion, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		key, err := findClientJWK(client, kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	}, jwtlib.WithLeeway(assertionMaxAge), jwtlib.WithValidMethods([]string{"RS256", "RS384", "RS512"}))

	if err != nil || !token.Valid {
		return nil, apperror.NewOAuthInvalidClient("client assertion is invalid")
	}

	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		return nil, apperror.NewOAuthInvalidClient("invalid assertion claims")
	}

	if err := validateAssertionClaims(claims, client); err != nil {
		return nil, apperror.NewOAuthInvalidClient(err.Error())
	}

	return client, nil
}

func authenticateClientSecretJWT(client *Client, creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
	if creds.ClientAssertionType != assertionTypeJWTBearer {
		return nil, apperror.NewOAuthInvalidClient("client_assertion_type must be " + assertionTypeJWTBearer)
	}
	if creds.ClientAssertion == "" {
		return nil, apperror.NewOAuthInvalidClient("client_assertion is required")
	}

	token, err := jwtlib.Parse(creds.ClientAssertion, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		secret := ptrOrEmpty(client.SecretHash)
		if secret == "" {
			return nil, fmt.Errorf("client has no registered secret")
		}
		return []byte(secret), nil
	}, jwtlib.WithLeeway(assertionMaxAge), jwtlib.WithValidMethods([]string{"HS256", "HS384", "HS512"}))

	if err != nil || !token.Valid {
		return nil, apperror.NewOAuthInvalidClient("client assertion is invalid")
	}

	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		return nil, apperror.NewOAuthInvalidClient("invalid assertion claims")
	}

	if err := validateAssertionClaims(claims, client); err != nil {
		return nil, apperror.NewOAuthInvalidClient(err.Error())
	}

	return client, nil
}

func validateAssertionClaims(claims jwtlib.MapClaims, client *Client) error {
	iss, _ := claims["iss"].(string)
	sub, _ := claims["sub"].(string)
	aud, _ := claims["aud"].(string)
	exp, _ := claims["exp"].(float64)

	if iss == "" || sub == "" || aud == "" || exp == 0 {
		return fmt.Errorf("assertion missing required claims")
	}

	if iss != ptrOrEmpty(client.Identifier) {
		return fmt.Errorf("assertion issuer does not match client_id")
	}
	if sub != ptrOrEmpty(client.Identifier) {
		return fmt.Errorf("assertion subject does not match client_id")
	}

	if client.Domain == nil || !strings.HasPrefix(aud, *client.Domain) {
		return fmt.Errorf("assertion audience invalid")
	}

	now := time.Now()
	expTime := time.Unix(int64(exp), 0)
	if now.After(expTime) {
		return fmt.Errorf("assertion expired")
	}

	return nil
}

func findClientJWK(client *Client, kid string) (interface{}, error) {
	if client.JWKS != nil {
		var jwks struct {
			Keys []json.RawMessage `json:"keys"`
		}
		if err := json.Unmarshal(client.JWKS, &jwks); err != nil {
			return nil, fmt.Errorf("invalid JWKS format")
		}
		for _, rawKey := range jwks.Keys {
			var key struct {
				KID string `json:"kid"`
				KTY string `json:"kty"`
				N   string `json:"n"`
				E   string `json:"e"`
				Alg string `json:"alg"`
			}
			if err := json.Unmarshal(rawKey, &key); err != nil {
				continue
			}
			if kid != "" && key.KID != kid {
				continue
			}
			if key.KTY == "RSA" {
				n := new(big.Int)
				nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
				if err != nil {
					continue
				}
				n.SetBytes(nBytes)

				eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
				if err != nil {
					continue
				}
				e := new(big.Int).SetBytes(eBytes)

				pubKey := &rsa.PublicKey{
					N: n,
					E: int(e.Int64()),
				}
				return pubKey, nil
			}
		}
		return nil, fmt.Errorf("no usable RSA key found in client JWKS")
	}
	return nil, fmt.Errorf("no JWKS configured for client")
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
