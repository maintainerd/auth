package user

import (
	"context"
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

// UserPoolServiceDataResult is the service-layer representation of a user pool.
type UserPoolServiceDataResult struct {
	UserPoolUUID uuid.UUID
	TenantID     int64
	Name         string
	DisplayName  string
	Identifier   string
	IsSystem     bool
	Status       string
	Metadata     datatypes.JSON
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserPoolService exposes tenant-scoped user pool management operations.
//
// A user pool is the isolation boundary for users, roles, and settings within a
// single tenant deployment (analogous to an AWS Cognito User Pool). All
// operations are tenant-isolated: a pool is only visible to its owning tenant.
type UserPoolService interface {
	List(ctx context.Context, tenantID int64) ([]*UserPoolServiceDataResult, error)
	GetByUUID(ctx context.Context, userPoolUUID uuid.UUID, tenantID int64) (*UserPoolServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name, displayName, status string, metadata datatypes.JSON, creatorUserID *int64) (*UserPoolServiceDataResult, error)
	Update(ctx context.Context, userPoolUUID uuid.UUID, tenantID int64, name, displayName, status string, metadata datatypes.JSON, updaterUserID *int64) (*UserPoolServiceDataResult, error)
	SetStatus(ctx context.Context, userPoolUUID uuid.UUID, tenantID int64, status string, updaterUserID *int64) (*UserPoolServiceDataResult, error)
	Delete(ctx context.Context, userPoolUUID uuid.UUID, tenantID int64) (*UserPoolServiceDataResult, error)
}

type userPoolService struct {
	db           *gorm.DB
	userPoolRepo UserPoolRepository
}

// NewUserPoolService returns a UserPoolService backed by the given repository.
func NewUserPoolService(db *gorm.DB, userPoolRepo UserPoolRepository) UserPoolService {
	return &userPoolService{
		db:           db,
		userPoolRepo: userPoolRepo,
	}
}

// List returns all user pools belonging to the given tenant.
func (s *userPoolService) List(ctx context.Context, tenantID int64) ([]*UserPoolServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_pool.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant_id", tenantID))

	pools, err := s.userPoolRepo.FindAllByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list user pools failed")
		return nil, err
	}

	results := make([]*UserPoolServiceDataResult, len(pools))
	for i := range pools {
		results[i] = toUserPoolServiceDataResult(&pools[i])
	}

	span.SetStatus(codes.Ok, "")
	return results, nil
}

// GetByUUID returns a single user pool, enforcing tenant ownership.
func (s *userPoolService) GetByUUID(ctx context.Context, userPoolUUID uuid.UUID, tenantID int64) (*UserPoolServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_pool.get_by_uuid")
	defer span.End()
	span.SetAttributes(attribute.String("user_pool_uuid", userPoolUUID.String()))

	pool, err := s.findOwnedPool(userPoolUUID, tenantID)
	if err != nil {
		span.SetStatus(codes.Error, "user pool not found")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserPoolServiceDataResult(pool), nil
}

// Create provisions a new (non-default, non-system) user pool for the tenant.
func (s *userPoolService) Create(ctx context.Context, tenantID int64, name, displayName, status string, metadata datatypes.JSON, creatorUserID *int64) (*UserPoolServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_pool.create")
	defer span.End()
	span.SetAttributes(attribute.String("user_pool.name", name))

	if status == "" {
		status = shared.StatusActive
	}

	identifier, err := crypto.GenerateIdentifier(24)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "identifier generation failed")
		return nil, apperror.NewInternal("failed to generate user pool identifier", err)
	}

	pool := UserPool{
		UserPoolUUID: uuid.New(),
		TenantID:     tenantID,
		Name:         name,
		DisplayName:  displayName,
		Identifier:   identifier,
		IsSystem:     false,
		Status:       status,
		Metadata:     metadata,
		CreatedBy:    creatorUserID,
		UpdatedBy:    creatorUserID,
	}

	created, err := s.userPoolRepo.Create(&pool)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create user pool failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserPoolServiceDataResult(created), nil
}

