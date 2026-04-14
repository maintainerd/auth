package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// OAuthConsentService manages user consent grants (the persisted records of
// which scopes a user has approved for a client).
type OAuthConsentService interface {
	// ListGrants returns all consent grants for the authenticated user.
	ListGrants(ctx context.Context, userID int64) ([]dto.OAuthConsentGrantResponseDTO, error)

	// RevokeGrant removes a consent grant, forcing the user to re-consent on
	// the next authorization request.
	RevokeGrant(ctx context.Context, grantUUID uuid.UUID, userID int64) error
}

type oauthConsentService struct {
	consentGrantRepo repository.OAuthConsentGrantRepository
}

// NewOAuthConsentService creates a new OAuthConsentService.
func NewOAuthConsentService(
	consentGrantRepo repository.OAuthConsentGrantRepository,
) OAuthConsentService {
	return &oauthConsentService{
		consentGrantRepo: consentGrantRepo,
	}
}

// ListGrants implements OAuthConsentService.
func (s *oauthConsentService) ListGrants(ctx context.Context, userID int64) ([]dto.OAuthConsentGrantResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_consent.list_grants")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	grants, err := s.consentGrantRepo.FindByUserID(userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "consent grants lookup failed")
		return nil, apperror.NewInternal("failed to retrieve consent grants", err)
	}

	result := make([]dto.OAuthConsentGrantResponseDTO, len(grants))
	for i, g := range grants {
		scopes := strings.Fields(g.Scopes)
		clientName := ""
		clientUUID := ""
		if g.Client != nil {
			clientName = g.Client.DisplayName
			clientUUID = g.Client.ClientUUID.String()
		}
		result[i] = dto.OAuthConsentGrantResponseDTO{
			ConsentGrantUUID: g.OAuthConsentGrantUUID.String(),
			ClientName:       clientName,
			ClientUUID:       clientUUID,
			Scopes:           scopes,
			GrantedAt:        g.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:        g.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// RevokeGrant implements OAuthConsentService.
func (s *oauthConsentService) RevokeGrant(ctx context.Context, grantUUID uuid.UUID, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "oauth_consent.revoke_grant")
	defer span.End()
	span.SetAttributes(
		attribute.String("consent.grant_uuid", grantUUID.String()),
		attribute.Int64("user.id", userID),
	)

	// Look up the grant to verify ownership.
	grants, err := s.consentGrantRepo.FindByUserID(userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "consent grant lookup failed")
		return apperror.NewInternal("failed to revoke consent grant", err)
	}

	var found *int64
	for _, g := range grants {
		if g.OAuthConsentGrantUUID == grantUUID {
			found = &g.ClientID
			break
		}
	}

	if found == nil {
		span.SetStatus(codes.Error, "consent grant not found or not owned by user")
		return apperror.NewNotFoundWithReason("consent grant not found")
	}

	if err := s.consentGrantRepo.DeleteByUserAndClient(userID, *found); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "consent grant deletion failed")
		return apperror.NewInternal("failed to revoke consent grant", err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
