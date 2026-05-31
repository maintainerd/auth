package user

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"gorm.io/gorm"
)

type BaseRepository[T any] = database.BaseRepository[T]
type BaseRepositoryMethods[T any] = database.BaseRepositoryMethods[T]
type PaginationResult[T any] = database.PaginationResult[T]

type PaginationRequestDTO = pagination.PaginationRequestDTO
type PaginatedResponseDTO[T any] = pagination.PaginatedResponseDTO[T]
type SuccessResponseDTO = pagination.SuccessResponseDTO

var generateIDTokenFn = jwt.GenerateIDToken
var generateRefreshTokenFn = jwt.GenerateRefreshToken

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

func toTenantServiceDataResult(t *Tenant) *TenantServiceDataResult {
	if t == nil {
		return nil
	}
	return &TenantServiceDataResult{
		TenantUUID:  t.TenantUUID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		Description: t.Description,
		Identifier:  t.Identifier,
		Status:      t.Status,
		IsPublic:    t.IsPublic,
		IsSystem:    t.IsSystem,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func toRoleServiceDataResult(role *Role) *RoleServiceDataResult {
	if role == nil {
		return nil
	}
	return &RoleServiceDataResult{
		RoleUUID:    role.RoleUUID,
		Name:        role.Name,
		Description: role.Description,
		IsDefault:   role.IsDefault,
		IsSystem:    role.IsSystem,
		Status:      role.Status,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

func ToClientServiceDataResult(client *Client) *ClientServiceDataResult {
	if client == nil {
		return nil
	}
	return &ClientServiceDataResult{
		ClientUUID:  client.ClientUUID,
		Name:        client.Name,
		DisplayName: client.DisplayName,
		ClientType:  client.ClientType,
		Domain:      client.Domain,
		Status:      client.Status,
		IsDefault:   client.IsDefault,
		IsSystem:    client.IsSystem,
		CreatedAt:   client.CreatedAt,
		UpdatedAt:   client.UpdatedAt,
	}
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
