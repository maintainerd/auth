package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"gorm.io/gorm"
)

type authnRoleRepoAdapter struct {
	repo iam.RoleRepository
}

func newAuthnRoleRepoAdapter(repo iam.RoleRepository) authn.RoleRepository {
	return &authnRoleRepoAdapter{repo: repo}
}

func (a *authnRoleRepoAdapter) WithTx(tx *gorm.DB) authn.RoleRepository {
	return &authnRoleRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnRoleRepoAdapter) Create(e *authn.Role) (*authn.Role, error) {
	r, err := a.repo.Create(toIamRole(e))
	return toAuthnRole(r), err
}

func (a *authnRoleRepoAdapter) CreateOrUpdate(e *authn.Role) (*authn.Role, error) {
	r, err := a.repo.CreateOrUpdate(toIamRole(e))
	return toAuthnRole(r), err
}

func (a *authnRoleRepoAdapter) FindAll(p ...string) ([]authn.Role, error) {
	r, err := a.repo.FindAll(p...)
	return mapAuthnRoles(r), err
}

func (a *authnRoleRepoAdapter) FindByUUID(id any, p ...string) (*authn.Role, error) {
	r, err := a.repo.FindByUUID(id, p...)
	return toAuthnRole(r), err
}

func (a *authnRoleRepoAdapter) FindByUUIDs(ids []string, p ...string) ([]authn.Role, error) {
	r, err := a.repo.FindByUUIDs(ids, p...)
	return mapAuthnRoles(r), err
}

func (a *authnRoleRepoAdapter) FindByID(id any, p ...string) (*authn.Role, error) {
	r, err := a.repo.FindByID(id, p...)
	return toAuthnRole(r), err
}

func (a *authnRoleRepoAdapter) UpdateByUUID(id, data any) (*authn.Role, error) {
	r, err := a.repo.UpdateByUUID(id, data)
	return toAuthnRole(r), err
}

func (a *authnRoleRepoAdapter) UpdateByID(id, data any) (*authn.Role, error) {
	r, err := a.repo.UpdateByID(id, data)
	return toAuthnRole(r), err
}

func (a *authnRoleRepoAdapter) DeleteByUUID(id any) error {
	return a.repo.DeleteByUUID(id)
}

func (a *authnRoleRepoAdapter) DeleteByID(id any) error {
	return a.repo.DeleteByID(id)
}

func (a *authnRoleRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.Role], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.Role]{Data: mapAuthnRoles(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnRoleRepoAdapter) FindPaginated(f authn.RoleRepositoryGetFilter) (*authn.PaginationResult[authn.Role], error) {
	var status []string
	if f.Status != nil {
		status = []string{*f.Status}
	}
	rf := iam.RoleRepositoryGetFilter{
		Name: f.Name, Description: f.Description, IsDefault: f.IsDefault, IsSystem: f.IsSystem,
		Status: status, TenantID: f.TenantID, Page: f.Page, Limit: f.Limit, SortBy: f.SortBy, SortOrder: f.SortOrder,
	}
	r, err := a.repo.FindPaginated(rf)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.Role]{Data: mapAuthnRoles(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnRoleRepoAdapter) FindByNameAndTenantID(name string, tenantID int64) (*authn.Role, error) {
	r, err := a.repo.FindByNameAndTenantID(name, tenantID)
	return toAuthnRole(r), err
}

type authnUserRoleRepoAdapter struct {
	repo user.UserRoleRepository
}

func newAuthnUserRoleRepoAdapter(repo user.UserRoleRepository) authn.UserRoleRepository {
	return &authnUserRoleRepoAdapter{repo: repo}
}

func (a *authnUserRoleRepoAdapter) WithTx(tx *gorm.DB) authn.UserRoleRepository {
	return &authnUserRoleRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnUserRoleRepoAdapter) Create(e *authn.UserRole) (*authn.UserRole, error) {
	r, err := a.repo.Create(toUserUserRole(e))
	return toAuthnUserRole(r), err
}

func (a *authnUserRoleRepoAdapter) CreateOrUpdate(e *authn.UserRole) (*authn.UserRole, error) {
	r, err := a.repo.CreateOrUpdate(toUserUserRole(e))
	return toAuthnUserRole(r), err
}

func (a *authnUserRoleRepoAdapter) FindAll(p ...string) ([]authn.UserRole, error) {
	r, err := a.repo.FindAll(p...)
	return mapAuthnUserRoles(r), err
}

func (a *authnUserRoleRepoAdapter) FindByUUID(id any, p ...string) (*authn.UserRole, error) {
	r, err := a.repo.FindByUUID(id, p...)
	return toAuthnUserRole(r), err
}

func (a *authnUserRoleRepoAdapter) FindByUUIDs(ids []string, p ...string) ([]authn.UserRole, error) {
	r, err := a.repo.FindByUUIDs(ids, p...)
	return mapAuthnUserRoles(r), err
}

func (a *authnUserRoleRepoAdapter) FindByID(id any, p ...string) (*authn.UserRole, error) {
	r, err := a.repo.FindByID(id, p...)
	return toAuthnUserRole(r), err
}

func (a *authnUserRoleRepoAdapter) UpdateByUUID(id, data any) (*authn.UserRole, error) {
	r, err := a.repo.UpdateByUUID(id, data)
	return toAuthnUserRole(r), err
}

func (a *authnUserRoleRepoAdapter) UpdateByID(id, data any) (*authn.UserRole, error) {
	r, err := a.repo.UpdateByID(id, data)
	return toAuthnUserRole(r), err
}

func (a *authnUserRoleRepoAdapter) DeleteByUUID(id any) error {
	return a.repo.DeleteByUUID(id)
}

func (a *authnUserRoleRepoAdapter) DeleteByID(id any) error {
	return a.repo.DeleteByID(id)
}

func (a *authnUserRoleRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.UserRole], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.UserRole]{Data: mapAuthnUserRoles(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnUserRoleRepoAdapter) FindByUserIDAndRoleID(userID, roleID int64) (*authn.UserRole, error) {
	r, err := a.repo.FindByUserIDAndRoleID(userID, roleID)
	return toAuthnUserRole(r), err
}
