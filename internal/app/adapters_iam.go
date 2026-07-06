package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

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
