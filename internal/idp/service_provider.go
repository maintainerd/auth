package idp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/shared"
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
	Issuer               string
	ProviderClientID     string
	AllowJITProvisioning bool
	EmailDomains         []string
	Config               *datatypes.JSON
	Tenant               *TenantServiceDataResult
	Status               string
	IsDefault            bool
	IsSystem             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// IdentityProviderCreateInput carries every create-time input. issuer, provider_client_id,
// provider_client_secret, allow_jit_provisioning and email_domains are promoted out of the
// config JSONB blob and threaded here as first-class fields.
type IdentityProviderCreateInput struct {
	Name                 string
	DisplayName          string
	Provider             string
	ProviderType         string
	Issuer               string
	ProviderClientID     string
	ProviderClientSecret string
	AllowJITProvisioning bool
	EmailDomains         []string
	Config               datatypes.JSON
	Status               string
	TenantUUID           string
	TenantID             int64
	ActorUserUUID        uuid.UUID
}

// IdentityProviderUpdateInput carries every update-time input. ProviderClientSecret obeys
// the write-only contract: blank or the redaction sentinel preserves the stored
// secret.
type IdentityProviderUpdateInput struct {
	IdpUUID              uuid.UUID
	Name                 string
	DisplayName          string
	Provider             string
	ProviderType         string
	Issuer               string
	ProviderClientID     string
	ProviderClientSecret string
	AllowJITProvisioning bool
	EmailDomains         []string
	Config               datatypes.JSON
	Status               string
	TenantID             int64
	ActorUserUUID        uuid.UUID
}

