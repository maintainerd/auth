package client

import (
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type ClientRoleRepository interface {
	BaseRepositoryMethods[ClientRole]
	WithTx(tx *gorm.DB) ClientRoleRepository
	AssignRole(clientID int64, roleID int64, createdBy *int64) (*ClientRole, error)
	RemoveRole(clientID int64, roleID int64) error
	ListRoles(clientID int64) ([]ClientRole, error)
	ResolvePermissions(clientID int64) ([]int64, error)
}

type clientRoleRepository struct {
	*BaseRepository[ClientRole]
}

func NewClientRoleRepository(db *gorm.DB) ClientRoleRepository {
	return &clientRoleRepository{
		BaseRepository: database.NewBaseRepository[ClientRole](db, "client_role_uuid", "client_role_id"),
	}
}

func (r *clientRoleRepository) WithTx(tx *gorm.DB) ClientRoleRepository {
	return &clientRoleRepository{
		BaseRepository: database.NewBaseRepository[ClientRole](tx, "client_role_uuid", "client_role_id"),
	}
}

func (r *clientRoleRepository) AssignRole(clientID int64, roleID int64, createdBy *int64) (*ClientRole, error) {
	cr := &ClientRole{
		ClientID:  clientID,
		RoleID:    roleID,
		CreatedBy: createdBy,
	}
	if err := r.DB().Create(cr).Error; err != nil {
		return nil, err
	}
	return cr, nil
}

func (r *clientRoleRepository) RemoveRole(clientID int64, roleID int64) error {
	return r.DB().
		Where("client_id = ? AND role_id = ?", clientID, roleID).
		Delete(&ClientRole{}).Error
}

func (r *clientRoleRepository) ListRoles(clientID int64) ([]ClientRole, error) {
	var roles []ClientRole
	err := r.DB().
		Where("client_id = ?", clientID).
		Find(&roles).Error
	return roles, err
}

func (r *clientRoleRepository) ResolvePermissions(clientID int64) ([]int64, error) {
	var permIDs []int64
	err := r.DB().
		Table("client_roles").
		Select("DISTINCT role_permissions.permission_id").
		Joins("JOIN role_permissions ON client_roles.role_id = role_permissions.role_id").
		Where("client_roles.client_id = ?", clientID).
		Pluck("role_permissions.permission_id", &permIDs).Error
	return permIDs, err
}
