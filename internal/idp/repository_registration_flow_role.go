package idp

import (
	"errors"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type RegistrationFlowRoleRepository interface {
	BaseRepositoryMethods[RegistrationFlowRole]
	WithTx(tx *gorm.DB) RegistrationFlowRoleRepository
	FindByRegistrationFlowID(registrationFlowID int64) ([]RegistrationFlowRole, error)
	FindByRegistrationFlowIDPaginated(registrationFlowID int64, page, limit int) (*PaginationResult[RegistrationFlowRole], error)
	DeleteByRegistrationFlowIDAndRoleID(registrationFlowID, roleID int64) error
	DeleteByRegistrationFlowID(registrationFlowID int64) error
	FindByRegistrationFlowIDAndRoleID(registrationFlowID, roleID int64) (*RegistrationFlowRole, error)
}

type registrationFlowRoleRepository struct {
	*BaseRepository[RegistrationFlowRole]
}

func NewRegistrationFlowRoleRepository(db *gorm.DB) RegistrationFlowRoleRepository {
	return &registrationFlowRoleRepository{
		BaseRepository: database.NewBaseRepository[RegistrationFlowRole](db, "registration_flow_role_uuid", "registration_flow_role_id"),
	}
}

func (r *registrationFlowRoleRepository) WithTx(tx *gorm.DB) RegistrationFlowRoleRepository {
	return &registrationFlowRoleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *registrationFlowRoleRepository) FindByRegistrationFlowID(registrationFlowID int64) ([]RegistrationFlowRole, error) {
	var registrationFlowRoles []RegistrationFlowRole
	err := r.DB().Where("registration_flow_id = ?", registrationFlowID).Preload("Role").Find(&registrationFlowRoles).Error
	if err != nil {
		return nil, err
	}
	return registrationFlowRoles, nil
}

func (r *registrationFlowRoleRepository) FindByRegistrationFlowIDPaginated(registrationFlowID int64, page, limit int) (*PaginationResult[RegistrationFlowRole], error) {
	query := r.DB().
		Model(&RegistrationFlowRole{}).
		Where("registration_flow_id = ?", registrationFlowID).
		Order("registration_flow_role_id ASC").
		Preload("Role")

	return database.PaginateQuery[RegistrationFlowRole](query, page, limit)
}

func (r *registrationFlowRoleRepository) DeleteByRegistrationFlowIDAndRoleID(registrationFlowID, roleID int64) error {
	return r.DB().Where("registration_flow_id = ? AND role_id = ?", registrationFlowID, roleID).Delete(&RegistrationFlowRole{}).Error
}

// DeleteByRegistrationFlowID clears a flow's entire role membership. Needed on
// flow delete: registration_flows is soft-deleted, and a soft delete does NOT
// fire the ON DELETE CASCADE on registration_flow_roles, so the children would
// otherwise outlive the parent.
func (r *registrationFlowRoleRepository) DeleteByRegistrationFlowID(registrationFlowID int64) error {
	return r.DB().Where("registration_flow_id = ?", registrationFlowID).Delete(&RegistrationFlowRole{}).Error
}

func (r *registrationFlowRoleRepository) FindByRegistrationFlowIDAndRoleID(registrationFlowID, roleID int64) (*RegistrationFlowRole, error) {
	var registrationFlowRole RegistrationFlowRole
	err := r.DB().Where("registration_flow_id = ? AND role_id = ?", registrationFlowID, roleID).First(&registrationFlowRole).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &registrationFlowRole, nil
}
