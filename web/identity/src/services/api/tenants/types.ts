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

export interface BrandingColors {
  primary?: string
  secondary?: string
  accent?: string
  appBackground?: string
  topPanelBackground?: string
  sidePanelBackground?: string
  cardBackground?: string
  textPrimary?: string
  textMuted?: string
  border?: string
  authPageBackground?: string
  authFormPanelBackground?: string
  authFormPanelBorder?: string
  authFormPanelText?: string
  authVisualPanelBackground?: string
  authVisualPanelText?: string
  authVisualPanelOverlay?: string
  authDecorativeLight?: string
  authDecorativeDark?: string
  authProgressPanelBackground?: string
  authSecurityPanelBackground?: string
}

export interface BrandingFont {
  family?: string
}

export interface BrandingMetadata {
  colors?: BrandingColors
  font?: BrandingFont
  /** Optional future override; appBackground remains the current console token. */
  background?: string
  /** Optional CSS gradient used ahead of background/appBackground when present. */
  gradient?: string
  [key: string]: unknown
}

export type BrandingLayout = 'centered' | 'full_page' | 'split'

export interface BrandingPublic {
  layout: BrandingLayout
  company_name: string
  logo_label?: string
  logo_detail?: string
  show_logo_label?: boolean
  /** Public-facing logo label shown on identity surfaces. Falls back to logo_label/company_name. */
  identity_logo_label?: string
  /** Whether the identity surface renders the logo label + detail. */
  identity_show_logo_label?: boolean
  logo_url: string
  favicon_url: string
  support_url: string
  privacy_policy_url: string
  terms_of_service_url: string
  metadata: BrandingMetadata | null
}

/**
 * Tenant entity from API
 */
export interface TenantEntity {
  tenant_id: string
  /** Tenant slug — also the subdomain label used to resolve this tenant. */
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
 * Which frontend surface a host maps to. Returned by the domain-bootstrap
 * endpoint so the app knows whether it is serving the identity or console UI.
 */
export type TenantSurface = 'identity' | 'console'

/**
 * Public projection of the tenant's seeded system client for a surface, returned
 * by the domain-bootstrap endpoint. For an identity host this is the tenant's
 * default IDENTITY client — the client_id first-party login should use without
 * the user having to supply anything.
 */
export interface TenantBootstrapClient {
  client_id: string
  name: string
  display_name: string
  client_type: string
}

/**
 * Public tenant identity projection embedded in the bootstrap response —
 * identifying fields only. Password/registration policy live as siblings on the
 * bootstrap payload (see TenantBootstrap), not on this projection.
 */
export interface TenantBootstrapTenant {
  tenant_id: string
  name: string
  display_name: string
  description: string
  status: TenantStatus
  is_system: boolean
}

/**
 * Domain-resolved bootstrap payload returned by
 * `GET /tenant?domain=<host>`. The backend resolves the full host to a tenant
 * and surface — the frontend no longer parses the host itself.
 */
export interface TenantBootstrap {
  tenant: TenantBootstrapTenant
  surface: TenantSurface
  identity_url: string
  console_url: string
  /** Tenant-managed password policy, enforced client-side for UX. */
  password_config?: PasswordConfigPublic
  /** Tenant-managed registration gating (self-signup, email verification, captcha). */
  registration_config?: RegistrationConfigPublic
  branding?: BrandingPublic | null
  client?: TenantBootstrapClient | null
  /**
   * Federated login options enabled on `client`, ordered by display_order.
   * Carried on the bootstrap so the login page can render provider buttons on
   * first paint — no second round trip, and no in-flight OAuth authorize request
   * needed to justify the lookup. Optional only for resilience against an older
   * backend: treat a missing value exactly like an empty array.
   */
  connections?: TenantBootstrapConnection[]
  /**
   * Whether this surface offers passwordless email sign-in. Opt-in per client,
   * so treat a missing value as off.
   */
  magic_link_enabled?: boolean
}

/**
 * One federated login option (identity provider) on the resolved surface
 * client. Structurally identical to the `/oauth/connections` projection so the
 * login form can render either source through the same code path.
 */
export interface TenantBootstrapConnection {
  identifier: string
  display_name: string
  provider: string
  provider_type: string
  is_default: boolean
  display_order: number
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
 * Tenant API response
 */
export type TenantResponse = ApiResponse<TenantEntity>

/**
 * Tenant list API response
 */
export type TenantListResponse = ApiResponse<PaginatedResponse<TenantEntity>>
