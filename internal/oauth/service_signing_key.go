package oauth

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type KeyRotationService interface {
	ListJWKS(ctx context.Context, tenantID int64) ([]JWKKeyDTO, error)
	GetActiveSigningKey(ctx context.Context, tenantID int64) (*SigningKey, error)
}

type keyRotationService struct {
	repo SigningKeyRepository
}

func NewKeyRotationService(repo SigningKeyRepository) KeyRotationService {
	return &keyRotationService{repo: repo}
}

func (s *keyRotationService) ListJWKS(ctx context.Context, tenantID int64) ([]JWKKeyDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "signing_keys.list_jwks")
	defer span.End()

	keys, err := s.repo.FindActiveByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list jwks failed")
		return nil, err
	}

	jwks := make([]JWKKeyDTO, 0, len(keys))
	for _, k := range keys {
		jwk, err := pemToJWK(k.PublicKeyPEM, k.KID, k.Algorithm, k.Use)
		if err != nil {
			// Skip keys that cannot be serialised (e.g. unsupported algorithm).
			continue
		}
		jwks = append(jwks, jwk)
	}

	span.SetStatus(codes.Ok, "")
	return jwks, nil
}

func (s *keyRotationService) GetActiveSigningKey(ctx context.Context, tenantID int64) (*SigningKey, error) {
	_, span := otel.Tracer("service").Start(ctx, "signing_keys.get_active")
	defer span.End()

	keys, err := s.repo.FindActiveByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get active signing key failed")
		return nil, err
	}
	if len(keys) == 0 {
		err := fmt.Errorf("no active signing key for tenant %d", tenantID)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &keys[0], nil
}

// pemToJWK converts a PEM-encoded RSA public key into a JWKKeyDTO.
// Non-RSA algorithms are not yet supported and return an error so callers can skip them.
func pemToJWK(pemStr, kid, alg, use string) (JWKKeyDTO, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return JWKKeyDTO{}, fmt.Errorf("pemToJWK: failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return JWKKeyDTO{}, fmt.Errorf("pemToJWK: failed to parse public key: %w", err)
	}

	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return JWKKeyDTO{}, fmt.Errorf("pemToJWK: algorithm %s is not RSA; EC/EdDSA JWK serialisation not yet implemented", alg)
	}

	resolvedKid := kid
	if resolvedKid == "" {
		h := sha256.Sum256(block.Bytes)
		resolvedKid = base64.RawURLEncoding.EncodeToString(h[:])
	}

	return JWKKeyDTO{
		Kty: "RSA",
		Use: use,
		Kid: resolvedKid,
		Alg: alg,
		N:   base64.RawURLEncoding.EncodeToString(rsaKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(rsaKey.E)).Bytes()),
	}, nil
}
