package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type TenantMemberServiceListFilter struct {
	TenantID  int64
	Role      *string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type TenantMemberServiceListResult struct {
	Data       []TenantMemberServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type TenantMemberServiceDataResult struct {
	TenantMemberUUID uuid.UUID
	TenantID         int64
	UserID           int64
	Role             string
	User             *MemberUser
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TenantMemberService interface {
	Create(ctx context.Context, tenantID int64, userID int64, role string, actorUserID int64) (*TenantMemberServiceDataResult, error)
	CreateByUserUUID(ctx context.Context, tenantID int64, userUUID uuid.UUID, role string, actorUserID int64) (*TenantMemberServiceDataResult, error)
	GetByUUID(ctx context.Context, tenantMemberUUID uuid.UUID) (*TenantMemberServiceDataResult, error)
	GetByTenantAndUser(ctx context.Context, tenantID int64, userID int64) (*TenantMemberServiceDataResult, error)
	ListByTenant(ctx context.Context, filter TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error)
	ListByUser(ctx context.Context, userID int64) ([]TenantMemberServiceDataResult, error)
	UpdateRole(ctx context.Context, tenantID int64, tenantMemberUUID uuid.UUID, role string, actorUserID int64) (*TenantMemberServiceDataResult, error)
	DeleteByUUID(ctx context.Context, tenantID int64, tenantMemberUUID uuid.UUID, actorUserID int64) error
	ResolveUserID(ctx context.Context, userUUID uuid.UUID) (int64, error)
	IsUserInTenant(ctx context.Context, userID int64, tenantUUID uuid.UUID) (bool, error)
	// CanManageTenant reports whether the user may perform tenant-management
	// operations (update, members) on the target tenant: true when the user is a
	// member of that tenant OR a member of the system tenant (the system-tenant
	// override, scoped to tenant-management only).
	CanManageTenant(ctx context.Context, userID int64, tenantUUID uuid.UUID) (bool, error)
}

type tenantMemberService struct {
	tenantMemberRepo TenantMemberRepository
	userRepo         UserReader
	userProvisioner  UserProvisioner
	tenantRepo       TenantRepository
	uow              UnitOfWork
	eventService     event.EventService
}

func NewTenantMemberService(tenantMemberRepo TenantMemberRepository, userRepo UserReader, tenantRepo TenantRepository, uow UnitOfWork, eventService event.EventService, userProvisioner ...UserProvisioner) TenantMemberService {
	if uow == nil {
		uow = newDirectUnitOfWork(tenantRepo, tenantMemberRepo)
	}
	var up UserProvisioner
	if len(userProvisioner) > 0 {
		up = userProvisioner[0]
	}
	return &tenantMemberService{
		tenantMemberRepo: tenantMemberRepo,
		userRepo:         userRepo,
		userProvisioner:  up,
		tenantRepo:       tenantRepo,
		uow:              uow,
		eventService:     eventService,
	}
}

func (s *tenantMemberService) Create(ctx context.Context, tenantID int64, userID int64, role string, actorUserID int64) (*TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID), attribute.Int64("user.id", userID))

	_, actorIsSystem, err := s.authorizeManager(tenantID, actorUserID)
	if err != nil {
		return nil, err
	}
	if role == shared.TenantRoleOwner && !actorIsSystem {
		return nil, apperror.NewForbidden("only system tenant administrators can assign a tenant owner")
	}
	if !actorIsSystem {
		actorMember, aErr := s.tenantMemberRepo.FindByTenantAndUser(tenantID, actorUserID)
		if aErr != nil {
			return nil, apperror.NewInternal("failed to verify actor role", aErr)
		}
		if actorMember == nil || actorMember.Role != shared.TenantRoleOwner {
			return nil, apperror.NewForbidden("only system tenant administrators or the tenant owner can manage members")
		}
	}

	var created *TenantMember
	err = s.uow.Do(ctx, func(tx Transaction) error {
		repo := tx.TenantMemberRepository()
		if role == shared.TenantRoleOwner {
			tenantRecord, err := tx.TenantRepository().FindByID(tenantID)
			if err != nil {
				return err
			}
			if tenantRecord == nil {
				return apperror.NewNotFound("tenant not found")
			}
			if tenantRecord.IsSystem {
				return apperror.NewValidation("system tenant ownership can only be established during initial setup")
			}
			existingOwner, err := repo.FindOwnerByTenantID(tenantID)
			if err != nil {
				return apperror.NewInternal("failed to check existing owner", err)
			}
			if existingOwner != nil {
				return apperror.NewConflict("tenant already has an owner — transfer ownership instead")
			}
		}
		tu := &TenantMember{
			TenantID: tenantID,
			UserID:   userID,
			Role:     role,
		}
		var err error
		created, err = repo.Create(tu)
		if err != nil {
			return err
		}
		if role == shared.TenantRoleOwner && s.userProvisioner != nil {
			if err := s.userProvisioner.GrantRoleByName(ctx, tx.Tx(), userID, tenantID, shared.RoleSuperAdmin); err != nil {
				return apperror.NewInternal("failed to assign super-admin role to owner", err)
			}
		}
		if role == shared.TenantRoleOwner {
			tenantRecord, err := tx.TenantRepository().FindByID(tenantID)
			if err != nil {
				return err
			}
			if tenantRecord == nil {
				return apperror.NewNotFound("tenant not found")
			}
			if !tenantRecord.IsCompleted {
				tenantRecord.IsCompleted = true
				if _, err = tx.TenantRepository().CreateOrUpdate(tenantRecord); err != nil {
					return err
				}
			}
		}

		// Emit tenant_member.added inside the transaction
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx.Tx(), event.NewIntegrationEvent(
				event.EventTypeTenantMemberAdded, 1, tenantID,
			).SetSubject(&created.TenantMemberUUID, "tenant_member")); emitErr != nil {
				return emitErr
			}
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create tenant member failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return toTenantMemberServiceDataResult(created), nil
}

func (s *tenantMemberService) CreateByUserUUID(ctx context.Context, tenantID int64, userUUID uuid.UUID, role string, actorUserID int64) (*TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.createByUserUUID")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID), attribute.String("user.uuid", userUUID.String()))

	_, actorIsSystem, err := s.authorizeManager(tenantID, actorUserID)
	if err != nil {
		return nil, err
	}
	if role == shared.TenantRoleOwner && !actorIsSystem {
		return nil, apperror.NewForbidden("only system tenant administrators can assign a tenant owner")
	}
	if !actorIsSystem {
		actorMember, aErr := s.tenantMemberRepo.FindByTenantAndUser(tenantID, actorUserID)
		if aErr != nil {
			return nil, apperror.NewInternal("failed to verify actor role", aErr)
		}
		if actorMember == nil || actorMember.Role != shared.TenantRoleOwner {
			return nil, apperror.NewForbidden("only system tenant administrators or the tenant owner can manage members")
		}
	}

	// First get the user to retrieve the user_id
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewNotFound("user not found")
	}

	// Ensure the user has a record in the target tenant (copy credentials if needed)
	userID := user.UserID
	if s.userProvisioner != nil {
		provisionedID, err := s.userProvisioner.EnsureUserInTenant(ctx, userUUID, tenantID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "user provisioning failed")
			return nil, err
		}
		userID = provisionedID
	}

	// Check if user is already a member of this tenant
	existing, err := s.tenantMemberRepo.FindByTenantAndUser(tenantID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "tenant member duplicate check failed")
		return nil, apperror.NewInternal("failed to check tenant membership", err)
	}
	if existing != nil {
		span.SetStatus(codes.Error, "user already a member of this tenant")
		return nil, apperror.NewConflict("user is already a member of this tenant")
	}

	result, err := s.Create(ctx, tenantID, userID, role, actorUserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create tenant member failed")
		return nil, err
	}

	// Populate user information in the result
	result.User = user

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *tenantMemberService) GetByUUID(ctx context.Context, tenantMemberUUID uuid.UUID) (*TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("tenantMember.uuid", tenantMemberUUID.String()))

	tu, err := s.tenantMemberRepo.FindByTenantMemberUUID(tenantMemberUUID)
	if err != nil || tu == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant member not found")
		return nil, apperror.NewNotFoundWithReason("tenant member not found")
	}
	span.SetStatus(codes.Ok, "")
	return toTenantMemberServiceDataResult(tu), nil
}

