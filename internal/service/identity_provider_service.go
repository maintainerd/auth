package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IdentityProviderServiceDataResult struct {
	IdentityProviderUUID uuid.UUID
	Name                 string
	DisplayName          string
	Provider             string
	ProviderType         string
	Identifier           string
	Config               *datatypes.JSON
	Tenant               *TenantServiceDataResult
	Status               string
	IsDefault            bool
	IsSystem             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type IdentityProviderServiceGetFilter struct {
	Name         *string
	DisplayName  *string
	Provider     []string
	ProviderType *string
	Identifier   *string
	TenantUUID   *string
	Status       []string
	IsDefault    *bool
	IsSystem     *bool
	Page         int
	Limit        int
	SortBy       string
	SortOrder    string
}

type IdentityProviderServiceGetResult struct {
	Data       []IdentityProviderServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type IdentityProviderService interface {
	Get(filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error)
	GetByUUID(idpUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	Create(name string, displayName string, provider string, providerType string, config datatypes.JSON, status string, tenantUUID string, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	Update(idpUUID uuid.UUID, name string, displayName string, provider string, providerType string, config datatypes.JSON, status string, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	SetStatusByUUID(idpUUID uuid.UUID, status string, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	DeleteByUUID(idpUUID uuid.UUID, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
}

type identityProviderService struct {
	db         *gorm.DB
	idpRepo    repository.IdentityProviderRepository
	tenantRepo repository.TenantRepository
	userRepo   repository.UserRepository
}

func NewIdentityProviderService(
	db *gorm.DB,
	idpRepo repository.IdentityProviderRepository,
	tenantRepo repository.TenantRepository,
	userRepo repository.UserRepository,
) IdentityProviderService {
	return &identityProviderService{
		db:         db,
		idpRepo:    idpRepo,
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
	}
}

func (s *identityProviderService) Get(filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
	var tenantID *int64

	// Get tenant if uuid exist
	if filter.TenantUUID != nil {
		tenantUUIDParsed, err := uuid.Parse(*filter.TenantUUID)
		if err != nil {
			return nil, errors.New("invalid tenant UUID")
		}
		tenant, err := s.tenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || tenant == nil {
			return nil, errors.New("tenant not found")
		}
		tenantID = &tenant.TenantID
	}

	// Build query filter
	queryFilter := repository.IdentityProviderRepositoryGetFilter{
		Name:         filter.Name,
		DisplayName:  filter.DisplayName,
		Provider:     filter.Provider,
		ProviderType: filter.ProviderType,
		Identifier:   filter.Identifier,
		TenantID:     tenantID,
		Status:       filter.Status,
		IsDefault:    filter.IsDefault,
		IsSystem:     filter.IsSystem,
		Page:         filter.Page,
		Limit:        filter.Limit,
		SortBy:       filter.SortBy,
		SortOrder:    filter.SortOrder,
	}

	result, err := s.idpRepo.FindPaginated(queryFilter)
	if err != nil {
		return nil, err
	}

	idps := make([]IdentityProviderServiceDataResult, len(result.Data))
	for i, idp := range result.Data {
		idps[i] = *toIdpServiceDataResult(&idp)
	}

	return &IdentityProviderServiceGetResult{
		Data:       idps,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *identityProviderService) GetByUUID(idpUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	idp, err := s.idpRepo.FindByUUID(idpUUID, "Tenant")
	if err != nil || idp == nil {
		return nil, errors.New("idp not found")
	}

	return toIdpServiceDataResult(idp), nil
}

func (s *identityProviderService) Create(name string, displayName string, provider string, providerType string, config datatypes.JSON, status string, tenantUUID string, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	var createdIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txTenantRepo := s.tenantRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Parse tenant UUID
		tenantUUIDParsed, err := uuid.Parse(tenantUUID)
		if err != nil {
			return errors.New("invalid tenant UUID")
		}

		// Check if tenant exist
		tenant, err := txTenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || tenant == nil {
			return errors.New("tenant not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, tenant); err != nil {
			return err
		}

		// Check if idp already exists
		existingIdp, err := txIdpRepo.FindByName(name, tenant.TenantID)
		if err != nil {
			return err
		}
		if existingIdp != nil {
			return errors.New(name + " idp already exists")
		}

		// Generate identifier
		identifier := fmt.Sprintf("idp-%s", util.GenerateIdentifier(12))

		// Create idp
		newIdp := &model.IdentityProvider{
			Name:         name,
			DisplayName:  displayName,
			Provider:     provider,
			ProviderType: providerType,
			Identifier:   identifier,
			Config:       config,
			TenantID:     tenant.TenantID,
			Status:       status,
			IsDefault:    false, // System-managed field, always default to false for user-created providers
			IsSystem:     false, // System-managed field, always default to false for user-created providers
		}

		_, err = txIdpRepo.CreateOrUpdate(newIdp)
		if err != nil {
			return err
		}

		// Fetch idp with Tenant preloaded
		createdIdp, err = txIdpRepo.FindByUUID(newIdp.IdentityProviderUUID, "Tenant")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toIdpServiceDataResult(createdIdp), nil
}

func (s *identityProviderService) Update(idpUUID uuid.UUID, name string, displayName string, provider string, providerType string, config datatypes.JSON, status string, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	var updatedIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get idp
		idp, err := txIdpRepo.FindByUUID(idpUUID, "Tenant")
		if err != nil || idp == nil {
			return errors.New("idp not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, idp.Tenant); err != nil {
			return err
		}

		// Check if system or default (cannot be updated)
		if idp.IsSystem {
			return errors.New("system idp cannot be updated")
		}
		if idp.IsDefault {
			return errors.New("default idp cannot be updated")
		}

		// Check if idp already exist
		if idp.Name != name {
			existingIdp, err := txIdpRepo.FindByName(name, idp.TenantID)
			if err != nil {
				return err
			}
			if existingIdp != nil && existingIdp.IdentityProviderUUID != idpUUID {
				return errors.New(name + " idp already exists")
			}
		}

		// Set values
		idp.Name = name
		idp.DisplayName = displayName
		idp.Provider = provider
		idp.ProviderType = providerType
		idp.Config = config
		idp.Status = status
		// IsDefault and IsSystem are system-managed, don't update them in user requests

		// Update
		_, err = txIdpRepo.CreateOrUpdate(idp)
		if err != nil {
			return err
		}

		updatedIdp = idp

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toIdpServiceDataResult(updatedIdp), nil
}

func (s *identityProviderService) SetStatusByUUID(idpUUID uuid.UUID, status string, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	var updatedIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get idp
		idp, err := txIdpRepo.FindByUUID(idpUUID, "Tenant")
		if err != nil || idp == nil {
			return errors.New("idp not found")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "Tenant")
		if err != nil || actorUser == nil {
			return errors.New("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, idp.Tenant); err != nil {
			return err
		}

		// Check if system or default (cannot be updated)
		if idp.IsSystem {
			return errors.New("system idp cannot be updated")
		}
		if idp.IsDefault {
			return errors.New("default idp cannot be updated")
		}

		// Set status
		idp.Status = status

		_, err = txIdpRepo.CreateOrUpdate(idp)
		if err != nil {
			return err
		}

		updatedIdp = idp

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toIdpServiceDataResult(updatedIdp), nil
}

func (s *identityProviderService) DeleteByUUID(idpUUID uuid.UUID, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	// Get idp
	idp, err := s.idpRepo.FindByUUID(idpUUID, "Tenant")
	if err != nil || idp == nil {
		return nil, errors.New("idp not found")
	}

	// Get actor user with tenant info
	actorUser, err := s.userRepo.FindByUUID(actorUserUUID, "Tenant")
	if err != nil || actorUser == nil {
		return nil, errors.New("actor user not found")
	}

	// Validate tenant access permissions
	if err := ValidateTenantAccess(actorUser, idp.Tenant); err != nil {
		return nil, err
	}

	// Check if system or default (cannot be deleted)
	if idp.IsSystem {
		return nil, errors.New("system idp cannot be deleted")
	}
	if idp.IsDefault {
		return nil, errors.New("default idp cannot be deleted")
	}

	err = s.idpRepo.DeleteByUUID(idpUUID)
	if err != nil {
		return nil, err
	}

	return toIdpServiceDataResult(idp), nil
}

// Reponse builder
func toIdpServiceDataResult(idp *model.IdentityProvider) *IdentityProviderServiceDataResult {
	if idp == nil {
		return nil
	}

	result := &IdentityProviderServiceDataResult{
		IdentityProviderUUID: idp.IdentityProviderUUID,
		Name:                 idp.Name,
		DisplayName:          idp.DisplayName,
		Provider:             idp.Provider,
		ProviderType:         idp.ProviderType,
		Identifier:           idp.Identifier,
		Config:               &idp.Config,
		Status:               idp.Status,
		IsDefault:            idp.IsDefault,
		IsSystem:             idp.IsSystem,
		CreatedAt:            idp.CreatedAt,
		UpdatedAt:            idp.UpdatedAt,
	}

	if idp.Tenant != nil {
		result.Tenant = toTenantServiceDataResult(idp.Tenant)
	}

	return result
}
