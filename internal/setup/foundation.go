package setup

import (
	"net/http"

	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/maintainerd/auth/internal/user"
	"gorm.io/gorm"
)

type BaseRepository[T any] = database.BaseRepository[T]
type BaseRepositoryMethods[T any] = database.BaseRepositoryMethods[T]
type PaginationResult[T any] = database.PaginationResult[T]

type PaginationRequestDTO = pagination.PaginationRequestDTO
type PaginatedResponseDTO[T any] = pagination.PaginatedResponseDTO[T]
type SuccessResponseDTO = pagination.SuccessResponseDTO

type LoginResponseDTO = authn.LoginResponseDTO
type UserResponseDTO = user.UserResponseDTO
type ProfileResponseDTO = user.ProfileResponseDTO
type TenantResponseDTO = tenant.TenantResponseDTO

type Tenant = tenant.Tenant
type TenantMember = tenant.TenantMember
type User = user.User
type UserIdentity = user.UserIdentity
type UserRole = user.UserRole
type Profile = user.Profile
type Client = client.Client
type IdentityProvider = client.IdentityProvider
type Role = iam.Role
type RoleRepositoryGetFilter = iam.RoleRepositoryGetFilter

type UserRepository = user.UserRepository
type TenantRepository = tenant.TenantRepository
type TenantMemberRepository = tenant.TenantMemberRepository
type ClientRepository = client.ClientRepository
type IdentityProviderRepository = any
type RoleRepository = iam.RoleRepository
type UserRoleRepository = user.UserRoleRepository
type UserTokenRepository = user.UserTokenRepository
type UserIdentityRepository = user.UserIdentityRepository
type ProfileRepository = user.ProfileRepository

type RegisterService = authn.RegisterService

var NewProfileResponseDTO = user.NewProfileResponseDTO

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