func (s *tenantMemberService) GetByTenantAndUser(ctx context.Context, tenantID int64, userID int64) (*TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.getByTenantAndUser")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID), attribute.Int64("user.id", userID))

	tu, err := s.tenantMemberRepo.FindByTenantAndUser(tenantID, userID)
	if err != nil || tu == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant member not found")
		return nil, apperror.NewNotFoundWithReason("tenant member not found")
	}
	span.SetStatus(codes.Ok, "")
	return toTenantMemberServiceDataResult(tu), nil
}

func (s *tenantMemberService) ListByTenant(ctx context.Context, filter TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.listByTenant")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))

	tus, err := s.tenantMemberRepo.FindByTenant(TenantMemberRepositoryListFilter(filter))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list tenant members failed")
		return nil, err
	}

	result := make([]TenantMemberServiceDataResult, len(tus.Data))
	for i, tu := range tus.Data {
		dr := toTenantMemberServiceDataResult(&tu)

		// Fetch user information
		user, err := s.userRepo.FindByID(tu.UserID)
		if err == nil && user != nil {
			dr.User = user
		}

		result[i] = *dr
	}
	span.SetStatus(codes.Ok, "")
	return &TenantMemberServiceListResult{
		Data:       result,
		Total:      tus.Total,
		Page:       tus.Page,
		Limit:      tus.Limit,
		TotalPages: tus.TotalPages,
	}, nil
}

