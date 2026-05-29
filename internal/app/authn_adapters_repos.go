package app

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/invite"
	"github.com/maintainerd/auth/internal/user"
	"gorm.io/gorm"
)

// ===========================================================================
// Conversion helpers — owner aggregate structs → authn local projections
// ===========================================================================

func toAuthnUser(u *user.User) *authn.User {
	if u == nil {
		return nil
	}
	return &authn.User{
		UserID:              u.UserID,
		UserUUID:            u.UserUUID,
		Username:            u.Username,
		Fullname:            u.Fullname,
		Email:               u.Email,
		Phone:               u.Phone,
		Password:            u.Password,
		IsEmailVerified:     u.IsEmailVerified,
		IsPhoneVerified:     u.IsPhoneVerified,
		IsProfileCompleted:  u.IsProfileCompleted,
		IsAccountCompleted:  u.IsAccountCompleted,
		Status:              u.Status,
		ForcePasswordChange: u.ForcePasswordChange,
		PasswordChangedAt:   u.PasswordChangedAt,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}

func toUserUser(u *authn.User) *user.User {
	if u == nil {
		return nil
	}
	return &user.User{
		UserID:              u.UserID,
		UserUUID:            u.UserUUID,
		Username:            u.Username,
		Fullname:            u.Fullname,
		Email:               u.Email,
		Phone:               u.Phone,
		Password:            u.Password,
		IsEmailVerified:     u.IsEmailVerified,
		IsPhoneVerified:     u.IsPhoneVerified,
		IsProfileCompleted:  u.IsProfileCompleted,
		IsAccountCompleted:  u.IsAccountCompleted,
		Status:              u.Status,
		ForcePasswordChange: u.ForcePasswordChange,
		PasswordChangedAt:   u.PasswordChangedAt,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}

func mapAuthnUsers(items []user.User) []authn.User {
	out := make([]authn.User, len(items))
	for i := range items {
		out[i] = *toAuthnUser(&items[i])
	}
	return out
}

func toAuthnTenantFromClient(t *client.Tenant) *authn.Tenant {
	if t == nil {
		return nil
	}
	return &authn.Tenant{
		TenantID: t.TenantID, TenantUUID: t.TenantUUID, Name: t.Name,
		DisplayName: t.DisplayName, Description: t.Description, Identifier: t.Identifier,
		Status: t.Status, IsPublic: t.IsPublic, IsSystem: t.IsSystem,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toAuthnIDPFromClient(p *client.IdentityProvider) *authn.IdentityProvider {
	if p == nil {
		return nil
	}
	return &authn.IdentityProvider{
		IdentityProviderID: p.IdentityProviderID, IdentityProviderUUID: p.IdentityProviderUUID,
		TenantID: p.TenantID, Name: p.Name, Provider: p.Provider, ProviderType: p.ProviderType,
		Identifier: p.Identifier, Status: p.Status, IsDefault: p.IsDefault, IsSystem: p.IsSystem,
		Tenant:    toAuthnTenantFromClient(p.Tenant),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toAuthnClient(c *client.Client) *authn.Client {
	if c == nil {
		return nil
	}
	return &authn.Client{
		ClientID: c.ClientID, ClientUUID: c.ClientUUID, TenantID: c.TenantID,
		IdentityProviderID: c.IdentityProviderID, Name: c.Name, DisplayName: c.DisplayName,
		ClientType: c.ClientType, Domain: c.Domain, Identifier: c.Identifier,
		Status: c.Status, IsDefault: c.IsDefault, IsSystem: c.IsSystem,
		IdentityProvider: toAuthnIDPFromClient(c.IdentityProvider),
		CreatedAt:        c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toClientClient(c *authn.Client) *client.Client {
	if c == nil {
		return nil
	}
	return &client.Client{
		ClientID: c.ClientID, ClientUUID: c.ClientUUID, TenantID: c.TenantID,
		IdentityProviderID: c.IdentityProviderID, Name: c.Name, DisplayName: c.DisplayName,
		ClientType: c.ClientType, Domain: c.Domain, Identifier: c.Identifier,
		Status: c.Status, IsDefault: c.IsDefault, IsSystem: c.IsSystem,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func mapAuthnClients(items []client.Client) []authn.Client {
	out := make([]authn.Client, len(items))
	for i := range items {
		out[i] = *toAuthnClient(&items[i])
	}
	return out
}

func toAuthnUserIdentity(u *user.UserIdentity) *authn.UserIdentity {
	if u == nil {
		return nil
	}
	return &authn.UserIdentity{
		UserIdentityID: u.UserIdentityID, UserIdentityUUID: u.UserIdentityUUID,
		TenantID: u.TenantID, UserID: u.UserID, ClientID: u.ClientID,
		Provider: u.Provider, Sub: u.Sub, Metadata: u.Metadata,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func toUserUserIdentity(u *authn.UserIdentity) *user.UserIdentity {
	if u == nil {
		return nil
	}
	return &user.UserIdentity{
		UserIdentityID: u.UserIdentityID, UserIdentityUUID: u.UserIdentityUUID,
		TenantID: u.TenantID, UserID: u.UserID, ClientID: u.ClientID,
		Provider: u.Provider, Sub: u.Sub, Metadata: u.Metadata,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func mapAuthnUserIdentities(items []user.UserIdentity) []authn.UserIdentity {
	out := make([]authn.UserIdentity, len(items))
	for i := range items {
		out[i] = *toAuthnUserIdentity(&items[i])
	}
	return out
}

func toAuthnUserRole(u *user.UserRole) *authn.UserRole {
	if u == nil {
		return nil
	}
	return &authn.UserRole{UserRoleID: u.UserRoleID, UserID: u.UserID, RoleID: u.RoleID, CreatedAt: u.CreatedAt}
}

func toUserUserRole(u *authn.UserRole) *user.UserRole {
	if u == nil {
		return nil
	}
	return &user.UserRole{UserRoleID: u.UserRoleID, UserID: u.UserID, RoleID: u.RoleID, CreatedAt: u.CreatedAt}
}

func mapAuthnUserRoles(items []user.UserRole) []authn.UserRole {
	out := make([]authn.UserRole, len(items))
	for i := range items {
		out[i] = *toAuthnUserRole(&items[i])
	}
	return out
}

func toAuthnRole(r *iam.Role) *authn.Role {
	if r == nil {
		return nil
	}
	return &authn.Role{
		RoleID: r.RoleID, RoleUUID: r.RoleUUID, TenantID: r.TenantID, Name: r.Name,
		Description: r.Description, Status: r.Status, IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toIamRole(r *authn.Role) *iam.Role {
	if r == nil {
		return nil
	}
	return &iam.Role{
		RoleID: r.RoleID, RoleUUID: r.RoleUUID, TenantID: r.TenantID, Name: r.Name,
		Description: r.Description, Status: r.Status, IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func mapAuthnRoles(items []iam.Role) []authn.Role {
	out := make([]authn.Role, len(items))
	for i := range items {
		out[i] = *toAuthnRole(&items[i])
	}
	return out
}

func toAuthnTenantFromIdp(t *idp.Tenant) *authn.Tenant {
	if t == nil {
		return nil
	}
	return &authn.Tenant{
		TenantID: t.TenantID, TenantUUID: t.TenantUUID, Name: t.Name,
		DisplayName: t.DisplayName, Description: t.Description, Identifier: t.Identifier,
		Status: t.Status, IsPublic: t.IsPublic, IsSystem: t.IsSystem,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toAuthnIDPFromIdp(p *idp.IdentityProvider) *authn.IdentityProvider {
	if p == nil {
		return nil
	}
	return &authn.IdentityProvider{
		IdentityProviderID: p.IdentityProviderID, IdentityProviderUUID: p.IdentityProviderUUID,
		TenantID: p.TenantID, Name: p.Name, Provider: p.Provider, ProviderType: p.ProviderType,
		Identifier: p.Identifier, Status: p.Status, IsDefault: p.IsDefault, IsSystem: p.IsSystem,
		Tenant:    toAuthnTenantFromIdp(p.Tenant),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toIdpIDP(p *authn.IdentityProvider) *idp.IdentityProvider {
	if p == nil {
		return nil
	}
	return &idp.IdentityProvider{
		IdentityProviderID: p.IdentityProviderID, IdentityProviderUUID: p.IdentityProviderUUID,
		TenantID: p.TenantID, Name: p.Name, Provider: p.Provider, ProviderType: p.ProviderType,
		Identifier: p.Identifier, Status: p.Status, IsDefault: p.IsDefault, IsSystem: p.IsSystem,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func mapAuthnIDPs(items []idp.IdentityProvider) []authn.IdentityProvider {
	out := make([]authn.IdentityProvider, len(items))
	for i := range items {
		out[i] = *toAuthnIDPFromIdp(&items[i])
	}
	return out
}

func toAuthnInvite(i *invite.Invite) *authn.Invite {
	if i == nil {
		return nil
	}
	return &authn.Invite{
		InviteID: i.InviteID, InviteUUID: i.InviteUUID, TenantID: i.TenantID,
		InvitedEmail: i.InvitedEmail, Status: i.Status, ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func toInviteInvite(i *authn.Invite) *invite.Invite {
	if i == nil {
		return nil
	}
	return &invite.Invite{
		InviteID: i.InviteID, InviteUUID: i.InviteUUID, TenantID: i.TenantID,
		InvitedEmail: i.InvitedEmail, Status: i.Status, ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func mapAuthnInvites(items []invite.Invite) []authn.Invite {
	out := make([]authn.Invite, len(items))
	for i := range items {
		out[i] = *toAuthnInvite(&items[i])
	}
	return out
}

// ===========================================================================
// authn.UserRepository  ← user.UserRepository
// ===========================================================================

type authnUserRepoAdapter struct{ repo user.UserRepository }

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
func (a *authnUserRepoAdapter) DeleteByUUID(id any) error { return a.repo.DeleteByUUID(id) }
func (a *authnUserRepoAdapter) DeleteByID(id any) error   { return a.repo.DeleteByID(id) }
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
func (a *authnUserRepoAdapter) SetStatus(id uuid.UUID, s string) error { return a.repo.SetStatus(id, s) }
func (a *authnUserRepoAdapter) SetForcePasswordChange(id uuid.UUID, f bool) error {
	return a.repo.SetForcePasswordChange(id, f)
}
func (a *authnUserRepoAdapter) SetPendingEmail(id uuid.UUID, pendingEmail, token string, expiresAt time.Time) error {
	return a.repo.SetPendingEmail(id, pendingEmail, token, expiresAt)
}
func (a *authnUserRepoAdapter) ClearEmailChange(id uuid.UUID) error { return a.repo.ClearEmailChange(id) }
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

// ===========================================================================
// authn.ClientRepository  ← client.ClientRepository
// ===========================================================================

type authnClientRepoAdapter struct{ repo client.ClientRepository }

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
func (a *authnClientRepoAdapter) DeleteByUUID(id any) error { return a.repo.DeleteByUUID(id) }
func (a *authnClientRepoAdapter) DeleteByID(id any) error   { return a.repo.DeleteByID(id) }
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

// ===========================================================================
// authn.UserIdentityRepository  ← user.UserIdentityRepository
// ===========================================================================

type authnUserIdentityRepoAdapter struct{ repo user.UserIdentityRepository }

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
func (a *authnUserIdentityRepoAdapter) DeleteByUUID(id any) error { return a.repo.DeleteByUUID(id) }
func (a *authnUserIdentityRepoAdapter) DeleteByID(id any) error   { return a.repo.DeleteByID(id) }
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
func (a *authnUserIdentityRepoAdapter) FindByProviderAndSub(provider, sub string) (*authn.UserIdentity, error) {
	r, err := a.repo.FindByProviderAndSub(provider, sub)
	return toAuthnUserIdentity(r), err
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

// ===========================================================================
// authn.IdentityProviderRepository  ← idp.IdentityProviderRepository
// ===========================================================================

type authnIDPRepoAdapter struct{ repo idp.IdentityProviderRepository }

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
func (a *authnIDPRepoAdapter) DeleteByUUID(id any) error { return a.repo.DeleteByUUID(id) }
func (a *authnIDPRepoAdapter) DeleteByID(id any) error   { return a.repo.DeleteByID(id) }
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

// ===========================================================================
// authn.RoleRepository  ← iam.RoleRepository
// ===========================================================================

type authnRoleRepoAdapter struct{ repo iam.RoleRepository }

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
func (a *authnRoleRepoAdapter) DeleteByUUID(id any) error { return a.repo.DeleteByUUID(id) }
func (a *authnRoleRepoAdapter) DeleteByID(id any) error   { return a.repo.DeleteByID(id) }
func (a *authnRoleRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.Role], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.Role]{Data: mapAuthnRoles(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}
func (a *authnRoleRepoAdapter) FindPaginated(f authn.RoleRepositoryGetFilter) (*authn.PaginationResult[authn.Role], error) {
	rf := iam.RoleRepositoryGetFilter{
		Name: f.Name, Description: f.Description, IsDefault: f.IsDefault, IsSystem: f.IsSystem,
		Status: f.Status, TenantID: f.TenantID, Page: f.Page, Limit: f.Limit, SortBy: f.SortBy, SortOrder: f.SortOrder,
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

// ===========================================================================
// authn.UserRoleRepository  ← user.UserRoleRepository
// ===========================================================================

type authnUserRoleRepoAdapter struct{ repo user.UserRoleRepository }

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
func (a *authnUserRoleRepoAdapter) DeleteByUUID(id any) error { return a.repo.DeleteByUUID(id) }
func (a *authnUserRoleRepoAdapter) DeleteByID(id any) error   { return a.repo.DeleteByID(id) }
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

// ===========================================================================
// authn.InviteRepository  ← invite.InviteRepository
// ===========================================================================

type authnInviteRepoAdapter struct{ repo invite.InviteRepository }

func newAuthnInviteRepoAdapter(repo invite.InviteRepository) authn.InviteRepository {
	return &authnInviteRepoAdapter{repo: repo}
}

func (a *authnInviteRepoAdapter) WithTx(tx *gorm.DB) authn.InviteRepository {
	return &authnInviteRepoAdapter{repo: a.repo.WithTx(tx)}
}
func (a *authnInviteRepoAdapter) Create(e *authn.Invite) (*authn.Invite, error) {
	r, err := a.repo.Create(toInviteInvite(e))
	return toAuthnInvite(r), err
}
func (a *authnInviteRepoAdapter) CreateOrUpdate(e *authn.Invite) (*authn.Invite, error) {
	r, err := a.repo.CreateOrUpdate(toInviteInvite(e))
	return toAuthnInvite(r), err
}
func (a *authnInviteRepoAdapter) FindAll(p ...string) ([]authn.Invite, error) {
	r, err := a.repo.FindAll(p...)
	return mapAuthnInvites(r), err
}
func (a *authnInviteRepoAdapter) FindByUUID(id any, p ...string) (*authn.Invite, error) {
	r, err := a.repo.FindByUUID(id, p...)
	return toAuthnInvite(r), err
}
func (a *authnInviteRepoAdapter) FindByUUIDs(ids []string, p ...string) ([]authn.Invite, error) {
	r, err := a.repo.FindByUUIDs(ids, p...)
	return mapAuthnInvites(r), err
}
func (a *authnInviteRepoAdapter) FindByID(id any, p ...string) (*authn.Invite, error) {
	r, err := a.repo.FindByID(id, p...)
	return toAuthnInvite(r), err
}
func (a *authnInviteRepoAdapter) UpdateByUUID(id, data any) (*authn.Invite, error) {
	r, err := a.repo.UpdateByUUID(id, data)
	return toAuthnInvite(r), err
}
func (a *authnInviteRepoAdapter) UpdateByID(id, data any) (*authn.Invite, error) {
	r, err := a.repo.UpdateByID(id, data)
	return toAuthnInvite(r), err
}
func (a *authnInviteRepoAdapter) DeleteByUUID(id any) error { return a.repo.DeleteByUUID(id) }
func (a *authnInviteRepoAdapter) DeleteByID(id any) error   { return a.repo.DeleteByID(id) }
func (a *authnInviteRepoAdapter) Paginate(c map[string]any, page, limit int, p ...string) (*authn.PaginationResult[authn.Invite], error) {
	r, err := a.repo.Paginate(c, page, limit, p...)
	if err != nil || r == nil {
		return nil, err
	}
	return &authn.PaginationResult[authn.Invite]{Data: mapAuthnInvites(r.Data), Total: r.Total, Page: r.Page, Limit: r.Limit, TotalPages: r.TotalPages}, nil
}
func (a *authnInviteRepoAdapter) FindByToken(token string) (*authn.Invite, error) {
	r, err := a.repo.FindByToken(token)
	return toAuthnInvite(r), err
}
func (a *authnInviteRepoAdapter) MarkAsUsed(inviteUUID uuid.UUID) error {
	return a.repo.MarkAsUsed(inviteUUID)
}

// ===========================================================================
// authn.UserPasswordHistoryRepository  ← user.UserPasswordHistoryRepository
// ===========================================================================

type authnPasswordHistoryRepoAdapter struct{ repo user.UserPasswordHistoryRepository }

func newAuthnPasswordHistoryRepoAdapter(repo user.UserPasswordHistoryRepository) authn.UserPasswordHistoryRepository {
	return &authnPasswordHistoryRepoAdapter{repo: repo}
}

func (a *authnPasswordHistoryRepoAdapter) WithTx(tx *gorm.DB) authn.UserPasswordHistoryRepository {
	return &authnPasswordHistoryRepoAdapter{repo: a.repo.WithTx(tx)}
}
func (a *authnPasswordHistoryRepoAdapter) AddEntry(userID int64, hash string) error {
	return a.repo.AddEntry(userID, hash)
}
func (a *authnPasswordHistoryRepoAdapter) FindRecentHashes(userID int64, count int) ([]string, error) {
	return a.repo.FindRecentHashes(userID, count)
}
func (a *authnPasswordHistoryRepoAdapter) PruneExcess(userID int64, keepCount int) error {
	return a.repo.PruneExcess(userID, keepCount)
}
