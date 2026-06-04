package dpop

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testMethod     = "POST"
	testRequestURL = "https://auth.example.com/oauth/token?ignored=true"
	testAccessTok  = "access-token"
)

type memoryJTIStore struct {
	denied   map[string]time.Duration
	checkErr error
	denyErr  error
}

func (s *memoryJTIStore) DenyJTI(_ context.Context, jti string, ttl time.Duration) error {
	if s.denied == nil {
		s.denied = make(map[string]time.Duration)
	}
	if s.denyErr != nil {
		return s.denyErr
	}
	s.denied[jti] = ttl
	return nil
}

func (s *memoryJTIStore) IsJTIDenied(_ context.Context, jti string) (bool, error) {
	if s.checkErr != nil {
		return false, s.checkErr
	}
	_, ok := s.denied[jti]
	return ok, nil
}

func TestValidateProof(t *testing.T) {
	key := newTestP256Key(t)
	validProof := signDPoPProof(t, key, proofOptions{
		jti:    "jti-valid",
		htm:    testMethod,
		htu:    "https://auth.example.com/oauth/token",
		iat:    time.Now(),
		ath:    AccessTokenHash(testAccessTok),
		typ:    "dpop+jwt",
		header: ecJWK(t, &key.PublicKey),
	})

	tests := []struct {
		name            string
		proofHeader     string
		method          string
		requestURL      string
		accessTokenHash string
		store           *memoryJTIStore
		wantErr         string
		assertClaims    func(t *testing.T, claims *Claims)
	}{
		{
			name:            "valid proof records replay jti and returns claims",
			proofHeader:     validProof,
			method:          testMethod,
			requestURL:      testRequestURL,
			accessTokenHash: AccessTokenHash(testAccessTok),
			store:           &memoryJTIStore{},
			assertClaims: func(t *testing.T, claims *Claims) {
				t.Helper()
				assert.Equal(t, "jti-valid", claims.JTI)
				assert.Equal(t, testMethod, claims.HTTPMethod)
				assert.Equal(t, "https://auth.example.com/oauth/token", claims.HTTPURL)
				assert.NotEmpty(t, claims.Thumbprint)
			},
		},
		{
			name:       "missing header",
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "header is missing",
		},
		{
			name:        "malformed jwt",
			proofHeader: "not-a-jwt",
			method:      testMethod,
			requestURL:  testRequestURL,
			wantErr:     "not a valid JWT",
		},
		{
			name: "wrong typ",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-wrong-typ", htm: testMethod, htu: testRequestURL, iat: time.Now(), typ: "JWT", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "typ must be 'dpop+jwt'",
		},
		{
			name: "missing jwk",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-missing-jwk", htm: testMethod, htu: testRequestURL, iat: time.Now(), typ: "dpop+jwt",
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "missing 'jwk'",
		},
		{
			name: "invalid jwk",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-invalid-jwk", htm: testMethod, htu: testRequestURL, iat: time.Now(), typ: "dpop+jwt", header: map[string]any{"kty": "oct"},
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "unsupported JWK kty",
		},
		{
			name: "unsupported signing algorithm",
			proofHeader: signHSProof(t, proofOptions{
				jti: "jti-hs", htm: testMethod, htu: testRequestURL, iat: time.Now(), typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "unsupported DPoP signing algorithm",
		},
		{
			name: "missing jti",
			proofHeader: signDPoPProof(t, key, proofOptions{
				htm: testMethod, htu: testRequestURL, iat: time.Now(), typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "missing jti",
		},
		{
			name: "method mismatch",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-method", htm: "GET", htu: testRequestURL, iat: time.Now(), typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "does not match request method",
		},
		{
			name: "url mismatch",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-url", htm: testMethod, htu: "https://evil.example.com/oauth/token", iat: time.Now(), typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "does not match request URL",
		},
		{
			name: "missing iat",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-missing-iat", htm: testMethod, htu: testRequestURL, typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "iat invalid",
		},
		{
			name: "old iat",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-old", htm: testMethod, htu: testRequestURL, iat: time.Now().Add(-6 * time.Minute), typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "too old or in the future",
		},
		{
			name: "future iat",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-future", htm: testMethod, htu: testRequestURL, iat: time.Now().Add(time.Minute), typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			wantErr:    "too old or in the future",
		},
		{
			name: "ath mismatch",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-ath", htm: testMethod, htu: testRequestURL, iat: time.Now(), ath: "wrong", typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:          testMethod,
			requestURL:      testRequestURL,
			accessTokenHash: AccessTokenHash(testAccessTok),
			wantErr:         "ath does not match",
		},
		{
			name: "replay check error fails closed",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-cache-error", htm: testMethod, htu: testRequestURL, iat: time.Now(), typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			store:      &memoryJTIStore{checkErr: errors.New("redis down")},
			wantErr:    "replay check failed",
		},
		{
			name: "replayed jti rejected",
			proofHeader: signDPoPProof(t, key, proofOptions{
				jti: "jti-replay", htm: testMethod, htu: testRequestURL, iat: time.Now(), typ: "dpop+jwt", header: ecJWK(t, &key.PublicKey),
			}),
			method:     testMethod,
			requestURL: testRequestURL,
			store:      &memoryJTIStore{denied: map[string]time.Duration{"dpop:jti-replay": replayWindowTTL}},
			wantErr:    "replay detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateProof(context.Background(), tt.proofHeader, tt.method, tt.requestURL, tt.accessTokenHash, tt.store)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, claims)
			if tt.assertClaims != nil {
				tt.assertClaims(t, claims)
			}
			if tt.store != nil {
				assert.Contains(t, tt.store.denied, "dpop:"+claims.JTI)
			}
		})
	}
}

func TestValidateProof_NilStore(t *testing.T) {
	key := newTestP256Key(t)
	proof := signDPoPProof(t, key, proofOptions{
		jti:    "jti-nil-store",
		htm:    testMethod,
		htu:    "https://auth.example.com/oauth/token",
		iat:    time.Now(),
		typ:    "dpop+jwt",
		header: ecJWK(t, &key.PublicKey),
	})

	claims, err := ValidateProof(context.Background(), proof, testMethod, testRequestURL, "", nil)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "jti-nil-store", claims.JTI)
}

func TestValidateResourceRequest(t *testing.T) {
	key := newTestP256Key(t)
	proof := signDPoPProof(t, key, proofOptions{
		jti:    "jti-resource",
		htm:    "GET",
		htu:    "https://api.example.com/resource",
		iat:    time.Now(),
		ath:    AccessTokenHash(testAccessTok),
		typ:    "dpop+jwt",
		header: ecJWK(t, &key.PublicKey),
	})
	_, thumbprint, err := extractPublicKeyAndThumbprint(ecJWK(t, &key.PublicKey))
	require.NoError(t, err)

	claims, err := ValidateResourceRequest(context.Background(), proof, "GET", "https://api.example.com/resource", testAccessTok, thumbprint, nil)
	require.NoError(t, err)
	assert.Equal(t, thumbprint, claims.Thumbprint)

	_, err = ValidateResourceRequest(context.Background(), proof, "GET", "https://api.example.com/resource", testAccessTok, "different-thumbprint", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match token binding")

	claims, err = ValidateResourceRequest(context.Background(), proof, "GET", "https://api.example.com/resource", testAccessTok, "", nil)
	require.NoError(t, err)
	assert.Equal(t, thumbprint, claims.Thumbprint)

	_, err = ValidateResourceRequest(context.Background(), "not-a-jwt", "GET", "https://api.example.com/resource", testAccessTok, thumbprint, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid JWT")
}

func TestAccessTokenHash(t *testing.T) {
	assert.Equal(t, "Pxa-1wifRlPl7yG_0oJNfzqq7MelmOfonFgOFgapzFI", AccessTokenHash("access-token"))
}

func TestExtractNumericDate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	tests := []struct {
		name    string
		claims  jwtlib.MapClaims
		want    time.Time
		wantErr string
	}{
		{name: "float64", claims: jwtlib.MapClaims{"iat": float64(now.Unix())}, want: now},
		{name: "json number", claims: jwtlib.MapClaims{"iat": json.Number("170")}, want: time.Unix(170, 0)},
		{name: "numeric date", claims: jwtlib.MapClaims{"iat": jwtlib.NewNumericDate(now)}, want: now},
		{name: "missing", claims: jwtlib.MapClaims{}, wantErr: "missing"},
		{name: "bad json number", claims: jwtlib.MapClaims{"iat": json.Number("bad")}, wantErr: "invalid syntax"},
		{name: "wrong type", claims: jwtlib.MapClaims{"iat": "170"}, wantErr: "unexpected type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractNumericDate(tt.claims, "iat")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripQuery(t *testing.T) {
	assert.Equal(t, "https://example.com/path", stripQuery("https://example.com/path?a=b"))
	assert.Equal(t, "https://example.com/path", stripQuery("https://example.com/path"))
	assert.Equal(t, "https://example.com", stripQuery("https://example.com"))
}

func TestExtractPublicKeyAndThumbprint_Errors(t *testing.T) {
	t.Run("invalid json marshal", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(make(chan int))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot marshal JWK")
	})

	t.Run("invalid kty", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{"kty": "unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported JWK kty")
	})

	t.Run("EC missing x", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "EC",
			"crv": "P-256",
			"y":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing crv, x, or y")
	})

	t.Run("EC missing y", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing crv, x, or y")
	})

	t.Run("EC invalid x base64", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "EC",
			"crv": "P-256",
			"x":   "!!!",
			"y":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "x decode")
	})

	t.Run("EC invalid y base64", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
			"y":   "!!!",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "y decode")
	})

	t.Run("RSA missing n", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "RSA",
			"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing n or e")
	})

	t.Run("RSA missing e", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "RSA",
			"n":   base64.RawURLEncoding.EncodeToString(make([]byte, 256)),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing n or e")
	})

	t.Run("RSA invalid n base64", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "RSA",
			"n":   "!!!",
			"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "n decode")
	})

	t.Run("RSA invalid e base64", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "RSA",
			"n":   base64.RawURLEncoding.EncodeToString(make([]byte, 256)),
			"e":   "!!!",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "e decode")
	})

	t.Run("EC point not on curve", func(t *testing.T) {
		_, _, err := extractPublicKeyAndThumbprint(map[string]any{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
			"y":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not on curve")
	})
}

type proofOptions struct {
	jti    string
	htm    string
	htu    string
	iat    time.Time
	ath    string
	typ    string
	header map[string]any
}

func newTestP256Key(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

func signDPoPProof(t *testing.T, key *ecdsa.PrivateKey, opts proofOptions) string {
	t.Helper()
	claims := jwtlib.MapClaims{}
	if opts.jti != "" {
		claims["jti"] = opts.jti
	}
	if opts.htm != "" {
		claims["htm"] = opts.htm
	}
	if opts.htu != "" {
		claims["htu"] = opts.htu
	}
	if !opts.iat.IsZero() {
		claims["iat"] = opts.iat.Unix()
	}
	if opts.ath != "" {
		claims["ath"] = opts.ath
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodES256, claims)
	if opts.typ != "" {
		token.Header["typ"] = opts.typ
	}
	if opts.header != nil {
		token.Header["jwk"] = opts.header
	}
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func signHSProof(t *testing.T, opts proofOptions) string {
	t.Helper()
	claims := jwtlib.MapClaims{
		"jti": opts.jti,
		"htm": opts.htm,
		"htu": opts.htu,
		"iat": opts.iat.Unix(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	token.Header["typ"] = opts.typ
	token.Header["jwk"] = opts.header
	signed, err := token.SignedString([]byte("secret"))
	require.NoError(t, err)
	return signed
}

func ecJWK(t *testing.T, pub *ecdsa.PublicKey) map[string]any {
	t.Helper()
	size := (pub.Curve.Params().BitSize + 7) / 8
	return map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(leftPad(pub.X.Bytes(), size)),
		"y":   base64.RawURLEncoding.EncodeToString(leftPad(pub.Y.Bytes(), size)),
	}
}

func rsaJWK(t *testing.T, pub *rsa.PublicKey) map[string]any {
	t.Helper()
	return map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}

func leftPad(in []byte, size int) []byte {
	if len(in) >= size {
		return in
	}
	out := make([]byte, size)
	copy(out[size-len(in):], in)
	return out
}

func assertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), substr), "expected %q to contain %q", err.Error(), substr)
}
