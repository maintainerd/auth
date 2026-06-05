package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

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
	Metadata    datatypes.JSON
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Tenant) TableName() string { return "tenants" }

type TenantServiceDataResult struct {
	TenantID    int64
	TenantUUID  uuid.UUID
	Name        string
	DisplayName string
	Description string
	Identifier  string
	Status      string
	IsPublic    bool
	IsSystem    bool
	Metadata    datatypes.JSON
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
	Tenant          *Tenant          `gorm:"foreignKey:TenantID;references:TenantID"`
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
	Tenant               *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
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
