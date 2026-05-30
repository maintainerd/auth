package client

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

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

func toPermissionServiceDataResult(p *Permission) PermissionServiceDataResult {
	return PermissionServiceDataResult{
		PermissionUUID: p.PermissionUUID,
		Name:           p.Name,
		Description:    p.Description,
		Status:         p.Status,
		IsDefault:      p.IsDefault,
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
