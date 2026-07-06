package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

type userSessionServiceAdapter struct {
	sessionService authn.SessionService
}

func newUserSessionServiceAdapter(sessionService authn.SessionService) user.SessionService {
	return &userSessionServiceAdapter{sessionService: sessionService}
}

func (a *userSessionServiceAdapter) ListSessions(ctx context.Context, userID int64) ([]*user.SessionDataResult, error) {
	sessions, err := a.sessionService.ListSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*user.SessionDataResult, len(sessions))
	for i, session := range sessions {
		result[i] = &user.SessionDataResult{
			SessionID:         session.SessionID,
			IPAddress:         session.IPAddress,
			UserAgent:         session.UserAgent,
			LastUsedAt:        session.LastActiveAt,
			ExpiresAt:         session.ExpiresAt,
			AbsoluteExpiresAt: session.ExpiresAt,
			CreatedAt:         session.CreatedAt,
		}
	}
	return result, nil
}

func (a *userSessionServiceAdapter) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	return a.sessionService.RevokeSession(ctx, userID, sessionUUID)
}

func (a *userSessionServiceAdapter) RevokeAllSessions(ctx context.Context, userID int64) error {
	return a.sessionService.RevokeAllSessions(ctx, userID)
}

func (a *userSessionServiceAdapter) CreateSession(ctx context.Context, userID, tenantID int64, ipAddress, userAgent string) (*user.UserToken, error) {
	session, err := a.sessionService.CreateSession(ctx, userID, tenantID, ipAddress, userAgent)
	if err != nil || session == nil {
		return nil, err
	}
	return &user.UserToken{
		UserTokenUUID:      session.UserSessionUUID,
		UserID:             session.UserID,
		TokenType:          "user:session",
		ExpiresAt:          &session.ExpiresAt,
		IsRevoked:          false,
		IPAddress:          session.IPAddress,
		UserAgent:          session.UserAgent,
		LastUsedAt:         &session.LastActiveAt,
		IdleTimeoutSeconds: &session.IdleTimeoutSeconds,
		AbsoluteExpiresAt:  &session.ExpiresAt,
		CreatedAt:          session.CreatedAt,
	}, nil
}

func (a *userSessionServiceAdapter) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	return a.sessionService.EnforceConcurrentLimit(ctx, userUUID, userID)
}

func (a *userSessionServiceAdapter) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	return a.sessionService.ValidateAndTouch(ctx, sessionUUID, userID)
}
