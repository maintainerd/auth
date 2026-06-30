package idp

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"gorm.io/datatypes"
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
	TenantID           int64
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
	IdentityProviderID int64             `gorm:"-"`
	IdentityProvider   *IdentityProvider `gorm:"-"`
	Name               string
	DisplayName        string
	ClientType         string
	Domain             *string
	Identifier         *string
	Status             string
	IsDefault          bool
	IsSystem           bool
	Tenant             *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
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

// ClientURI is a lightweight read-model of a client's registered URI, used to
// resolve/preload callback URIs attached to a registration flow.
type ClientURI struct {
	ClientURIID   int64     `gorm:"column:client_uri_id;primaryKey"`
	ClientURIUUID uuid.UUID `gorm:"column:client_uri_uuid"`
	TenantID      int64     `gorm:"column:tenant_id"`
	ClientID      int64     `gorm:"column:client_id"`
	URI           string    `gorm:"column:uri"`
	Type          string    `gorm:"column:type"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (ClientURI) TableName() string { return "client_uris" }

// Branding is a lightweight read-model used to resolve a branding template by
// UUID and surface its UUID/name on a registration flow.
type Branding struct {
	BrandingID   int64     `gorm:"column:branding_id;primaryKey"`
	BrandingUUID uuid.UUID `gorm:"column:branding_uuid"`
	TenantID     int64     `gorm:"column:tenant_id"`
	Name         string    `gorm:"column:name"`
}

func (Branding) TableName() string { return "branding" }

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
	FindByEmailAndTenantID(email string, tenantID int64) (*User, error)
}

type UserIdentityRepository interface {
	BaseRepositoryMethods[UserIdentity]
	DeleteByID(id any) error
	WithTx(tx *gorm.DB) UserIdentityRepository
	FindByUserID(userID int64) ([]UserIdentity, error)
	FindByTenantProviderAndSub(tenantID int64, provider, sub string) (*UserIdentity, error)
	FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error)
	CreateByTenantProviderSubIfAbsent(identity *UserIdentity) (*UserIdentity, bool, error)
	DeleteByUserID(userID int64) error
}

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByUUID(uuid any, preloads ...string) (*Client, error)
	FindByID(id any, preloads ...string) (*Client, error)
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
	// Tenant isolation: access is granted only to the actor's own tenant(s).
	// System-tenant identities do NOT get a cross-tenant override here — that
	// override is confined to the tenant package (tenant-management ops only).
	for _, identity := range actor.UserIdentities {
		if identity.TenantID == target.TenantID {
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
		TenantID:    t.TenantID,
		TenantUUID:  t.TenantUUID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		Description: t.Description,
		Identifier:  t.Identifier,
		Status:      t.Status,
		IsSystem:    t.IsSystem,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
