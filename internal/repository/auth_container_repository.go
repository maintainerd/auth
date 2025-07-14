package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthContainerRepository interface {
	BaseRepositoryMethods[model.AuthContainer]
	FindByName(name string) (*model.AuthContainer, error)
	FindByIdentifier(identifier string) (*model.AuthContainer, error)
	FindAllByOrganizationID(organizationID int64) ([]model.AuthContainer, error)
	FindDefaultByOrganizationID(organizationID int64) (*model.AuthContainer, error)
	SetActiveStatusByUUID(authContainerUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(authContainerUUID uuid.UUID, isDefault bool) error
}

type authContainerRepository struct {
	*BaseRepository[model.AuthContainer]
	db *gorm.DB
}

func NewAuthContainerRepository(db *gorm.DB) AuthContainerRepository {
	return &authContainerRepository{
		BaseRepository: NewBaseRepository[model.AuthContainer](db, "auth_container_uuid", "auth_container_id"),
		db:             db,
	}
}

func (r *authContainerRepository) FindByName(name string) (*model.AuthContainer, error) {
	var container model.AuthContainer
	err := r.db.
		Where("name = ?", name).
		First(&container).Error
	return &container, err
}

func (r *authContainerRepository) FindByIdentifier(identifier string) (*model.AuthContainer, error) {
	var container model.AuthContainer
	err := r.db.
		Where("identifier = ?", identifier).
		First(&container).Error
	return &container, err
}

func (r *authContainerRepository) FindAllByOrganizationID(organizationID int64) ([]model.AuthContainer, error) {
	var containers []model.AuthContainer
	err := r.db.
		Where("organization_id = ?", organizationID).
		Find(&containers).Error
	return containers, err
}

func (r *authContainerRepository) FindDefaultByOrganizationID(organizationID int64) (*model.AuthContainer, error) {
	var container model.AuthContainer
	err := r.db.
		Where("organization_id = ? AND is_default = true", organizationID).
		First(&container).Error
	return &container, err
}

func (r *authContainerRepository) SetActiveStatusByUUID(authContainerUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.AuthContainer{}).
		Where("auth_container_uuid = ?", authContainerUUID).
		Update("is_active", isActive).Error
}

func (r *authContainerRepository) SetDefaultStatusByUUID(authContainerUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.AuthContainer{}).
		Where("auth_container_uuid = ?", authContainerUUID).
		Update("is_default", isDefault).Error
}
