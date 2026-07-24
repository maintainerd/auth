package idp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RegistrationFlowServiceDataResult struct {
	RegistrationFlowUUID uuid.UUID
	Name                 string
	Description          string
	Status               string
	ClientUUID           *uuid.UUID
	ClientName           string
	ClientDisplayName    string
	ClientIdentifier     string
	ClientStatus         string
	VerificationRequired bool
	RequiredFields       datatypes.JSON
	IsSystem             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type RegistrationFlowServiceListResult struct {
	Data       []RegistrationFlowServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type RegistrationFlowRoleServiceDataResult struct {
	RegistrationFlowRoleUUID uuid.UUID
	RegistrationFlowUUID     uuid.UUID
	RoleUUID                 uuid.UUID
	RoleName                 string
	RoleDescription          string
	RoleStatus               string
	RoleIsDefault            bool
	RoleIsSystem             bool
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type RegistrationFlowRoleServiceListResult struct {
	Data       []RegistrationFlowRoleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

// RegistrationFlowServiceGetFilter is the list query contract.
type RegistrationFlowServiceGetFilter struct {
	TenantID   int64
	Name       *string
	Search     *string
	Status     []string
	ClientUUID *uuid.UUID
	IsSystem   *bool
	Page       int
	Limit      int
	SortBy     string
	SortOrder  string
}

// RegistrationFlowCreateInput is the create contract. Name doubles as the public
// registration-link selector, so it is slug-validated and tenant-unique.
type RegistrationFlowCreateInput struct {
	TenantID             int64
	ActorUserUUID        uuid.UUID
	Name                 string
	Description          string
	Status               string
	ClientUUID           uuid.UUID
	RoleUUIDs            []uuid.UUID
	VerificationRequired bool
	RequiredFields       *[]string
}

// RegistrationFlowUpdateInput is the update contract. Every pointer field
// carries omitted-means-unchanged semantics so a partial update cannot silently
// downgrade a security control.
type RegistrationFlowUpdateInput struct {
	RegistrationFlowUUID uuid.UUID
	TenantID             int64
	ActorUserUUID        uuid.UUID
	Name                 *string
	Description          *string
	Status               *string
	RoleUUIDs            []uuid.UUID
	VerificationRequired *bool
	RequiredFields       *[]string
}

type RegistrationFlowService interface {
	Get(ctx context.Context, filter RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error)
	GetByUUID(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64) (*RegistrationFlowServiceDataResult, error)
	Create(ctx context.Context, in RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error)
	Update(ctx context.Context, in RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error)
	SetStatus(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, status string) (*RegistrationFlowServiceDataResult, error)
	Delete(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error)
	AssignRoles(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, roleUUIDs []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error)
	GetRoles(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, page, limit int) (*RegistrationFlowRoleServiceListResult, error)
	RemoveRole(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, roleUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error)
}

type registrationFlowService struct {
	db                       *gorm.DB
	registrationFlowRepo     RegistrationFlowRepository
	registrationFlowRoleRepo RegistrationFlowRoleRepository
	roleRepo                 RoleRepository
	clientRepo               ClientRepository
	userRepo                 UserRepository
	userRoleRepo             UserRoleRepository
	inviteCounter            RegistrationFlowInviteCounter
	rolePermissionReader     RolePermissionNameReader
}

func NewRegistrationFlowService(
	db *gorm.DB,
	registrationFlowRepo RegistrationFlowRepository,
	registrationFlowRoleRepo RegistrationFlowRoleRepository,
	roleRepo RoleRepository,
	clientRepo ClientRepository,
	userRepo UserRepository,
	userRoleRepo UserRoleRepository,
	inviteCounter RegistrationFlowInviteCounter,
	rolePermissionReader RolePermissionNameReader,
) RegistrationFlowService {
	return &registrationFlowService{
		db:                       db,
		registrationFlowRepo:     registrationFlowRepo,
		registrationFlowRoleRepo: registrationFlowRoleRepo,
		roleRepo:                 roleRepo,
		clientRepo:               clientRepo,
		userRepo:                 userRepo,
		userRoleRepo:             userRoleRepo,
		inviteCounter:            inviteCounter,
		rolePermissionReader:     rolePermissionReader,
	}
}

// loadActorForTenant resolves the acting user and asserts they hold an identity
// in the target tenant. This is the defence-in-depth layer behind the
// middleware-supplied tenant: the tenant in context proves the request was
// routed for a tenant, not that this user belongs to it.
func (s *registrationFlowService) loadActorForTenant(tx *gorm.DB, actorUserUUID uuid.UUID, tenantID int64) (*User, error) {
	actorUser, err := s.userRepo.WithTx(tx).FindByUUID(actorUserUUID, "UserIdentities.Tenant")
	if err != nil || actorUser == nil {
		return nil, apperror.NewNotFoundWithReason("actor user not found")
	}
	tenant, err := s.tenantOf(tx, tenantID)
	if err != nil {
		return nil, err
	}
	if err := ValidateTenantAccess(actorUser, tenant); err != nil {
		return nil, err
	}
	return actorUser, nil
}

func (s *registrationFlowService) tenantOf(tx *gorm.DB, tenantID int64) (*Tenant, error) {
	var tenant Tenant
	if err := tx.Where("tenant_id = ?", tenantID).First(&tenant).Error; err != nil {
		return nil, apperror.NewNotFoundWithReason("tenant not found")
	}
	return &tenant, nil
}

// assertFlowMutable blocks every mutation of a system-managed flow. Seeded
// system flows carry privileged role grants; letting them be renamed,
// re-activated, or re-pointed at other roles through the ordinary admin API
// would be a durable backdoor.
func assertFlowMutable(flow *RegistrationFlow, action string) error {
	if flow.IsSystem {
		return apperror.NewValidation("system registration flow is not allowed to be " + action)
	}
	return nil
}

// assertRolesGrantable caps what a registration flow may hand out.
//
// A registration flow is redeemed from a PUBLIC link whose selector is the flow
// name — deliberately readable, therefore guessable. Nothing about the link is
// secret, so the flow itself must not be capable of granting privilege that a
// stranger should not be able to claim by typing a plausible name. Four caps:
//
//  1. system roles can never be attached;
//  2. the role must be active — an inactive role must not be revived by signup;
//  3. the role must carry NO management-plane permission (shared.IsElevatedPermission).
//     Privileged onboarding goes through an invite, which is signed, single-use
//     and email-bound. This is the cap that makes a guessable selector safe;
//  4. the acting admin may only grant roles they themselves possess — without
//     it, `registration-flow:update` would confer strictly more power than
//     `user:invite`, which is the real escalation path.
func (s *registrationFlowService) assertRolesGrantable(tx *gorm.DB, actorUser *User, roles []*Role) error {
	txUserRoleRepo := s.userRoleRepo.WithTx(tx)
	txPermReader := s.rolePermissionReader.WithTx(tx)

	for _, role := range roles {
		if role.IsSystem {
			return apperror.NewForbidden("system roles cannot be assigned to a registration flow")
		}
		if role.Status != shared.StatusActive {
			return apperror.NewValidation("role \"" + role.Name + "\" is inactive and cannot be assigned to a registration flow")
		}

		permNames, err := txPermReader.FindPermissionNamesByRoleID(role.RoleID)
		if err != nil {
			return err
		}
		if elevated := shared.FirstElevatedPermission(permNames); elevated != "" {
			return apperror.NewForbidden(
				"role \"" + role.Name + "\" grants the administrative permission \"" + elevated +
					"\" and cannot be attached to a self-service registration flow; use an invite instead",
			)
		}

		held, err := txUserRoleRepo.FindByUserIDAndRoleID(actorUser.UserID, role.RoleID)
		if err != nil {
			return err
		}
		if held == nil {
			return apperror.NewValidation("you cannot grant roles you do not possess")
		}
	}
	return nil
}

// resolveGrantableRoles loads each role, asserts same-tenant ownership, and
// applies the grantable cap.
func (s *registrationFlowService) resolveGrantableRoles(tx *gorm.DB, actorUser *User, tenantID int64, roleUUIDs []uuid.UUID) ([]*Role, error) {
	txRoleRepo := s.roleRepo.WithTx(tx)
	roles := make([]*Role, 0, len(roleUUIDs))
	for _, ru := range roleUUIDs {
		role, err := txRoleRepo.FindByUUID(ru)
		if err != nil || role == nil {
			return nil, apperror.NewNotFoundWithReason("role not found: " + ru.String())
		}
		if role.TenantID != tenantID {
			return nil, apperror.NewForbidden("role does not belong to the same tenant as the registration flow")
		}
		roles = append(roles, role)
	}
	if err := s.assertRolesGrantable(tx, actorUser, roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// syncRoles replaces the flow's role membership to exactly roleUUIDs (adds
// missing, removes extra). Runs inside the supplied transaction.
func (s *registrationFlowService) syncRoles(tx *gorm.DB, actorUser *User, registrationFlow *RegistrationFlow, roleUUIDs []uuid.UUID) error {
	txAFRRepo := s.registrationFlowRoleRepo.WithTx(tx)

	roles, err := s.resolveGrantableRoles(tx, actorUser, registrationFlow.TenantID, roleUUIDs)
	if err != nil {
		return err
	}

	desired := make(map[int64]bool, len(roles))
	for _, role := range roles {
		desired[role.RoleID] = true
	}

	existing, err := txAFRRepo.FindByRegistrationFlowID(registrationFlow.RegistrationFlowID)
	if err != nil {
		return err
	}
	existingSet := make(map[int64]bool, len(existing))
	for _, e := range existing {
		existingSet[e.RoleID] = true
		if !desired[e.RoleID] {
			if err := txAFRRepo.DeleteByRegistrationFlowIDAndRoleID(registrationFlow.RegistrationFlowID, e.RoleID); err != nil {
				return err
			}
		}
	}
	for _, role := range roles {
		if !existingSet[role.RoleID] {
			if _, err := txAFRRepo.Create(&RegistrationFlowRole{RegistrationFlowID: registrationFlow.RegistrationFlowID, RoleID: role.RoleID}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *registrationFlowService) Get(ctx context.Context, filter RegistrationFlowServiceGetFilter) (*RegistrationFlowServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))

	var clientID *int64
	if filter.ClientUUID != nil {
		client, err := s.clientRepo.FindByUUID(*filter.ClientUUID)
		if err != nil || client == nil {
			return nil, apperror.NewNotFoundWithReason("auth client not found")
		}
		if client.TenantID != filter.TenantID {
			return nil, apperror.NewForbidden("client does not belong to your tenant")
		}
		clientID = &client.ClientID
	}

	repoFilter := RegistrationFlowRepositoryGetFilter{
		Name:      filter.Name,
		Search:    filter.Search,
		Status:    filter.Status,
		TenantID:  &filter.TenantID,
		ClientID:  clientID,
		IsSystem:  filter.IsSystem,
		Page:      filter.Page,
		Limit:     filter.Limit,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
	}

	result, err := s.registrationFlowRepo.FindPaginated(repoFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list registration flows failed")
		return nil, err
	}

	data := make([]RegistrationFlowServiceDataResult, len(result.Data))
	for i := range result.Data {
		data[i] = *toRegistrationFlowServiceDataResult(&result.Data[i])
	}

	span.SetStatus(codes.Ok, "")
	return &RegistrationFlowServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *registrationFlowService) GetByUUID(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	registrationFlow, err := s.registrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID, "Client")
	if err != nil || registrationFlow == nil {
		span.SetStatus(codes.Error, "registration flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("registration flow not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toRegistrationFlowServiceDataResult(registrationFlow), nil
}

func (s *registrationFlowService) Create(ctx context.Context, in RegistrationFlowCreateInput) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", in.TenantID))

	var createdRegistrationFlow *RegistrationFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		actorUser, err := s.loadActorForTenant(tx, in.ActorUserUUID, in.TenantID)
		if err != nil {
			return err
		}

		// Find auth client
		client, err := txClientRepo.FindByUUID(in.ClientUUID)
		if err != nil || client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}
		if client.TenantID != in.TenantID {
			return apperror.NewForbidden("client does not belong to your tenant")
		}
		if client.Status != shared.StatusActive {
			return apperror.NewValidation("auth client is inactive or deleted")
		}

		// Check if name already exists
		existingName, err := txRegistrationFlowRepo.FindByNameAndTenantID(in.Name, in.TenantID)
		if err != nil {
			return err
		}
		if existingName != nil {
			return apperror.NewConflict("registration flow with this name already exists")
		}

		// Create registration flow
		registrationFlow := &RegistrationFlow{
			TenantID:             in.TenantID,
			Name:                 in.Name,
			Description:          in.Description,
			Status:               in.Status,
			ClientID:             client.ClientID,
			VerificationRequired: in.VerificationRequired,
			RequiredFields:       requiredFieldsToJSON(in.RequiredFields),
			IsSystem:             false,
			CreatedBy:            &actorUser.UserID,
			UpdatedBy:            &actorUser.UserID,
		}

		created, err := txRegistrationFlowRepo.Create(registrationFlow)
		if err != nil {
			return err
		}

		if len(in.RoleUUIDs) > 0 {
			if err := s.syncRoles(tx, actorUser, created, in.RoleUUIDs); err != nil {
				return err
			}
		}

		createdRegistrationFlow = created
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create registration flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, createdRegistrationFlow.RegistrationFlowUUID, in.TenantID)
}

func (s *registrationFlowService) Update(ctx context.Context, in RegistrationFlowUpdateInput) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.update")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", in.RegistrationFlowUUID.String()), attribute.Int64("tenant.id", in.TenantID))

	var updatedRegistrationFlow *RegistrationFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)

		// Find existing registration flow and validate tenant ownership
		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(in.RegistrationFlowUUID, in.TenantID)
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		actorUser, err := s.loadActorForTenant(tx, in.ActorUserUUID, in.TenantID)
		if err != nil {
			return err
		}

		if err := assertFlowMutable(registrationFlow, "updated"); err != nil {
			return err
		}

		// Check if name is being changed and if it conflicts
		if in.Name != nil && *in.Name != registrationFlow.Name {
			existingName, err := txRegistrationFlowRepo.FindByNameAndTenantID(*in.Name, in.TenantID)
			if err != nil {
				return err
			}
			if existingName != nil && existingName.RegistrationFlowID != registrationFlow.RegistrationFlowID {
				return apperror.NewConflict("registration flow with this name already exists")
			}
		}

		// Omitted means unchanged. Renaming changes the flow's public
		// registration link, which is why the name is slug-validated.
		if in.Name != nil {
			registrationFlow.Name = *in.Name
		}
		if in.Description != nil {
			registrationFlow.Description = *in.Description
		}
		if in.Status != nil {
			registrationFlow.Status = *in.Status
		}
		if in.VerificationRequired != nil {
			registrationFlow.VerificationRequired = *in.VerificationRequired
		}
		if in.RequiredFields != nil {
			registrationFlow.RequiredFields = requiredFieldsToJSON(in.RequiredFields)
		}
		registrationFlow.UpdatedBy = &actorUser.UserID

		updated, err := txRegistrationFlowRepo.CreateOrUpdate(registrationFlow)
		if err != nil {
			return err
		}

		// nil = leave membership untouched; non-nil (incl. empty) = replace it.
		if in.RoleUUIDs != nil {
			if err := s.syncRoles(tx, actorUser, updated, in.RoleUUIDs); err != nil {
				return err
			}
		}

		updatedRegistrationFlow = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update registration flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, updatedRegistrationFlow.RegistrationFlowUUID, in.TenantID)
}

