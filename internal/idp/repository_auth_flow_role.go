package idp

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type AuthFlowRoleRepository interface {
	BaseRepositoryMethods[AuthFlowRole]
	WithTx(tx *gorm.DB) AuthFlowRoleRepository
	FindByAuthFlowID(authFlowID int64) ([]AuthFlowRole, error)
	FindByAuthFlowIDPaginated(authFlowID int64, page, limit int) ([]AuthFlowRole, int64, error)
	DeleteByAuthFlowIDAndRoleID(authFlowID, roleID int64) error
	FindByAuthFlowIDAndRoleID(authFlowID, roleID int64) (*AuthFlowRole, error)
}

type authFlowRoleRepository struct {
	*BaseRepository[AuthFlowRole]
}

func NewAuthFlowRoleRepository(db *gorm.DB) AuthFlowRoleRepository {
	return &authFlowRoleRepository{
		BaseRepository: database.NewBaseRepository[AuthFlowRole](db, "auth_flow_role_uuid", "auth_flow_role_id"),
	}
}

func (r *authFlowRoleRepository) WithTx(tx *gorm.DB) AuthFlowRoleRepository {
	return &authFlowRoleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *authFlowRoleRepository) FindByAuthFlowID(authFlowID int64) ([]AuthFlowRole, error) {
	var authFlowRoles []AuthFlowRole
	err := r.DB().Where("auth_flow_id = ?", authFlowID).Preload("Role").Find(&authFlowRoles).Error
	if err != nil {
		return nil, err
	}
	return authFlowRoles, nil
}

func (r *authFlowRoleRepository) FindByAuthFlowIDPaginated(authFlowID int64, page, limit int) ([]AuthFlowRole, int64, error) {
	var authFlowRoles []AuthFlowRole
	var total int64

	query := r.DB().Where("auth_flow_id = ?", authFlowID)

	// Get total count
	if err := query.Model(&AuthFlowRole{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated data
	offset := (page - 1) * limit
	err := query.Preload("Role").Offset(offset).Limit(limit).Find(&authFlowRoles).Error
	if err != nil {
		return nil, 0, err
	}

	return authFlowRoles, total, nil
}

func (r *authFlowRoleRepository) DeleteByAuthFlowIDAndRoleID(authFlowID, roleID int64) error {
	return r.DB().Where("auth_flow_id = ? AND role_id = ?", authFlowID, roleID).Delete(&AuthFlowRole{}).Error
}

func (r *authFlowRoleRepository) FindByAuthFlowIDAndRoleID(authFlowID, roleID int64) (*AuthFlowRole, error) {
	var authFlowRole AuthFlowRole
	err := r.DB().Where("auth_flow_id = ? AND role_id = ?", authFlowID, roleID).First(&authFlowRole).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &authFlowRole, nil
}
