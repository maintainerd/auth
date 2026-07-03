package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type iamTenantServiceRepo struct {
	*database.BaseRepository[iam.TenantService]
}

func newIAMTenantServiceRepo(db *gorm.DB) iam.TenantServiceRepository {
	return &iamTenantServiceRepo{database.NewBaseRepository[iam.TenantService](db, "", "tenant_service_id")}
}

func (r *iamTenantServiceRepo) WithTx(tx *gorm.DB) iam.TenantServiceRepository {
	return &iamTenantServiceRepo{r.BaseRepository.WithTx(tx)}
}

func (r *iamTenantServiceRepo) FindPaginated(filter iam.TenantServiceRepositoryGetFilter) (*iam.PaginationResult[iam.TenantService], error) {
	conditions := map[string]any{}
	if filter.TenantID != nil {
		conditions["tenant_id"] = *filter.TenantID
	}
	if filter.ServiceID != nil {
		conditions["service_id"] = *filter.ServiceID
	}
	return r.Paginate(conditions, filter.Page, filter.Limit)
}

func (r *iamTenantServiceRepo) FindByTenantAndService(tenantID int64, serviceID int64) (*iam.TenantService, error) {
	var ts iam.TenantService
	err := r.DB().Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).First(&ts).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &ts, nil
}

func (r *iamTenantServiceRepo) DeleteByTenantAndService(tenantID int64, serviceID int64) error {
	return r.DB().Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).Delete(&iam.TenantService{}).Error
}

type iamClientRepo struct {
	*database.BaseRepository[iam.Client]
}

func newIAMClientRepo(db *gorm.DB) iam.ClientRepository {
	return &iamClientRepo{database.NewBaseRepository[iam.Client](db, "client_uuid", "client_id")}
}

func (r *iamClientRepo) WithTx(tx *gorm.DB) iam.ClientRepository {
	return &iamClientRepo{r.BaseRepository.WithTx(tx)}
}

type iamTenantRepo struct {
	*database.BaseRepository[iam.Tenant]
}

func newIAMTenantRepo(db *gorm.DB) iam.TenantRepository {
	return &iamTenantRepo{database.NewBaseRepository[iam.Tenant](db, "tenant_uuid", "tenant_id")}
}

func (r *iamTenantRepo) WithTx(tx *gorm.DB) iam.TenantRepository {
	return &iamTenantRepo{r.BaseRepository.WithTx(tx)}
}

type iamUserRepo struct {
	*database.BaseRepository[iam.User]
}

func newIAMUserRepo(db *gorm.DB) iam.UserRepository {
	return &iamUserRepo{database.NewBaseRepository[iam.User](db, "user_uuid", "user_id")}
}

func (r *iamUserRepo) WithTx(tx *gorm.DB) iam.UserRepository {
	return &iamUserRepo{r.BaseRepository.WithTx(tx)}
}
