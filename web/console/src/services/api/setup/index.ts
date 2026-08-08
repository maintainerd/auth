/**
 * Setup Service
 * Service layer for setup-related API calls
 */
import { get, post } from '../client'
import { API_ENDPOINTS } from '../config'
import type {
  CreateTenantRequest,
  CreateTenantResponse,
  CreateAdminRequest,
  CreateAdminResponse,
  SetupCreateProfileRequest,
  CreateProfileResponse,
  SetupStatusResponse,
  CompleteSetupResponse,
} from './types'

export async function createTenant(data: CreateTenantRequest): Promise<CreateTenantResponse> {
  return post<CreateTenantResponse>(API_ENDPOINTS.SETUP.CREATE_TENANT, data)
}

export async function createAdmin(data: CreateAdminRequest): Promise<CreateAdminResponse> {
  return post<CreateAdminResponse>(API_ENDPOINTS.SETUP.CREATE_ADMIN, data)
}

export async function createProfile(data: SetupCreateProfileRequest): Promise<CreateProfileResponse> {
  return post<CreateProfileResponse>(API_ENDPOINTS.SETUP.CREATE_PROFILE, data)
}

export async function getSetupStatus(): Promise<SetupStatusResponse> {
  return get<SetupStatusResponse>(API_ENDPOINTS.SETUP.STATUS)
}

export async function completeSetup(): Promise<CompleteSetupResponse> {
  return post<CompleteSetupResponse>(API_ENDPOINTS.SETUP.COMPLETE, {})
}

export async function getDefaultTenantMetadata() {
  return {
    application_logo_url: '',
    favicon_url: '',
    language: 'en',
    timezone: 'UTC',
    date_format: 'YYYY-MM-DD',
    time_format: '24h',
    privacy_policy_url: '',
    term_of_service_url: '',
  }
}

export async function createTenantWithDefaults(
  name: string,
  display_name: string,
  description: string,
): Promise<CreateTenantResponse> {
  const defaultMetadata = await getDefaultTenantMetadata()
  return createTenant({
    name,
    display_name,
    description,
    metadata: defaultMetadata,
  })
}

export async function isSetupCompleted(): Promise<boolean> {
  try {
    const response = await getSetupStatus()
    return response.success && response.data != null && response.data.is_setup_complete === true
  } catch {
    return false
  }
}

export const setupService = {
  createTenant,
  createAdmin,
  getSetupStatus,
  completeSetup,
  getDefaultTenantMetadata,
  createTenantWithDefaults,
  isSetupCompleted,
}
