package app

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/user"
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
	query := r.DB().Model(&user.Client{}).Where("clients.identifier = ?", clientID)
	if identityProviderIdentifier != "" {
		query = query.Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
			Where("identity_providers.identifier = ?", identityProviderIdentifier)
	}
	var c user.Client
	err := query.First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
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

type userBackupCodeRepo struct {
	*database.BaseRepository[user.UserBackupCode]
}

func newUserBackupCodeRepo(db *gorm.DB) user.UserBackupCodeRepository {
	return &userBackupCodeRepo{database.NewBaseRepository[user.UserBackupCode](db, "backup_code_uuid", "backup_code_id")}
}

func (r *userBackupCodeRepo) WithTx(tx *gorm.DB) user.UserBackupCodeRepository {
	return &userBackupCodeRepo{r.BaseRepository.WithTx(tx)}
}

func (r *userBackupCodeRepo) CreateBulk(codes []*user.UserBackupCode) error {
	return r.DB().Create(&codes).Error
}

func (r *userBackupCodeRepo) FindUnusedByUserID(userID int64) ([]user.UserBackupCode, error) {
	var codes []user.UserBackupCode
	err := r.DB().Where("user_id = ? AND used = ?", userID, false).Find(&codes).Error
	return codes, err
}

func (r *userBackupCodeRepo) FindByUserIDAndCodeHash(userID int64, codeHash string) (*user.UserBackupCode, error) {
	var code user.UserBackupCode
	err := r.DB().Where("user_id = ? AND code_hash = ?", userID, codeHash).First(&code).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &code, nil
}

func (r *userBackupCodeRepo) MarkUsed(id int64) error {
	return r.DB().Model(&user.UserBackupCode{}).Where("backup_code_id = ?", id).Updates(map[string]any{"used": true}).Error
}

func (r *userBackupCodeRepo) DeleteAllByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&user.UserBackupCode{}).Error
}
