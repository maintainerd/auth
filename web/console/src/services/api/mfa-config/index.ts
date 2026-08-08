import { get, put } from '@/services'
import type {
  MfaConfig,
  MfaConfigPayload,
  MfaConfigResponse,
} from './types'

const API_ENDPOINTS = {
  MFA_CONFIG: '/security-settings/mfa',
}

export async function fetchMfaConfig(): Promise<MfaConfig> {
  const response = await get<MfaConfigResponse>(API_ENDPOINTS.MFA_CONFIG)

  if (!response.success) {
    throw new Error(response.message || 'Failed to fetch MFA config')
  }

  return response.data
}

export async function updateMfaConfig(
  data: MfaConfigPayload
): Promise<MfaConfig> {
  const response = await put<MfaConfigResponse>(
    API_ENDPOINTS.MFA_CONFIG,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update MFA config')
  }

  return response.data
}
