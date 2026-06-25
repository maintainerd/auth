package app

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/client"
	"gorm.io/gorm"
)

type authnClientRepoAdapter struct {
	repo client.ClientRepository
}

func newAuthnClientRepoAdapter(repo client.ClientRepository) authn.ClientRepository {
	return &authnClientRepoAdapter{repo: repo}
}

func (a *authnClientRepoAdapter) WithTx(tx *gorm.DB) authn.ClientRepository {
	return &authnClientRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnClientRepoAdapter) Create(e *authn.Client) (*authn.Client, error) {
	r, err := a.repo.Create(toClientClient(e))
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) CreateOrUpdate(e *authn.Client) (*authn.Client, error) {
	r, err := a.repo.CreateOrUpdate(toClientClient(e))
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindAll(p ...string) ([]authn.Client, error) {
	r, err := a.repo.FindAll(p...)
	return mapAuthnClients(r), err
}

func (a *authnClientRepoAdapter) FindByUUID(id any, p ...string) (*authn.Client, error) {
	r, err := a.repo.FindByUUID(id, p...)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindByUUIDs(ids []string, p ...string) ([]authn.Client, error) {
	r, err := a.repo.FindByUUIDs(ids, p...)
	return mapAuthnClients(r), err
}

func (a *authnClientRepoAdapter) FindByID(id any, p ...string) (*authn.Client, error) {
	r, err := a.repo.FindByID(id, p...)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) UpdateByUUID(id, data any) (*authn.Client, error) {
	r, err := a.repo.UpdateByUUID(id, data)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) UpdateByID(id, data any) (*authn.Client, error) {
	r, err := a.repo.UpdateByID(id, data)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) DeleteByUUID(id any) error {
	return a.repo.DeleteByUUID(id)
}

func (a *authnClientRepoAdapter) DeleteByID(id any) error {
	return a.repo.DeleteByID(id)
}

func (a *authnClientRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.Client], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.Client]{Data: mapAuthnClients(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnClientRepoAdapter) FindSystem() (*authn.Client, error) {
	r, err := a.repo.FindSystem()
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindByIdentifier(identifier string) (*authn.Client, error) {
	r, err := a.repo.FindByIdentifier(identifier)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindSystemByTenantIdentifier(tenantIdentifier string) (*authn.Client, error) {
	r, err := a.repo.FindSystemByTenantIdentifier(tenantIdentifier)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindSystemByTenantIdentifierAndName(tenantIdentifier, name string) (*authn.Client, error) {
	r, err := a.repo.FindSystemByTenantIdentifierAndName(tenantIdentifier, name)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindByClientIDAndIdentityProvider(clientID, providerID string) (*authn.Client, error) {
	r, err := a.repo.FindByClientIDAndIdentityProvider(clientID, providerID)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64) (*authn.Client, error) {
	r, err := a.repo.FindByUUIDAndTenantID(id, tenantID)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindByNameAndIdentityProvider(name string, ipID, tenantID int64) (*authn.Client, error) {
	r, err := a.repo.FindByNameAndIdentityProvider(name, ipID, tenantID)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindByNameAndTenantID(name string, tenantID int64) (*authn.Client, error) {
	r, err := a.repo.FindByNameAndTenantID(name, tenantID)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindByClientID(clientID string, tenantID int64) (*authn.Client, error) {
	r, err := a.repo.FindByClientID(clientID, tenantID)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindAllByTenantID(tenantID int64) ([]authn.Client, error) {
	r, err := a.repo.FindAllByTenantID(tenantID)
	return mapAuthnClients(r), err
}

func (a *authnClientRepoAdapter) FindDefaultByTenantID(tenantID int64) (*authn.Client, error) {
	r, err := a.repo.FindDefaultByTenantID(tenantID)
	return toAuthnClient(r), err
}

func (a *authnClientRepoAdapter) FindPaginated(f authn.ClientRepositoryGetFilter) (*authn.PaginationResult[authn.Client], error) {
	cf := client.ClientRepositoryGetFilter{TenantID: f.TenantID, Name: f.Name, Page: f.Page, Limit: f.Limit, SortBy: f.SortBy, SortOrder: f.SortOrder}
	if f.IdentityProviderID > 0 {
		cf.IdentityProviderID = &f.IdentityProviderID
	}
	if f.Status != nil {
		cf.Status = []string{*f.Status}
	}
	r, err := a.repo.FindPaginated(cf)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.Client]{Data: mapAuthnClients(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnClientRepoAdapter) SetStatusByUUID(id uuid.UUID, tenantID int64, status string) error {
	return a.repo.SetStatusByUUID(id, tenantID, status)
}

func (a *authnClientRepoAdapter) DeleteByUUIDAndTenantID(id uuid.UUID, tenantID int64) error {
	return a.repo.DeleteByUUIDAndTenantID(id, tenantID)
}
