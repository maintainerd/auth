package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"gorm.io/gorm"
)

type oauthClientRepo struct {
	*database.BaseRepository[oauth.Client]
}

func newOAuthClientRepo(db *gorm.DB) oauth.ClientRepository {
	return &oauthClientRepo{database.NewBaseRepository[oauth.Client](db, "client_uuid", "client_id")}
}

func (r *oauthClientRepo) WithTx(tx *gorm.DB) oauth.ClientRepository {
	return &oauthClientRepo{r.BaseRepository.WithTx(tx)}
}

func (r *oauthClientRepo) FindSystem() (*oauth.Client, error) {
	var c oauth.Client
	err := r.DB().Where("is_system = ?", true).First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

func (r *oauthClientRepo) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*oauth.Client, error) {
	query := r.DB().Model(&oauth.Client{}).
		Where("clients.identifier = ?", clientID).
		Where("clients.status = ?", shared.StatusActive)
	if identityProviderIdentifier != "" {
		query = query.
			Joins("JOIN client_identity_providers ON client_identity_providers.client_id = clients.client_id").
			Joins("JOIN identity_providers ON identity_providers.identity_provider_id = client_identity_providers.identity_provider_id").
			Where("identity_providers.identifier = ?", identityProviderIdentifier).
			Where("identity_providers.status = ? AND client_identity_providers.enabled = ? AND client_identity_providers.deleted_at IS NULL", shared.StatusActive, true)
	}
	var c oauth.Client
	err := query.
		Preload("Tenant").
		Preload("ClientURIs").
		First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

func (r *oauthClientRepo) FindByIdentifier(identifier string) (*oauth.Client, error) {
	return r.FindByClientIDAndIdentityProvider(identifier, "")
}

func (r *oauthClientRepo) FindSystemByTenantIdentifierAndName(tenantIdentifier, name string) (*oauth.Client, error) {
	var c oauth.Client
	err := r.DB().
		Joins("JOIN tenants ON tenants.tenant_id = clients.tenant_id").
		Where("clients.is_system = ? AND clients.status = ?", true, shared.StatusActive).
		Where("clients.name = ?", name).
		Where("tenants.identifier = ?", tenantIdentifier).
		Preload("Tenant").
		First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

type oauthClientURIRepo struct {
	*database.BaseRepository[oauth.ClientURI]
}

func newOAuthClientURIRepo(db *gorm.DB) oauth.ClientURIRepository {
	return &oauthClientURIRepo{database.NewBaseRepository[oauth.ClientURI](db, "client_uri_uuid", "client_uri_id")}
}

func (r *oauthClientURIRepo) WithTx(tx *gorm.DB) oauth.ClientURIRepository {
	return &oauthClientURIRepo{r.BaseRepository.WithTx(tx)}
}

type oauthTenantRepo struct {
	*database.BaseRepository[oauth.Tenant]
}

func newOAuthTenantRepo(db *gorm.DB) oauth.TenantRepository {
	return &oauthTenantRepo{database.NewBaseRepository[oauth.Tenant](db, "tenant_uuid", "tenant_id")}
}

func (r *oauthTenantRepo) WithTx(tx *gorm.DB) oauth.TenantRepository {
	return &oauthTenantRepo{r.BaseRepository.WithTx(tx)}
}

func (r *oauthTenantRepo) FindSystem() (*oauth.Tenant, error) {
	var t oauth.Tenant
	err := r.DB().Where("is_system = ?", true).First(&t).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &t, nil
}

type oauthUserRepo struct {
	*database.BaseRepository[oauth.User]
}

func newOAuthUserRepo(db *gorm.DB) oauth.UserRepository {
	return &oauthUserRepo{database.NewBaseRepository[oauth.User](db, "user_uuid", "user_id")}
}

func (r *oauthUserRepo) WithTx(tx *gorm.DB) oauth.UserRepository {
	return &oauthUserRepo{r.BaseRepository.WithTx(tx)}
}

// FindByID loads a user (with roles + permissions + profile) by user_id and
// maps it to the oauth/authctx user shape used for token claims.
//
// The oauth user type (oauth.User = authctx.AuthUser) is a context/DTO struct,
// not a GORM table model — its Roles/Permissions/Profile fields have no DB
// columns or foreign keys, so querying GORM directly into it fails schema
// parsing. These finders therefore query the real `users` table model and map
// the result instead of relying on the embedded BaseRepository.
func (r *oauthUserRepo) FindByID(id any, _ ...string) (*oauth.User, error) {
	var u user.User
	err := userWithRolesQuery(r.DB()).Where("user_id = ?", id).First(&u).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return mapUserToOAuthUser(&u), nil
}

func (r *oauthUserRepo) FindByEmailAndTenantID(email string, tenantID int64) (*oauth.User, error) {
	var u user.User
	err := userWithRolesQuery(r.DB()).Where("email = ? AND tenant_id = ?", email, tenantID).First(&u).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return mapUserToOAuthUser(&u), nil
}

func (r *oauthUserRepo) FindBySubAndClientID(sub, clientID string) (*oauth.User, error) {
	var u user.User
	err := userWithRolesQuery(r.DB()).
		Joins("JOIN user_identities ON user_identities.user_id = users.user_id").
		Joins("JOIN clients ON clients.client_id = user_identities.client_id").
		Where("user_identities.sub = ? AND clients.identifier = ?", sub, clientID).
		First(&u).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return mapUserToOAuthUser(&u), nil
}

// userWithRolesQuery preloads the associations needed to populate an
// authctx.AuthUser with its roles, permissions, and profile.
func userWithRolesQuery(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Profile").
		Preload("UserRoles.Role.RolePermissions.Permission")
}

// mapUserToOAuthUser maps the persisted user model to the oauth/authctx user
// shape. It mirrors the canonical toAuthUser mapping used by the auth
// middleware so token claims and request context agree on roles/permissions.
func mapUserToOAuthUser(u *user.User) *oauth.User {
	if u == nil {
		return nil
	}

	roles := make([]authctx.AuthRole, 0, len(u.UserRoles))
	for _, ur := range u.UserRoles {
		if ur.Role == nil {
			continue
		}
		role := ur.Role
		perms := make([]authctx.AuthPermission, 0, len(role.RolePermissions))
		for _, rp := range role.RolePermissions {
			perms = append(perms, authctx.AuthPermission{
				PermissionID:   rp.Permission.PermissionID,
				PermissionUUID: rp.Permission.PermissionUUID,
				Name:           rp.Permission.Name,
			})
		}
		roles = append(roles, authctx.AuthRole{
			RoleID:      role.RoleID,
			RoleUUID:    role.RoleUUID,
			Name:        role.Name,
			Permissions: perms,
		})
	}

	var profile *authctx.AuthProfile
	if u.Profile != nil {
		profile = &authctx.AuthProfile{
			DisplayName: u.Profile.DisplayName,
			FirstName:   u.Profile.FirstName,
			LastName:    u.Profile.LastName,
			ProfileURL:  u.Profile.ProfileURL,
		}
	}

	return &authctx.AuthUser{
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

type oauthUserIdentityRepo struct {
	*database.BaseRepository[oauth.UserIdentity]
}

func newOAuthUserIdentityRepo(db *gorm.DB) oauth.UserIdentityRepository {
	return &oauthUserIdentityRepo{database.NewBaseRepository[oauth.UserIdentity](db, "user_identity_uuid", "user_identity_id")}
}

func (r *oauthUserIdentityRepo) WithTx(tx *gorm.DB) oauth.UserIdentityRepository {
	return &oauthUserIdentityRepo{r.BaseRepository.WithTx(tx)}
}

func (r *oauthUserIdentityRepo) FindByUserIDAndClientID(userID, clientID int64) (*oauth.UserIdentity, error) {
	var identity oauth.UserIdentity
	err := r.DB().Where("user_id = ? AND client_id = ?", userID, clientID).First(&identity).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &identity, nil
}
