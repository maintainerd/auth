package repositories

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/models"
	"gorm.io/gorm"
)

type RoleRepository interface {
	Create(role *models.Role) error
	FindAll() ([]models.Role, error)
	FindByUUID(roleUUID uuid.UUID) (*models.Role, error)
	UpdateByUUID(roleUUID uuid.UUID, updatedRole *models.Role) error
	DeleteByUUID(roleUUID uuid.UUID) error
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db}
}

func (r *roleRepository) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

func (r *roleRepository) FindAll() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Find(&roles).Error
	return roles, err
}

func (r *roleRepository) FindByUUID(roleUUID uuid.UUID) (*models.Role, error) {
	var role models.Role
	err := r.db.Where("role_uuid = ?", roleUUID).First(&role).Error
	return &role, err
}

func (r *roleRepository) UpdateByUUID(roleUUID uuid.UUID, updatedRole *models.Role) error {
	// Ensure we target the correct row by UUID
	return r.db.Model(&models.Role{}).
		Where("role_uuid = ?", roleUUID).
		Updates(updatedRole).Error
}

func (r *roleRepository) DeleteByUUID(roleUUID uuid.UUID) error {
	return r.db.Where("role_uuid = ?", roleUUID).Delete(&models.Role{}).Error
}
