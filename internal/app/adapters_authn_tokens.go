package app

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/user"
	"gorm.io/gorm"
)

// authnUserTokenRepoAdapter bridges the user-owned repository to authn's local
// session/token projection types.
type authnUserTokenRepoAdapter struct {
	repo user.UserTokenRepository
}

func newAuthnUserTokenRepoAdapter(repo user.UserTokenRepository) authn.UserTokenRepository {
	return &authnUserTokenRepoAdapter{repo: repo}
}

func (a *authnUserTokenRepoAdapter) WithTx(tx *gorm.DB) authn.UserTokenRepository {
	return &authnUserTokenRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnUserTokenRepoAdapter) Create(token *authn.UserToken) (*authn.UserToken, error) {
	created, err := a.repo.Create(toUserUserToken(token))
	if err != nil {
		return nil, err
	}
	return toAuthnUserToken(created), nil
}

func (a *authnUserTokenRepoAdapter) CreateOrUpdate(token *authn.UserToken) (*authn.UserToken, error) {
	created, err := a.repo.CreateOrUpdate(toUserUserToken(token))
	if err != nil {
		return nil, err
	}
	return toAuthnUserToken(created), nil
}

func (a *authnUserTokenRepoAdapter) FindAll(preloads ...string) ([]authn.UserToken, error) {
	items, err := a.repo.FindAll(preloads...)
	if err != nil {
		return nil, err
	}
	return mapUserTokensToAuthn(items), nil
}

func (a *authnUserTokenRepoAdapter) FindByUUID(id any, preloads ...string) (*authn.UserToken, error) {
	item, err := a.repo.FindByUUID(id, preloads...)
	if err != nil || item == nil {
		return nil, err
	}
	return toAuthnUserToken(item), nil
}

func (a *authnUserTokenRepoAdapter) FindByUUIDs(ids []string, preloads ...string) ([]authn.UserToken, error) {
	items, err := a.repo.FindByUUIDs(ids, preloads...)
	if err != nil {
		return nil, err
	}
	return mapUserTokensToAuthn(items), nil
}

func (a *authnUserTokenRepoAdapter) FindByID(id any, preloads ...string) (*authn.UserToken, error) {
	item, err := a.repo.FindByID(id, preloads...)
	if err != nil || item == nil {
		return nil, err
	}
	return toAuthnUserToken(item), nil
}

func (a *authnUserTokenRepoAdapter) UpdateByUUID(id, data any) (*authn.UserToken, error) {
	if token, ok := data.(*authn.UserToken); ok {
		updated, err := a.repo.UpdateByUUID(id, toUserUserToken(token))
		if err != nil || updated == nil {
			return nil, err
		}
		return toAuthnUserToken(updated), nil
	}
	updated, err := a.repo.UpdateByUUID(id, data)
	if err != nil || updated == nil {
		return nil, err
	}
	return toAuthnUserToken(updated), nil
}

func (a *authnUserTokenRepoAdapter) UpdateByID(id, data any) (*authn.UserToken, error) {
	if token, ok := data.(*authn.UserToken); ok {
		updated, err := a.repo.UpdateByID(id, toUserUserToken(token))
		if err != nil || updated == nil {
			return nil, err
		}
		return toAuthnUserToken(updated), nil
	}
	updated, err := a.repo.UpdateByID(id, data)
	if err != nil || updated == nil {
		return nil, err
	}
	return toAuthnUserToken(updated), nil
}

func (a *authnUserTokenRepoAdapter) DeleteByUUID(id any) error { return a.repo.DeleteByUUID(id) }
func (a *authnUserTokenRepoAdapter) DeleteByID(id any) error   { return a.repo.DeleteByID(id) }

func (a *authnUserTokenRepoAdapter) Paginate(c map[string]any, page, limit int, preloads ...string) (*authn.PaginationResult[authn.UserToken], error) {
	result, err := a.repo.Paginate(c, page, limit, preloads...)
	if err != nil || result == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.UserToken]{
		Data:       mapUserTokensToAuthn(result.Data),
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (a *authnUserTokenRepoAdapter) FindByUserID(userID int64) ([]authn.UserToken, error) {
	items, err := a.repo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	return mapUserTokensToAuthn(items), nil
}

func (a *authnUserTokenRepoAdapter) FindActiveTokensByUserID(userID int64) ([]authn.UserToken, error) {
	items, err := a.repo.FindActiveTokensByUserID(userID)
	if err != nil {
		return nil, err
	}
	return mapUserTokensToAuthn(items), nil
}

func (a *authnUserTokenRepoAdapter) FindByUserIDAndTokenType(userID int64, tokenType string) ([]authn.UserToken, error) {
	items, err := a.repo.FindByUserIDAndTokenType(userID, tokenType)
	if err != nil {
		return nil, err
	}
	return mapUserTokensToAuthn(items), nil
}

func (a *authnUserTokenRepoAdapter) RevokeByUUID(id uuid.UUID) error { return a.repo.RevokeByUUID(id) }
func (a *authnUserTokenRepoAdapter) RevokeAllByUserID(userID int64) error {
	return a.repo.RevokeAllByUserID(userID)
}
func (a *authnUserTokenRepoAdapter) DeleteByUserID(userID int64) error {
	return a.repo.DeleteByUserID(userID)
}
func (a *authnUserTokenRepoAdapter) DeleteExpiredTokens(before time.Time) error {
	return a.repo.DeleteExpiredTokens(before)
}
func (a *authnUserTokenRepoAdapter) CountActiveSessions(userID int64) (int64, error) {
	return a.repo.CountActiveSessions(userID)
}
func (a *authnUserTokenRepoAdapter) TouchSession(userID int64, sessionUUID uuid.UUID, now time.Time) error {
	return a.repo.TouchSession(userID, sessionUUID, now)
}
func (a *authnUserTokenRepoAdapter) RevokeSessionByUUID(userID int64, sessionUUID uuid.UUID) error {
	return a.repo.RevokeSessionByUUID(userID, sessionUUID)
}
func (a *authnUserTokenRepoAdapter) RevokeAllSessionsByUserID(userID int64) error {
	return a.repo.RevokeAllSessionsByUserID(userID)
}

func (a *authnUserTokenRepoAdapter) FindActiveSessions(userID int64) ([]authn.UserToken, error) {
	items, err := a.repo.FindActiveSessions(userID)
	if err != nil {
		return nil, err
	}
	return mapUserTokensToAuthn(items), nil
}

func (a *authnUserTokenRepoAdapter) FindActiveSessionByUUID(userID int64, sessionUUID uuid.UUID) (*authn.UserToken, error) {
	item, err := a.repo.FindActiveSessionByUUID(userID, sessionUUID)
	if err != nil || item == nil {
		return nil, err
	}
	return toAuthnUserToken(item), nil
}

func toAuthnUserToken(token *user.UserToken) *authn.UserToken {
	if token == nil {
		return nil
	}
	return &authn.UserToken{
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
	}
}

func toUserUserToken(token *authn.UserToken) *user.UserToken {
	if token == nil {
		return nil
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
	}
}

func mapUserTokensToAuthn(items []user.UserToken) []authn.UserToken {
	mapped := make([]authn.UserToken, len(items))
	for i := range items {
		mapped[i] = *toAuthnUserToken(&items[i])
	}
	return mapped
}