func (s *tenantMemberService) ListByUser(ctx context.Context, userID int64) ([]TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.listByUser")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	tus, err := s.tenantMemberRepo.FindAllByUser(userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list user tenant memberships failed")
		return nil, err
	}

	result := make([]TenantMemberServiceDataResult, len(tus))
	for i, tu := range tus {
		result[i] = *toTenantMemberServiceDataResult(&tu)
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *tenantMemberService) UpdateRole(ctx context.Context, tenantID int64, tenantMemberUUID uuid.UUID, role string, actorUserID int64) (*TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.updateRole")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID), attribute.String("tenantMember.uuid", tenantMemberUUID.String()))

	_, actorIsSystem, err := s.authorizeManager(tenantID, actorUserID)
	if err != nil {
		return nil, err
	}

	var updated *TenantMember
	err = s.uow.Do(ctx, func(tx Transaction) error {
		repo := tx.TenantMemberRepository()
		tu, err := repo.FindByTenantMemberUUID(tenantMemberUUID)
		if err != nil {
			return err
		}
		if tu == nil {
			return apperror.NewNotFoundWithReason("tenant member not found")
		}
		if tu.TenantID != tenantID {
			return apperror.NewNotFoundWithReason("tenant member not found")
		}

		if tu.Role == shared.TenantRoleOwner && role != shared.TenantRoleOwner {
			return apperror.NewValidation("cannot demote a tenant owner directly — transfer ownership instead")
		}

		if role == shared.TenantRoleOwner && tu.Role != shared.TenantRoleOwner {
			if !actorIsSystem {
				return apperror.NewForbidden("only system tenant administrators can transfer tenant ownership")
			}
			tenantRecord, tErr := tx.TenantRepository().FindByID(tenantID)
			if tErr != nil {
				return tErr
			}
			if tenantRecord == nil {
				return apperror.NewNotFound("tenant not found")
			}
			if tenantRecord.IsSystem {
				return apperror.NewValidation("cannot transfer ownership of the system tenant")
			}
			if s.userProvisioner != nil {
				ownerUser, uErr := s.userRepo.FindByID(tu.UserID)
				if uErr != nil || ownerUser == nil {
					return apperror.NewInternal("failed to resolve user for ownership transfer", uErr)
				}
				sysTenant, sErr := s.tenantRepo.FindSystem()
				if sErr != nil {
					return apperror.NewInternal("failed to resolve system tenant for provisioning", sErr)
				}
				if sysTenant != nil {
					if _, pErr := s.userProvisioner.EnsureUserInTenant(ctx, ownerUser.UserUUID, sysTenant.TenantID); pErr != nil {
						return apperror.NewInternal("failed to provision new owner in system tenant", pErr)
					}
				}
			}
			existingOwner, oErr := repo.FindOwnerByTenantID(tenantID)
			if oErr != nil {
				return apperror.NewInternal("failed to check existing owner", oErr)
			}
			if existingOwner != nil {
				existingOwner.Role = shared.TenantRoleMember
				if _, oErr = repo.CreateOrUpdate(existingOwner); oErr != nil {
					return oErr
				}
				if s.userProvisioner != nil {
					if oErr = s.userProvisioner.RevokeRoleByName(ctx, tx.Tx(), existingOwner.UserID, tenantID, shared.RoleSuperAdmin); oErr != nil {
						return apperror.NewInternal("failed to revoke super-admin role from previous owner", oErr)
					}
				}
			}
			if s.userProvisioner != nil {
				if oErr = s.userProvisioner.GrantRoleByName(ctx, tx.Tx(), tu.UserID, tenantID, shared.RoleSuperAdmin); oErr != nil {
					return apperror.NewInternal("failed to assign super-admin role to new owner", oErr)
				}
			}
		}

		tu.Role = role
		updated, err = repo.CreateOrUpdate(tu)
		return err
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update tenant member role failed")
		return nil, err
	}

	result := toTenantMemberServiceDataResult(updated)

	// Fetch and populate user information
	user, err := s.userRepo.FindByID(updated.UserID)
	if err == nil && user != nil {
		result.User = user
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *tenantMemberService) DeleteByUUID(ctx context.Context, tenantID int64, tenantMemberUUID uuid.UUID, actorUserID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.delete")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID), attribute.String("tenantMember.uuid", tenantMemberUUID.String()))

	_, actorIsSystem, err := s.authorizeManager(tenantID, actorUserID)
	if err != nil {
		return err
	}
	if !actorIsSystem {
		actorMember, aErr := s.tenantMemberRepo.FindByTenantAndUser(tenantID, actorUserID)
		if aErr != nil {
			return apperror.NewInternal("failed to verify actor role", aErr)
		}
		if actorMember == nil || actorMember.Role != shared.TenantRoleOwner {
			return apperror.NewForbidden("only system tenant administrators or the tenant owner can manage members")
		}
	}

	tu, err := s.tenantMemberRepo.FindByTenantMemberUUID(tenantMemberUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find tenant member failed")
		return err
	}
	if tu == nil {
		span.SetStatus(codes.Error, "tenant member not found")
		return apperror.NewNotFoundWithReason("tenant member not found")
	}
	if tu.TenantID != tenantID {
		span.SetStatus(codes.Error, "tenant member not found")
		return apperror.NewNotFoundWithReason("tenant member not found")
	}

	if tu.Role == shared.TenantRoleOwner {
		return apperror.NewValidation("cannot remove a tenant owner directly — transfer ownership first")
	}

	err = s.uow.Do(ctx, func(tx Transaction) error {
		repo := tx.TenantMemberRepository()
		if err := repo.DeleteByUUID(tenantMemberUUID); err != nil {
			return err
		}

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx.Tx(), event.NewIntegrationEvent(
				event.EventTypeTenantMemberRemoved, 1, tenantID,
			).SetSubject(&tenantMemberUUID, "tenant_member")); emitErr != nil {
				return emitErr
			}
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete tenant member failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *tenantMemberService) ResolveUserID(_ context.Context, userUUID uuid.UUID) (int64, error) {
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil {
		return 0, err
	}
	if user == nil {
		return 0, apperror.NewNotFound("user not found")
	}
	return user.UserID, nil
}

func (s *tenantMemberService) authorizeManager(tenantID, actorUserID int64) (bool, bool, error) {
	if actorUserID <= 0 {
		return false, false, apperror.NewForbidden("actor user is required")
	}
	targetMember, err := s.tenantMemberRepo.FindByTenantAndUser(tenantID, actorUserID)
	if err != nil {
		return false, false, apperror.NewInternal("failed to verify tenant membership", err)
	}
	systemTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return false, false, apperror.NewInternal("failed to resolve system tenant", err)
	}
	var systemMember *TenantMember
	if systemTenant != nil {
		systemMember, err = s.tenantMemberRepo.FindByTenantAndUser(systemTenant.TenantID, actorUserID)
		if err != nil {
			return false, false, apperror.NewInternal("failed to verify system tenant membership", err)
		}
	}
	if targetMember == nil && systemMember == nil {
		return false, false, apperror.NewForbidden("actor cannot manage this tenant")
	}
	return targetMember != nil, systemMember != nil, nil
}

// IsUserInTenant checks if a user is a member of the specified tenant
func (s *tenantMemberService) IsUserInTenant(ctx context.Context, userID int64, tenantUUID uuid.UUID) (bool, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.isUserInTenant")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID), attribute.String("tenant.uuid", tenantUUID.String()))

	// First get the tenant to retrieve tenant_id
	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant not found")
		return false, apperror.NewNotFound("tenant not found")
	}

	// Check if user is in tenant_members
	tenantMember, err := s.tenantMemberRepo.FindByTenantAndUser(tenant.TenantID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "check user in tenant failed")
		return false, err
	}

	span.SetStatus(codes.Ok, "")
	return tenantMember != nil, nil
}

