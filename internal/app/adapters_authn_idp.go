package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/idp"
	"gorm.io/gorm"
)

type authnIDPRepoAdapter struct {
	repo idp.IdentityProviderRepository
}

func newAuthnIDPRepoAdapter(repo idp.IdentityProviderRepository) authn.IdentityProviderRepository {
	return &authnIDPRepoAdapter{repo: repo}
}

func (a *authnIDPRepoAdapter) WithTx(tx *gorm.DB) authn.IdentityProviderRepository {
	return &authnIDPRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnIDPRepoAdapter) Create(e *authn.IdentityProvider) (*authn.IdentityProvider, error) {
	r, err := a.repo.Create(toIdpIDP(e))
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) CreateOrUpdate(e *authn.IdentityProvider) (*authn.IdentityProvider, error) {
	r, err := a.repo.CreateOrUpdate(toIdpIDP(e))
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) FindAll(p ...string) ([]authn.IdentityProvider, error) {
	r, err := a.repo.FindAll(p...)
	return mapAuthnIDPs(r), err
}

func (a *authnIDPRepoAdapter) FindByUUID(id any, p ...string) (*authn.IdentityProvider, error) {
	r, err := a.repo.FindByUUID(id, p...)
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) FindByUUIDs(ids []string, p ...string) ([]authn.IdentityProvider, error) {
	r, err := a.repo.FindByUUIDs(ids, p...)
	return mapAuthnIDPs(r), err
}

func (a *authnIDPRepoAdapter) FindByID(id any, p ...string) (*authn.IdentityProvider, error) {
	r, err := a.repo.FindByID(id, p...)
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) UpdateByUUID(id, data any) (*authn.IdentityProvider, error) {
	r, err := a.repo.UpdateByUUID(id, data)
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) UpdateByID(id, data any) (*authn.IdentityProvider, error) {
	r, err := a.repo.UpdateByID(id, data)
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) DeleteByUUID(id any) error {
	return a.repo.DeleteByUUID(id)
}

func (a *authnIDPRepoAdapter) DeleteByID(id any) error {
	return a.repo.DeleteByID(id)
}

func (a *authnIDPRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.IdentityProvider], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.IdentityProvider]{Data: mapAuthnIDPs(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnIDPRepoAdapter) FindByIdentifier(identifier string) (*authn.IdentityProvider, error) {
	r, err := a.repo.FindByIdentifier(identifier)
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) FindByName(name string, tenantID int64) (*authn.IdentityProvider, error) {
	r, err := a.repo.FindByName(name, tenantID)
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) FindDefaultByTenantID(tenantID int64) (*authn.IdentityProvider, error) {
	r, err := a.repo.FindDefaultByTenantID(tenantID)
	return toAuthnIDPFromIdp(r), err
}

func (a *authnIDPRepoAdapter) FindPaginated(f authn.IdentityProviderRepositoryGetFilter) (*authn.PaginationResult[authn.IdentityProvider], error) {
	df := idp.IdentityProviderRepositoryGetFilter{Page: f.Page, Limit: f.Limit, SortBy: f.SortBy, SortOrder: f.SortOrder}
	if f.TenantID > 0 {
		df.TenantID = &f.TenantID
	}
	if f.Status != nil {
		df.Status = []string{*f.Status}
	}
	r, err := a.repo.FindPaginated(df)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.IdentityProvider]{Data: mapAuthnIDPs(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnIDPRepoAdapter) FindAllByTenantID(tenantID int64) ([]authn.IdentityProvider, error) {
	r, err := a.repo.FindAllByTenantID(tenantID)
	return mapAuthnIDPs(r), err
}

func (a *authnIDPRepoAdapter) FindByTenantAndProvider(tenantID int64, provider string) (*authn.IdentityProvider, error) {
	r, err := a.repo.FindByTenantAndProvider(tenantID, provider)
	return toAuthnIDPFromIdp(r), err
}
