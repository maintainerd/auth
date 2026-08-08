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
export type * from './api/oauth/types'
export type {
  TenantStatus,
  TenantEntity,
  TenantListParams,
  UpdateTenantRequest,
  TenantResponse,
  TenantListResponse,
} from './api/tenants/types'

// Authentication service functions
export {
  login,
  verifyMFALogin,
  sendMFALoginSMS,
  beginMFALoginWebAuthn,
  register,
  registerInvite,
  logout,
  fetchProfile,
  fetchAccount,
  createUserProfile,
  createRegisterProfile,
  validateAuthentication,
  forgotPassword,
  resetPassword,
} from './api/auth'

// OAuth2 / OIDC browser interaction service functions
export {
  authorizeOAuth,
  fetchOAuthConsentChallenge,
  submitOAuthConsent,
  listOAuthConsentGrants,
  revokeOAuthConsentGrant,
  approveOAuthDevice,
  denyOAuthDevice,
  approveOAuthCIBA,
  denyOAuthCIBA,
  oauthEndSessionURL,
} from './api/oauth'

// Tenant service functions
export {
  fetchDefaultTenant,
  fetchTenantBySlug,
  fetchTenant,
  fetchTenantBootstrap,
  bootstrapToTenantEntity
} from './api/tenants'
