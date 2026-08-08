/**
 * Authentication API Types
 * Type definitions for authentication-related API requests and responses
 */

import type { ApiResponse } from '../types'

/**
 * Profile entity from API
 */
export interface ProfileEntity {
  profile_id: string
  first_name: string
  last_name: string
  display_name: string
  bio?: string
  birthdate?: string
  gender?: string
  phone?: string
  email: string
  address?: string
  city?: string
  country?: string
  timezone?: string
  language?: string
  created_at: string
  updated_at: string
}

/**
 * Account entity — consolidated user info returned by GET /account
 */
export interface AccountEntity {
  user_id: string
  email: string
  phone: string
  email_verified: boolean
  phone_verified: boolean
  profiles: AccountProfileEntity[]
  roles: string[]
  permissions: string[]
  tenant: { tenant_id: string; name: string; display_name: string }
}

export interface AccountProfileEntity {
  profile_id: string
  first_name: string
  last_name?: string | null
  display_name?: string | null
  default: boolean
}

export interface LoginRequest {
  username: string
  password: string
}

export type LoginResponse = ApiResponse<{
  access_token?: string
  id_token?: string
  refresh_token?: string
  expires_in?: number
  token_type?: string
  issued_at?: number
  // Present when the account has MFA enrolled: the password step succeeded but
  // a second factor is required to finish login (and elevate the session to acr=2).
  mfa_required?: boolean
  mfa_challenge_token?: string
  mfa_allowed_methods?: string[]
  mfa_totp_digits?: number
}>

// MFALoginVerifyRequest completes the login MFA second step. Exactly one proof
// is sent: `code` (totp/sms/backup_code) or `assertion` (passkey).
export interface MFALoginVerifyRequest {
  mfa_challenge_token: string
  method: string
  code?: string
  assertion?: unknown
  // When true, the backend records this device as trusted so future logins from
  // it can skip the MFA step until the tenant's trusted-device window expires.
  remember_device?: boolean
}

export interface RefreshTokenRequest {
  refresh_token: string
}

export type RefreshTokenResponse = ApiResponse<{
  access_token: string
  expires_in: number
  token_type: string
  issued_at: number
}>

export interface LogoutRequest {
  refresh_token?: string
}

export type LogoutResponse = ApiResponse

export interface RegisterRequest {
  username: string
  email: string
  password: string
  // Sent only when the resolved registration context requires them. The server
  // enforces the requirement (enforceRequiredRegistrationFields); these fields
  // exist so the form can satisfy it rather than 400 with no way forward.
  fullname?: string
  phone?: string
}

// RegistrationContext is what the signup form must collect, resolved from the
// client and (optionally) the registration flow named in the link.
export interface RegistrationContext {
  registration_flow: string
  required_fields: RegistrationRequiredField[]
  verification_required: boolean
}

export type RegistrationRequiredField = 'fullname' | 'email' | 'phone'


export type RegisterResponse = ApiResponse<{
  user_id: string
  username: string
  email: string
  fullname: string
  phone?: string
  created_at: string
}>

// RegisterInviteRequest — body for POST /register/invite
export interface RegisterInviteRequest {
  username: string
  password: string
}

// RegisterInviteQueryParams — signed query params included in the invite URL
export interface RegisterInviteQueryParams {
  invite_token: string
  expires: string
  sig: string
  email?: string
  callback_url?: string
  client_id?: string
  tenant_id?: string
}

export type ProfileResponse = ApiResponse<ProfileEntity>

export interface ForgotPasswordRequest {
  email: string
}

export type ForgotPasswordResponse = ApiResponse

export interface ResetPasswordRequest {
  new_password: string
}

export interface ResetPasswordQueryParams {
  client_id?: string
  tenant_id?: string
  expires: string
  sig: string
  token: string
}

export type ResetPasswordResponse = ApiResponse

// Profile creation types for authenticated users (using /profiles endpoint)
export interface CreateProfileRequest {
  first_name: string
  last_name: string
  display_name: string
  bio?: string
  birthdate?: string
  gender?: string
  phone?: string
  email: string
  address?: string
  city?: string
  country?: string
  timezone?: string
  language?: string
}

export type CreateProfileResponse = ApiResponse<ProfileEntity>
