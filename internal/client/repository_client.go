package client

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

type ClientRepositoryGetFilter struct {
	TenantID           int64
	Name               *string
	DisplayName        *string
	ClientType         []string
	Status             []string
	IsDefault          *bool
	IsSystem           *bool
	IdentityProviderID *int64
	Page               int
	Limit              int
	SortBy             string
	SortOrder          string
}

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) (*Client, error)
	FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*Client, error)
	FindByNameAndTenantID(name string, tenantID int64) (*Client, error)
	FindByClientID(clientID string, tenantID int64) (*Client, error)
	FindByIdentifier(identifier string) (*Client, error)
	FindAllByTenantID(tenantID int64) ([]Client, error)
	FindSystem() (*Client, error)
	FindSystemByTenantIdentifier(tenantIdentifier string) (*Client, error)
	FindDefaultByTenantID(tenantID int64) (*Client, error)
	FindPaginated(filter ClientRepositoryGetFilter) (*PaginationResult[Client], error)
	SetStatusByUUID(clientUUID uuid.UUID, tenantID int64, status string) error
	FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error)
	DeleteByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) error
}

type clientRepository struct {
	*BaseRepository[Client]
}

func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{
		BaseRepository: database.NewBaseRepository[Client](db, "client_uuid", "client_id"),
	}
}

func (r *clientRepository) WithTx(tx *gorm.DB) ClientRepository {
	return &clientRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientRepository) FindByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().
		Preload("IdentityProvider").
		Preload("ClientURIs").
		Where("client_uuid = ? AND tenant_id = ?", clientUUID, tenantID).
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *clientRepository) FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().Where("name = ? AND identity_provider_id = ? AND tenant_id = ?", name, identityProviderID, tenantID).First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

// FindByIdentifier returns the active client with the given globally-unique
// identifier string, preloading its IdentityProvider and tenant chain.
func (r *clientRepository) FindByIdentifier(identifier string) (*Client, error) {
	var client Client
	err := r.DB().
		Where("identifier = ? AND status = ?", identifier, shared.StatusActive).
		Preload("IdentityProvider").
		Preload("IdentityProvider.Tenant").
		First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) FindByClientID(clientID string, tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().Where("client_id = ? AND tenant_id = ?", clientID, tenantID).First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) FindAllByTenantID(tenantID int64) ([]Client, error) {
	var clients []Client
	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("identity_providers.tenant_id = ?", tenantID).
		Find(&clients).Error
	return clients, err
}

// FindSystem returns the active system client belonging to the system tenant
// (is_system = true on both tenant and client). The seeded system client is the
// auth-console SPA used to bootstrap the management surface on port 8080.
func (r *clientRepository) FindSystem() (*Client, error) {
	var client Client
	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Joins("JOIN tenants ON tenants.tenant_id = identity_providers.tenant_id").
		Where("clients.is_system = ? AND clients.status = ?", true, shared.StatusActive).
		Where("identity_providers.status = ?", shared.StatusActive).
		Where("tenants.is_system = ?", true).
		Preload("IdentityProvider").
		Preload("IdentityProvider.Tenant").
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// FindSystemByTenantIdentifier returns the active is_system client belonging
// to the tenant identified by its identifier string. This is used when the
// API consumer provides a tenant_id instead of a client_id.
func (r *clientRepository) FindSystemByTenantIdentifier(tenantIdentifier string) (*Client, error) {
	var client Client
	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Joins("JOIN tenants ON tenants.tenant_id = identity_providers.tenant_id").
		Where("clients.is_system = ? AND clients.status = ?", true, shared.StatusActive).
		Where("identity_providers.status = ?", shared.StatusActive).
		Where("tenants.identifier = ?", tenantIdentifier).
		Preload("IdentityProvider").
		Preload("IdentityProvider.Tenant").
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// FindByNameAndTenantID returns the client with the given name within the
// tenant, regardless of which identity provider it is attached to.
func (r *clientRepository) FindByNameAndTenantID(name string, tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().
		Where("name = ? AND tenant_id = ?", name, tenantID).
		Preload("IdentityProvider").
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) FindDefaultByTenantID(tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("identity_providers.tenant_id = ? AND clients.is_default = true AND clients.status = ?", tenantID, shared.StatusActive).
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *clientRepository) FindPaginated(filter ClientRepositoryGetFilter) (*PaginationResult[Client], error) {
	query := r.DB().Model(&Client{}).Where("tenant_id = ?", filter.TenantID)

	// Filters with LIKE
	query = database.ApplyILike(query, "name", filter.Name)
	query = database.ApplyILike(query, "display_name", filter.DisplayName)

	// Filters with exact match
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}
	if len(filter.ClientType) > 0 {
		query = query.Where("client_type IN ?", filter.ClientType)
	}
	if filter.IdentityProviderID != nil {
		query = query.Where("identity_provider_id = ?", *filter.IdentityProviderID)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC")).
		Preload("IdentityProvider").
		Preload("ClientURIs")

	return database.PaginateQuery[Client](query, filter.Page, filter.Limit)
}

func (r *clientRepository) SetStatusByUUID(clientUUID uuid.UUID, tenantID int64, status string) error {
	return r.DB().Model(&Client{}).
		Where("client_uuid = ? AND tenant_id = ?", clientUUID, tenantID).
		Update("status", status).Error
}

func (r *clientRepository) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error) {
	var client Client

	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("clients.client_id = ? AND identity_providers.identifier = ?", clientID, identityProviderIdentifier).
		Where("clients.status = ? AND identity_providers.status = ?", shared.StatusActive, shared.StatusActive).
		Preload("IdentityProvider.Tenant").
		Preload("IdentityProvider").
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) DeleteByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) error {
	result := r.DB().Where("client_uuid = ? AND tenant_id = ?", clientUUID, tenantID).Delete(&Client{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
