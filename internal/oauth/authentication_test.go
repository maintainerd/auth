package oauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/lib/pq"
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
		ClientID:   1,
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

	t.Run("valid assertion", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		assertion := signJWTWithRSA(t, claims, privKey, kid)

		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}

		result, oerr := authenticatePrivateKeyJWT(client, creds)
		require.Nil(t, oerr)
		assert.Equal(t, client, result)
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
	secretHash := "$2a$10$not-the-hmac-secret"
	secretEncrypted := "encrypted-current-secret"

	origDecrypt := crypto.DecryptAtRest
	t.Cleanup(func() { crypto.DecryptAtRest = origDecrypt })
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		switch ciphertext {
		case secretEncrypted:
			return secret, nil
		case "encrypted-previous-secret":
			return "previous-client-secret", nil
		default:
			return "", assert.AnError
		}
	}

	client := &Client{
		ClientID:        1,
		Identifier:      &clientID,
		Domain:          &domain,
		SecretHash:      &secretHash,
		SecretEncrypted: &secretEncrypted,
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

	t.Run("valid assertion uses encrypted secret not bcrypt hash", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
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
		require.Nil(t, oerr)
		assert.Equal(t, client, result)
	})

	t.Run("assertion signed with secret hash is rejected", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
		assertion, err := token.SignedString([]byte(secretHash))
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

	t.Run("valid assertion can use previous encrypted secret during grace period", func(t *testing.T) {
		previousEncrypted := "encrypted-previous-secret"
		expires := time.Now().Add(time.Hour)
		c := *client
		c.PreviousSecretEncrypted = &previousEncrypted
		c.PreviousSecretExpiresAt = &expires

		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
		assertion, err := token.SignedString([]byte("previous-client-secret"))
		require.NoError(t, err)

		creds := OAuthClientCredentials{
			ClientAssertionType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
			ClientAssertion:     assertion,
		}
		result, oerr := authenticateClientSecretJWT(&c, creds)
		require.Nil(t, oerr)
		assert.Equal(t, &c, result)
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

func TestFindClientJWK(t *testing.T) {
	privKey := genRSAKey(t)
	jwks := buildRSAPublicKeyJWK(t, privKey, "key-1")

	t.Run("finds matching RSA key", func(t *testing.T) {
		key, err := findClientJWK(&Client{JWKS: jwks}, "key-1")
		require.NoError(t, err)
		assert.IsType(t, &rsa.PublicKey{}, key)
	})

	t.Run("uses first RSA key when kid is empty", func(t *testing.T) {
		key, err := findClientJWK(&Client{JWKS: jwks}, "")
		require.NoError(t, err)
		assert.IsType(t, &rsa.PublicKey{}, key)
	})

	t.Run("invalid JWKS JSON", func(t *testing.T) {
		key, err := findClientJWK(&Client{JWKS: []byte("{")}, "key-1")
		assert.Nil(t, key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JWKS")
	})

	t.Run("skips malformed and non matching keys", func(t *testing.T) {
		raw, err := json.Marshal(map[string]any{
			"keys": []any{
				"{",
				map[string]string{"kid": "other", "kty": "RSA", "n": "!", "e": "!"},
				map[string]string{"kid": "key-1", "kty": "EC"},
				map[string]string{"kid": "key-1", "kty": "RSA", "n": "!", "e": "AQAB"},
				map[string]string{"kid": "key-1", "kty": "RSA", "n": base64.RawURLEncoding.EncodeToString([]byte{1}), "e": "!"},
			},
		})
		require.NoError(t, err)

		key, err := findClientJWK(&Client{JWKS: raw}, "key-1")
		assert.Nil(t, key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no usable RSA")
	})

	t.Run("no JWKS configured", func(t *testing.T) {
		key, err := findClientJWK(&Client{}, "key-1")
		assert.Nil(t, key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no JWKS")
	})
}

func TestPtrOrEmpty(t *testing.T) {
	assert.Equal(t, "", ptrOrEmpty(nil))
	value := "client"
	assert.Equal(t, "client", ptrOrEmpty(&value))
}

func TestAuthenticateOAuthClient_EmptyClientID(t *testing.T) {
	result, oerr := authenticateOAuthClient(nil, OAuthClientCredentials{})
	assert.Nil(t, result)
	require.NotNil(t, oerr)
	assert.Equal(t, "invalid_client", oerr.Code)
	assert.Contains(t, oerr.Description, "required")
}

func TestClientSecretJWTSecrets_DecryptError(t *testing.T) {
	origDecrypt := crypto.DecryptAtRest
	t.Cleanup(func() { crypto.DecryptAtRest = origDecrypt })
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		return "", assert.AnError
	}

	secretEncrypted := "encrypted-secret"
	client := &Client{SecretEncrypted: &secretEncrypted}

	secrets, err := clientSecretJWTSecrets(client)
	assert.Nil(t, secrets)
	require.Error(t, err)
}

func TestClientSecretJWTSecrets_PreviousSecretDecryptError(t *testing.T) {
	origDecrypt := crypto.DecryptAtRest
	t.Cleanup(func() { crypto.DecryptAtRest = origDecrypt })
	expireTime := time.Now().Add(time.Hour)
	callCount := 0
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		callCount++
		if callCount == 1 {
			return "valid-secret", nil
		}
		return "", assert.AnError
	}

	currentEncrypted := "encrypted-current"
	previousEncrypted := "encrypted-previous"
	client := &Client{
		SecretEncrypted:         &currentEncrypted,
		PreviousSecretEncrypted: &previousEncrypted,
		PreviousSecretExpiresAt: &expireTime,
	}

	_, err := clientSecretJWTSecrets(client)
	require.Error(t, err)
}

func TestClientSecretJWTSecrets_PreviousSecretEmpty(t *testing.T) {
	origDecrypt := crypto.DecryptAtRest
	t.Cleanup(func() { crypto.DecryptAtRest = origDecrypt })
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		return ciphertext, nil
	}

	currentEncrypted := "encrypted-current"
	previousEncrypted := ""
	expireTime := time.Now().Add(time.Hour)
	client := &Client{
		SecretEncrypted:         &currentEncrypted,
		PreviousSecretEncrypted: &previousEncrypted,
		PreviousSecretExpiresAt: &expireTime,
	}

	secrets, err := clientSecretJWTSecrets(client)
	require.NoError(t, err)
	assert.Len(t, secrets, 1)
}

func TestClientSecretJWTSecrets_PreviousSecretExpired(t *testing.T) {
	origDecrypt := crypto.DecryptAtRest
	t.Cleanup(func() { crypto.DecryptAtRest = origDecrypt })
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		return "decrypted-" + ciphertext, nil
	}

	currentEncrypted := "encrypted-current"
	previousEncrypted := "encrypted-previous"
	pastTime := time.Now().Add(-time.Hour)
	client := &Client{
		SecretEncrypted:         &currentEncrypted,
		PreviousSecretEncrypted: &previousEncrypted,
		PreviousSecretExpiresAt: &pastTime,
	}

	secrets, err := clientSecretJWTSecrets(client)
	require.NoError(t, err)
	assert.Len(t, secrets, 1)
}

func TestAuthenticateClientSecretJWT_DecryptError(t *testing.T) {
	clientID := "test-app"
	domain := "test-app.example.com"

	origDecrypt := crypto.DecryptAtRest
	t.Cleanup(func() { crypto.DecryptAtRest = origDecrypt })
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		return "", assert.AnError
	}

	secretEncrypted := "encrypted-secret"
	client := &Client{
		ClientID:        1,
		Identifier:      &clientID,
		Domain:          &domain,
		SecretEncrypted: &secretEncrypted,
	}

	claims := jwtlib.MapClaims{
		"iss": clientID, "sub": clientID, "aud": domain,
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	assertion, err := token.SignedString([]byte("doesnt-matter"))
	require.NoError(t, err)

	creds := OAuthClientCredentials{
		ClientAssertionType: assertionTypeJWTBearer,
		ClientAssertion:     assertion,
	}

	result, oerr := authenticateClientSecretJWT(client, creds)
	assert.Nil(t, result)
	require.NotNil(t, oerr)
	assert.Equal(t, "invalid_client", oerr.Code)
}

func TestValidateAssertionClaims_DomainNil(t *testing.T) {
	clientID := "test-app"
	client := &Client{ClientID: 1, Identifier: &clientID, Domain: nil}

	claims := jwtlib.MapClaims{
		"iss": clientID, "sub": clientID, "aud": "any-audience.example.com",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}
	err := validateAssertionClaims(claims, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")
}

func TestAuthenticatePrivateKeyJWT_BadClaims(t *testing.T) {
	privKey := genRSAKey(t)
	clientID := "test-app"
	domain := "test-app.example.com"
	kid := "key-1"

	client := &Client{
		ClientID:   1,
		Identifier: &clientID,
		Domain:     &domain,
		JWKS:       buildRSAPublicKeyJWK(t, privKey, kid),
	}

	t.Run("expired assertion", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
			"exp": time.Now().Add(-time.Hour).Unix(), "iat": time.Now().Add(-2 * time.Hour).Unix(),
		}
		assertion := signJWTWithRSA(t, claims, privKey, kid)
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}
		result, oerr := authenticatePrivateKeyJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("issuer mismatch in assertion", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": "wrong-client", "sub": clientID, "aud": domain,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		assertion := signJWTWithRSA(t, claims, privKey, kid)
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}
		result, oerr := authenticatePrivateKeyJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("missing claims in assertion", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID,
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		assertion := signJWTWithRSA(t, claims, privKey, kid)
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}
		result, oerr := authenticatePrivateKeyJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("audience invalid", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": "wrong-domain.example.com",
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		assertion := signJWTWithRSA(t, claims, privKey, kid)
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}
		result, oerr := authenticatePrivateKeyJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("JWKSUri only (no JWKS)", func(t *testing.T) {
		jwksURI := "https://example.com/jwks"
		c := &Client{
			ClientID:   1,
			Identifier: &clientID,
			Domain:     &domain,
			JWKSUri:    &jwksURI,
		}
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     "some-jwt",
		}
		result, oerr := authenticatePrivateKeyJWT(c, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})
}

func TestAuthenticateClientSecretJWT_BadClaims(t *testing.T) {
	clientID := "test-app"
	domain := "test-app.example.com"
	secret := "my-client-secret"
	secretEncrypted := "encrypted-current-secret"

	origDecrypt := crypto.DecryptAtRest
	t.Cleanup(func() { crypto.DecryptAtRest = origDecrypt })
	crypto.DecryptAtRest = func(ciphertext string) (string, error) {
		return secret, nil
	}

	client := &Client{
		ClientID:        1,
		Identifier:      &clientID,
		Domain:          &domain,
		SecretEncrypted: &secretEncrypted,
	}

	t.Run("expired assertion", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": domain,
			"exp": time.Now().Add(-time.Hour).Unix(), "iat": time.Now().Add(-2 * time.Hour).Unix(),
		}
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
		assertion, err := token.SignedString([]byte(secret))
		require.NoError(t, err)
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}
		result, oerr := authenticateClientSecretJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("subject mismatch in assertion", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": "wrong-sub", "aud": domain,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
		assertion, err := token.SignedString([]byte(secret))
		require.NoError(t, err)
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}
		result, oerr := authenticateClientSecretJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("missing claims in assertion", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID,
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
		assertion, err := token.SignedString([]byte(secret))
		require.NoError(t, err)
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}
		result, oerr := authenticateClientSecretJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})

	t.Run("audience invalid in assertion", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"iss": clientID, "sub": clientID, "aud": "wrong-domain.example.com",
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
		assertion, err := token.SignedString([]byte(secret))
		require.NoError(t, err)
		creds := OAuthClientCredentials{
			ClientAssertionType: assertionTypeJWTBearer,
			ClientAssertion:     assertion,
		}
		result, oerr := authenticateClientSecretJWT(client, creds)
		assert.Nil(t, result)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_client", oerr.Code)
	})
}

