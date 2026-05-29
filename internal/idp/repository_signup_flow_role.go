package idp

import (
	"errors"

	"gorm.io/gorm"
)

type SignupFlowRoleRepository interface {
	BaseRepositoryMethods[SignupFlowRole]
	WithTx(tx *gorm.DB) SignupFlowRoleRepository
	FindBySignupFlowID(signupFlowID int64) ([]SignupFlowRole, error)
	FindBySignupFlowIDPaginated(signupFlowID int64, page, limit int) ([]SignupFlowRole, int64, error)
	DeleteBySignupFlowIDAndRoleID(signupFlowID, roleID int64) error
	FindBySignupFlowIDAndRoleID(signupFlowID, roleID int64) (*SignupFlowRole, error)
}

type signupFlowRoleRepository struct {
	*BaseRepository[SignupFlowRole]
}

func NewSignupFlowRoleRepository(db *gorm.DB) SignupFlowRoleRepository {
	return &signupFlowRoleRepository{
		BaseRepository: NewBaseRepository[SignupFlowRole](db, "signup_flow_role_uuid", "signup_flow_role_id"),
	}
}

func (r *signupFlowRoleRepository) WithTx(tx *gorm.DB) SignupFlowRoleRepository {
	return &signupFlowRoleRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *signupFlowRoleRepository) FindBySignupFlowID(signupFlowID int64) ([]SignupFlowRole, error) {
	var signupFlowRoles []SignupFlowRole
	err := r.DB().Where("signup_flow_id = ?", signupFlowID).Preload("Role").Find(&signupFlowRoles).Error
	if err != nil {
		return nil, err
	}
	return signupFlowRoles, nil
}

func (r *signupFlowRoleRepository) FindBySignupFlowIDPaginated(signupFlowID int64, page, limit int) ([]SignupFlowRole, int64, error) {
	var signupFlowRoles []SignupFlowRole
	var total int64

	query := r.DB().Where("signup_flow_id = ?", signupFlowID)

	// Get total count
	if err := query.Model(&SignupFlowRole{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated data
	offset := (page - 1) * limit
	err := query.Preload("Role").Offset(offset).Limit(limit).Find(&signupFlowRoles).Error
	if err != nil {
		return nil, 0, err
	}

	return signupFlowRoles, total, nil
}

func (r *signupFlowRoleRepository) DeleteBySignupFlowIDAndRoleID(signupFlowID, roleID int64) error {
	return r.DB().Where("signup_flow_id = ? AND role_id = ?", signupFlowID, roleID).Delete(&SignupFlowRole{}).Error
}

func (r *signupFlowRoleRepository) FindBySignupFlowIDAndRoleID(signupFlowID, roleID int64) (*SignupFlowRole, error) {
	var signupFlowRole SignupFlowRole
	err := r.DB().Where("signup_flow_id = ? AND role_id = ?", signupFlowID, roleID).First(&signupFlowRole).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlowRole, nil
}
