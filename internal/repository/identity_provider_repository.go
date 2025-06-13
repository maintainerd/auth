package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type IdentityProviderRepository interface {
	BaseRepositoryMethods[model.IdentityProvider]
	FindByName(providerName string, authContainerID int64) (*model.IdentityProvider, error)
	FindByIdentifier(identifier string, authContainerID int64) (*model.IdentityProvider, error)
	FindAllByAuthContainerID(authContainerID int64) ([]model.IdentityProvider, error)
	FindDefaultByAuthContainerID(authContainerID int64) (*model.IdentityProvider, error)
	SetActiveStatusByUUID(identityProviderUUID uuid.UUID, isActive bool) error
	SetDefaultStatusByUUID(identityProviderUUID uuid.UUID, isDefault bool) error
}

type identityProviderRepository struct {
	*BaseRepository[model.IdentityProvider]
	db *gorm.DB
}

func NewIdentityProviderRepository(db *gorm.DB) IdentityProviderRepository {
	return &identityProviderRepository{
		BaseRepository: NewBaseRepository[model.IdentityProvider](db, "identity_provider_uuid", "identity_provider_id"),
		db:             db,
	}
}

func (r *identityProviderRepository) FindByName(providerName string, authContainerID int64) (*model.IdentityProvider, error) {
	var provider model.IdentityProvider
	err := r.db.
		Where("provider_name = ? AND auth_container_id = ?", providerName, authContainerID).
		First(&provider).Error
	return &provider, err
}

func (r *identityProviderRepository) FindByIdentifier(identifier string, authContainerID int64) (*model.IdentityProvider, error) {
	var provider model.IdentityProvider
	err := r.db.
		Where("identifier = ? AND auth_container_id = ?", identifier, authContainerID).
		First(&provider).Error
	return &provider, err
}

func (r *identityProviderRepository) FindAllByAuthContainerID(authContainerID int64) ([]model.IdentityProvider, error) {
	var providers []model.IdentityProvider
	err := r.db.
		Where("auth_container_id = ?", authContainerID).
		Find(&providers).Error
	return providers, err
}

func (r *identityProviderRepository) FindDefaultByAuthContainerID(authContainerID int64) (*model.IdentityProvider, error) {
	var provider model.IdentityProvider
	err := r.db.
		Where("auth_container_id = ? AND is_default = true", authContainerID).
		First(&provider).Error
	return &provider, err
}

func (r *identityProviderRepository) SetActiveStatusByUUID(identityProviderUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.IdentityProvider{}).
		Where("identity_provider_uuid = ?", identityProviderUUID).
		Update("is_active", isActive).Error
}

func (r *identityProviderRepository) SetDefaultStatusByUUID(identityProviderUUID uuid.UUID, isDefault bool) error {
	return r.db.Model(&model.IdentityProvider{}).
		Where("identity_provider_uuid = ?", identityProviderUUID).
		Update("is_default", isDefault).Error
}
