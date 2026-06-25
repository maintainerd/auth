package app

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/idp"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type idpTenantRepo struct {
	*database.BaseRepository[idp.Tenant]
}

func newIDPTenantRepo(db *gorm.DB) idp.TenantRepository {
	return &idpTenantRepo{database.NewBaseRepository[idp.Tenant](db, "tenant_uuid", "tenant_id")}
}

func (r *idpTenantRepo) WithTx(tx *gorm.DB) idp.TenantRepository {
	return &idpTenantRepo{r.BaseRepository.WithTx(tx)}
}

type idpUserRepo struct {
	*database.BaseRepository[idp.User]
}

func newIDPUserRepo(db *gorm.DB) idp.UserRepository {
	return &idpUserRepo{database.NewBaseRepository[idp.User](db, "user_uuid", "user_id")}
}

func (r *idpUserRepo) WithTx(tx *gorm.DB) idp.UserRepository {
	return &idpUserRepo{r.BaseRepository.WithTx(tx)}
}

func (r *idpUserRepo) FindByEmail(email string) (*idp.User, error) {
	var u idp.User
	err := r.DB().Where("email = ?", email).First(&u).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &u, nil
}

func (r *idpUserRepo) FindByEmailAndTenantID(email string, tenantID int64) (*idp.User, error) {
	var u idp.User
	err := r.DB().Joins("JOIN user_identities ON user_identities.user_id = users.user_id").
		Where("users.email = ? AND user_identities.tenant_id = ?", email, tenantID).
		First(&u).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &u, nil
}

type idpUserIdentityRepo struct {
	*database.BaseRepository[idp.UserIdentity]
}

func newIDPUserIdentityRepo(db *gorm.DB) idp.UserIdentityRepository {
	return &idpUserIdentityRepo{database.NewBaseRepository[idp.UserIdentity](db, "user_identity_uuid", "user_identity_id")}
}

func (r *idpUserIdentityRepo) WithTx(tx *gorm.DB) idp.UserIdentityRepository {
	return &idpUserIdentityRepo{r.BaseRepository.WithTx(tx)}
}

func (r *idpUserIdentityRepo) FindByUserID(userID int64) ([]idp.UserIdentity, error) {
	var identities []idp.UserIdentity
	err := r.DB().Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

func (r *idpUserIdentityRepo) FindByTenantProviderAndSub(tenantID int64, provider, sub string) (*idp.UserIdentity, error) {
	var identity idp.UserIdentity
	err := r.DB().Where("tenant_id = ? AND provider = ? AND sub = ?", tenantID, provider, sub).First(&identity).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &identity, nil
}

func (r *idpUserIdentityRepo) CreateByTenantProviderSubIfAbsent(identity *idp.UserIdentity) (*idp.UserIdentity, bool, error) {
	if identity.UserIdentityUUID == uuid.Nil {
		identity.UserIdentityUUID = uuid.New()
	}
	result := r.DB().Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "provider"},
			{Name: "sub"},
		},
		DoNothing: true,
	}).Create(identity)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected > 0 {
		return identity, true, nil
	}
	existing, err := r.FindByTenantProviderAndSub(identity.TenantID, identity.Provider, identity.Sub)
	return existing, false, err
}

func (r *idpUserIdentityRepo) FindByUserIDAndProvider(userID int64, provider string) (*idp.UserIdentity, error) {
	var identity idp.UserIdentity
	err := r.DB().Where("user_id = ? AND provider = ?", userID, provider).First(&identity).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &identity, nil
}

func (r *idpUserIdentityRepo) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&idp.UserIdentity{}).Error
}

type idpClientRepo struct {
	*database.BaseRepository[idp.Client]
}

func newIDPClientRepo(db *gorm.DB) idp.ClientRepository {
	return &idpClientRepo{database.NewBaseRepository[idp.Client](db, "client_uuid", "client_id")}
}

func (r *idpClientRepo) WithTx(tx *gorm.DB) idp.ClientRepository {
	return &idpClientRepo{r.BaseRepository.WithTx(tx)}
}

func (r *idpClientRepo) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*idp.Client, error) {
	query := r.DB().Model(&idp.Client{}).Where("clients.identifier = ?", clientID)
	if identityProviderIdentifier != "" {
		query = query.
			Joins("JOIN client_identity_providers ON client_identity_providers.client_id = clients.client_id").
			Joins("JOIN identity_providers ON identity_providers.identity_provider_id = client_identity_providers.identity_provider_id").
			Where("identity_providers.identifier = ?", identityProviderIdentifier)
	}
	var c idp.Client
	err := query.First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

type idpUserRoleRepo struct {
	*database.BaseRepository[idp.UserRole]
}

func newIDPUserRoleRepo(db *gorm.DB) idp.UserRoleRepository {
	return &idpUserRoleRepo{database.NewBaseRepository[idp.UserRole](db, "", "user_role_id")}
}

func (r *idpUserRoleRepo) WithTx(tx *gorm.DB) idp.UserRoleRepository {
	return &idpUserRoleRepo{r.BaseRepository.WithTx(tx)}
}

func (r *idpUserRoleRepo) FindByUserID(userID int64) ([]idp.UserRole, error) {
	var roles []idp.UserRole
	err := r.DB().Where("user_id = ?", userID).Find(&roles).Error
	return roles, err
}

func (r *idpUserRoleRepo) FindByUserIDAndRoleID(userID, roleID int64) (*idp.UserRole, error) {
	var role idp.UserRole
	err := r.DB().Where("user_id = ? AND role_id = ?", userID, roleID).First(&role).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &role, nil
}

func (r *idpUserRoleRepo) DeleteByUserIDAndRoleID(userID, roleID int64) error {
	return r.DB().Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&idp.UserRole{}).Error
}

type idpRoleRepo struct {
	*database.BaseRepository[idp.Role]
}

func newIDPRoleRepo(db *gorm.DB) idp.RoleRepository {
	return &idpRoleRepo{database.NewBaseRepository[idp.Role](db, "role_uuid", "role_id")}
}

func (r *idpRoleRepo) WithTx(tx *gorm.DB) idp.RoleRepository {
	return &idpRoleRepo{r.BaseRepository.WithTx(tx)}
}

func (r *idpRoleRepo) FindByNameAndTenantID(name string, tenantID int64) (*idp.Role, error) {
	var role idp.Role
	err := r.DB().Where("name = ? AND tenant_id = ?", name, tenantID).First(&role).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &role, nil
}

func (r *idpRoleRepo) FindPaginated(filter idp.RoleRepositoryGetFilter) (*idp.PaginationResult[idp.Role], error) {
	conditions := map[string]any{"tenant_id": filter.TenantID}
	if filter.Name != nil {
		conditions["name"] = *filter.Name
	}
	if filter.Status != nil {
		conditions["status"] = *filter.Status
	}
	if filter.IsDefault != nil {
		conditions["is_default"] = *filter.IsDefault
	}
	if filter.IsSystem != nil {
		conditions["is_system"] = *filter.IsSystem
	}
	return r.Paginate(conditions, filter.Page, filter.Limit)
}
