package setup

import (
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/maintainerd/auth/internal/user"
)

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
