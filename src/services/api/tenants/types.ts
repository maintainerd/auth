/**
 * Tenant API Types
 * Type definitions for tenant API operations
 */

import type { ApiResponse, PaginatedResponse } from '../types'

/**
 * Tenant status type
 */
export type TenantStatus = 'active' | 'inactive' | 'suspended'

export interface RegistrationConfigPublic {
  self_registration_enabled: boolean
  require_email_verification: boolean
  captcha_on_signup: boolean
}

export interface PasswordConfigPublic {
  min_length: number
  max_length: number
  require_uppercase: boolean
  require_lowercase: boolean
  require_number: boolean
  require_symbol: boolean
  // Backend-authoritative checks, surfaced so password fields can show them as
  // requirements. Common-password and HIBP screening run only on the backend;
  // the frontend displays the backend's error. min_strength_score 0 = disabled.
  min_strength_score: number
  reject_common_passwords: boolean
  check_hibp: boolean
}

export interface BrandingPublic {
  layout: string
  company_name: string
  logo_label: string
  logo_detail: string
  show_logo_label: boolean
  identity_logo_label: string
  identity_show_logo_label: boolean
  logo_url: string
  favicon_url: string
  support_url: string
  privacy_policy_url: string
  terms_of_service_url: string
  metadata: Record<string, unknown> | null
}

/**
 * Tenant entity from API
 */
export interface TenantEntity {
  tenant_id: string
  name: string
  display_name: string
  description: string
  status: TenantStatus
  is_default: boolean
  is_system: boolean
  created_at: string
  updated_at: string
  password_config?: PasswordConfigPublic
  registration_config?: RegistrationConfigPublic
  branding?: BrandingPublic
}

/**
 * Tenant list query parameters
 */
export interface TenantListParams {
  name?: string
  display_name?: string
  description?: string
  status?: TenantStatus
  is_default?: boolean
  is_system?: boolean
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Create tenant request
 */
export interface CreateTenantRequest {
  name: string
  display_name: string
  description: string
  status: TenantStatus
}

/**
 * Update tenant request
 */
export interface UpdateTenantRequest {
  name: string
  display_name: string
  description: string
  status: TenantStatus
}

/**
 * OAuth client summary returned by the tenant-bootstrap endpoint. For a console
 * host this is the tenant's CONSOLE client — used to start the OAuth2 flow
 * without a separate `/client/console` lookup.
 */
export interface TenantBootstrapClient {
  client_id: string
  name: string
  display_name: string
  client_type: string
}

/**
 * Tenant summary embedded in the bootstrap response. Note the id field is
 * `tenant_uuid` here (the public surface names it differently from the internal
 * `tenant_id`); it is normalized to `tenant_id` when stored in tenant state.
 */
export interface TenantBootstrapTenant {
  tenant_uuid: string
  name: string
  display_name: string
  description: string
  status: TenantStatus
  is_system: boolean
}

/**
 * Payload of the CONTROL-plane `GET /api/v1/tenant?domain=<host>`. The backend
 * resolves the tenant from the FULL host (no client-side slug parsing) and
 * derives the per-tenant identity/console origins plus the console OAuth client.
 */
export interface TenantBootstrapData {
  tenant: TenantBootstrapTenant
  surface: 'identity' | 'console'
  identity_url: string
  console_url: string
  password_config?: PasswordConfigPublic
  registration_config?: RegistrationConfigPublic
  branding?: BrandingPublic
  client?: TenantBootstrapClient
}

export type TenantBootstrapResponse = ApiResponse<TenantBootstrapData>

/**
 * Tenant API response
 */
export type TenantResponse = ApiResponse<TenantEntity>

/**
 * Tenant list API response
 */
export type TenantListResponse = ApiResponse<PaginatedResponse<TenantEntity>>
