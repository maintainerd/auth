package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type TenantMemberServiceDataResult struct {
	TenantMemberUUID uuid.UUID
	TenantID         int64
	UserID           int64
	Role             string
	User             *UserServiceDataResult
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TenantMemberService interface {
	Create(ctx context.Context, tenantID int64, userID int64, role string) (*TenantMemberServiceDataResult, error)
	CreateByUserUUID(ctx context.Context, tenantID int64, userUUID uuid.UUID, role string) (*TenantMemberServiceDataResult, error)
	GetByUUID(ctx context.Context, tenantMemberUUID uuid.UUID) (*TenantMemberServiceDataResult, error)
	GetByTenantAndUser(ctx context.Context, tenantID int64, userID int64) (*TenantMemberServiceDataResult, error)
	ListByTenant(ctx context.Context, tenantID int64) ([]TenantMemberServiceDataResult, error)
	ListByUser(ctx context.Context, userID int64) ([]TenantMemberServiceDataResult, error)
	UpdateRole(ctx context.Context, tenantMemberUUID uuid.UUID, role string) (*TenantMemberServiceDataResult, error)
	DeleteByUUID(ctx context.Context, tenantMemberUUID uuid.UUID) error
	IsUserInTenant(ctx context.Context, userID int64, tenantUUID uuid.UUID) (bool, error)
}

type tenantMemberService struct {
	db               *gorm.DB
	tenantMemberRepo repository.TenantMemberRepository
	userRepo         repository.UserRepository
	tenantRepo       repository.TenantRepository
}

func NewTenantMemberService(db *gorm.DB, tenantMemberRepo repository.TenantMemberRepository, userRepo repository.UserRepository, tenantRepo repository.TenantRepository) TenantMemberService {
	return &tenantMemberService{
		db:               db,
		tenantMemberRepo: tenantMemberRepo,
		userRepo:         userRepo,
		tenantRepo:       tenantRepo,
	}
}

func (s *tenantMemberService) Create(ctx context.Context, tenantID int64, userID int64, role string) (*TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID), attribute.Int64("user.id", userID))

	var created *model.TenantMember
	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantMemberRepo.WithTx(tx)
		tu := &model.TenantMember{
			TenantID: tenantID,
			UserID:   userID,
			Role:     role,
		}
		var err error
		created, err = repo.Create(tu)
		return err
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create tenant member failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return toTenantMemberServiceDataResult(created), nil
}

func (s *tenantMemberService) CreateByUserUUID(ctx context.Context, tenantID int64, userUUID uuid.UUID, role string) (*TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.createByUserUUID")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID), attribute.String("user.uuid", userUUID.String()))

	// First get the user to retrieve the user_id
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil || user == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewNotFound("user not found")
	}

	// Check if user is already a member of this tenant
	existing, _ := s.tenantMemberRepo.FindByTenantAndUser(tenantID, user.UserID)
	if existing != nil {
		span.SetStatus(codes.Error, "user already a member of this tenant")
		return nil, apperror.NewConflict("user is already a member of this tenant")
	}

	result, err := s.Create(ctx, tenantID, user.UserID, role)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create tenant member failed")
		return nil, err
	}

	// Populate user information in the result
	result.User = toUserServiceDataResult(user)

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

func (s *tenantMemberService) ListByTenant(ctx context.Context, tenantID int64) ([]TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.listByTenant")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	tus, err := s.tenantMemberRepo.FindAllByTenant(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list tenant members failed")
		return nil, err
	}

	result := make([]TenantMemberServiceDataResult, len(tus))
	for i, tu := range tus {
		dr := toTenantMemberServiceDataResult(&tu)

		// Fetch user information
		user, err := s.userRepo.FindByID(tu.UserID)
		if err == nil && user != nil {
			dr.User = toUserServiceDataResult(user)
		}

		result[i] = *dr
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
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

func (s *tenantMemberService) UpdateRole(ctx context.Context, tenantMemberUUID uuid.UUID, role string) (*TenantMemberServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.updateRole")
	defer span.End()
	span.SetAttributes(attribute.String("tenantMember.uuid", tenantMemberUUID.String()))

	var updated *model.TenantMember
	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantMemberRepo.WithTx(tx)
		tu, err := repo.FindByTenantMemberUUID(tenantMemberUUID)
		if err != nil {
			return err
		}
		if tu == nil {
			return apperror.NewNotFoundWithReason("tenant member not found")
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
		result.User = toUserServiceDataResult(user)
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *tenantMemberService) DeleteByUUID(ctx context.Context, tenantMemberUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "tenantMember.delete")
	defer span.End()
	span.SetAttributes(attribute.String("tenantMember.uuid", tenantMemberUUID.String()))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.tenantMemberRepo.WithTx(tx)
		tu, err := repo.FindByTenantMemberUUID(tenantMemberUUID)
		if err != nil {
			return err
		}
		if tu == nil {
			return apperror.NewNotFoundWithReason("tenant member not found")
		}
		return repo.DeleteByUUID(tenantMemberUUID)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete tenant member failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
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

func toTenantMemberServiceDataResult(tu *model.TenantMember) *TenantMemberServiceDataResult {
	return &TenantMemberServiceDataResult{
		TenantMemberUUID: tu.TenantMemberUUID,
		TenantID:         tu.TenantID,
		UserID:           tu.UserID,
		Role:             tu.Role,
		CreatedAt:        tu.CreatedAt,
		UpdatedAt:        tu.UpdatedAt,
	}
}
