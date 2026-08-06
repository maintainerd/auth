package idp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
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
	AllowRegistration    bool
	AllowTokenFederation bool
	AllowedAudiences     []string
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
// provider_client_secret, allow_jit_provisioning, allow_token_federation, allowed_audiences
// and email_domains are promoted out of the config JSONB blob and threaded here as
// first-class fields.
type IdentityProviderCreateInput struct {
	Name                 string
	DisplayName          string
	Provider             string
	ProviderType         string
	Issuer               string
	ProviderClientID     string
	ProviderClientSecret string
	AllowJITProvisioning bool
	AllowRegistration    bool
	AllowTokenFederation bool
	AllowedAudiences     []string
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
	AllowRegistration    bool
	AllowTokenFederation bool
	AllowedAudiences     []string
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
	db                  *gorm.DB
	idpRepo             IdentityProviderRepository
	emailDomainRepo     IdentityProviderEmailDomainRepository
	allowedAudienceRepo IdentityProviderAllowedAudienceRepository
	tenantRepo          TenantRepository
	userRepo            UserRepository
	// cacheInvalidator clears cached user contexts. Deactivating or deleting a
	// provider revokes every identity it issued, so it must take effect on the
	// next request rather than after the cache TTL.
	cacheInvalidator cache.Invalidator
}

func (s *identityProviderService) invalidateUserContexts(ctx context.Context) {
	if s.cacheInvalidator != nil {
		s.cacheInvalidator.InvalidateAllUsers(ctx)
	}
}

