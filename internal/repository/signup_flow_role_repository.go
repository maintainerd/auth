package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type SignupFlowRoleRepository interface {
	BaseRepositoryMethods[model.SignupFlowRole]
	WithTx(tx *gorm.DB) SignupFlowRoleRepository
	FindBySignupFlowID(signupFlowID int64) ([]model.SignupFlowRole, error)
	FindBySignupFlowIDPaginated(signupFlowID int64, page, limit int) ([]model.SignupFlowRole, int64, error)
	DeleteBySignupFlowIDAndRoleID(signupFlowID, roleID int64) error
	FindBySignupFlowIDAndRoleID(signupFlowID, roleID int64) (*model.SignupFlowRole, error)
}

type signupFlowRoleRepository struct {
	*BaseRepository[model.SignupFlowRole]
	db *gorm.DB
}

func NewSignupFlowRoleRepository(db *gorm.DB) SignupFlowRoleRepository {
	return &signupFlowRoleRepository{
		BaseRepository: NewBaseRepository[model.SignupFlowRole](db, "signup_flow_role_uuid", "signup_flow_role_id"),
		db:             db,
	}
}

func (r *signupFlowRoleRepository) WithTx(tx *gorm.DB) SignupFlowRoleRepository {
	return &signupFlowRoleRepository{
		BaseRepository: NewBaseRepository[model.SignupFlowRole](tx, "signup_flow_role_uuid", "signup_flow_role_id"),
		db:             tx,
	}
}

func (r *signupFlowRoleRepository) FindBySignupFlowID(signupFlowID int64) ([]model.SignupFlowRole, error) {
	var signupFlowRoles []model.SignupFlowRole
	err := r.db.Where("signup_flow_id = ?", signupFlowID).Preload("Role").Find(&signupFlowRoles).Error
	if err != nil {
		return nil, err
	}
	return signupFlowRoles, nil
}

func (r *signupFlowRoleRepository) FindBySignupFlowIDPaginated(signupFlowID int64, page, limit int) ([]model.SignupFlowRole, int64, error) {
	var signupFlowRoles []model.SignupFlowRole
	var total int64

	query := r.db.Where("signup_flow_id = ?", signupFlowID)

	// Get total count
	if err := query.Model(&model.SignupFlowRole{}).Count(&total).Error; err != nil {
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
	return r.db.Where("signup_flow_id = ? AND role_id = ?", signupFlowID, roleID).Delete(&model.SignupFlowRole{}).Error
}

func (r *signupFlowRoleRepository) FindBySignupFlowIDAndRoleID(signupFlowID, roleID int64) (*model.SignupFlowRole, error) {
	var signupFlowRole model.SignupFlowRole
	err := r.db.Where("signup_flow_id = ? AND role_id = ?", signupFlowID, roleID).First(&signupFlowRole).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &signupFlowRole, nil
}
