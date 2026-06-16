package idp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type AuthFlowServiceDataResult struct {
	AuthFlowUUID uuid.UUID
	Name         string
	Description  string
	Identifier   string
	Destination string
	Status       string
	ClientUUID   uuid.UUID
	BrandingUUID *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AuthFlowCallbackURIServiceDataResult struct {
	AuthFlowCallbackURIUUID uuid.UUID
	AuthFlowUUID            uuid.UUID
	ClientURIUUID           uuid.UUID
	URI                     string
	CreatedAt               time.Time
}

type AuthFlowCallbackURIServiceListResult struct {
	Data       []AuthFlowCallbackURIServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type AuthFlowServiceListResult struct {
	Data       []AuthFlowServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type AuthFlowRoleServiceDataResult struct {
	AuthFlowRoleUUID uuid.UUID
	AuthFlowUUID     uuid.UUID
	RoleUUID         uuid.UUID
	RoleName         string
	RoleDescription  string
	RoleStatus       string
	RoleIsDefault    bool
	RoleIsSystem     bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AuthFlowRoleServiceListResult struct {
	Data       []AuthFlowRoleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type AuthFlowService interface {
	GetAll(ctx context.Context, tenantID int64, name, identifier *string, status []string, ClientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*AuthFlowServiceListResult, error)
	GetByUUID(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64) (*AuthFlowServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name, description, status, destination string, ClientUUID uuid.UUID, brandingUUID *uuid.UUID, roleUUIDs, callbackClientURIUUIDs []uuid.UUID) (*AuthFlowServiceDataResult, error)
	Update(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, name, description, status string, brandingUUID *uuid.UUID, roleUUIDs, callbackClientURIUUIDs []uuid.UUID) (*AuthFlowServiceDataResult, error)
	UpdateStatus(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, status string) (*AuthFlowServiceDataResult, error)
	Delete(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64) (*AuthFlowServiceDataResult, error)
	AssignRoles(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]AuthFlowRoleServiceDataResult, error)
	GetRoles(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, page, limit int) (*AuthFlowRoleServiceListResult, error)
	RemoveRole(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, roleUUID uuid.UUID) error
	AssignCallbackURIs(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, clientURIUUIDs []uuid.UUID) ([]AuthFlowCallbackURIServiceDataResult, error)
	GetCallbackURIs(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, page, limit int) (*AuthFlowCallbackURIServiceListResult, error)
	RemoveCallbackURI(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, clientURIUUID uuid.UUID) error
}

type authFlowService struct {
	db                      *gorm.DB
	authFlowRepo            AuthFlowRepository
	authFlowRoleRepo        AuthFlowRoleRepository
	authFlowCallbackURIRepo AuthFlowCallbackURIRepository
	roleRepo                RoleRepository
	clientRepo              ClientRepository
}

func NewAuthFlowService(
	db *gorm.DB,
	authFlowRepo AuthFlowRepository,
	authFlowRoleRepo AuthFlowRoleRepository,
	authFlowCallbackURIRepo AuthFlowCallbackURIRepository,
	roleRepo RoleRepository,
	clientRepo ClientRepository,
) AuthFlowService {
	return &authFlowService{
		db:                      db,
		authFlowRepo:            authFlowRepo,
		authFlowRoleRepo:        authFlowRoleRepo,
		authFlowCallbackURIRepo: authFlowCallbackURIRepo,
		roleRepo:                roleRepo,
		clientRepo:              clientRepo,
	}
}

// resolveBrandingID maps an optional branding UUID to its internal id within the
// tenant. Returns (nil, nil) when no branding is supplied.
func (s *authFlowService) resolveBrandingID(tx *gorm.DB, tenantID int64, brandingUUID *uuid.UUID) (*int64, error) {
	if brandingUUID == nil {
		return nil, nil
	}
	var b Branding
	err := tx.Where("branding_uuid = ? AND tenant_id = ?", *brandingUUID, tenantID).First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundWithReason("branding not found")
		}
		return nil, err
	}
	return &b.BrandingID, nil
}

// syncRoles replaces the flow's role membership to exactly roleUUIDs (adds
// missing, removes extra). Runs inside the supplied transaction.
func (s *authFlowService) syncRoles(tx *gorm.DB, authFlow *AuthFlow, roleUUIDs []uuid.UUID) error {
	txRoleRepo := s.roleRepo.WithTx(tx)
	txAFRRepo := s.authFlowRoleRepo.WithTx(tx)

	desired := make(map[int64]bool, len(roleUUIDs))
	for _, ru := range roleUUIDs {
		role, err := txRoleRepo.FindByUUID(ru)
		if err != nil || role == nil {
			return apperror.NewNotFoundWithReason("role not found: " + ru.String())
		}
		if role.TenantID != authFlow.TenantID {
			return apperror.NewForbidden("role does not belong to the same tenant as the auth flow")
		}
		desired[role.RoleID] = true
	}

	existing, err := txAFRRepo.FindByAuthFlowID(authFlow.AuthFlowID)
	if err != nil {
		return err
	}
	existingSet := make(map[int64]bool, len(existing))
	for _, e := range existing {
		existingSet[e.RoleID] = true
		if !desired[e.RoleID] {
			if err := txAFRRepo.DeleteByAuthFlowIDAndRoleID(authFlow.AuthFlowID, e.RoleID); err != nil {
				return err
			}
		}
	}
	for roleID := range desired {
		if !existingSet[roleID] {
			if _, err := txAFRRepo.Create(&AuthFlowRole{AuthFlowID: authFlow.AuthFlowID, RoleID: roleID}); err != nil {
				return err
			}
		}
	}
	return nil
}

// syncCallbackURIs replaces the flow's callback-URI membership to exactly the
// given client-URI UUIDs. Each must belong to the flow's client. Runs inside the
// supplied transaction.
func (s *authFlowService) syncCallbackURIs(tx *gorm.DB, authFlow *AuthFlow, tenantID int64, clientURIUUIDs []uuid.UUID) error {
	if len(clientURIUUIDs) > 0 && authFlow.ClientID == nil {
		return apperror.NewValidation("auth flow has no client; attach a client before adding callback URIs")
	}
	txCallbackRepo := s.authFlowCallbackURIRepo.WithTx(tx)

	desired := make(map[int64]bool, len(clientURIUUIDs))
	for _, cu := range clientURIUUIDs {
		uri, err := s.findClientURI(tx, cu, tenantID)
		if err != nil {
			return err
		}
		if uri == nil {
			return apperror.NewNotFoundWithReason("client URI not found: " + cu.String())
		}
		if authFlow.ClientID == nil || uri.ClientID != *authFlow.ClientID {
			return apperror.NewValidation("callback URI must belong to the auth flow's client")
		}
		desired[uri.ClientURIID] = true
	}

	existing, err := txCallbackRepo.FindByAuthFlowID(authFlow.AuthFlowID)
	if err != nil {
		return err
	}
	existingSet := make(map[int64]bool, len(existing))
	for _, e := range existing {
		existingSet[e.ClientURIID] = true
		if !desired[e.ClientURIID] {
			if err := txCallbackRepo.DeleteByAuthFlowIDAndClientURIID(authFlow.AuthFlowID, e.ClientURIID); err != nil {
				return err
			}
		}
	}
	for clientURIID := range desired {
		if !existingSet[clientURIID] {
			if _, err := txCallbackRepo.Create(&AuthFlowCallbackURI{AuthFlowID: authFlow.AuthFlowID, ClientURIID: clientURIID}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *authFlowService) GetAll(ctx context.Context, tenantID int64, name, identifier *string, status []string, ClientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*AuthFlowServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	var ClientID *int64
	if ClientUUID != nil {
		Client, err := s.clientRepo.FindByUUID(*ClientUUID)
		if err != nil || Client == nil {
			return nil, apperror.NewNotFoundWithReason("auth client not found")
		}
		if Client.TenantID != tenantID {
			return nil, apperror.NewForbidden("client does not belong to your tenant")
		}
		ClientID = &Client.ClientID
	}

	filter := AuthFlowRepositoryGetFilter{
		Name:       name,
		Identifier: identifier,
		Status:     status,
		TenantID:   &tenantID,
		ClientID:   ClientID,
		Page:       page,
		Limit:      limit,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
	}

	result, err := s.authFlowRepo.FindPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list signup flows failed")
		return nil, err
	}

	data := make([]AuthFlowServiceDataResult, len(result.Data))
	for i, sf := range result.Data {
		data[i] = *toAuthFlowServiceDataResult(&sf)
	}

	span.SetStatus(codes.Ok, "")
	return &AuthFlowServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *authFlowService) GetByUUID(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64) (*AuthFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	authFlow, err := s.authFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID, "Client", "Branding")
	if err != nil || authFlow == nil {
		span.SetStatus(codes.Error, "signup flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("signup flow not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toAuthFlowServiceDataResult(authFlow), nil
}

func (s *authFlowService) Create(ctx context.Context, tenantID int64, name, description, status, destination string, ClientUUID uuid.UUID, brandingUUID *uuid.UUID, roleUUIDs, callbackClientURIUUIDs []uuid.UUID) (*AuthFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	var createdAuthFlow *AuthFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthFlowRepo := s.authFlowRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Find auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}
		if Client.TenantID != tenantID {
			return apperror.NewForbidden("client does not belong to your tenant")
		}

		// Check if name already exists
		existingName, err := txAuthFlowRepo.FindByNameAndTenantID(name, tenantID)
		if err != nil {
			return err
		}
		if existingName != nil {
			return apperror.NewConflict("signup flow with this name already exists")
		}

		// Generate unique identifier
		var identifier string
		for {
			identifier, err = crypto.GenerateIdentifier(16)
			if err != nil {
				return err
			}
			existing, err := txAuthFlowRepo.FindByIdentifierAndClientID(identifier, Client.ClientID)
			if err != nil {
				return err
			}
			if existing == nil {
				break
			}
		}

		// Resolve optional branding
		brandingID, err := s.resolveBrandingID(tx, tenantID, brandingUUID)
		if err != nil {
			return err
		}

		// Create auth flow
		authFlow := &AuthFlow{
			TenantID:     tenantID,
			Name:         name,
			Description:  description,
			Identifier:   identifier,
			Destination: destination,
			Status:       status,
			ClientID:     &Client.ClientID,
			BrandingID:   brandingID,
		}

		created, err := txAuthFlowRepo.Create(authFlow)
		if err != nil {
			return err
		}

		// Attach roles + callback URIs in the same transaction.
		if len(roleUUIDs) > 0 {
			if err := s.syncRoles(tx, created, roleUUIDs); err != nil {
				return err
			}
		}
		if len(callbackClientURIUUIDs) > 0 {
			if err := s.syncCallbackURIs(tx, created, tenantID, callbackClientURIUUIDs); err != nil {
				return err
			}
		}

		createdAuthFlow = created
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create signup flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, createdAuthFlow.AuthFlowUUID, tenantID)
}

func (s *authFlowService) Update(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, name, description, status string, brandingUUID *uuid.UUID, roleUUIDs, callbackClientURIUUIDs []uuid.UUID) (*AuthFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.update")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedAuthFlow *AuthFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthFlowRepo := s.authFlowRepo.WithTx(tx)

		// Find existing signup flow and validate tenant ownership
		authFlow, err := txAuthFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID)
		if err != nil || authFlow == nil {
			return apperror.NewNotFoundWithReason("signup flow not found or access denied")
		}

		// Check if name is being changed and if it conflicts
		if name != authFlow.Name {
			existingName, err := txAuthFlowRepo.FindByNameAndTenantID(name, tenantID)
			if err != nil {
				return err
			}
			if existingName != nil && existingName.AuthFlowID != authFlow.AuthFlowID {
				return apperror.NewConflict("signup flow with this name already exists")
			}
		}

		// Resolve optional branding (nil clears it)
		brandingID, err := s.resolveBrandingID(tx, tenantID, brandingUUID)
		if err != nil {
			return err
		}

		// Update fields (identifier remains unchanged)
		authFlow.Name = name
		authFlow.Description = description
		authFlow.Status = status
		authFlow.BrandingID = brandingID

		updated, err := txAuthFlowRepo.CreateOrUpdate(authFlow)
		if err != nil {
			return err
		}

		// nil = leave membership untouched; non-nil (incl. empty) = replace it.
		if roleUUIDs != nil {
			if err := s.syncRoles(tx, updated, roleUUIDs); err != nil {
				return err
			}
		}
		if callbackClientURIUUIDs != nil {
			if err := s.syncCallbackURIs(tx, updated, tenantID, callbackClientURIUUIDs); err != nil {
				return err
			}
		}

		updatedAuthFlow = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update signup flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, updatedAuthFlow.AuthFlowUUID, tenantID)
}

func (s *authFlowService) UpdateStatus(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, status string) (*AuthFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.updateStatus")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedAuthFlow *AuthFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthFlowRepo := s.authFlowRepo.WithTx(tx)

		authFlow, err := txAuthFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID)
		if err != nil || authFlow == nil {
			return apperror.NewNotFoundWithReason("signup flow not found or access denied")
		}

		authFlow.Status = status

		updated, err := txAuthFlowRepo.CreateOrUpdate(authFlow)
		if err != nil {
			return err
		}

		updatedAuthFlow = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update signup flow status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, updatedAuthFlow.AuthFlowUUID, tenantID)
}