func NewIdentityProviderService(
	db *gorm.DB,
	idpRepo IdentityProviderRepository,
	emailDomainRepo IdentityProviderEmailDomainRepository,
	allowedAudienceRepo IdentityProviderAllowedAudienceRepository,
	tenantRepo TenantRepository,
	userRepo UserRepository,
	// Variadic so existing call sites need no change.
	cacheInvalidator ...cache.Invalidator,
) IdentityProviderService {
	var invalidator cache.Invalidator
	if len(cacheInvalidator) > 0 {
		invalidator = cacheInvalidator[0]
	}
	return &identityProviderService{
		db:                  db,
		idpRepo:             idpRepo,
		emailDomainRepo:     emailDomainRepo,
		allowedAudienceRepo: allowedAudienceRepo,
		tenantRepo:          tenantRepo,
		userRepo:            userRepo,
		cacheInvalidator:    invalidator,
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

	// Safe read: never loads the encrypted secret columns. Preload the same
	// relations the create/update paths return so a re-opened provider shows its
	// saved email domains AND allowed audiences (the detail page + edit form both
	// load through here).
	idp, err := s.idpRepo.FindByUUIDSafe(idpUUID, "Tenant", "EmailDomains", "AllowedAudiences")
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

	// The 'system' provider type is reserved for the seeded, built-in provider —
	// exactly one per tenant (is_system, undeletable), provisioned only by the
	// seeder. It can never be created through the API; every user-created
	// provider (including additional external maintainerd instances) is non-system.
	if in.ProviderType == shared.IDPTypeSystem {
		return nil, apperror.NewValidation("the 'system' provider type is reserved for the built-in provider and cannot be created")
	}

	// config is JSONB NOT NULL; a nil/empty datatypes.JSON serializes to SQL NULL
	// on write and violates the constraint, so normalize an omitted config to '{}'.
	if len(in.Config) == 0 {
		in.Config = datatypes.JSON([]byte("{}"))
	}

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
			AllowRegistration:             in.AllowRegistration,
			AllowTokenFederation:          in.AllowTokenFederation,
			Config:                        in.Config,
			TenantID:                      tenant.TenantID,
			Status:                        in.Status,
			IsDefault:                     false,
			IsSystem:                      false,
		}

		if in.ProviderType == shared.IDPTypeSAML {
			var samlCfg SAMLProviderConfig
			if jsonErr := json.Unmarshal(in.Config, &samlCfg); jsonErr == nil && samlCfg.Certificate != "" {
				if expiry, certErr := ParsePEMCertExpiry(samlCfg.Certificate); certErr == nil {
					newIdp.CertificateExpiresAt = expiry
				}
			}
		}

		if _, err := txIdpRepo.CreateOrUpdate(newIdp); err != nil {
			return err
		}

		if err := txEmailDomainRepo.ReplaceForProvider(tenant.TenantID, newIdp.IdentityProviderID, in.EmailDomains); err != nil {
			return err
		}
		if s.allowedAudienceRepo != nil {
			if err := s.allowedAudienceRepo.WithTx(tx).ReplaceForProvider(tenant.TenantID, newIdp.IdentityProviderID, in.AllowedAudiences); err != nil {
				return err
			}
		}

		createdIdp, err = txIdpRepo.FindByUUIDSafe(newIdp.IdentityProviderUUID, "Tenant", "EmailDomains", "AllowedAudiences")
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

	// The 'system' provider type is reserved for the built-in provider and can
	// never be assigned via the API (the built-in itself is update-blocked below).
	if in.ProviderType == shared.IDPTypeSystem {
		return nil, apperror.NewValidation("the 'system' provider type is reserved for the built-in provider and cannot be assigned")
	}

	// config is JSONB NOT NULL; a nil/empty datatypes.JSON serializes to SQL NULL
	// on write and violates the constraint, so normalize an omitted config to '{}'.
	if len(in.Config) == 0 {
		in.Config = datatypes.JSON([]byte("{}"))
	}

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
		idp.AllowRegistration = in.AllowRegistration
		idp.AllowTokenFederation = in.AllowTokenFederation
		idp.Config = in.Config
		idp.Status = in.Status

		if in.ProviderType == shared.IDPTypeSAML {
			var samlCfg SAMLProviderConfig
			if jsonErr := json.Unmarshal(in.Config, &samlCfg); jsonErr == nil && samlCfg.Certificate != "" {
				if expiry, certErr := ParsePEMCertExpiry(samlCfg.Certificate); certErr == nil {
					idp.CertificateExpiresAt = expiry
				}
			}
		}

		if _, err := txIdpRepo.CreateOrUpdate(idp); err != nil {
			return err
		}

		if err := txEmailDomainRepo.ReplaceForProvider(idp.TenantID, idp.IdentityProviderID, in.EmailDomains); err != nil {
			return err
		}
		if s.allowedAudienceRepo != nil {
			if err := s.allowedAudienceRepo.WithTx(tx).ReplaceForProvider(idp.TenantID, idp.IdentityProviderID, in.AllowedAudiences); err != nil {
				return err
			}
		}

		updatedIdp, err = txIdpRepo.FindByUUIDSafe(in.IdpUUID, "Tenant", "EmailDomains", "AllowedAudiences")
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
	// Update writes idp.Status, so an admin can deactivate a provider here and
	// not only through SetStatusByUUID.
	s.invalidateUserContexts(ctx)
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

		// Activating a provider must not bypass the structural validation the
		// create/update DTOs enforce: a draft (missing issuer/client_id/config)
		// could otherwise be flipped ACTIVE and become an unconfigured live IdP.
		// Validate the STORED provider before flipping status to active. Deactivating
		// is never gated.
		if status == shared.StatusActive {
			if err := validateStoredProviderForActivation(idp); err != nil {
				return err
			}
		}

		// Set status
		idp.Status = status

		if _, err := txIdpRepo.CreateOrUpdate(idp); err != nil {
			return err
		}

		// Re-fetch safe (no secret) for the response.
		updatedIdp, err = txIdpRepo.FindByUUIDSafe(idpUUID, "Tenant", "EmailDomains", "AllowedAudiences")
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
	s.invalidateUserContexts(ctx)
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

	// A soft delete does NOT fire the FK ON DELETE CASCADE, so the child email_domains
	// and allowed_audiences rows would be left live — later blocking domain reuse
	// (unique-constraint violation) and misrouting home-realm discovery to a deleted
	// IdP. Clear the children and soft-delete the provider atomically.
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.emailDomainRepo.WithTx(tx).ReplaceForProvider(idp.TenantID, idp.IdentityProviderID, nil); err != nil {
			return err
		}
		if s.allowedAudienceRepo != nil {
			if err := s.allowedAudienceRepo.WithTx(tx).ReplaceForProvider(idp.TenantID, idp.IdentityProviderID, nil); err != nil {
				return err
			}
		}
		// client_identity_providers rows survive the soft delete for the same
		// reason. Every reachability query gates on the provider being live, so
		// they are no longer an access path — but leaving them would show a
		// deleted provider as still connected in the admin UI, and they would
		// come back to life if the row were ever restored.
		if err := tx.Exec(`
			UPDATE client_identity_providers
			SET deleted_at = now(), enabled = FALSE
			WHERE identity_provider_id = ? AND deleted_at IS NULL`,
			idp.IdentityProviderID).Error; err != nil {
			return err
		}
		return s.idpRepo.WithTx(tx).DeleteByUUID(idpUUID)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete identity provider")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.invalidateUserContexts(ctx)
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

// validateStoredProviderForActivation runs the same structural + config rules the
// create/update DTOs enforce, but against the STORED provider row, so a draft can
// never be flipped ACTIVE through the status endpoint without being fully
// configured. It mirrors the DTO wiring: the column rule always runs, then the
// SAML config rule (SAML providers) or the external OIDC/OAuth2 config rule
// (social/enterprise providers). Config rule errors are ozzo validation errors, so
// they are wrapped in apperror.NewValidation to map to HTTP 400 at the service edge.
func validateStoredProviderForActivation(idp *IdentityProvider) error {
	if err := validateExternalProviderColumns(idp.ProviderType, shared.StatusActive, idp.IssuerOrEmpty(), idp.ProviderClientIDOrEmpty()); err != nil {
		return err
	}
	switch {
	case isSAMLProviderType(idp.ProviderType):
		if err := validateSAMLConfig(true)(idp.Config); err != nil {
			return apperror.NewValidation(err.Error())
		}
	case isExternalProviderType(idp.ProviderType):
		if err := validateExternalProviderConfig(idp.Provider, isOAuth2OnlyProvider(idp.Provider))(idp.Config); err != nil {
			return apperror.NewValidation(err.Error())
		}
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

	audiences := make([]string, 0, len(idp.AllowedAudiences))
	for _, a := range idp.AllowedAudiences {
		audiences = append(audiences, a.Audience)
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
		AllowRegistration:    idp.AllowRegistration,
		AllowTokenFederation: idp.AllowTokenFederation,
		AllowedAudiences:     audiences,
		EmailDomains:         domains,
		Config:               cfg,
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
