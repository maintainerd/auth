package oauth

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type TokenRevocationService interface {
	Revoke(ctx context.Context, tenantID int64, jti, tokenType, reason string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, tenantID int64, jti string) (bool, error)
}

type tokenRevocationService struct {
	repo OAuthTokenRevocationRepository
}

func NewTokenRevocationService(repo OAuthTokenRevocationRepository) TokenRevocationService {
	return &tokenRevocationService{repo: repo}
}

func (s *tokenRevocationService) Revoke(ctx context.Context, tenantID int64, jti, tokenType, reason string, expiresAt time.Time) error {
	_, span := otel.Tracer("service").Start(ctx, "token_revocation.revoke")
	defer span.End()

	rec := &OAuthTokenRevocation{
		TenantID:  tenantID,
		JTI:       jti,
		TokenType: tokenType,
		Reason:    reason,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.Revoke(rec); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "revoke failed")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *tokenRevocationService) IsRevoked(ctx context.Context, tenantID int64, jti string) (bool, error) {
	_, span := otel.Tracer("service").Start(ctx, "token_revocation.is_revoked")
	defer span.End()

	revoked, err := s.repo.IsRevoked(tenantID, jti)
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	return revoked, nil
}
