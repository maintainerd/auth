package client

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type API struct {
	APIID       int64     `gorm:"column:api_id"`
	APIUUID     uuid.UUID `gorm:"column:api_uuid"`
	TenantID    int64
	Name        string
	DisplayName string
	Description string
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
	APIID          int64 `gorm:"column:api_id"`
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
	IsSystem    bool
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
	AllowRegistration    bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Tenant               *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

func (IdentityProvider) TableName() string { return "identity_providers" }

type User struct {
	UserID         int64          `gorm:"primaryKey"`
	UserUUID       uuid.UUID      `gorm:"column:user_uuid"`
	Username       string         `gorm:"column:username"`
	Email          string         `gorm:"column:email"`
	UserIdentities []UserIdentity `gorm:"foreignKey:UserID;references:UserID"`
}

func (User) TableName() string { return "users" }

type UserIdentity struct {
	UserIdentityID int64   `gorm:"primaryKey"`
	TenantID       int64   `gorm:"column:tenant_id"`
	UserID         int64   `gorm:"column:user_id"`
	ClientID       *int64  `gorm:"column:client_id"`
	Tenant         *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	Client         *Client `gorm:"foreignKey:ClientID;references:ClientID;constraint:OnDelete:SET NULL"`
}

func (UserIdentity) TableName() string { return "user_identities" }

type APIRepository interface {
	BaseRepositoryMethods[API]
	WithTx(tx *gorm.DB) APIRepository
	FindByUUID(uuid any, preloads ...string) (*API, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]API, error)
}

type PermissionRepository interface {
	BaseRepositoryMethods[Permission]
	WithTx(tx *gorm.DB) PermissionRepository
	FindByUUID(uuid any, preloads ...string) (*Permission, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]Permission, error)
}

type Role struct {
	RoleID      int64     `gorm:"column:role_id"`
	RoleUUID    uuid.UUID `gorm:"column:role_uuid"`
	TenantID    int64
	Name        string
	Description string
	Status      string
	IsDefault   bool
	IsSystem    bool
}

func (Role) TableName() string { return "roles" }

type RoleRepository interface {
	BaseRepositoryMethods[Role]
	WithTx(tx *gorm.DB) RoleRepository
	FindByUUID(uuid any, preloads ...string) (*Role, error)
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

type APIServiceDataResult struct {
	APIUUID     uuid.UUID
	Name        string
	DisplayName string
	Description string
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
	IsSystem       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func toPermissionServiceDataResult(p *Permission) PermissionServiceDataResult {
	return PermissionServiceDataResult{
		PermissionUUID: p.PermissionUUID,
		Name:           p.Name,
		Description:    p.Description,
		Status:         p.Status,
		IsSystem:       p.IsSystem,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
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

type ClientIdentityProviderServiceDataResult struct {
	ClientIdentityProviderUUID uuid.UUID
	IdentityProvider           IdentityProviderServiceDataResult
	IsDefault                  bool
	Enabled                    bool
	DisplayOrder               int
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}
