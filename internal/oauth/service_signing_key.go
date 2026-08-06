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
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type KeyRotationService interface {
	ListJWKS(ctx context.Context, tenantID int64) ([]JWKKeyDTO, error)
	GetActiveSigningKey(ctx context.Context, tenantID int64) (*SigningKey, error)

	// ListKeys returns the metadata operators need to see what is signing tokens.
	// It never returns private key material.
	ListKeys(ctx context.Context, tenantID int64) ([]SigningKeyResponseDTO, error)

	// Rotate mints and persists a new global signing key and installs it.
	Rotate(ctx context.Context) error

	// Retire takes a key out of the published key set. Tokens already signed with
	// it stop verifying once it leaves JWKS, so this is for keys whose tokens have
	// aged out.
	Retire(ctx context.Context, kid string) error

	// MarkCompromised is the emergency control: the key stops being published and
	// stops being usable immediately, accepting that live tokens signed with it
	// break. That is the point — a compromised key must not keep validating.
	MarkCompromised(ctx context.Context, kid string) error
}

// SigningKeyResponseDTO is the operator-facing view of a signing key. It
// deliberately has no field for PrivateKeyEncrypted: the management surface must
// not be able to read key material back out, encrypted or not.
type SigningKeyResponseDTO struct {
	KID          string  `json:"kid"`
	Algorithm    string  `json:"algorithm"`
	Use          string  `json:"use"`
	Status       string  `json:"status"`
	PublicKeyPEM string  `json:"public_key_pem"`
	RotatedAt    *string `json:"rotated_at,omitempty"`
	ExpiresAt    *string `json:"expires_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type keyRotationService struct {
	repo SigningKeyRepository
	db   *gorm.DB
}

// NewKeyRotationService builds the service. db is variadic so existing call
// sites that only need JWKS serving keep compiling; the rotation entry point
// needs it because rotation writes a new row and installs the key.
func NewKeyRotationService(repo SigningKeyRepository, db ...*gorm.DB) KeyRotationService {
	svc := &keyRotationService{repo: repo}
	if len(db) > 0 {
		svc.db = db[0]
	}
	return svc
}

func (s *keyRotationService) ListKeys(ctx context.Context, tenantID int64) ([]SigningKeyResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "signing_keys.list")
	defer span.End()

	keys, err := s.repo.FindActiveByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list signing keys failed")
		return nil, err
	}

	out := make([]SigningKeyResponseDTO, 0, len(keys))
	for _, k := range keys {
		dto := SigningKeyResponseDTO{
			KID:          k.KID,
			Algorithm:    k.Algorithm,
			Use:          k.Use,
			Status:       k.Status,
			PublicKeyPEM: k.PublicKeyPEM,
			CreatedAt:    k.CreatedAt.UTC().Format(time.RFC3339),
		}
		if k.RotatedAt != nil {
			dto.RotatedAt = ptr.Ptr(k.RotatedAt.UTC().Format(time.RFC3339))
		}
		if k.ExpiresAt != nil {
			dto.ExpiresAt = ptr.Ptr(k.ExpiresAt.UTC().Format(time.RFC3339))
		}
		out = append(out, dto)
	}

	span.SetStatus(codes.Ok, "")
	return out, nil
}

func (s *keyRotationService) Rotate(ctx context.Context) error {
	_, span := otel.Tracer("service").Start(ctx, "signing_keys.rotate")
	defer span.End()

	if s.db == nil {
		err := fmt.Errorf("signing key rotation is not available: no database handle")
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := rotateGlobalSigningKey(ctx, s.repo); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "rotation failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *keyRotationService) Retire(ctx context.Context, kid string) error {
	_, span := otel.Tracer("service").Start(ctx, "signing_keys.retire")
	defer span.End()

	if err := s.guardNotLastActiveKey(kid); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := s.repo.RetireByKID(kid); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "retire failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *keyRotationService) MarkCompromised(ctx context.Context, kid string) error {
	_, span := otel.Tracer("service").Start(ctx, "signing_keys.mark_compromised")
	defer span.End()

	// No last-key guard here on purpose. Refusing to disown a key because it is
	// the only one would keep a KNOWN-compromised key signing and verifying, which
	// is strictly worse than an outage the operator can end by rotating.
	if err := s.repo.MarkCompromised(kid); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "mark compromised failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// guardNotLastActiveKey refuses to retire the only remaining active key.
// Retiring it would empty the published key set while tokens are still being
// issued, so every verification — internal and external — would start failing.
func (s *keyRotationService) guardNotLastActiveKey(kid string) error {
	keys, err := s.repo.FindActiveByTenantID(0)
	if err != nil {
		return err
	}
	remaining := 0
	for _, k := range keys {
		if k.KID != kid {
			remaining++
		}
	}
	if remaining == 0 {
		return fmt.Errorf("refusing to retire the last active signing key; rotate first")
	}
	return nil
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
