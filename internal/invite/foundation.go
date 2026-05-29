package invite

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/platform/pagination"
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

// ---------------------------------------------------------------------------
// Local aggregate structs — same underlying tables as owning domains.
// ---------------------------------------------------------------------------

// Tenant and User are type aliases for the cache auth types so test helpers
// can pass them directly into middleware.AuthContext.
type Tenant = cache.AuthTenant
type User = cache.AuthUser

type TenantRecord struct {
	TenantID   int64
	TenantUUID uuid.UUID
	Name       string
	Identifier string
	Status     string
	IsSystem   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (TenantRecord) TableName() string { return "tenants" }

type IdentityProvider struct {
	IdentityProviderID int64
	TenantID           int64
	Identifier         string
	Status             string
	Tenant             *Tenant `gorm:"-"` // navigational, not a real FK join in this package
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (IdentityProvider) TableName() string { return "identity_providers" }

type Client struct {
	ClientID           int64
	ClientUUID         uuid.UUID
	TenantID           int64
	IdentityProviderID int64
	Name               string
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
	RoleID    int64
	RoleUUID  uuid.UUID
	TenantID  int64
	Name      string
	Status    string
	IsDefault bool
	IsSystem  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Role) TableName() string { return "roles" }

// ---------------------------------------------------------------------------
// Consumer repository interfaces
// ---------------------------------------------------------------------------

type ClientRepository interface {
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindSystem() (*Client, error)
}

type RoleRepository interface {
	BaseRepositoryMethods[Role]
	WithTx(tx *gorm.DB) RoleRepository
	FindByUUIDs(uuids []string, preloads ...string) ([]Role, error)
}
