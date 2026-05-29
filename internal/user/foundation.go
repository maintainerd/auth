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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaseRepository[T any] = database.BaseRepository[T]
type BaseRepositoryMethods[T any] = database.BaseRepositoryMethods[T]
type PaginationResult[T any] = database.PaginationResult[T]

type PaginationRequestDTO = pagination.PaginationRequestDTO
type PaginatedResponseDTO[T any] = pagination.PaginatedResponseDTO[T]
type SuccessResponseDTO = pagination.SuccessResponseDTO

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

type SessionDataResult struct {
	SessionID         string     `json:"session_id"`
	IPAddress         *string    `json:"ip_address,omitempty"`
	UserAgent         *string    `json:"user_agent,omitempty"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	AbsoluteExpiresAt *time.Time `json:"absolute_expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type SessionService interface {
	ListSessions(ctx context.Context, userID int64) ([]*SessionDataResult, error)
	RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error
	RevokeAllSessions(ctx context.Context, userID int64) error
	CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*UserToken, error)
	EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error
	ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error
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
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsPublic    bool      `json:"is_public"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Role struct {
	RoleID          int64
	RoleUUID        uuid.UUID
	TenantID        int64
	Name            string
	Description     string
	IsDefault       bool
	IsSystem        bool
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Tenant          *Tenant
	RolePermissions []RolePermission `gorm:"foreignKey:RoleID;references:RoleID"`
}

func (Role) TableName() string { return "roles" }

type RolePermission struct {
	RolePermissionID int64
	RoleID           int64
	PermissionID     int64
}

func (RolePermission) TableName() string { return "role_permissions" }

type RoleServiceDataResult struct {
	RoleUUID    uuid.UUID
	Name        string
	Description string
	IsDefault   bool
	IsSystem    bool
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
	CreatedAt          time.Time
	UpdatedAt          time.Time
	IdentityProvider   *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
}

func (Client) TableName() string { return "clients" }

type ClientServiceDataResult struct {
	ClientUUID  uuid.UUID
	Name        string
	DisplayName string
	ClientType  string
	Domain      *string
	Status      string
	IsDefault   bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ClientResponseDTO struct {
	ClientUUID  uuid.UUID `json:"client_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	ClientType  string    `json:"client_type"`
	Domain      *string   `json:"domain,omitempty"`
	Status      string    `json:"status"`
	IsDefault   bool      `json:"is_default"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

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

type UserBackupCode struct {
	BackupCodeID   int64
	BackupCodeUUID uuid.UUID
	UserID         int64
	CodeHash       string
	Used           bool
	UsedAt         *time.Time
	CreatedAt      time.Time
}

func (UserBackupCode) TableName() string { return "user_backup_codes" }

type TenantRepository interface {
	BaseRepositoryMethods[Tenant]
	WithTx(tx *gorm.DB) TenantRepository
	FindByUUID(uuid any, preloads ...string) (*Tenant, error)
}

type RoleRepository interface {
	BaseRepositoryMethods[Role]
	WithTx(tx *gorm.DB) RoleRepository
	FindByUUID(uuid any, preloads ...string) (*Role, error)
	FindByNameAndTenantID(name string, tenantID int64) (*Role, error)
	FindPaginated(filter RoleRepositoryGetFilter) (*PaginationResult[Role], error)
}

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByID(id any, preloads ...string) (*Client, error)
	FindByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) (*Client, error)
	FindDefaultByTenantID(tenantID int64) (*Client, error)
	FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error)
}

type IdentityProviderRepository interface {
	BaseRepositoryMethods[IdentityProvider]
	WithTx(tx *gorm.DB) IdentityProviderRepository
	FindByIdentifier(identifier string) (*IdentityProvider, error)
}

type UserBackupCodeRepository interface {
	BaseRepositoryMethods[UserBackupCode]
	WithTx(tx *gorm.DB) UserBackupCodeRepository
	CreateBulk(codes []*UserBackupCode) error
	FindUnusedByUserID(userID int64) ([]UserBackupCode, error)
	FindByUserIDAndCodeHash(userID int64, codeHash string) (*UserBackupCode, error)
	MarkUsed(id int64) error
	DeleteAllByUserID(userID int64) error
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
