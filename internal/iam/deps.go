package iam

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tenant struct {
	TenantID    int64
	TenantUUID  uuid.UUID
	Name        string
	DisplayName string
	Description string
	Identifier  string
	Status      string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Tenant) TableName() string { return "tenants" }

type UserIdentity struct {
	UserIdentityID   int64     `gorm:"column:user_identity_id;primaryKey"`
	UserIdentityUUID uuid.UUID `gorm:"column:user_identity_uuid;unique"`
	TenantID         int64     `gorm:"column:tenant_id"`
	UserID           int64     `gorm:"column:user_id"`
	Sub              string    `gorm:"column:sub"`
	Tenant           *Tenant   `gorm:"foreignKey:TenantID;references:TenantID"`
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

// TenantService is DEPRECATED — the tenant_services table has been removed.
// services.tenant_id is the authoritative tenant-scope relationship.
// This type is kept only for test compatibility.
type TenantService struct {
	TenantServiceID int64
	TenantID        int64
	ServiceID       int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (TenantService) TableName() string { return "tenant_services" }

type TenantServiceRepositoryGetFilter struct {
	TenantID  *int64
	ServiceID *int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

// TenantServiceRepository is DEPRECATED.
type TenantServiceRepository interface {
	BaseRepositoryMethods[TenantService]
	WithTx(tx *gorm.DB) TenantServiceRepository
	FindPaginated(filter TenantServiceRepositoryGetFilter) (*PaginationResult[TenantService], error)
	FindByTenantAndService(tenantID int64, serviceID int64) (*TenantService, error)
	DeleteByTenantAndService(tenantID int64, serviceID int64) error
}

type UserRole struct {
	UserRoleID uuid.UUID
	UserID     int64
	RoleID     int64
}

func (UserRole) TableName() string { return "user_roles" }

type UserToken struct {
	UserTokenID int64
	UserID      int64
	IsRevoked   bool
}

func (UserToken) TableName() string { return "user_tokens" }

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
