package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

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
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no active signing key for tenant %d", tenantID)
	}

	span.SetStatus(codes.Ok, "")
	return &keys[0], nil
}

func pemToJWK(pem string, kid, alg, use string) (JWKKeyDTO, error) {
	k := JWKKeyDTO{
		Kty: "RSA",
		Use: use,
		Kid: kid,
		Alg: alg,
	}
	kidHash := sha256.Sum256([]byte(pem))
	k.Kid = base64.RawURLEncoding.EncodeToString(kidHash[:])
	if kid != "" {
		k.Kid = kid
	}
	return k, nil
}
