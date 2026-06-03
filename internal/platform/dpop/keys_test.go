package dpop

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildECPublicKey(t *testing.T) {
	key := newTestP256Key(t)
	x := leftPad(key.PublicKey.X.Bytes(), 32)
	y := leftPad(key.PublicKey.Y.Bytes(), 32)
	p384 := mustGenerateECPoint(t, elliptic.P384())
	p521 := mustGenerateECPoint(t, elliptic.P521())

	tests := []struct {
		name    string
		curve   string
		x       []byte
		y       []byte
		wantErr string
	}{
		{name: "valid P-256", curve: "P-256", x: x, y: y},
		{name: "valid P-384", curve: "P-384", x: p384.x, y: p384.y},
		{name: "valid P-521", curve: "P-521", x: p521.x, y: p521.y},
		{name: "unsupported curve", curve: "P-224", x: x, y: y, wantErr: "unsupported EC curve"},
		{name: "point off curve", curve: "P-256", x: []byte{1}, y: []byte{1}, wantErr: "not on curve"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub, err := buildECPublicKey(tt.curve, tt.x, tt.y)
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.True(t, pub.Curve.IsOnCurve(pub.X, pub.Y))
		})
	}
}

func TestBuildRSAPublicKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	nb64 := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
	eb64 := base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})

	tests := []struct {
		name    string
		n       string
		e       string
		wantErr string
	}{
		{name: "valid key", n: nb64, e: eb64},
		{name: "invalid n", n: "%%%", e: eb64, wantErr: "n decode"},
		{name: "invalid e", n: nb64, e: "%%%", wantErr: "e decode"},
		{name: "zero e", n: nb64, e: base64.RawURLEncoding.EncodeToString([]byte{}), wantErr: "decoded to zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub, err := buildRSAPublicKey(tt.n, tt.e)
			if tt.wantErr != "" {
				assertErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, key.PublicKey.N, pub.N)
			assert.Equal(t, 65537, pub.E)
		})
	}
}

func TestExtractPublicKeyAndThumbprint_RSA(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pub, thumbprint, err := extractPublicKeyAndThumbprint(rsaJWK(t, &key.PublicKey))
	require.NoError(t, err)
	require.IsType(t, &rsa.PublicKey{}, pub)
	assert.NotEmpty(t, thumbprint)
}

type ecPoint struct {
	x []byte
	y []byte
}

func mustGenerateECPoint(t *testing.T, curve elliptic.Curve) ecPoint {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	size := (curve.Params().BitSize + 7) / 8
	return ecPoint{x: leftPad(key.PublicKey.X.Bytes(), size), y: leftPad(key.PublicKey.Y.Bytes(), size)}
}
