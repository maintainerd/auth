package iam

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
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