type IdentityProviderServiceGetFilter struct {
	Search       *string
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
	Create(ctx context.Context, in IdentityProviderCreateInput) (*IdentityProviderServiceDataResult, error)
	Update(ctx context.Context, in IdentityProviderUpdateInput) (*IdentityProviderServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, idpUUID uuid.UUID, status string, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	DeleteByUUID(ctx context.Context, idpUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
}

type identityProviderService struct {
	db              *gorm.DB
	idpRepo         IdentityProviderRepository
	emailDomainRepo IdentityProviderEmailDomainRepository
	tenantRepo      TenantRepository
	userRepo        UserRepository
}

func NewIdentityProviderService(
	db *gorm.DB,
	idpRepo IdentityProviderRepository,
	emailDomainRepo IdentityProviderEmailDomainRepository,
	tenantRepo TenantRepository,
	userRepo UserRepository,
) IdentityProviderService {
	return &identityProviderService{
		db:              db,
		idpRepo:         idpRepo,
		emailDomainRepo: emailDomainRepo,
		tenantRepo:      tenantRepo,
		userRepo:        userRepo,
	}
}

func (s *identityProviderService) Get(ctx context.Context, filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))

	// Build query filter
	queryFilter := IdentityProviderRepositoryGetFilter{
		Search:       filter.Search,
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

	// Safe read: never loads the encrypted secret columns.
	idp, err := s.idpRepo.FindByUUIDSafe(idpUUID, "Tenant", "EmailDomains")
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

func (s *identityProviderService) Create(ctx context.Context, in IdentityProviderCreateInput) (*IdentityProviderServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.create")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", in.TenantID),
		attribute.String("idp.name", in.Name),
	)

	// Enforce the active-gated structural rule on every surface (HTTP + gRPC).
	if err := validateExternalProviderColumns(in.ProviderType, in.Status, in.Issuer, in.ProviderClientID); err != nil {
		return nil, err
	}

	var createdIdp *IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txEmailDomainRepo := s.emailDomainRepo.WithTx(tx)
		txTenantRepo := s.tenantRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Parse and check if tenant UUID is valid
		tenantUUIDParsed, err := uuid.Parse(in.TenantUUID)
		if err != nil {
			return apperror.NewValidation("invalid tenant UUID")
		}

		// Check if tenant exist
		tenant, err := txTenantRepo.FindByUUID(tenantUUIDParsed)
		if err != nil || tenant == nil {
			return apperror.NewNotFound("tenant not found")
		}

		// Validate tenant ownership
		if tenant.TenantID != in.TenantID {
			return apperror.NewForbidden("access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(in.ActorUserUUID, "UserIdentities.Tenant")
		if err != nil || actorUser == nil {
			return apperror.NewNotFoundWithReason("actor user not found")
		}

		// Validate tenant access permissions
		if err := ValidateTenantAccess(actorUser, tenant); err != nil {
			return err
		}

		// Check if idp already exists
		existingIdp, err := txIdpRepo.FindByName(in.Name, tenant.TenantID)
		if err != nil {
			return err
		}
		if existingIdp != nil {
			return apperror.NewConflict(in.Name + " idp already exists")
		}

		// Generate identifier
		idSuffix, err := crypto.GenerateIdentifier(12)
		if err != nil {
			return err
		}
		identifier := fmt.Sprintf("idp-%s", idSuffix)

		// Encrypt the upstream client secret into the dedicated column.
		encSecret, encErr := encryptProviderClientSecret(in.ProviderClientSecret)
		if encErr != nil {
			return encErr
		}

		newIdp := &IdentityProvider{
			Name:                          in.Name,
			DisplayName:                   in.DisplayName,
			Provider:                      in.Provider,
			ProviderType:                  in.ProviderType,
			Identifier:                    identifier,
			Issuer:                        ptrIfNotBlank(in.Issuer),
			ProviderClientID:              ptrIfNotBlank(in.ProviderClientID),
			ProviderClientSecretEncrypted: encSecret,
			AllowJITProvisioning:          in.AllowJITProvisioning,
			Config:                        in.Config,
			TenantID:                      tenant.TenantID,
			Status:                        in.Status,
			IsDefault:                     false, // System-managed field, always default to false for user-created providers
			IsSystem:                      false, // System-managed field, always default to false for user-created providers
		}

		if _, err := txIdpRepo.CreateOrUpdate(newIdp); err != nil {
			return err
		}

		// Replace email-domain membership transactionally.
		if err := txEmailDomainRepo.ReplaceForProvider(tenant.TenantID, newIdp.IdentityProviderID, in.EmailDomains); err != nil {
			return err
		}

		// Fetch idp (safe — no secret) with Tenant + EmailDomains preloaded.
		createdIdp, err = txIdpRepo.FindByUUIDSafe(newIdp.IdentityProviderUUID, "Tenant", "EmailDomains")
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

func (s *identityProviderService) Update(ctx context.Context, in IdentityProviderUpdateInput) (*IdentityProviderServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "identityProvider.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("idp.uuid", in.IdpUUID.String()),
		attribute.Int64("tenant.id", in.TenantID),
	)

	// Enforce the active-gated structural rule on every surface (HTTP + gRPC).
	if err := validateExternalProviderColumns(in.ProviderType, in.Status, in.Issuer, in.ProviderClientID); err != nil {
		return nil, err
	}

	var updatedIdp *IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txEmailDomainRepo := s.emailDomainRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Full read — needs the existing encrypted secret to honor the write-only
		// preserve contract, and re-saving the full row avoids nulling it.
		idp, err := txIdpRepo.FindByUUID(in.IdpUUID, "Tenant")
		if err != nil || idp == nil {
			return apperror.NewNotFoundWithReason("identity provider not found or access denied")
		}

		// Validate tenant ownership
		if idp.TenantID != in.TenantID {
			return apperror.NewNotFoundWithReason("identity provider not found or access denied")
		}

		// Get actor user with tenant info
		actorUser, err := txUserRepo.FindByUUID(in.ActorUserUUID, "UserIdentities.Tenant")
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
		if idp.Name != in.Name {
			existingIdp, err := txIdpRepo.FindByName(in.Name, idp.TenantID)
			if err != nil {
				return err
			}
			if existingIdp != nil && existingIdp.IdentityProviderUUID != in.IdpUUID {
				return apperror.NewConflict(in.Name + " idp already exists")
			}
		}

		// Preserve the stored secret when the request omits/blanks/redacts it,
		// otherwise encrypt the newly provided plaintext.
		encSecret, encErr := preserveProviderClientSecret(in.ProviderClientSecret, idp.ProviderClientSecretEncrypted)
		if encErr != nil {
			return encErr
		}

		// Set values
		idp.Name = in.Name
		idp.DisplayName = in.DisplayName
		idp.Provider = in.Provider
		idp.ProviderType = in.ProviderType
		idp.Issuer = ptrIfNotBlank(in.Issuer)
		idp.ProviderClientID = ptrIfNotBlank(in.ProviderClientID)
		idp.ProviderClientSecretEncrypted = encSecret
		idp.AllowJITProvisioning = in.AllowJITProvisioning
		idp.Config = in.Config
		idp.Status = in.Status
		// IsDefault and IsSystem are system-managed, don't update them in user requests

		if _, err := txIdpRepo.CreateOrUpdate(idp); err != nil {
			return err
		}

		// Replace email-domain membership transactionally.
		if err := txEmailDomainRepo.ReplaceForProvider(idp.TenantID, idp.IdentityProviderID, in.EmailDomains); err != nil {
			return err
		}

		// Re-fetch safe (no secret) for the response.
		updatedIdp, err = txIdpRepo.FindByUUIDSafe(in.IdpUUID, "Tenant", "EmailDomains")
		if err != nil {
			return err
		}
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

	var updatedIdp *IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Full read so re-saving the row does not null the encrypted secret.
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

		if _, err := txIdpRepo.CreateOrUpdate(idp); err != nil {
			return err
		}

		// Re-fetch safe (no secret) for the response.
		updatedIdp, err = txIdpRepo.FindByUUIDSafe(idpUUID, "Tenant", "EmailDomains")
		if err != nil {
			return err
		}
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

	// Safe read (no secret) — delete does not re-save the row.
	idp, err := s.idpRepo.FindByUUIDSafe(idpUUID, "Tenant", "EmailDomains")
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

	if err := s.idpRepo.DeleteByUUID(idpUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete identity provider")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toIdpServiceDataResult(idp), nil
}

// validateExternalProviderColumns enforces, at the service layer (covering HTTP
// and gRPC surfaces), that an ACTIVE social/enterprise provider carries the
// issuer and provider_client_id columns. Inactive/draft providers are intentionally not
// constrained so they can be created before being fully configured. The DB does
// not enforce this (drafts are common); validation lives here and in the DTO.
func validateExternalProviderColumns(providerType, status, issuer, clientID string) error {
	if status != shared.StatusActive {
		return nil
	}
	if providerType != shared.IDPTypeSocial && providerType != shared.IDPTypeEnterprise {
		return nil
	}
	if strings.TrimSpace(issuer) == "" {
		return apperror.NewValidation("issuer is required for active social/enterprise identity providers")
	}
	if strings.TrimSpace(clientID) == "" {
		return apperror.NewValidation("provider_client_id is required for active social/enterprise identity providers")
	}
	return nil
}

func ptrIfNotBlank(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// Reponse builder
func toIdpServiceDataResult(idp *IdentityProvider) *IdentityProviderServiceDataResult {
	if idp == nil {
		return nil
	}

	var cfg *datatypes.JSON
	if len(idp.Config) > 0 {
		c := idp.Config
		cfg = &c
	}

	domains := make([]string, 0, len(idp.EmailDomains))
	for _, d := range idp.EmailDomains {
		domains = append(domains, d.Domain)
	}

	result := &IdentityProviderServiceDataResult{
		IdentityProviderUUID: idp.IdentityProviderUUID,
		Name:                 idp.Name,
		DisplayName:          idp.DisplayName,
		Provider:             idp.Provider,
		ProviderType:         idp.ProviderType,
		Identifier:           idp.Identifier,
		Issuer:               idp.IssuerOrEmpty(),
		ProviderClientID:     idp.ProviderClientIDOrEmpty(),
		AllowJITProvisioning: idp.AllowJITProvisioning,
		EmailDomains:         domains,
		// The encrypted secret column is never selected on read paths, so it
		// never reaches this result. Config holds only non-secret JSONB fields.
		Config:    cfg,
		Status:    idp.Status,
		IsDefault: idp.IsDefault,
		IsSystem:  idp.IsSystem,
		CreatedAt: idp.CreatedAt,
		UpdatedAt: idp.UpdatedAt,
	}

	if idp.Tenant != nil {
		result.Tenant = toTenantServiceDataResult(idp.Tenant)
	}

	return result
}
