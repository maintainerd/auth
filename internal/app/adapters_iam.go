package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
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

// EffectivePermissionNames resolves what the user can actually do in a tenant
// right now. Every hop is filtered the same way the request auth context is
// filtered, so the escalation guard cannot be satisfied by a role or permission
// that has been deleted or deactivated.
func (r *iamUserRepo) EffectivePermissionNames(userID, tenantID int64) ([]string, error) {
	var names []string
	err := r.DB().
		Table("user_roles").
		Joins("JOIN roles ON roles.role_id = user_roles.role_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = roles.role_id").
		Joins("JOIN permissions ON permissions.permission_id = role_permissions.permission_id").
		Where("user_roles.user_id = ?", userID).
		Where("roles.tenant_id = ? AND roles.deleted_at IS NULL AND roles.status = ?", tenantID, shared.StatusActive).
		Where("permissions.deleted_at IS NULL AND permissions.status = ?", shared.StatusActive).
		Distinct().
		Pluck("permissions.name", &names).Error
	return names, err
}