func (s *authFlowService) Delete(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64) (*AuthFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.delete")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	authFlow, err := s.authFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID, "Client")
	if err != nil || authFlow == nil {
		span.SetStatus(codes.Error, "auth flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("auth flow not found or access denied")
	}

	// Guard: prevent deletion if the flow is referenced by any pending invites.
	var pendingCount int64
	if err := s.db.
		Table("invites").
		Where("auth_flow_id = ? AND status = ?", authFlow.AuthFlowID, "pending").
		Count(&pendingCount).Error; err != nil {
		span.RecordError(err)
		return nil, err
	}
	if pendingCount > 0 {
		return nil, apperror.NewConflict("cannot delete auth flow that is referenced by pending invites")
	}

	if authFlow.IsSystem {
		return nil, apperror.NewValidation("cannot delete system auth flow")
	}

	result := toAuthFlowServiceDataResult(authFlow)

	err = s.authFlowRepo.DeleteByUUID(authFlowUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete auth flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func toAuthFlowServiceDataResult(sf *AuthFlow) *AuthFlowServiceDataResult {
	if sf == nil {
		return nil
	}

	var ClientUUID uuid.UUID
	if sf.Client != nil {
		ClientUUID = sf.Client.ClientUUID
	}

	var brandingUUID *uuid.UUID
	if sf.Branding != nil {
		brandingUUID = &sf.Branding.BrandingUUID
	}

	return &AuthFlowServiceDataResult{
		AuthFlowUUID: sf.AuthFlowUUID,
		Name:         sf.Name,
		Description:  sf.Description,
		Identifier:   sf.Identifier,
		Destination: sf.Destination,
		Status:       sf.Status,
		ClientUUID:   ClientUUID,
		BrandingUUID: brandingUUID,
		CreatedAt:    sf.CreatedAt,
		UpdatedAt:    sf.UpdatedAt,
	}
}

func (s *authFlowService) AssignRoles(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]AuthFlowRoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.assignRoles")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var assignedRoles []AuthFlowRoleServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthFlowRepo := s.authFlowRepo.WithTx(tx)
		txAuthFlowRoleRepo := s.authFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify signup flow exists and belongs to tenant
		authFlow, err := txAuthFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID)
		if err != nil || authFlow == nil {
			return apperror.NewNotFoundWithReason("signup flow not found or access denied")
		}

		// Assign each role
		for _, roleUUID := range roleUUIDs {
			role, err := txRoleRepo.FindByUUID(roleUUID)
			if err != nil || role == nil {
				return apperror.NewNotFoundWithReason("role not found: " + roleUUID.String())
			}
			if role.TenantID != authFlow.TenantID {
				return apperror.NewForbidden("role does not belong to the same tenant as the auth flow")
			}

			// Check if already assigned
			existing, err := txAuthFlowRoleRepo.FindByAuthFlowIDAndRoleID(authFlow.AuthFlowID, role.RoleID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue // Skip if already assigned
			}

			// Create signup flow role
			authFlowRole := &AuthFlowRole{
				AuthFlowID: authFlow.AuthFlowID,
				RoleID:     role.RoleID,
			}

			created, err := txAuthFlowRoleRepo.Create(authFlowRole)
			if err != nil {
				return err
			}

			assignedRoles = append(assignedRoles, AuthFlowRoleServiceDataResult{
				AuthFlowRoleUUID: created.AuthFlowRoleUUID,
				AuthFlowUUID:     authFlow.AuthFlowUUID,
				RoleUUID:         role.RoleUUID,
				RoleName:         role.Name,
				RoleDescription:  role.Description,
				RoleStatus:       role.Status,
				RoleIsDefault:    role.IsDefault,
				RoleIsSystem:     role.IsSystem,
				CreatedAt:        created.CreatedAt,
				UpdatedAt:        role.UpdatedAt,
			})
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "assign roles failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return assignedRoles, nil
}

func (s *authFlowService) GetRoles(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, page, limit int) (*AuthFlowRoleServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.getRoles")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Verify signup flow exists and belongs to tenant
	authFlow, err := s.authFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID)
	if err != nil || authFlow == nil {
		span.SetStatus(codes.Error, "signup flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("signup flow not found or access denied")
	}

	// Get paginated signup flow roles
	authFlowRoles, total, err := s.authFlowRoleRepo.FindByAuthFlowIDPaginated(authFlow.AuthFlowID, page, limit)
	if err != nil {
		return nil, err
	}

	roles := make([]AuthFlowRoleServiceDataResult, len(authFlowRoles))
	for i, sfr := range authFlowRoles {
		if sfr.Role != nil {
			roles[i] = AuthFlowRoleServiceDataResult{
				AuthFlowRoleUUID: sfr.AuthFlowRoleUUID,
				AuthFlowUUID:     authFlow.AuthFlowUUID,
				RoleUUID:         sfr.Role.RoleUUID,
				RoleName:         sfr.Role.Name,
				RoleDescription:  sfr.Role.Description,
				RoleStatus:       sfr.Role.Status,
				RoleIsDefault:    sfr.Role.IsDefault,
				RoleIsSystem:     sfr.Role.IsSystem,
				CreatedAt:        sfr.CreatedAt,
				UpdatedAt:        sfr.Role.UpdatedAt,
			}
		}
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	span.SetStatus(codes.Ok, "")
	return &AuthFlowRoleServiceListResult{
		Data:       roles,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (s *authFlowService) RemoveRole(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, roleUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.removeRole")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthFlowRepo := s.authFlowRepo.WithTx(tx)
		txAuthFlowRoleRepo := s.authFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify signup flow exists and belongs to tenant
		authFlow, err := txAuthFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID)
		if err != nil || authFlow == nil {
			return apperror.NewNotFoundWithReason("signup flow not found or access denied")
		}

		// Verify role exists
		role, err := txRoleRepo.FindByUUID(roleUUID)
		if err != nil || role == nil {
			return apperror.NewNotFound("role not found")
		}

		// Delete signup flow role
		return txAuthFlowRoleRepo.DeleteByAuthFlowIDAndRoleID(authFlow.AuthFlowID, role.RoleID)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove role failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

// findClientURI resolves a client URI by UUID within the tenant.
func (s *authFlowService) findClientURI(tx *gorm.DB, clientURIUUID uuid.UUID, tenantID int64) (*ClientURI, error) {
	var cu ClientURI
	err := tx.Where("client_uri_uuid = ? AND tenant_id = ?", clientURIUUID, tenantID).First(&cu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cu, nil
}

// AssignCallbackURIs attaches one or more of the flow's client's registered URIs
// to the flow. Each URI must belong to the flow's client (the allowlist stays
// owned by the client). Already-attached URIs are skipped.
func (s *authFlowService) AssignCallbackURIs(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, clientURIUUIDs []uuid.UUID) ([]AuthFlowCallbackURIServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.assignCallbackURIs")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var assigned []AuthFlowCallbackURIServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthFlowRepo := s.authFlowRepo.WithTx(tx)
		txCallbackRepo := s.authFlowCallbackURIRepo.WithTx(tx)

		authFlow, err := txAuthFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID)
		if err != nil || authFlow == nil {
			return apperror.NewNotFoundWithReason("auth flow not found or access denied")
		}
		if authFlow.ClientID == nil {
			return apperror.NewValidation("auth flow has no client; attach a client before adding callback URIs")
		}

		for _, clientURIUUID := range clientURIUUIDs {
			cu, err := s.findClientURI(tx, clientURIUUID, tenantID)
			if err != nil {
				return err
			}
			if cu == nil {
				return apperror.NewNotFoundWithReason("client URI not found: " + clientURIUUID.String())
			}
			if cu.ClientID != *authFlow.ClientID {
				return apperror.NewValidation("callback URI must belong to the auth flow's client")
			}

			existing, err := txCallbackRepo.FindByAuthFlowIDAndClientURIID(authFlow.AuthFlowID, cu.ClientURIID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue
			}

			created, err := txCallbackRepo.Create(&AuthFlowCallbackURI{
				AuthFlowID:  authFlow.AuthFlowID,
				ClientURIID: cu.ClientURIID,
			})
			if err != nil {
				return err
			}

			assigned = append(assigned, AuthFlowCallbackURIServiceDataResult{
				AuthFlowCallbackURIUUID: created.AuthFlowCallbackURIUUID,
				AuthFlowUUID:            authFlow.AuthFlowUUID,
				ClientURIUUID:           cu.ClientURIUUID,
				URI:                     cu.URI,
				CreatedAt:               created.CreatedAt,
			})
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "assign callback URIs failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return assigned, nil
}

func (s *authFlowService) GetCallbackURIs(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, page, limit int) (*AuthFlowCallbackURIServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.getCallbackURIs")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	authFlow, err := s.authFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID)
	if err != nil || authFlow == nil {
		span.SetStatus(codes.Error, "auth flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("auth flow not found or access denied")
	}

	rows, total, err := s.authFlowCallbackURIRepo.FindByAuthFlowIDPaginated(authFlow.AuthFlowID, page, limit)
	if err != nil {
		return nil, err
	}

	data := make([]AuthFlowCallbackURIServiceDataResult, 0, len(rows))
	for _, row := range rows {
		item := AuthFlowCallbackURIServiceDataResult{
			AuthFlowCallbackURIUUID: row.AuthFlowCallbackURIUUID,
			AuthFlowUUID:            authFlow.AuthFlowUUID,
			CreatedAt:               row.CreatedAt,
		}
		if row.ClientURI != nil {
			item.ClientURIUUID = row.ClientURI.ClientURIUUID
			item.URI = row.ClientURI.URI
		}
		data = append(data, item)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	span.SetStatus(codes.Ok, "")
	return &AuthFlowCallbackURIServiceListResult{
		Data:       data,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (s *authFlowService) RemoveCallbackURI(ctx context.Context, authFlowUUID uuid.UUID, tenantID int64, clientURIUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "authFlow.removeCallbackURI")
	defer span.End()
	span.SetAttributes(attribute.String("authFlow.uuid", authFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthFlowRepo := s.authFlowRepo.WithTx(tx)
		txCallbackRepo := s.authFlowCallbackURIRepo.WithTx(tx)

		authFlow, err := txAuthFlowRepo.FindByUUIDAndTenantID(authFlowUUID, tenantID)
		if err != nil || authFlow == nil {
			return apperror.NewNotFoundWithReason("auth flow not found or access denied")
		}

		cu, err := s.findClientURI(tx, clientURIUUID, tenantID)
		if err != nil {
			return err
		}
		if cu == nil {
			return apperror.NewNotFound("client URI not found")
		}

		return txCallbackRepo.DeleteByAuthFlowIDAndClientURIID(authFlow.AuthFlowID, cu.ClientURIID)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove callback URI failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
