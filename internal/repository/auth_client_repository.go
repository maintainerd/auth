package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthClientRepository interface {
	BaseRepositoryMethods[model.AuthClient]
	FindByClientID(clientID string) (*model.AuthClient, error)
	FindAllByAuthContainerID(authContainerID int64) ([]model.AuthClient, error)
	FindDefaultByAuthContainerID(authContainerID int64) (*model.AuthClient, error)
	SetActiveStatusByUUID(authClientUUID uuid.UUID, isActive bool) error
	FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*model.AuthClient, error)
}

type authClientRepository struct {
	*BaseRepository[model.AuthClient]
	db *gorm.DB
}

func NewAuthClientRepository(db *gorm.DB) AuthClientRepository {
	return &authClientRepository{
		BaseRepository: NewBaseRepository[model.AuthClient](db, "auth_client_uuid", "auth_client_id"),
		db:             db,
	}
}

func (r *authClientRepository) FindByClientID(clientID string) (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.Where("client_id = ?", clientID).First(&client).Error
	return &client, err
}

func (r *authClientRepository) FindAllByAuthContainerID(authContainerID int64) ([]model.AuthClient, error) {
	var clients []model.AuthClient
	err := r.db.
		Where("auth_container_id = ?", authContainerID).
		Find(&clients).Error
	return clients, err
}

func (r *authClientRepository) FindDefaultByAuthContainerID(authContainerID int64) (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.
		Where("auth_container_id = ? AND is_default = true", authContainerID).
		First(&client).Error
	return &client, err
}

func (r *authClientRepository) SetActiveStatusByUUID(authClientUUID uuid.UUID, isActive bool) error {
	return r.db.Model(&model.AuthClient{}).
		Where("auth_client_uuid = ?", authClientUUID).
		Update("is_active", isActive).Error
}

func (r *authClientRepository) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*model.AuthClient, error) {
	var client model.AuthClient

	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.auth_container_id = auth_clients.auth_container_id").
		Where("auth_clients.client_id = ? AND identity_providers.identifier = ?", clientID, identityProviderIdentifier).
		Where("auth_clients.is_active = true AND identity_providers.is_active = true").
		Preload("AuthContainer").
		First(&client).Error

	return &client, err
}
