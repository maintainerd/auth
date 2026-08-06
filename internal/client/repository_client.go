package client

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// clientSortColumns is this table's own sort allowlist. The global set in
// platform/database is a union across every table, so it contains columns
// `clients` does not have — email, username, first_name, event_type, type and
// others. Ordering by one of those reached Postgres as an undefined column and
// surfaced as a 500 rather than a 400.
var clientSortColumns = map[string]struct{}{
	"created_at": {}, "updated_at": {}, "name": {}, "display_name": {},
	"client_type": {}, "status": {}, "identifier": {}, "domain": {},
	"is_default": {}, "is_system": {}, "tenant_id": {},
}

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
	FindByUUID(uuid any, preloads ...string) (*Client, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]Client, error)
	FindAll(preloads ...string) ([]Client, error)
	FindByID(id any, preloads ...string) (*Client, error)
	UpdateByUUID(uuid any, updatedData any) (*Client, error)
	UpdateByID(id any, updatedData any) (*Client, error)
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[Client], error)
	WithTx(tx *gorm.DB) ClientRepository
	FindByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) (*Client, error)
	FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*Client, error)
	FindByNameAndTenantID(name string, tenantID int64) (*Client, error)
	FindByClientID(clientID string, tenantID int64) (*Client, error)
	FindByIdentifier(identifier string) (*Client, error)
	ExistsByIdentifier(identifier string) (bool, error)
	FindAllByTenantID(tenantID int64) ([]Client, error)
	FindSystem() (*Client, error)
	FindSystemByTenantIdentifier(tenantIdentifier string) (*Client, error)
	FindSystemByTenantIdentifierAndName(tenantIdentifier, name string) (*Client, error)
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
		Preload("Tenant").
		Preload("Branding").
		Preload("ConnectedProviders.IdentityProvider").
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
	query := r.DB().Where("clients.name = ? AND clients.tenant_id = ?", name, tenantID)
	if identityProviderID > 0 {
		query = query.
			Joins("JOIN client_identity_providers ON client_identity_providers.client_id = clients.client_id").
			Where("client_identity_providers.identity_provider_id = ? AND client_identity_providers.deleted_at IS NULL", identityProviderID)
	}
	err := query.First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

