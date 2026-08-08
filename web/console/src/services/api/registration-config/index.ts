import { get, put } from '@/services'
import type {
  RegistrationConfig,
  RegistrationConfigPayload,
  RegistrationConfigResponse,
} from './types'

const API_ENDPOINTS = {
  REGISTRATION_CONFIG: '/security-settings/registration',
}

export async function fetchRegistrationConfig(): Promise<RegistrationConfig> {
  const response = await get<RegistrationConfigResponse>(API_ENDPOINTS.REGISTRATION_CONFIG)

  if (!response.success) {
    throw new Error(response.message || 'Failed to fetch registration config')
  }

  return response.data
}

export async function updateRegistrationConfig(
  data: RegistrationConfigPayload
): Promise<RegistrationConfig> {
  const response = await put<RegistrationConfigResponse>(
    API_ENDPOINTS.REGISTRATION_CONFIG,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update registration config')
  }

  return response.data
}
