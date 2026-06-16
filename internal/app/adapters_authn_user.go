package app

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/user"
	"gorm.io/gorm"
)

type authnUserRepoAdapter struct {
	repo user.UserRepository
}

func newAuthnUserRepoAdapter(repo user.UserRepository) authn.UserRepository {
	return &authnUserRepoAdapter{repo: repo}
}

func (a *authnUserRepoAdapter) WithTx(tx *gorm.DB) authn.UserRepository {
	return &authnUserRepoAdapter{repo: a.repo.WithTx(tx)}
}

func (a *authnUserRepoAdapter) Create(e *authn.User) (*authn.User, error) {
	r, err := a.repo.Create(toUserUser(e))
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) CreateOrUpdate(e *authn.User) (*authn.User, error) {
	r, err := a.repo.CreateOrUpdate(toUserUser(e))
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindAll(p ...string) ([]authn.User, error) {
	r, err := a.repo.FindAll(p...)
	return mapAuthnUsers(r), err
}

func (a *authnUserRepoAdapter) FindByUUID(id any, p ...string) (*authn.User, error) {
	r, err := a.repo.FindByUUID(id, p...)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindByUUIDs(ids []string, p ...string) ([]authn.User, error) {
	r, err := a.repo.FindByUUIDs(ids, p...)
	return mapAuthnUsers(r), err
}

func (a *authnUserRepoAdapter) FindByID(id any, p ...string) (*authn.User, error) {
	r, err := a.repo.FindByID(id, p...)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) UpdateByUUID(id, data any) (*authn.User, error) {
	r, err := a.repo.UpdateByUUID(id, data)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) UpdateByID(id, data any) (*authn.User, error) {
	r, err := a.repo.UpdateByID(id, data)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) DeleteByUUID(id any) error {
	return a.repo.DeleteByUUID(id)
}

func (a *authnUserRepoAdapter) DeleteByID(id any) error {
	return a.repo.DeleteByID(id)
}

func (a *authnUserRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.User], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.User]{Data: mapAuthnUsers(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnUserRepoAdapter) FindByUsername(username string) (*authn.User, error) {
	r, err := a.repo.FindByUsername(username)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindByEmail(email string) (*authn.User, error) {
	r, err := a.repo.FindByEmail(email)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindByEmailAndTenantID(email string, tenantID int64) (*authn.User, error) {
	r, err := a.repo.FindByEmailAndTenantID(email, tenantID)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindByUsernameAndTenantID(username string, tenantID int64) (*authn.User, error) {
	r, err := a.repo.FindByUsernameAndTenantID(username, tenantID)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindByPhoneAndTenantID(phone string, tenantID int64) (*authn.User, error) {
	r, err := a.repo.FindByPhoneAndTenantID(phone, tenantID)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindByPendingEmailAndTenantID(email string, tenantID int64) (*authn.User, error) {
	r, err := a.repo.FindByPendingEmailAndTenantID(email, tenantID)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindByPhone(phone string) (*authn.User, error) {
	r, err := a.repo.FindByPhone(phone)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindSuperAdmin() (*authn.User, error) {
	r, err := a.repo.FindSuperAdmin()
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindRoles(userID int64) ([]authn.Role, error) {
	r, err := a.repo.FindRoles(userID)
	out := make([]authn.Role, len(r))
	for i := range r {
		out[i] = authn.Role{
			RoleID: r[i].RoleID, RoleUUID: r[i].RoleUUID, TenantID: r[i].TenantID, Name: r[i].Name,
			Description: r[i].Description, Status: r[i].Status, IsDefault: r[i].IsDefault, IsSystem: r[i].IsSystem,
			CreatedAt: r[i].CreatedAt, UpdatedAt: r[i].UpdatedAt,
		}
	}
	return out, err
}

func (a *authnUserRepoAdapter) FindBySubAndClientID(sub, clientID string) (*authn.User, error) {
	r, err := a.repo.FindBySubAndClientID(sub, clientID)
	return toAuthnUser(r), err
}

func (a *authnUserRepoAdapter) FindPaginated(f authn.UserRepositoryGetFilter) (*authn.PaginationResult[authn.User], error) {
	uf := user.UserRepositoryGetFilter{Page: f.Page, Limit: f.Limit, SortBy: f.SortBy, SortOrder: f.SortOrder}
	if f.TenantID > 0 {
		uf.TenantID = &f.TenantID
	}
	if f.Status != nil {
		uf.Status = []string{*f.Status}
	}
	r, err := a.repo.FindPaginated(uf)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.User]{Data: mapAuthnUsers(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}

func (a *authnUserRepoAdapter) SetEmailVerified(id uuid.UUID, v bool) error {
	return a.repo.SetEmailVerified(id, v)
}

func (a *authnUserRepoAdapter) SetStatus(id uuid.UUID, s string) error {
	return a.repo.SetStatus(id, s)
}

func (a *authnUserRepoAdapter) SetForcePasswordChange(id uuid.UUID, f bool) error {
	return a.repo.SetForcePasswordChange(id, f)
}

func (a *authnUserRepoAdapter) SetPendingEmail(id uuid.UUID, pendingEmail, token string, expiresAt time.Time) error {
	return a.repo.SetPendingEmail(id, pendingEmail, token, expiresAt)
}

func (a *authnUserRepoAdapter) ClearEmailChange(id uuid.UUID) error {
	return a.repo.ClearEmailChange(id)
}

func (a *authnUserRepoAdapter) UpdateEmail(id uuid.UUID, email string) error {
	return a.repo.UpdateEmail(id, email)
}

func (a *authnUserRepoAdapter) UpdateUsername(id uuid.UUID, username string) error {
	return a.repo.UpdateUsername(id, username)
}

func (a *authnUserRepoAdapter) FindByPendingEmail(email string) (*authn.User, error) {
	r, err := a.repo.FindByPendingEmail(email)
	return toAuthnUser(r), err
}
