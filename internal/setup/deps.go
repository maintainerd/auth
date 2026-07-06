package setup

import (
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/maintainerd/maintainerd-auth/internal/user"
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
type Service = iam.Service
type Policy = iam.Policy
type ServicePolicy = iam.ServicePolicy
type RoleRepositoryGetFilter = iam.RoleRepositoryGetFilter
type ServiceRepositoryGetFilter = iam.ServiceRepositoryGetFilter
type PolicyRepositoryGetFilter = iam.PolicyRepositoryGetFilter
type ServicePolicyRepositoryGetFilter = iam.ServicePolicyRepositoryGetFilter

type UserRepository = user.UserRepository
type TenantRepository = tenant.TenantRepository
type TenantMemberRepository = tenant.TenantMemberRepository
type ClientRepository = client.ClientRepository
type RoleRepository = iam.RoleRepository
type UserRoleRepository = user.UserRoleRepository
type UserIdentityRepository = user.UserIdentityRepository
type ProfileRepository = user.ProfileRepository
type ServiceRepository = iam.ServiceRepository
type PolicyRepository = iam.PolicyRepository
type ServicePolicyRepository = iam.ServicePolicyRepository

var NewProfileResponseDTO = user.NewProfileResponseDTO
