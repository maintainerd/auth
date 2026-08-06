/**
 * User API Types
 */


import type { Status } from '@/types/status'

/**
 * User metadata type - define the structure of metadata here
 */
export type UserMetadata = {
  // Add specific fields as needed, or use string | number | boolean for values
  [key: string]: string | number | boolean | null;
};

/**
 * User status type - defines valid statuses for users
 */
export type UserStatus = Extract<Status, 'active' | 'inactive' | 'pending' | 'suspended'>

/**
 * User Profile type
 */
export type UserProfile = {
  profile_id: string
  first_name: string
  // The backend returns these as optional (omitempty) top-level fields.
  middle_name?: string
  last_name?: string
  display_name?: string
  birthdate?: string
  gender?: string
  email?: string
  timezone?: string
  language?: string
  profile_url?: string
  is_default: boolean
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

/**
 * User type
 */
export type User = {
  user_id: string
  username: string
  fullname: string
  email: string
  phone: string
  is_email_verified: boolean
  is_phone_verified: boolean
  is_profile_completed: boolean
  is_account_completed: boolean
  status: UserStatus
  metadata: UserMetadata
  tenant?: {
    tenant_id: string
    name: string
    display_name: string
    description: string
    status: string
    is_public: boolean
    is_system: boolean
    metadata: Record<string, unknown> | null
    created_at: string
    updated_at: string
  }
  created_at: string
  updated_at: string
  last_login_at?: string | null
  login_count?: number
  email_verified_at?: string | null
  phone_verified_at?: string | null
  external_id?: string | null
  created_by?: string | null
  updated_by?: string | null
}

/**
 * User Role type
 */
export type UserRole = {
  role_id: string
  name: string
  description: string
  is_default: boolean
  is_system: boolean
  status: string
  created_at: string
  updated_at: string
}

/**
 * User Identity type
 */
export type UserIdentity = {
  user_identity_id: string
  provider: string
  sub: string
  metadata: Record<string, unknown> | null
  // An identity belongs to the identity provider that issued it, never to a
  // single application: every client connected to that provider authenticates
  // with the same identity. The backend omits these when the provider could
  // not be resolved for the caller's tenant.
  identity_provider_id?: string
  identity_provider_name?: string
  created_at: string
  updated_at: string
  jit_provisioned_at?: string | null
  provisioning_source?: 'jit' | 'scim' | 'manual' | 'invite' | 'import' | null
}

/**
 * User list query parameters interface
 */
export interface UserQueryParams {
  search?: string
  username?: string
  email?: string
  phone?: string
  fullname?: string
  status?: string
  tenant_id?: string
  role_id?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  after_id?: number
}

/**
 * Paginated User list response interface
 */
export interface UserListResponse {
  rows: User[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Create User request interface
 */
export interface CreateUserRequest {
  username: string
  email: string
  phone?: string
  password: string
  status: UserStatus
  metadata?: UserMetadata
}

/**
 * Update User request interface
 */
export interface UpdateUserRequest {
  username?: string
  email?: string
  phone?: string
  status?: UserStatus
  metadata?: UserMetadata
}

/**
 * Update User status request interface
 */
export interface UpdateUserStatusRequest {
  status: UserStatus
}

/**
 * User Roles query parameters interface
 */
export interface UserRolesQueryParams {
  name?: string
  description?: string
  status?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Paginated User Roles response interface
 */
export interface UserRolesResponse {
  rows: UserRole[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * User Identities query parameters interface
 */
export interface UserIdentitiesQueryParams {
  provider?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Paginated User Identities response interface
 */
export interface UserIdentitiesResponse {
  rows: UserIdentity[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * User Profiles query parameters interface
 */
export interface UserProfilesQueryParams {
  first_name?: string
  last_name?: string
  email?: string
  phone?: string
  city?: string
  country?: string
  is_default?: boolean
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Paginated User Profiles response interface
 */
export interface UserProfilesResponse {
  rows: UserProfile[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Create User Profile request interface
 */
export interface CreateUserProfileRequest {
  first_name: string
  middle_name?: string
  last_name?: string
  display_name?: string
  birthdate?: string
  gender?: string
  email?: string
  timezone?: string
  language?: string
  profile_url?: string
  metadata?: Record<string, unknown>
}

/**
 * Update User Profile request interface
 */
export interface UpdateUserProfileRequest {
  first_name?: string
  middle_name?: string
  last_name?: string
  display_name?: string
  birthdate?: string
  gender?: string
  email?: string
  timezone?: string
  language?: string
  profile_url?: string
  metadata?: Record<string, unknown>
}

/**
 * Activity (auth-event) types — the read-only audit trail for a user.
 */
export interface UserActivityQueryParams {
  category?: string
  event_type?: string
  severity?: string
  result?: string
  date_from?: string
  date_to?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface AuthEvent {
  auth_event_id: string
  category: string
  event_type: string
  severity: string
  result: string
  ip_address: string
  user_agent?: string | null
  description?: string | null
  error_reason?: string | null
  created_at: string
}

export interface UserActivityResponse {
  rows: AuthEvent[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Session type — an active sign-in session for a user (admin view).
 */
export interface UserSession {
  session_id: string
  ip_address?: string | null
  user_agent?: string | null
  last_used_at?: string | null
  expires_at?: string | null
  absolute_expires_at?: string | null
  created_at: string
}

/** MFA configuration for a user (admin view). */
export interface UserMFAResponse {
  is_totp_enabled: boolean
  is_webauthn_enabled: boolean
  is_sms_enabled: boolean
  is_email_otp_enabled: boolean
  backup_codes_count: number
  webauthn_keys?: UserMFAWebAuthnKey[]
  mfa_enabled_at?: string | null
}

export interface UserMFAWebAuthnKey {
  credential_uuid: string
  name: string
  /** Scalar on the wire (user.UserMFAWebAuthnKeyDTO.Transport is a string). */
  transport?: string
  is_discoverable_credential?: boolean
  last_used_at?: string | null
  created_at: string
}

export interface UserConsent {
  uuid: string
  consent_type: string
  policy_version: string
  accepted: boolean
  ip_address?: string | null
  user_agent?: string | null
  created_at: string
}

export interface UserConsentsResponse {
  rows: UserConsent[]
  total: number
  page: number
  limit: number
  total_pages: number
}

export interface TrustedDevice {
  uuid: string
  device_fingerprint?: string
  device_name?: string
  location?: string
  ip_address?: string
  user_agent?: string
  last_seen_at?: string | null
  trusted_until: string | null
  created_at: string
}

export interface TrustedDevicesResponse {
  rows: TrustedDevice[]
  total: number
  page: number
  limit: number
  total_pages: number
}