// FindByIdentifier returns the active client with the given globally-unique
// identifier string, preloading its tenant and enabled provider connections.
func (r *clientRepository) FindByIdentifier(identifier string) (*Client, error) {
	var client Client
	err := r.DB().
		Where("identifier = ? AND status = ?", identifier, shared.StatusActive).
		Preload("Tenant").
		Preload("ConnectedProviders", "enabled = ?", true).
		Preload("ConnectedProviders.IdentityProvider").
		First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// ExistsByIdentifier reports whether any client already holds this OAuth
// client_id. Deliberately unscoped and status-agnostic: FindByIdentifier filters
// to active rows, which is the wrong question when allocating a new identifier —
// an inactive or soft-deleted client still owns the value, and reusing it would
// re-point anything still holding the old client_id at a different client.
func (r *clientRepository) ExistsByIdentifier(identifier string) (bool, error) {
	var count int64
	err := r.DB().Unscoped().
		Model(&Client{}).
		Where("identifier = ?", identifier).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
		Where("clients.tenant_id = ?", tenantID).
		Find(&clients).Error
	return clients, err
}

// FindSystem returns the active system client belonging to the system tenant
// (is_system = true on both tenant and client). The seeded system client is the
// auth-console SPA used to bootstrap the management surface on port 8080.
func (r *clientRepository) FindSystem() (*Client, error) {
	var client Client
	err := r.DB().
		Joins("JOIN tenants ON tenants.tenant_id = clients.tenant_id").
		Where("clients.is_system = ? AND clients.status = ?", true, shared.StatusActive).
		Where("clients.name = ?", shared.SystemClientNameAuthConsole).
		Where("tenants.is_system = ?", true).
		Preload("Tenant").
		Preload("ConnectedProviders", "enabled = ?", true).
		Preload("ConnectedProviders.IdentityProvider").
		Preload("ConnectedProviders.IdentityProvider.Tenant").
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
// to the tenant identified by its name (the DNS slug). This is used when the
// API consumer provides a tenant_id (tenant name) instead of a client_id.
func (r *clientRepository) FindSystemByTenantIdentifier(tenantIdentifier string) (*Client, error) {
	return r.FindSystemByTenantIdentifierAndName(tenantIdentifier, shared.SystemClientNameAuthConsole)
}

// FindSystemByTenantIdentifierAndName returns the named active is_system client
// belonging to the tenant identified by its name (the DNS slug). The
// tenantIdentifier argument now carries the tenant name.
func (r *clientRepository) FindSystemByTenantIdentifierAndName(tenantIdentifier, name string) (*Client, error) {
	var client Client
	err := r.DB().
		Joins("JOIN tenants ON tenants.tenant_id = clients.tenant_id").
		Where("clients.is_system = ? AND clients.status = ?", true, shared.StatusActive).
		Where("clients.name = ?", name).
		Where("tenants.name = ?", tenantIdentifier).
		Preload("Tenant").
		Preload("ConnectedProviders", "enabled = ?", true).
		Preload("ConnectedProviders.IdentityProvider").
		Preload("ConnectedProviders.IdentityProvider.Tenant").
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
		Preload("Tenant").
		// Callers create user identities against this client, and an identity
		// names its provider (migration 030). Without these preloads the
		// connection is invisible and every such caller fails closed.
		Preload("ConnectedProviders", "enabled = ?", true).
		Preload("ConnectedProviders.IdentityProvider").
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
		Where("tenant_id = ? AND is_default = true AND status = ?", tenantID, shared.StatusActive).
		Preload("Tenant").
		Preload("ConnectedProviders", "enabled = ?", true).
		Preload("ConnectedProviders.IdentityProvider").
		Preload("ConnectedProviders.IdentityProvider.Tenant").
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
	query := r.DB().Model(&Client{}).Where("clients.tenant_id = ?", filter.TenantID)

	// Filters with LIKE
	query = database.ApplyILike(query, "clients.name", filter.Name)
	query = database.ApplyILike(query, "clients.display_name", filter.DisplayName)

	// Filters with exact match
	if len(filter.Status) > 0 {
		query = query.Where("clients.status IN ?", filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("clients.is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("clients.is_system = ?", *filter.IsSystem)
	}
	if len(filter.ClientType) > 0 {
		query = query.Where("clients.client_type IN ?", filter.ClientType)
	}
	if filter.IdentityProviderID != nil {
		query = query.
			Joins("JOIN client_identity_providers ON client_identity_providers.client_id = clients.client_id").
			Where("client_identity_providers.identity_provider_id = ? AND client_identity_providers.deleted_at IS NULL", *filter.IdentityProviderID)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(database.SanitizeOrderInPrefixed(clientSortColumns, "clients.", filter.SortBy, filter.SortOrder, "clients.created_at DESC")).
		Preload("Tenant").
		Preload("Branding").
		Preload("ConnectedProviders", "enabled = ?", true).
		Preload("ConnectedProviders.IdentityProvider").
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

	query := r.DB().
		Where("clients.identifier = ?", clientID).
		Where("clients.status = ?", shared.StatusActive).
		Preload("Tenant").
		Preload("ConnectedProviders", "enabled = ?", true).
		Preload("ConnectedProviders.IdentityProvider").
		Preload("ConnectedProviders.IdentityProvider.Tenant")
	if identityProviderIdentifier != "" {
		query = query.
			Joins("JOIN client_identity_providers ON client_identity_providers.client_id = clients.client_id").
			Joins("JOIN identity_providers ON identity_providers.identity_provider_id = client_identity_providers.identity_provider_id").
			Where("identity_providers.identifier = ?", identityProviderIdentifier).
			Where("identity_providers.status = ? AND client_identity_providers.enabled = ? AND client_identity_providers.deleted_at IS NULL", shared.StatusActive, true)
	}

	err := query.First(&client).Error

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
