/**
 * Setup API Types
 * Type definitions for setup-related API requests and responses
 */
import type { ApiResponse } from '../types'

export interface CreateTenantRequest {
  name: string
  display_name: string
  description?: string
  metadata?: {
    application_logo_url?: string
    favicon_url?: string
    language?: string
    timezone?: string
    date_format?: string
    time_format?: string
    privacy_policy_url?: string
    term_of_service_url?: string
  }
}

export interface TenantData {
  tenant_id: string
  name: string
  display_name: string
  description: string
  identifier: string
  status: string
  is_public: boolean
  is_system: boolean
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

export type CreateTenantResponse = ApiResponse<TenantData>

export interface CreateAdminRequest {
  username: string
  fullname?: string
  password: string
  email: string
}

export interface AdminData {
  user_id: string
  username: string
  fullname: string
  email: string
  phone: string
  is_email_verified: boolean
  is_phone_verified: boolean
  is_profile_completed: boolean
  is_account_completed: boolean
  status: string
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

export type CreateAdminResponse = ApiResponse<AdminData>

export interface SetupCreateProfileRequest {
  first_name: string
  middle_name?: string
  last_name?: string
  suffix?: string
  display_name?: string
  birthdate?: string
  gender?: string
  bio?: string
  phone?: string
  email?: string
  address?: string
  city?: string
  country?: string
  timezone?: string
  language?: string
  profile_url?: string
  metadata?: Record<string, unknown>
}

export interface ProfileData {
  profile_id: string
  first_name: string
  middle_name?: string | null
  last_name?: string | null
  suffix?: string | null
  display_name?: string | null
  bio?: string | null
  birthdate?: string | null
  gender?: string | null
  phone?: string | null
  email?: string | null
  address?: string | null
  city?: string | null
  country?: string | null
  timezone?: string | null
  language?: string | null
  profile_url?: string | null
  is_default: boolean
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export type CreateProfileResponse = ApiResponse<ProfileData>

export interface SetupStatusData {
  is_tenant_setup: boolean
  is_admin_setup: boolean
  is_profile_setup: boolean
  is_setup_complete: boolean
}

export type SetupStatusResponse = ApiResponse<SetupStatusData>

export interface CompleteSetupData {
  is_setup_complete: boolean
}

export type CompleteSetupResponse = ApiResponse<CompleteSetupData>
