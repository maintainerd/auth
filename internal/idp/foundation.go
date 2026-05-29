package idp

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
	"gorm.io/datatypes"
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

type LoginResponseDTO struct {
	AccessToken           string  `json:"access_token"`
	IDToken               string  `json:"id_token"`
	RefreshToken          string  `json:"refresh_token,omitempty"`
	ExpiresIn             int64   `json:"expires_in"`
	TokenType             string  `json:"token_type"`
	IssuedAt              int64   `json:"issued_at"`
	RequirePasswordChange bool    `json:"require_password_change,omitempty"`
	SessionID             *string `json:"session_id,omitempty"`
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

type TenantServiceDataResult struct {
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

type TenantResponseDTO struct {
	TenantUUID  uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name,omitempty"`
	Description string    `json:"description"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsPublic    bool      `json:"is_public"`
	IsSystem    bool      `json:"is_system,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserIdentity struct {
	UserIdentityID     int64
	UserIdentityUUID   uuid.UUID
	TenantID           int64
	UserID             int64
	ClientID           int64
	IdentityProviderID *int64
	Provider           string
	Sub                string
	Metadata           datatypes.JSON
	Tenant             *Tenant           `gorm:"foreignKey:TenantID;references:TenantID"`
	Client             *Client           `gorm:"foreignKey:ClientID;references:ClientID"`
	IdentityProvider   *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (UserIdentity) TableName() string { return "user_identities" }

type User struct {
	UserID             int64
	UserUUID           uuid.UUID
	Username           string
	Fullname           string `gorm:"-"`
	Email              string
	Phone              string
	Password           *string
	IsEmailVerified    bool
	IsPhoneVerified    bool
	IsProfileCompleted bool
	IsAccountCompleted bool
	Status             string
	Metadata           datatypes.JSON
	UserIdentities     []UserIdentity `gorm:"foreignKey:UserID;references:UserID"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (User) TableName() string { return "users" }

type Client struct {
	ClientID           int64
	ClientUUID         uuid.UUID
	TenantID           int64
	IdentityProviderID int64
	Name               string
	DisplayName        string
	ClientType         string
	Domain             *string
	Identifier         *string
	Status             string
	IsDefault          bool
	IsSystem           bool
	IdentityProvider   *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (Client) TableName() string { return "clients" }

type Role struct {
	RoleID      int64
	RoleUUID    uuid.UUID
	TenantID    int64
	Name        string
	Description string
	Status      string
	IsDefault   bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Role) TableName() string { return "roles" }

type RoleRepositoryGetFilter struct {
	Name        *string
	Description *string
	IsDefault   *bool
	IsSystem    *bool
	Status      *string
	TenantID    int64
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type RoleResponseDTO struct {
	RoleUUID    uuid.UUID `json:"role_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	IsSystem    bool      `json:"is_system"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserRole struct {
	UserRoleID int64
	UserID     int64
	RoleID     int64
	CreatedAt  time.Time
}

func (UserRole) TableName() string { return "user_roles" }

type TenantRepository interface {
	BaseRepositoryMethods[Tenant]
	WithTx(tx *gorm.DB) TenantRepository
	FindByUUID(uuid any, preloads ...string) (*Tenant, error)
}

type UserRepository interface {
	BaseRepositoryMethods[User]
	WithTx(tx *gorm.DB) UserRepository
	FindByUUID(uuid any, preloads ...string) (*User, error)
	FindByID(id any, preloads ...string) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByEmailAndTenantID(email string, tenantID int64) (*User, error)
}

type UserIdentityRepository interface {
	BaseRepositoryMethods[UserIdentity]
	WithTx(tx *gorm.DB) UserIdentityRepository
	FindByUserID(userID int64) ([]UserIdentity, error)
	FindByProviderAndSub(provider, sub string) (*UserIdentity, error)
	FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error)
	DeleteByUserID(userID int64) error
}

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByUUID(uuid any, preloads ...string) (*Client, error)
	FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error)
}

type UserRoleRepository interface {
	BaseRepositoryMethods[UserRole]
	WithTx(tx *gorm.DB) UserRoleRepository
	FindByUserID(userID int64) ([]UserRole, error)
	FindByUserIDAndRoleID(userID, roleID int64) (*UserRole, error)
	DeleteByUserIDAndRoleID(userID, roleID int64) error
}

type RoleRepository interface {
	BaseRepositoryMethods[Role]
	WithTx(tx *gorm.DB) RoleRepository
	FindByUUID(uuid any, preloads ...string) (*Role, error)
	FindByNameAndTenantID(name string, tenantID int64) (*Role, error)
	FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[Role], error)
}

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