func TestAuthenticateOAuthClient_SecretMismatch(t *testing.T) {
	secretHash, err := security.HashClientSecret(t.Context(), "correct-secret")
	require.NoError(t, err)
	clientID := "test-app"

	db, mock := newMockDB(t)
	rows := sqlmock.NewRows([]string{
		"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
		"client_type", "domain", "identifier", "secret_hash", "status",
		"is_default", "is_system", "token_endpoint_auth_method",
		"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
		"require_consent", "created_at", "updated_at",
	}).AddRow(
		10, testResourceUUID, 1, int64(100), "test-client", "Test Client",
		"spa", nil, clientID, secretHash, "active",
		false, false, TokenAuthMethodSecretBasic,
		pq.StringArray{}, pq.StringArray{}, nil, nil,
		false, time.Now(), time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

	creds := OAuthClientCredentials{ClientID: clientID, ClientSecret: "wrong-secret"}
	result, oerr := authenticateOAuthClient(db, creds)
	assert.Nil(t, result)
	require.NotNil(t, oerr)
	assert.Equal(t, "invalid_client", oerr.Code)
	assert.Contains(t, oerr.Description, "authentication failed")
}

func TestAuthenticateOAuthClient_ClientSecretJWT(t *testing.T) {
	clientID := "test-app"
	domain := "test-app.example.com"
	secret := "my-client-secret"
	secretEncrypted := "test-enc:" + secret

	db, mock := newMockDB(t)
	rows := sqlmock.NewRows([]string{
		"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
		"client_type", "domain", "identifier", "secret_encrypted", "status",
		"is_default", "is_system", "token_endpoint_auth_method",
		"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
		"require_consent", "created_at", "updated_at",
	}).AddRow(
		10, testResourceUUID, 1, int64(100), "test-client", "Test Client",
		"spa", domain, clientID, secretEncrypted, "active",
		false, false, TokenAuthMethodClientSecretJWT,
		pq.StringArray{}, pq.StringArray{}, nil, nil,
		false, time.Now(), time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

	claims := jwtlib.MapClaims{
		"iss": clientID, "sub": clientID, "aud": domain,
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	assertion, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	creds := OAuthClientCredentials{
		ClientID:            clientID,
		ClientAssertionType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
		ClientAssertion:     assertion,
	}
	result, oerr := authenticateOAuthClient(db, creds)
	require.Nil(t, oerr)
	require.NotNil(t, result)
	assert.Equal(t, clientID, *result.Identifier)
}

func TestAuthenticateOAuthClient_UnsupportedAuthMethod(t *testing.T) {
	clientID := "test-app"

	db, mock := newMockDB(t)
	rows := sqlmock.NewRows([]string{
		"client_id", "client_uuid", "tenant_id", "identity_provider_id", "name", "display_name",
		"client_type", "domain", "identifier", "secret_hash", "status",
		"is_default", "is_system", "token_endpoint_auth_method",
		"grant_types", "response_types", "access_token_ttl", "refresh_token_ttl",
		"require_consent", "created_at", "updated_at",
	}).AddRow(
		10, testResourceUUID, 1, int64(100), "test-client", "Test Client",
		"spa", nil, clientID, ptrOrEmpty(&clientID), "active",
		false, false, "custom_unsupported_method",
		pq.StringArray{}, pq.StringArray{}, nil, nil,
		false, time.Now(), time.Now(),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).WillReturnRows(sqlmock.NewRows(nil))

	creds := OAuthClientCredentials{ClientID: clientID}
	result, oerr := authenticateOAuthClient(db, creds)
	assert.Nil(t, result)
	require.NotNil(t, oerr)
	assert.Equal(t, "invalid_client", oerr.Code)
	assert.Contains(t, oerr.Description, "unsupported")
}
