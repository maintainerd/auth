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

export type LogoutResponse = ApiResponse

export type ProfileResponse = ApiResponse<ProfileEntity>

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
