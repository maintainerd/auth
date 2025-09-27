package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthClientRepositoryGetFilter struct {
	Name               *string
	DisplayName        *string
	ClientType         *string
	IsActive           *bool
	IsDefault          *bool
	IdentityProviderID *int64
	Page               int
	Limit              int
	SortBy             string
	SortOrder          string
}

type AuthClientRepository interface {
	BaseRepositoryMethods[model.AuthClient]
	WithTx(tx *gorm.DB) AuthClientRepository
	FindByNameAndIdentityProvider(name string, identityProviderID int64) (*model.AuthClient, error)
	FindByClientID(clientID string) (*model.AuthClient, error)
	FindAllByAuthContainerID(authContainerID int64) ([]model.AuthClient, error)
	FindDefault() (*model.AuthClient, error)
	FindDefaultByAuthContainerID(authContainerID int64) (*model.AuthClient, error)
	FindPaginated(filter AuthClientRepositoryGetFilter) (*PaginationResult[model.AuthClient], error)
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

func (r *authClientRepository) WithTx(tx *gorm.DB) AuthClientRepository {
	return &authClientRepository{
		BaseRepository: NewBaseRepository[model.AuthClient](tx, "auth_client_uuid", "auth_client_id"),
		db:             tx,
	}
}

func (r *authClientRepository) FindByNameAndIdentityProvider(name string, identityProviderID int64) (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.Where("name = ? AND identity_provider_id = ?", name, identityProviderID).First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
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

func (r *authClientRepository) FindDefault() (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = auth_clients.identity_provider_id").
		Where("auth_clients.is_default = ? AND auth_clients.is_active = ?", true, true).
		Where("auth_clients.client_type = ?", "traditional").
		Where("identity_providers.is_active = ?", true).
		Preload("IdentityProvider").
		Preload("IdentityProvider.AuthContainer").
		First(&client).Error

	return &client, err
}

func (r *authClientRepository) FindDefaultByAuthContainerID(authContainerID int64) (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = auth_clients.identity_provider_id").
		Where("identity_providers.auth_container_id = ? AND auth_clients.is_default = true AND auth_clients.is_active = true", authContainerID).
		First(&client).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *authClientRepository) FindPaginated(filter AuthClientRepositoryGetFilter) (*PaginationResult[model.AuthClient], error) {
	query := r.db.Model(&model.AuthClient{})

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}

	// Filters with exact match
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.ClientType != nil {
		query = query.Where("client_type = ?", *filter.ClientType)
	}
	if filter.IdentityProviderID != nil {
		query = query.Where("identity_provider_id = ?", *filter.IdentityProviderID)
	}

	// Sorting
	orderBy := filter.SortBy + " " + filter.SortOrder
	query = query.Order(orderBy)

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var clients []model.AuthClient
	if err := query.
		Preload("IdentityProvider").
		Preload("AuthClientRedirectURIs").
		Limit(filter.Limit).
		Offset(offset).
		Find(&clients).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.AuthClient]{
		Data:       clients,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
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
		Preload("IdentityProvider").
		First(&client).Error

	return &client, err
}