func (s *registrationFlowService) SetStatus(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, status string) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedRegistrationFlow *RegistrationFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)

		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		actorUser, err := s.loadActorForTenant(tx, actorUserUUID, tenantID)
		if err != nil {
			return err
		}

		if err := assertFlowMutable(registrationFlow, "updated"); err != nil {
			return err
		}

		registrationFlow.Status = status
		registrationFlow.UpdatedBy = &actorUser.UserID

		updated, err := txRegistrationFlowRepo.CreateOrUpdate(registrationFlow)
		if err != nil {
			return err
		}

		updatedRegistrationFlow = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update registration flow status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, updatedRegistrationFlow.RegistrationFlowUUID, tenantID)
}

func (s *registrationFlowService) Delete(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.delete")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var result *RegistrationFlowServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)
		txRegistrationFlowRoleRepo := s.registrationFlowRoleRepo.WithTx(tx)

		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID, "Client")
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		if _, err := s.loadActorForTenant(tx, actorUserUUID, tenantID); err != nil {
			return err
		}

		if err := assertFlowMutable(registrationFlow, "deleted"); err != nil {
			return err
		}

		// Guard: a flow still referenced by pending invites cannot be deleted
		// out from under them. Inside the transaction so the count cannot go
		// stale between the check and the delete.
		pendingCount, err := s.inviteCounter.WithTx(tx).CountPendingByRegistrationFlowID(registrationFlow.RegistrationFlowID)
		if err != nil {
			return err
		}
		if pendingCount > 0 {
			return apperror.NewConflict("cannot delete registration flow that is referenced by pending invites")
		}

		result = toRegistrationFlowServiceDataResult(registrationFlow)

		// registration_flows is soft-deleted, which does NOT fire the FK
		// ON DELETE CASCADE, so the role membership must be cleared explicitly
		// or it would outlive the flow.
		if err := txRegistrationFlowRoleRepo.DeleteByRegistrationFlowID(registrationFlow.RegistrationFlowID); err != nil {
			return err
		}

		return txRegistrationFlowRepo.DeleteByUUID(registrationFlowUUID)
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete registration flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func requiredFieldsToJSON(fields *[]string) datatypes.JSON {
	if fields == nil {
		return datatypes.JSON([]byte("[]"))
	}
	normalized := make([]string, 0, len(*fields))
	for _, f := range *fields {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(f)))
	}
	b, err := json.Marshal(normalized)
	if err != nil {
		return datatypes.JSON([]byte("[]"))
	}
	return datatypes.JSON(b)
}

