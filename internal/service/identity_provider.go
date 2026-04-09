package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	TenantID     int64
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
	Get(ctx context.Context, filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error)
	GetByUUID(ctx context.Context, idpUUID uuid.UUID, tenantID int64) (*IdentityProviderServiceDataResult, error)
	Create(ctx context.Context, name string, displayName string, provider string, providerType string, config datatypes.JSON, status string, tenantUUID string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	Update(ctx context.Context, idpUUID uuid.UUID, name string, displayName string, provider string, providerType string, config datatypes.JSON, status string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, idpUUID uuid.UUID, status string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	DeleteByUUID(ctx context.Context, idpUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
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

func (s *identityProviderService) Get(ctx context.Context, filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))

	// Build query filter
	queryFilter := repository.IdentityProviderRepositoryGetFilter{
		Name:         filter.Name,
		DisplayName:  filter.DisplayName,
		Provider:     filter.Provider,
		ProviderType: filter.ProviderType,
		Identifier:   filter.Identifier,
		TenantID:     &filter.TenantID,
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to list identity providers")
		return nil, err
	}

	idps := make([]IdentityProviderServiceDataResult, len(result.Data))
	for i, idp := range result.Data {
		idps[i] = *toIdpServiceDataResult(&idp)
	}

	span.SetStatus(codes.Ok, "")
	return &IdentityProviderServiceGetResult{
		Data:       idps,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *identityProviderService) GetByUUID(ctx context.Context, idpUUID uuid.UUID, tenantID int64) (*IdentityProviderServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("idp.uuid", idpUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	idp, err := s.idpRepo.FindByUUID(idpUUID, "Tenant")
	if err != nil || idp == nil {
		span.SetStatus(codes.Error, "identity provider not found or access denied")
		return nil, apperror.NewNotFoundWithReason("identity provider not found or access denied")
	}

	// Validate tenant ownership
	if idp.TenantID != tenantID {
		span.SetStatus(codes.Error, "identity provider not found or access denied")
		return nil, apperror.NewNotFoundWithReason("identity provider not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toIdpServiceDataResult(idp), nil
}

func (s *identityProviderService) Create(ctx context.Context, name string, displayName string, provider string, providerType string, config datatypes.JSON, status string, tenantUUID string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.create")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.String("idp.name", name),
	)

	var createdIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txTenantRepo := s.tenantRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Parse and check if tenant UUID is valid
		tenantUUIDParsed, err := uuid.Parse(tenantUUID)
		if err != nil {
			return apperror.NewValidation("invalid tenant UUID")
		}

		// Check if tenant exist
		tenant, err := txTenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || tenant == nil {
			return apperror.NewNotFound("tenant not found")
		}

		// Validate tenant ownership
		if tenant.TenantID != tenantID {
			return apperror.NewForbidden("access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
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
			return apperror.NewConflict(name + " idp already exists")
		}

		// Generate identifier
		idSuffix, err := crypto.GenerateIdentifier(12)
		if err != nil {
			return err
		}
		identifier := fmt.Sprintf("idp-%s", idSuffix)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create identity provider")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toIdpServiceDataResult(createdIdp), nil
}

func (s *identityProviderService) Update(ctx context.Context, idpUUID uuid.UUID, name string, displayName string, provider string, providerType string, config datatypes.JSON, status string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("idp.uuid", idpUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	var updatedIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get idp
		idp, err := txIdpRepo.FindByUUID(idpUUID, "Tenant")
		if err != nil || idp == nil {
			return apperror.NewNotFoundWithReason("identity provider not found or access denied")
		}

		// Validate tenant ownership
		if idp.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("identity provider not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, idp.Tenant); err != nil {
			return err
		}

		// Check if system or default (cannot be updated)
		if idp.IsSystem {
			return apperror.NewValidation("system idp cannot be updated")
		}
		if idp.IsDefault {
			return apperror.NewValidation("default idp cannot be updated")
		}

		// Check if idp already exist
		if idp.Name != name {
			existingIdp, err := txIdpRepo.FindByName(name, idp.TenantID)
			if err != nil {
				return err
			}
			if existingIdp != nil && existingIdp.IdentityProviderUUID != idpUUID {
				return apperror.NewConflict(name + " idp already exists")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update identity provider")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toIdpServiceDataResult(updatedIdp), nil
}

func (s *identityProviderService) SetStatusByUUID(ctx context.Context, idpUUID uuid.UUID, status string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.setStatus")
	defer span.End()
	span.SetAttributes(
		attribute.String("idp.uuid", idpUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("idp.status", status),
	)

	var updatedIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Get idp
		idp, err := txIdpRepo.FindByUUID(idpUUID, "Tenant")
		if err != nil || idp == nil {
			return apperror.NewNotFoundWithReason("identity provider not found or access denied")
		}

		// Validate tenant ownership
		if idp.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("identity provider not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, idp.Tenant); err != nil {
			return err
		}

		// Check if system or default (cannot be updated)
		if idp.IsSystem {
			return apperror.NewValidation("system idp cannot be updated")
		}
		if idp.IsDefault {
			return apperror.NewValidation("default idp cannot be updated")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update identity provider status")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toIdpServiceDataResult(updatedIdp), nil
}

func (s *identityProviderService) DeleteByUUID(ctx context.Context, idpUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("idp.uuid", idpUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	// Get idp
	idp, err := s.idpRepo.FindByUUID(idpUUID, "Tenant")
	if err != nil || idp == nil {
		span.SetStatus(codes.Error, "identity provider not found or access denied")
		return nil, apperror.NewNotFoundWithReason("identity provider not found or access denied")
	}

	// Validate tenant ownership
	if idp.TenantID != tenantID {
		span.SetStatus(codes.Error, "identity provider not found or access denied")
		return nil, apperror.NewNotFoundWithReason("identity provider not found or access denied")
	}

	// Get actor user with tenant info
	actorUser, err := s.userRepo.FindByUUID(actorUserUUID, "UserIdentities.Tenant")
	if err != nil || actorUser == nil {
		span.SetStatus(codes.Error, "actor user not found")
		return nil, apperror.NewNotFoundWithReason("actor user not found")
	}

	// Validate tenant access permissions
	if err := ValidateTenantAccess(actorUser, idp.Tenant); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "tenant access denied")
		return nil, err
	}

	// Check if system or default (cannot be deleted)
	if idp.IsSystem {
		span.SetStatus(codes.Error, "system idp cannot be deleted")
		return nil, apperror.NewValidation("system idp cannot be deleted")
	}
	if idp.IsDefault {
		span.SetStatus(codes.Error, "default idp cannot be deleted")
		return nil, apperror.NewValidation("default idp cannot be deleted")
	}

	err = s.idpRepo.DeleteByUUID(idpUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete identity provider")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
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
