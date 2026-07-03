package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"gorm.io/gorm"
)

type authnUserIdentityRepoAdapter struct {
	repo user.UserIdentityRepository
}

func newAuthnUserIdentityRepoAdapter(repo user.UserIdentityRepository) authn.UserIdentityRepository {
	return &authnUserIdentityRepoAdapter{repo: repo}
}

func (a *authnUserIdentityRepoAdapter) WithTx(tx *gorm.DB) authn.UserIdentityRepository {
	return &authnUserIdentityRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnUserIdentityRepoAdapter) Create(e *authn.UserIdentity) (*authn.UserIdentity, error) {
	r, err := a.repo.Create(toUserUserIdentity(e))
	return toAuthnUserIdentity(r), err
}

func (a *authnUserIdentityRepoAdapter) CreateOrUpdate(e *authn.UserIdentity) (*authn.UserIdentity, error) {
	r, err := a.repo.CreateOrUpdate(toUserUserIdentity(e))
	return toAuthnUserIdentity(r), err
}

func (a *authnUserIdentityRepoAdapter) FindAll(p ...string) ([]authn.UserIdentity, error) {
	r, err := a.repo.FindAll(p...)
	return mapAuthnUserIdentities(r), err
}

func (a *authnUserIdentityRepoAdapter) FindByUUID(id any, p ...string) (*authn.UserIdentity, error) {
	r, err := a.repo.FindByUUID(id, p...)
	return toAuthnUserIdentity(r), err
}

func (a *authnUserIdentityRepoAdapter) FindByUUIDs(ids []string, p ...string) ([]authn.UserIdentity, error) {
	r, err := a.repo.FindByUUIDs(ids, p...)
	return mapAuthnUserIdentities(r), err
}

func (a *authnUserIdentityRepoAdapter) FindByID(id any, p ...string) (*authn.UserIdentity, error) {
	r, err := a.repo.FindByID(id, p...)
	return toAuthnUserIdentity(r), err
}

func (a *authnUserIdentityRepoAdapter) UpdateByUUID(id, data any) (*authn.UserIdentity, error) {
	r, err := a.repo.UpdateByUUID(id, data)
	return toAuthnUserIdentity(r), err
}

func (a *authnUserIdentityRepoAdapter) UpdateByID(id, data any) (*authn.UserIdentity, error) {
	r, err := a.repo.UpdateByID(id, data)
	return toAuthnUserIdentity(r), err
}

func (a *authnUserIdentityRepoAdapter) DeleteByUUID(id any) error {
	return a.repo.DeleteByUUID(id)
}

func (a *authnUserIdentityRepoAdapter) DeleteByID(id any) error {
	return a.repo.DeleteByID(id)
}

func (a *authnUserIdentityRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.UserIdentity], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.UserIdentity]{Data: mapAuthnUserIdentities(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnUserIdentityRepoAdapter) FindByUserIDAndClientID(userID, clientID int64) (*authn.UserIdentity, error) {
	r, err := a.repo.FindByUserIDAndClientID(userID, clientID)
	return toAuthnUserIdentity(r), err
}

func (a *authnUserIdentityRepoAdapter) FindByUserID(userID int64) ([]authn.UserIdentity, error) {
	r, err := a.repo.FindByUserID(userID)
	return mapAuthnUserIdentities(r), err
}

func (a *authnUserIdentityRepoAdapter) FindByUserIDAndProvider(userID int64, provider string) (*authn.UserIdentity, error) {
	r, err := a.repo.FindByUserIDAndProvider(userID, provider)
	return toAuthnUserIdentity(r), err
}

func (a *authnUserIdentityRepoAdapter) FindByIdentityProviderID(idpID int64) ([]authn.UserIdentity, error) {
	r, err := a.repo.FindByIdentityProviderID(idpID)
	return mapAuthnUserIdentities(r), err
}

func (a *authnUserIdentityRepoAdapter) DeleteByUserID(userID int64) error {
	return a.repo.DeleteByUserID(userID)
}
