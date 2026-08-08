/**
 * Services Index
 * Central export point for all services
 */

// API Client and utilities
export { ApiError, get, post, put, deleteRequest, patch } from './api/client'
export { API_CONFIG, API_ENDPOINTS } from './api/config'
export type * from './api/types'

// Service-specific API types
export type * from './api/auth/types'
// `setup/types` shares `CreateProfileRequest`/`CreateProfileResponse` with
// `auth/types`; auth's definitions win, so re-export setup's remaining members
// explicitly to avoid the ambiguous-wildcard conflict.
export type {
  CreateTenantRequest,
  TenantData,
  CreateTenantResponse,
  CreateAdminRequest,
  AdminData,
  CreateAdminResponse,
  ProfileData,
  SetupStatusData,
  SetupStatusResponse,
  CompleteSetupData,
  CompleteSetupResponse,
} from './api/setup/types'
// `tenants/types` shares `CreateTenantRequest` with `setup/types`; setup's
// definition wins, so re-export tenants' remaining members explicitly.
export type {
  TenantStatus,
  TenantEntity,
  TenantListParams,
  UpdateTenantRequest,
  TenantResponse,
  TenantListResponse,
} from './api/tenants/types'
export type * from './api/services/types'
export type * from './api/api/types'
export type * from './api/policies/types'
export type * from './api/clients/types'

// Setup service functions
export {
  createTenant,
  createAdmin,
  createProfile,
  getSetupStatus,
  completeSetup,
  getDefaultTenantMetadata,
  createTenantWithDefaults,
  isSetupCompleted,
} from './api/setup'

// Authentication service functions
export {
  fetchProfile,
  fetchAccount,
  createUserProfile,
  createRegisterProfile,
  validateAuthentication,
} from './api/auth'
// Tenant service functions
export {
  fetchDefaultTenant,
  fetchTenantByIdentifier,
  fetchTenant
} from './api/tenants'

// Service service functions
export {
  fetchServices,
  fetchServiceById,
  createService,
  updateService,
  deleteService,
  updateServiceStatus,
  assignPolicyToService,
  removePolicyFromService
} from './api/services'

// API service functions
export {
  fetchApis,
  fetchApiById,
  createApi,
  updateApi,
  deleteApi,
  updateApiStatus
} from './api/api'

// Policy service functions
export {
  fetchPolicies,
  fetchPolicyById,
  createPolicy,
  updatePolicy,
  deletePolicy,
  updatePolicyStatus,
  fetchServicesByPolicy
} from './api/policies'

// Client service functions
export {
  fetchClients,
  fetchClientById,
  createClient,
  updateClient,
  deleteClient,
  updateClientStatus
} from './api/clients'

