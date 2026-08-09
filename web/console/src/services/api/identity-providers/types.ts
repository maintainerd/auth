/**
 * Identity Provider API Types
 */

import type { Status } from '@/types/status'

/**
 * Identity provider status type
 */
export type IdentityProviderStatus = Extract<Status, 'active' | 'inactive'>

/**
 * Identity provider type
 */
export type ProviderType = 'system' | 'social' | 'enterprise' | 'saml'

/**
 * Provider options
 */
export type ProviderOption =
  | 'maintainerd'
  | 'cognito'
  | 'auth0'
  | 'google'
  | 'facebook'
  | 'github'
  | 'gitlab'
  | 'microsoft'
  | 'linkedin'
  | 'twitter'
  | 'saml'

export type IdentityProviderConfig = Record<string, unknown>

export type IdentityProviderConnectionFields = {
  issuer: string | null
  provider_client_id: string | null
  /**
   * The OAuth2/OIDC redirect URI to register in the upstream provider (e.g.
   * Cognito "Allowed callback URLs"). Computed by the backend; absent for SAML.
   */
  callback_url?: string | null
  allow_jit_provisioning: boolean
  allow_registration: boolean
  allow_token_federation: boolean
  allowed_audiences: string[]
  email_domains: string[]
}

/**
 * Identity Provider type
 */
export type IdentityProvider = IdentityProviderConnectionFields & {
  identity_provider_id: string
  name: string
  display_name: string
  provider: ProviderOption
  provider_type: ProviderType
  identifier: string
  status: IdentityProviderStatus
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
  certificate_expires_at?: string | null
}

/**
 * Tenant type for identity provider details
 */
export type Tenant = {
  tenant_id: string
  name: string
  description: string
  status: 'active' | 'inactive' | 'suspended'
  is_public: boolean
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Identity Provider Detail type (includes config and tenant)
 */
export type IdentityProviderDetail = IdentityProviderConnectionFields & {
  identity_provider_id: string
  name: string
  display_name: string
  provider: ProviderOption
  provider_type: ProviderType
  identifier: string
  config: IdentityProviderConfig | null
  tenant: Tenant | null
  status: IdentityProviderStatus
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
  certificate_expires_at?: string | null
}

/**
 * Identity provider list query parameters interface
 */
export interface IdentityProviderQueryParams {
  search?: string
  name?: string
  display_name?: string
  provider?: string
  provider_type?: ProviderType
  identifier?: string
  status?: string
  is_default?: boolean
  is_system?: boolean
  tenant_id?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Paginated identity provider list response interface
 */
export interface IdentityProviderListResponse {
  rows: IdentityProvider[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Single identity provider response interface (for list items)
 */
export interface IdentityProviderResponse extends IdentityProviderConnectionFields {
  identity_provider_id: string
  name: string
  display_name: string
  provider: ProviderOption
  provider_type: ProviderType
  identifier: string
  status: IdentityProviderStatus
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Identity provider detail response interface (includes config and tenant)
 */
export interface IdentityProviderDetailResponse extends IdentityProviderConnectionFields {
  identity_provider_id: string
  name: string
  display_name: string
  provider: ProviderOption
  provider_type: ProviderType
  identifier: string
  config: IdentityProviderConfig | null
  tenant: Tenant | null
  status: IdentityProviderStatus
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Create identity provider request interface
 */
export interface CreateIdentityProviderRequest {
  name: string
  display_name: string
  provider: ProviderOption
  provider_type: ProviderType
  issuer?: string | null
  provider_client_id?: string | null
  provider_client_secret?: string
  allow_jit_provisioning?: boolean
  allow_registration: boolean
  allow_token_federation: boolean
  allowed_audiences: string[]
  email_domains?: string[]
  config: IdentityProviderConfig
  status: IdentityProviderStatus
}

export interface UpdateIdentityProviderRequest {
  name: string
  display_name: string
  provider: ProviderOption
  provider_type: ProviderType
  issuer?: string | null
  provider_client_id?: string | null
  provider_client_secret?: string
  allow_jit_provisioning?: boolean
  allow_registration: boolean
  allow_token_federation: boolean
  allowed_audiences: string[]
  email_domains?: string[]
  config: IdentityProviderConfig
  status: IdentityProviderStatus
}

/**
 * Update identity provider status request interface
 */
export interface UpdateIdentityProviderStatusRequest {
  status: IdentityProviderStatus
}