// Update mutates a tenant-owned user pool. System pools are immutable.
func (s *userPoolService) Update(ctx context.Context, userPoolUUID uuid.UUID, tenantID int64, name, displayName, status string, metadata datatypes.JSON, updaterUserID *int64) (*UserPoolServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_pool.update")
	defer span.End()
	span.SetAttributes(attribute.String("user_pool_uuid", userPoolUUID.String()))

	pool, err := s.findOwnedPool(userPoolUUID, tenantID)
	if err != nil {
		span.SetStatus(codes.Error, "user pool not found")
		return nil, err
	}

	if pool.IsSystem {
		span.SetStatus(codes.Error, "system user pool is immutable")
		return nil, apperror.NewConflict("system user pool cannot be modified")
	}

	pool.Name = name
	pool.DisplayName = displayName
	pool.Status = status
	if metadata != nil {
		pool.Metadata = metadata
	}
	pool.UpdatedBy = updaterUserID

	if _, err := s.userPoolRepo.UpdateByUUID(userPoolUUID, pool); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update user pool failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserPoolServiceDataResult(pool), nil
}

// SetStatus updates only the status of a tenant-owned user pool. The system pool
// is protected because deactivating it would break authentication for the tenant.
func (s *userPoolService) SetStatus(ctx context.Context, userPoolUUID uuid.UUID, tenantID int64, status string, updaterUserID *int64) (*UserPoolServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_pool.set_status")
	defer span.End()
	span.SetAttributes(attribute.String("user_pool_uuid", userPoolUUID.String()))

	pool, err := s.findOwnedPool(userPoolUUID, tenantID)
	if err != nil {
		span.SetStatus(codes.Error, "user pool not found")
		return nil, err
	}

	if pool.IsSystem {
		span.SetStatus(codes.Error, "system user pool is immutable")
		return nil, apperror.NewConflict("system user pool status cannot be changed")
	}

	pool.Status = status
	pool.UpdatedBy = updaterUserID

	if _, err := s.userPoolRepo.UpdateByUUID(userPoolUUID, pool); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set user pool status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserPoolServiceDataResult(pool), nil
}

// Delete soft-deletes a tenant-owned pool. The system pool is protected.
func (s *userPoolService) Delete(ctx context.Context, userPoolUUID uuid.UUID, tenantID int64) (*UserPoolServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "user_pool.delete")
	defer span.End()
	span.SetAttributes(attribute.String("user_pool_uuid", userPoolUUID.String()))

	pool, err := s.findOwnedPool(userPoolUUID, tenantID)
	if err != nil {
		span.SetStatus(codes.Error, "user pool not found")
		return nil, err
	}

	if pool.IsSystem {
		span.SetStatus(codes.Error, "system user pool cannot be deleted")
		return nil, apperror.NewConflict("system user pool cannot be deleted")
	}

	if err := s.userPoolRepo.DeleteByUUID(userPoolUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete user pool failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toUserPoolServiceDataResult(pool), nil
}

// findOwnedPool fetches a pool by UUID and verifies it belongs to the tenant,
// returning a NotFound error otherwise so cross-tenant access is indistinguishable
// from a missing record.
func (s *userPoolService) findOwnedPool(userPoolUUID uuid.UUID, tenantID int64) (*UserPool, error) {
	pool, err := s.userPoolRepo.FindByUUID(userPoolUUID)
	if err != nil {
		return nil, err
	}
	if pool == nil || pool.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("user pool not found")
	}
	return pool, nil
}

func toUserPoolServiceDataResult(pool *UserPool) *UserPoolServiceDataResult {
	if pool == nil {
		return nil
	}
	return &UserPoolServiceDataResult{
		UserPoolUUID: pool.UserPoolUUID,
		TenantID:     pool.TenantID,
		Name:         pool.Name,
		DisplayName:  pool.DisplayName,
		Identifier:   pool.Identifier,
		IsSystem:     pool.IsSystem,
		Status:       pool.Status,
		Metadata:     pool.Metadata,
		CreatedAt:    pool.CreatedAt,
		UpdatedAt:    pool.UpdatedAt,
	}
}
