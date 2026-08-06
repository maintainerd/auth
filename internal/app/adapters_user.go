package app

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"gorm.io/gorm"
)

type userTenantRepo struct {
	*database.BaseRepository[user.Tenant]
}

func newUserTenantRepo(db *gorm.DB) user.TenantRepository {
	return &userTenantRepo{database.NewBaseRepository[user.Tenant](db, "tenant_uuid", "tenant_id")}
}

func (r *userTenantRepo) WithTx(tx *gorm.DB) user.TenantRepository {
	return &userTenantRepo{r.BaseRepository.WithTx(tx)}
}

// FindSystem resolves the single system tenant. Membership candidates are drawn
// from it, so it is looked up here rather than accepted as a caller-supplied id.
func (r *userTenantRepo) FindSystem() (*user.Tenant, error) {
	var t user.Tenant
	err := r.DB().Where("is_system = ?", true).First(&t).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &t, nil
}

type userRoleRepo struct {
	*database.BaseRepository[user.Role]
}

func newUserRoleRepo(db *gorm.DB) user.RoleRepository {
	return &userRoleRepo{database.NewBaseRepository[user.Role](db, "role_uuid", "role_id")}
}

func (r *userRoleRepo) WithTx(tx *gorm.DB) user.RoleRepository {
	return &userRoleRepo{r.BaseRepository.WithTx(tx)}
}

func (r *userRoleRepo) FindByNameAndTenantID(name string, tenantID int64) (*user.Role, error) {
	var role user.Role
	err := r.DB().Where("name = ? AND tenant_id = ?", name, tenantID).First(&role).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &role, nil
}

func (r *userRoleRepo) FindPaginated(filter user.RoleRepositoryGetFilter) (*user.PaginationResult[user.Role], error) {
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

type userClientRepo struct {
	*database.BaseRepository[user.Client]
}

func newUserClientRepo(db *gorm.DB) user.ClientRepository {
	return &userClientRepo{database.NewBaseRepository[user.Client](db, "client_uuid", "client_id")}
}

func (r *userClientRepo) WithTx(tx *gorm.DB) user.ClientRepository {
	return &userClientRepo{r.BaseRepository.WithTx(tx)}
}

func (r *userClientRepo) FindByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) (*user.Client, error) {
	var c user.Client
	err := r.DB().Where("client_uuid = ? AND tenant_id = ?", clientUUID, tenantID).First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

func (r *userClientRepo) FindDefaultByTenantID(tenantID int64) (*user.Client, error) {
	var c user.Client
	err := r.DB().Where("tenant_id = ? AND is_default = ?", tenantID, true).First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

func (r *userClientRepo) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*user.Client, error) {
	query := r.DB().Model(&user.Client{}).
		Where("clients.identifier = ?", clientID).
		// Same gates as client.clientRepository.FindByClientIDAndIdentityProvider.
		// This adapter backs the federated login path, so leaving them off meant a
		// disabled connection or a deactivated provider still minted tokens — the
		// later reachability checks only bite on subsequent API calls, by which
		// point a full-TTL access token has already been issued.
		Where("clients.status = ? AND clients.deleted_at IS NULL", shared.StatusActive)
	if identityProviderIdentifier != "" {
		query = query.
			Joins("JOIN client_identity_providers ON client_identity_providers.client_id = clients.client_id").
			Joins("JOIN identity_providers ON identity_providers.identity_provider_id = client_identity_providers.identity_provider_id").
			Where("identity_providers.identifier = ?", identityProviderIdentifier).
			Where("identity_providers.status = ? AND identity_providers.deleted_at IS NULL", shared.StatusActive).
			Where("client_identity_providers.enabled = TRUE AND client_identity_providers.deleted_at IS NULL")
	}
	var c user.Client
	err := query.First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

func (r *userClientRepo) FindByIdentifier(identifier string) (*user.Client, error) {
	var c user.Client
	err := r.DB().Where("identifier = ?", identifier).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *userClientRepo) FindByIDs(ids []int64) ([]user.Client, error) {
	var clients []user.Client
	err := r.DB().Model(&user.Client{}).Where("client_id IN ?", ids).Find(&clients).Error
	return clients, err
}

type userIDPRepo struct {
	*database.BaseRepository[user.IdentityProvider]
}

func newUserIDPRepo(db *gorm.DB) user.IdentityProviderRepository {
	return &userIDPRepo{database.NewBaseRepository[user.IdentityProvider](db, "identity_provider_uuid", "identity_provider_id")}
}

func (r *userIDPRepo) WithTx(tx *gorm.DB) user.IdentityProviderRepository {
	return &userIDPRepo{r.BaseRepository.WithTx(tx)}
}

func (r *userIDPRepo) FindByIdentifier(identifier string) (*user.IdentityProvider, error) {
	var provider user.IdentityProvider
	err := r.DB().Where("identifier = ?", identifier).First(&provider).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &provider, nil
}

func (r *userIDPRepo) FindDefaultByTenantID(tenantID int64) (*user.IdentityProvider, error) {
	var provider user.IdentityProvider
	err := r.DB().Where("tenant_id = ? AND is_default = ?", tenantID, true).First(&provider).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &provider, nil
}

type userBackupCodeRepo struct {
	*database.BaseRepository[user.UserMFABackupCode]
}

func newUserMFABackupCodeRepo(db *gorm.DB) user.UserMFABackupCodeRepository {
	return &userBackupCodeRepo{database.NewBaseRepository[user.UserMFABackupCode](db, "backup_code_uuid", "backup_code_id")}
}

func (r *userBackupCodeRepo) WithTx(tx *gorm.DB) user.UserMFABackupCodeRepository {
	return &userBackupCodeRepo{r.BaseRepository.WithTx(tx)}
}

func (r *userBackupCodeRepo) CreateBulk(codes []*user.UserMFABackupCode) error {
	return r.DB().Create(&codes).Error
}

func (r *userBackupCodeRepo) FindUnusedByUserID(userID int64) ([]user.UserMFABackupCode, error) {
	var codes []user.UserMFABackupCode
	err := r.DB().Where("user_id = ? AND used = ?", userID, false).Find(&codes).Error
	return codes, err
}

func (r *userBackupCodeRepo) FindByUserIDAndCodeHash(userID int64, codeHash string) (*user.UserMFABackupCode, error) {
	var code user.UserMFABackupCode
	err := r.DB().Where("user_id = ? AND code_hash = ?", userID, codeHash).First(&code).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &code, nil
}

func (r *userBackupCodeRepo) MarkUsed(id int64) error {
	return r.DB().Model(&user.UserMFABackupCode{}).Where("backup_code_id = ?", id).Updates(map[string]any{"used": true}).Error
}

func (r *userBackupCodeRepo) DeleteAllByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&user.UserMFABackupCode{}).Error
}
