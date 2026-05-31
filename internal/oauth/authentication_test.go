package oauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func buildRSAPublicKeyJWK(t *testing.T, privKey *rsa.PrivateKey, kid string) datatypes.JSON {
	t.Helper()
	pub := &privKey.PublicKey
	jwk := map[string]string{
		"kty": "RSA",
		"kid": kid,
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigEndianBytes(pub.E)),
	}
	jwks := map[string]interface{}{
		"keys": []interface{}{jwk},
	}
	b, err := json.Marshal(jwks)
	require.NoError(t, err)
	return datatypes.JSON(b)
}

func bigEndianBytes(e int) []byte {
	b := make([]byte, 4)
	b[0] = byte(e >> 24)
	b[1] = byte(e >> 16)
	b[2] = byte(e >> 8)
	b[3] = byte(e)
	for b[0] == 0 {
		b = b[1:]
	}
	return b
}

func signJWTWithRSA(t *testing.T, claims jwtlib.MapClaims, privKey *rsa.PrivateKey, kid string) string {
	t.Helper()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(privKey)
	require.NoError(t, err)
	return signed
}

func TestAuthenticatePrivateKeyJWT(t *testing.T) {
	privKey := genRSAKey(t)
	clientID := "test-app"
	domain := "test-app.example.com"
	kid := "key-1"

	client := &Client{
		ClientID:  1,
		Identifier: &clientID,
		Domain:     &domain,
		JWKS:       buildRSAPublicKeyJWK(t, privKey, kid),
	}

	t.Run("invalid assertion_type", func(t *testing.T) {
		creds := OAuthClientCredentials{ClientAssertionType: "wrong"}
		result, oerr := authenticatePrivateKeyJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("empty assertion", func(t *testing.T) {
		creds := OAuthClientCredentials{ClientAssertionType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"}
		result, oerr := authenticatePrivateKeyJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
	})

	t.Run("client has no JWKS", func(t *testing.T) {
		c := &Client{ClientID: 1, Identifier: &clientID, Domain: &domain}
		creds := OAuthClientCredentials{
			ClientAssertionType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
			ClientAssertion:     "some-jwt",
		}
		result, oerr := authenticatePrivateKeyJWT(c, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("invalid JWT signature", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		wrongKey := genRSAKey(t)
		assertion := signJWTWithRSA(t, claims, wrongKey, kid)

		creds := OAuthClientCredentials{
			ClientAssertionType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
			ClientAssertion:     assertion,
		}

		result, oerr := authenticatePrivateKeyJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})
}

func genRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func TestAuthenticateClientSecretJWT(t *testing.T) {
	clientID := "test-app"
	domain := "test-app.example.com"
	secret := "my-client-secret"

	client := &Client{
		ClientID:   1,
		Identifier: &clientID,
		Domain:     &domain,
		SecretHash: &secret,
	}

	t.Run("invalid assertion_type", func(t *testing.T) {
		creds := OAuthClientCredentials{ClientAssertionType: "wrong"}
		result, oerr := authenticateClientSecretJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
	})

	t.Run("empty assertion", func(t *testing.T) {
		creds := OAuthClientCredentials{ClientAssertionType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"}
		result, oerr := authenticateClientSecretJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
	})

	t.Run("client has no secret", func(t *testing.T) {
		c := &Client{ClientID: 1, Identifier: &clientID, Domain: &domain}
		creds := OAuthClientCredentials{
			ClientAssertionType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
			ClientAssertion:     "some-jwt",
		}
		result, oerr := authenticateClientSecretJWT(c, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
	})

	t.Run("assertion issuer mismatch", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": "wrong-client", "sub": "wrong-client", "aud": domain,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
		assertion, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		creds := OAuthClientCredentials{
			ClientAssertionType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
			ClientAssertion:     assertion,
		}
		result, oerr := authenticateClientSecretJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})
}

func TestValidateAssertionClaims(t *testing.T) {
	clientID := "test-app"
	domain := "test-app.example.com"
	client := &Client{ClientID: 1, Identifier: &clientID, Domain: &domain}

	t.Run("valid claims", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}
		err := validateAssertionClaims(claims, client)
		require.NoError(t, err)
	})

	t.Run("missing claims", func(t *testing.T) {
		err := validateAssertionClaims(jwtlib.MapClaims{}, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required claims")
	})

	t.Run("issuer mismatch", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": "other", "sub": clientID, "aud": domain,
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issuer")
	})

	t.Run("subject mismatch", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": "other", "aud": domain,
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subject")
	})

	t.Run("audience invalid", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": "wrong-domain.example.com",
			"exp": float64(time.Now().Add(time.Hour).Unix()),
		}
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audience")
	})

	t.Run("assertion expired", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
			"exp": float64(time.Now().Add(-time.Hour).Unix()),
		}
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})
}
