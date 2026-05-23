package dpop

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
)

// buildECPublicKey reconstructs an *ecdsa.PublicKey from raw x/y coordinates
// and a curve name (P-256, P-384, P-521).
func buildECPublicKey(crv string, x, y []byte) (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	switch crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve: %q", crv)
	}

	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(x),
		Y:     new(big.Int).SetBytes(y),
	}
	if !curve.IsOnCurve(pub.X, pub.Y) {
		return nil, fmt.Errorf("EC point (%s) is not on curve %q", crv, crv)
	}
	return pub, nil
}

// buildRSAPublicKey reconstructs an *rsa.PublicKey from base64url-encoded n and e.
func buildRSAPublicKey(nb64, eb64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nb64)
	if err != nil {
		return nil, fmt.Errorf("RSA JWK n decode: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eb64)
	if err != nil {
		return nil, fmt.Errorf("RSA JWK e decode: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	if e == 0 {
		return nil, fmt.Errorf("RSA JWK e decoded to zero")
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}