// CanManageTenant reports whether the user may manage the target tenant. Access
// is granted when the user is a member of the target tenant OR a member of the
// system tenant. The latter is the system-tenant override, deliberately scoped
// to tenant-management operations only (create/update/members) — it does NOT
// grant access to other tenants' non-tenant records (users, roles, clients,
// idps), which each tenant's own members administer.
func (s *tenantMemberService) CanManageTenant(ctx context.Context, userID int64, tenantUUID uuid.UUID) (bool, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.canManageTenant")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID), attribute.String("tenant.uuid", tenantUUID.String()))

	target, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || target == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant not found")
		return false, apperror.NewNotFound("tenant not found")
	}

	// Member of the target tenant?
	member, err := s.tenantMemberRepo.FindByTenantAndUser(target.TenantID, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "check target membership failed")
		return false, err
	}
	if member != nil {
		span.SetStatus(codes.Ok, "")
		return true, nil
	}

	// System-tenant override: members of the system tenant may manage any tenant.
	systemTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find system tenant failed")
		return false, err
	}
	if systemTenant != nil && systemTenant.TenantID != target.TenantID {
		systemMember, err := s.tenantMemberRepo.FindByTenantAndUser(systemTenant.TenantID, userID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "check system membership failed")
			return false, err
		}
		if systemMember != nil {
			span.SetStatus(codes.Ok, "")
			return true, nil
		}
	}

	span.SetStatus(codes.Ok, "")
	return false, nil
}

func toTenantMemberServiceDataResult(tu *TenantMember) *TenantMemberServiceDataResult {
	return &TenantMemberServiceDataResult{
		TenantMemberUUID: tu.TenantMemberUUID,
		TenantID:         tu.TenantID,
		UserID:           tu.UserID,
		Role:             tu.Role,
		CreatedAt:        tu.CreatedAt,
		UpdatedAt:        tu.UpdatedAt,
	}
}
