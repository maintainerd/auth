package idp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// idpSafeColumns lists every identity_providers column EXCEPT the encrypted
// secret columns. Admin read/list paths select these explicitly so the secret
// never even loads into memory — the secret is write-only and only the
// federation/broker flows (which fetch the full row) ever decrypt it.
var idpSafeColumns = []string{
	"identity_provider_id",
	"identity_provider_uuid",
	"tenant_id",
	"name",
	"display_name",
	"provider",
	"provider_type",
	"identifier",
	"issuer",
	"provider_client_id",
	"allow_jit_provisioning",
	"allow_registration",
	"allow_token_federation",
	"config",
	"status",
	"is_default",
	"is_system",
	"created_by",
	"updated_by",
	"created_at",
	"updated_at",
	"deleted_at",
}

type IdentityProviderRepositoryGetFilter struct {
	Search       *string
	Name         *string
	DisplayName  *string
	Provider     []string
	ProviderType *string
	Identifier   *string
	TenantID     *int64
	Status       []string
	IsDefault    *bool
	IsSystem     *bool
	Page         int
	Limit        int
	SortBy       string
	SortOrder    string
}

type IdentityProviderRepository interface {
	BaseRepositoryMethods[IdentityProvider]
	FindByUUID(uuid any, preloads ...string) (*IdentityProvider, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]IdentityProvider, error)
	FindAll(preloads ...string) ([]IdentityProvider, error)
	FindByID(id any, preloads ...string) (*IdentityProvider, error)
	UpdateByUUID(uuid any, updatedData any) (*IdentityProvider, error)
	UpdateByID(id any, updatedData any) (*IdentityProvider, error)
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[IdentityProvider], error)
	WithTx(tx *gorm.DB) IdentityProviderRepository
	FindByName(name string, tenantID int64) (*IdentityProvider, error)
	FindByIdentifier(identifier string) (*IdentityProvider, error)
	// FindByIssuer returns the active provider that claims the given issuer.
	// Issuer is unique among active providers (uq_identity_providers_issuer), so
	// this is the indexed lookup for federated multi-issuer token validation,
	// replacing config-blob scans. Returns (nil, nil) when no match.
	FindByIssuer(issuer string) (*IdentityProvider, error)
	// FindByUUIDSafe reads a provider by UUID without selecting the encrypted
	// secret columns. Used by admin get/read paths so the secret never loads.
	FindByUUIDSafe(uuidVal any, preloads ...string) (*IdentityProvider, error)
	FindDefaultByTenantID(tenantID int64) (*IdentityProvider, error)
	// FindByTenantAndProvider returns the active provider record matching the
	// tenant and provider slug (e.g. "google", "cognito").
	FindByTenantAndProvider(tenantID int64, provider string) (*IdentityProvider, error)
	// FindAllByTenantID returns every provider configured for a tenant.
	FindAllByTenantID(tenantID int64) ([]IdentityProvider, error)
	FindPaginated(filter IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error)
}

type identityProviderRepository struct {
	*BaseRepository[IdentityProvider]
}

func NewIdentityProviderRepository(db *gorm.DB) IdentityProviderRepository {
	return &identityProviderRepository{
		BaseRepository: database.NewBaseRepository[IdentityProvider](db, "identity_provider_uuid", "identity_provider_id"),
	}
}

func (r *identityProviderRepository) WithTx(tx *gorm.DB) IdentityProviderRepository {
	return &identityProviderRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *identityProviderRepository) FindByName(name string, tenantID int64) (*IdentityProvider, error) {
	var provider IdentityProvider
	err := r.DB().
		Where("name = ? AND tenant_id = ?", name, tenantID).
		First(&provider).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &provider, err
}

func (r *identityProviderRepository) FindByIdentifier(identifier string) (*IdentityProvider, error) {
	var provider IdentityProvider
	err := r.DB().
		Where("identifier = ?", identifier).
		First(&provider).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &provider, nil
}

func (r *identityProviderRepository) FindByIssuer(issuer string) (*IdentityProvider, error) {
	var provider IdentityProvider
	err := r.DB().
		Where("issuer = ? AND status = ? AND deleted_at IS NULL", issuer, shared.StatusActive).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

func (r *identityProviderRepository) FindByUUIDSafe(uuidVal any, preloads ...string) (*IdentityProvider, error) {
	query := r.DB().Select(idpSafeColumns)
	for _, p := range preloads {
		query = query.Preload(p)
	}
	var provider IdentityProvider
	err := query.Where("identity_provider_uuid = ?", uuidVal).First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

func (r *identityProviderRepository) FindDefaultByTenantID(tenantID int64) (*IdentityProvider, error) {
	var provider IdentityProvider
	err := r.DB().
		Where("tenant_id = ? AND is_default = true", tenantID).
		First(&provider).Error
	return &provider, err
}

func (r *identityProviderRepository) FindPaginated(filter IdentityProviderRepositoryGetFilter) (*PaginationResult[IdentityProvider], error) {
	if filter.TenantID == nil || *filter.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := r.DB().Model(&IdentityProvider{}).Select(idpSafeColumns)

	// Search across name, display_name, identifier with OR
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + strings.ToLower(*filter.Search) + "%"
		query = query.Where(
			"LOWER(identity_providers.name) LIKE ? OR LOWER(identity_providers.display_name) LIKE ? OR LOWER(identity_providers.identifier) LIKE ?",
			like, like, like,
		)
	} else {
		query = database.ApplyILike(query, "identity_providers.name", filter.Name)
		query = database.ApplyILike(query, "identity_providers.display_name", filter.DisplayName)
	}

	// Filters with exact match
	if len(filter.Provider) > 0 {
		query = query.Where("provider IN ?", filter.Provider)
	}
	if filter.ProviderType != nil {
		query = query.Where("provider_type = ?", *filter.ProviderType)
	}
	if filter.Identifier != nil {
		query = query.Where("identifier = ?", *filter.Identifier)
	}
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}

	// Sorting — protected against SQL injection via allowlist. EmailDomains is
	// intentionally NOT preloaded on the list path (it is returned on get/detail);
	// keeping list lean avoids an N-row child fan-out query.
	query = query.Order(database.SanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC")).Preload("Tenant")

	return database.PaginateQuery[IdentityProvider](query, filter.Page, filter.Limit)
}

func (r *identityProviderRepository) FindByTenantAndProvider(tenantID int64, provider string) (*IdentityProvider, error) {
	var idp IdentityProvider
	err := r.DB().
		Where("tenant_id = ? AND provider = ? AND deleted_at IS NULL", tenantID, provider).
		First(&idp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &idp, nil
}

func (r *identityProviderRepository) FindAllByTenantID(tenantID int64) ([]IdentityProvider, error) {
	var idps []IdentityProvider
	err := r.DB().
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Find(&idps).Error
	return idps, err
}