func toRegistrationFlowServiceDataResult(sf *RegistrationFlow) *RegistrationFlowServiceDataResult {
	if sf == nil {
		return nil
	}

	out := &RegistrationFlowServiceDataResult{
		RegistrationFlowUUID: sf.RegistrationFlowUUID,
		Name:                 sf.Name,
		Description:          sf.Description,
		Status:               sf.Status,
		VerificationRequired: sf.VerificationRequired,
		RequiredFields:       sf.RequiredFields,
		IsSystem:             sf.IsSystem,
		CreatedAt:            sf.CreatedAt,
		UpdatedAt:            sf.UpdatedAt,
	}

	if len(out.RequiredFields) == 0 {
		out.RequiredFields = datatypes.JSON([]byte("[]"))
	}

	if sf.Client != nil {
		clientUUID := sf.Client.ClientUUID
		out.ClientUUID = &clientUUID
		out.ClientName = sf.Client.Name
		out.ClientDisplayName = sf.Client.DisplayName
		out.ClientStatus = sf.Client.Status
		if sf.Client.Identifier != nil {
			out.ClientIdentifier = *sf.Client.Identifier
		}
	}

	return out
}

func (s *registrationFlowService) AssignRoles(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, roleUUIDs []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.assignRoles")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var assignedRoles []RegistrationFlowRoleServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)
		txRegistrationFlowRoleRepo := s.registrationFlowRoleRepo.WithTx(tx)

		// Verify registration flow exists and belongs to tenant
		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		actorUser, err := s.loadActorForTenant(tx, actorUserUUID, tenantID)
		if err != nil {
			return err
		}

		if err := assertFlowMutable(registrationFlow, "modified"); err != nil {
			return err
		}

		roles, err := s.resolveGrantableRoles(tx, actorUser, registrationFlow.TenantID, roleUUIDs)
		if err != nil {
			return err
		}

		assignedRoles = nil
		for _, role := range roles {
			// Check if already assigned
			existing, err := txRegistrationFlowRoleRepo.FindByRegistrationFlowIDAndRoleID(registrationFlow.RegistrationFlowID, role.RoleID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue // Skip if already assigned
			}

			created, err := txRegistrationFlowRoleRepo.Create(&RegistrationFlowRole{
				RegistrationFlowID: registrationFlow.RegistrationFlowID,
				RoleID:             role.RoleID,
			})
			if err != nil {
				return err
			}

			assignedRoles = append(assignedRoles, RegistrationFlowRoleServiceDataResult{
				RegistrationFlowRoleUUID: created.RegistrationFlowRoleUUID,
				RegistrationFlowUUID:     registrationFlow.RegistrationFlowUUID,
				RoleUUID:                 role.RoleUUID,
				RoleName:                 role.Name,
				RoleDescription:          role.Description,
				RoleStatus:               role.Status,
				RoleIsDefault:            role.IsDefault,
				RoleIsSystem:             role.IsSystem,
				CreatedAt:                created.CreatedAt,
				UpdatedAt:                role.UpdatedAt,
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
	if assignedRoles == nil {
		assignedRoles = []RegistrationFlowRoleServiceDataResult{}
	}
	return assignedRoles, nil
}

func (s *registrationFlowService) GetRoles(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, page, limit int) (*RegistrationFlowRoleServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.getRoles")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Verify registration flow exists and belongs to tenant
	registrationFlow, err := s.registrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
	if err != nil || registrationFlow == nil {
		span.SetStatus(codes.Error, "registration flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("registration flow not found or access denied")
	}

	result, err := s.registrationFlowRoleRepo.FindByRegistrationFlowIDPaginated(registrationFlow.RegistrationFlowID, page, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list registration flow roles failed")
		return nil, err
	}

	roles := make([]RegistrationFlowRoleServiceDataResult, 0, len(result.Data))
	for i := range result.Data {
		sfr := result.Data[i]
		if sfr.Role == nil {
			continue
		}
		roles = append(roles, RegistrationFlowRoleServiceDataResult{
			RegistrationFlowRoleUUID: sfr.RegistrationFlowRoleUUID,
			RegistrationFlowUUID:     registrationFlow.RegistrationFlowUUID,
			RoleUUID:                 sfr.Role.RoleUUID,
			RoleName:                 sfr.Role.Name,
			RoleDescription:          sfr.Role.Description,
			RoleStatus:               sfr.Role.Status,
			RoleIsDefault:            sfr.Role.IsDefault,
			RoleIsSystem:             sfr.Role.IsSystem,
			CreatedAt:                sfr.CreatedAt,
			UpdatedAt:                sfr.Role.UpdatedAt,
		})
	}

	span.SetStatus(codes.Ok, "")
	return &RegistrationFlowRoleServiceListResult{
		Data:       roles,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *registrationFlowService) RemoveRole(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, roleUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.removeRole")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)
		txRegistrationFlowRoleRepo := s.registrationFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify registration flow exists and belongs to tenant
		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		if _, err := s.loadActorForTenant(tx, actorUserUUID, tenantID); err != nil {
			return err
		}

		if err := assertFlowMutable(registrationFlow, "modified"); err != nil {
			return err
		}

		// Verify role exists
		role, err := txRoleRepo.FindByUUID(roleUUID)
		if err != nil || role == nil {
			return apperror.NewNotFound("role not found")
		}
		if role.TenantID != registrationFlow.TenantID {
			return apperror.NewNotFoundWithReason("role not found: " + roleUUID.String())
		}

		return txRegistrationFlowRoleRepo.DeleteByRegistrationFlowIDAndRoleID(registrationFlow.RegistrationFlowID, role.RoleID)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove role failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, registrationFlowUUID, tenantID)
}
