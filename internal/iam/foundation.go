package iam

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"gorm.io/gorm"
)

type BaseRepository[T any] = database.BaseRepository[T]
type BaseRepositoryMethods[T any] = database.BaseRepositoryMethods[T]
type PaginationResult[T any] = database.PaginationResult[T]

type PaginationRequestDTO = pagination.PaginationRequestDTO
type PaginatedResponseDTO[T any] = pagination.PaginatedResponseDTO[T]
type SuccessResponseDTO = pagination.SuccessResponseDTO

const (
	SortOrderAsc  = pagination.SortOrderAsc
	SortOrderDesc = pagination.SortOrderDesc
)

func NewBaseRepository[T any](db any, uuidFieldName, idFieldName string) *database.BaseRepository[T] {
	return database.NewBaseRepository[T](db.(*gorm.DB), uuidFieldName, idFieldName)
}

func parsePaginationQuery(r *http.Request) pagination.PaginationRequestDTO {
	return pagination.ParseQuery(r)
}

func sanitizeOrder(sortBy, sortOrder, defaultCol string) string {
	return database.SanitizeOrder(sortBy, sortOrder, defaultCol)
}

func sanitizeOrderPrefixed(prefix, sortBy, sortOrder, defaultCol string) string {
	return database.SanitizeOrderPrefixed(prefix, sortBy, sortOrder, defaultCol)
}

func normalizePagination(page, limit int) (int, int) {
	return database.NormalizePagination(page, limit)
}

type Tenant struct {
	TenantID    int64
	TenantUUID  uuid.UUID
	Name        string
	DisplayName string
	Description string
	Identifier  string
	Status      string
	IsPublic    bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Tenant) TableName() string { return "tenants" }

type UserIdentity struct {
	UserIdentityID uuid.UUID
	TenantID       int64
	UserID         int64
	Tenant         *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

func (UserIdentity) TableName() string { return "user_identities" }

type User struct {
	UserID         int64
	UserUUID       uuid.UUID
	Username       string
	Email          string
	Status         string
	UserIdentities []UserIdentity `gorm:"foreignKey:UserID;references:UserID"`
}

func (User) TableName() string { return "users" }

type Client struct {
	ClientID   int64
	ClientUUID uuid.UUID
	TenantID   int64
	Name       string
	Status     string
}

func (Client) TableName() string { return "clients" }

type TenantService struct {
	TenantServiceID int64
	TenantID        int64
	ServiceID       int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (TenantService) TableName() string { return "tenant_services" }

type TenantRepository interface {
	BaseRepositoryMethods[Tenant]
	WithTx(tx *gorm.DB) TenantRepository
	FindByUUID(uuid any, preloads ...string) (*Tenant, error)
}

type UserRepository interface {
	BaseRepositoryMethods[User]
	WithTx(tx *gorm.DB) UserRepository
	FindByUUID(uuid any, preloads ...string) (*User, error)
}

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByUUID(uuid any, preloads ...string) (*Client, error)
}

type TenantServiceRepositoryGetFilter struct {
	TenantID  *int64
	ServiceID *int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type TenantServiceRepository interface {
	BaseRepositoryMethods[TenantService]
	WithTx(tx *gorm.DB) TenantServiceRepository
	FindPaginated(filter TenantServiceRepositoryGetFilter) (*PaginationResult[TenantService], error)
	FindByTenantAndService(tenantID int64, serviceID int64) (*TenantService, error)
	DeleteByTenantAndService(tenantID int64, serviceID int64) error
}

type noopAuthEventService struct{}

func (noopAuthEventService) Log(_ context.Context, _ authevent.AuthEventInput) {}
func (noopAuthEventService) FindPaginated(_ context.Context, _ authevent.AuthEventRepositoryGetFilter) (*PaginationResult[authevent.AuthEventServiceDataResult], error) {
	return &PaginationResult[authevent.AuthEventServiceDataResult]{}, nil
}
func (noopAuthEventService) FindByUUID(_ context.Context, _ int64, _ uuid.UUID) (*authevent.AuthEventServiceDataResult, error) {
	return nil, nil
}
func (noopAuthEventService) CountByEventType(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (noopAuthEventService) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func coalesceAuthEventService(svc authevent.AuthEventService) authevent.AuthEventService {
	if svc != nil {
		return svc
	}
	return noopAuthEventService{}
}

func ValidateTenantAccess(actor *User, target *Tenant) error {
	if actor == nil {
		return apperror.NewUnauthorized("actor user not found")
	}
	if target == nil {
		return apperror.NewNotFoundWithReason("tenant not found")
	}
	if len(actor.UserIdentities) == 0 {
		return apperror.NewForbidden("actor user has no identities")
	}
	for _, identity := range actor.UserIdentities {
		if identity.TenantID == target.TenantID {
			return nil
		}
		if identity.Tenant != nil && identity.Tenant.IsSystem {
			return nil
		}
	}
	return apperror.NewForbidden("tenant access denied")
}
