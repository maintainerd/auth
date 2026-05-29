package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authn"
	platformcache "github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/user"
)

type middlewareUserContextProvider struct {
	userService user.UserService
}

func newMiddlewareUserContextProvider(userService user.UserService) *middlewareUserContextProvider {
	return &middlewareUserContextProvider{userService: userService}
}

func (p *middlewareUserContextProvider) FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*platformcache.AuthUser, error) {
	u, err := p.userService.FindBySubAndClientID(ctx, sub, clientID)
	if err != nil || u == nil {
		return nil, err
	}
	return toAuthUser(u), nil
}

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
			LastUsedAt:        session.LastUsedAt,
			ExpiresAt:         session.ExpiresAt,
			AbsoluteExpiresAt: session.AbsoluteExpiresAt,
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

func (a *userSessionServiceAdapter) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*user.UserToken, error) {
	token, err := a.sessionService.CreateSession(ctx, userID, ipAddress, userAgent)
	if err != nil || token == nil {
		return nil, err
	}
	return &user.UserToken{
		UserTokenID:        token.UserTokenID,
		UserTokenUUID:      token.UserTokenUUID,
		UserID:             token.UserID,
		TokenType:          token.TokenType,
		Token:              token.Token,
		ExpiresAt:          token.ExpiresAt,
		IsRevoked:          token.IsRevoked,
		IPAddress:          token.IPAddress,
		UserAgent:          token.UserAgent,
		LastUsedAt:         token.LastUsedAt,
		IdleTimeoutSeconds: token.IdleTimeoutSeconds,
		AbsoluteExpiresAt:  token.AbsoluteExpiresAt,
		CreatedAt:          token.CreatedAt,
		UpdatedAt:          token.UpdatedAt,
	}, nil
}

func (a *userSessionServiceAdapter) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	return a.sessionService.EnforceConcurrentLimit(ctx, userUUID, userID)
}

func (a *userSessionServiceAdapter) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	return a.sessionService.ValidateAndTouch(ctx, sessionUUID, userID)
}

func toAuthUser(u *user.User) *platformcache.AuthUser {
	if u == nil {
		return nil
	}

	roles := make([]platformcache.AuthRole, len(u.Roles))
	for i, role := range u.Roles {
		roles[i] = platformcache.AuthRole{
			RoleID:   role.RoleID,
			RoleUUID: role.RoleUUID,
			Name:     role.Name,
		}
	}

	var profile *platformcache.AuthProfile
	if u.Profile != nil {
		profile = &platformcache.AuthProfile{
			DisplayName: u.Profile.DisplayName,
			FirstName:   u.Profile.FirstName,
			LastName:    u.Profile.LastName,
			ProfileURL:  u.Profile.ProfileURL,
		}
	}

	return &platformcache.AuthUser{
		UserID:          u.UserID,
		UserUUID:        u.UserUUID,
		Roles:           roles,
		Email:           u.Email,
		IsEmailVerified: u.IsEmailVerified,
		Phone:           u.Phone,
		IsPhoneVerified: u.IsPhoneVerified,
		Fullname:        u.Fullname,
		UpdatedAt:       u.UpdatedAt,
		Profile:         profile,
	}
}
