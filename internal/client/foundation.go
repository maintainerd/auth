package client

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaseRepository[T any] = database.BaseRepository[T]
type BaseRepositoryMethods[T any] = database.BaseRepositoryMethods[T]
type PaginationResult[T any] = database.PaginationResult[T]

type PaginationRequestDTO = pagination.PaginationRequestDTO
type PaginatedResponseDTO[T any] = pagination.PaginatedResponseDTO[T]
type SuccessResponseDTO = pagination.SuccessResponseDTO
type AuthEventService = authevent.AuthEventService

type API struct {
	APIID       int64
	APIUUID     uuid.UUID
	TenantID    int64
	Name        string
	DisplayName string
	Description string
	APIType     string
	Identifier  string
	Status      string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (API) TableName() string { return "apis" }

type Permission struct {
	PermissionID   int64
	PermissionUUID uuid.UUID
	TenantID       int64
	APIID          int64
	Name           string
	Description    string
	Status         string
	IsDefault      bool
	IsSystem       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Permission) TableName() string { return "permissions" }

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

type IdentityProvider struct {
	IdentityProviderID   int64
	IdentityProviderUUID uuid.UUID
	TenantID             int64
	Name                 string
	DisplayName          string
	Provider             string
	ProviderType         string
	Identifier           string
	Config               datatypes.JSON
	Status               string
	IsDefault            bool
	IsSystem             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Tenant               *Tenant
}

func (IdentityProvider) TableName() string { return "identity_providers" }

type User struct {
	UserID         int64
	UserUUID       uuid.UUID
	Username       string
	Email          string
	UserIdentities []UserIdentity
}

func (User) TableName() string { return "users" }

type UserIdentity struct {
	UserIdentityID int64
	TenantID       int64
	UserID         int64
	ClientID       int64
	Tenant         *Tenant
	Client         *Client
}

func (UserIdentity) TableName() string { return "user_identities" }

type APIRepository interface {
	BaseRepositoryMethods[API]
	WithTx(tx *gorm.DB) APIRepository
	FindByUUID(uuid any, preloads ...string) (*API, error)
}

type PermissionRepository interface {
	BaseRepositoryMethods[Permission]
	WithTx(tx *gorm.DB) PermissionRepository
	FindByUUID(uuid any, preloads ...string) (*Permission, error)
}

type IdentityProviderRepository interface {
	BaseRepositoryMethods[IdentityProvider]
	WithTx(tx *gorm.DB) IdentityProviderRepository
	FindByUUID(uuid any, preloads ...string) (*IdentityProvider, error)
}

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

type APIResponseDTO struct {
	APIUUID     uuid.UUID `json:"api_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	APIType     string    `json:"api_type"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PermissionResponseDTO struct {
	PermissionUUID uuid.UUID `json:"permission_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	IsDefault      bool      `json:"is_default"`
	IsSystem       bool      `json:"is_system"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type IdentityProviderResponseDTO struct {
	IdentityProviderUUID uuid.UUID `json:"identity_provider_id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	Provider             string    `json:"provider"`
	ProviderType         string    `json:"provider_type"`
	Identifier           string    `json:"identifier"`
	Status               string    `json:"status"`
	IsDefault            bool      `json:"is_default"`
	IsSystem             bool      `json:"is_system"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type APIServiceDataResult struct {
	APIUUID     uuid.UUID
	Name        string
	DisplayName string
	Description string
	APIType     string
	Identifier  string
	Status      string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PermissionServiceDataResult struct {
	PermissionUUID uuid.UUID
	Name           string
	Description    string
	API            *APIServiceDataResult
	Status         string
	IsDefault      bool
	IsSystem       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type IdentityProviderServiceDataResult struct {
	IdentityProviderUUID uuid.UUID
	Name                 string
	DisplayName          string
	Provider             string
	ProviderType         string
	Identifier           string
	Status               string
	IsDefault            bool
	IsSystem             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
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
