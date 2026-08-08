import { get, put } from '@/services'
import type {
  LockoutConfig,
  LockoutConfigPayload,
  LockoutConfigResponse,
} from './types'

const API_ENDPOINTS = {
  LOCKOUT_CONFIG: '/security-settings/lockout',
}

export async function fetchLockoutConfig(): Promise<LockoutConfig> {
  const response = await get<LockoutConfigResponse>(API_ENDPOINTS.LOCKOUT_CONFIG)

  if (!response.success) {
    throw new Error(response.message || 'Failed to fetch lockout config')
  }

  return response.data
}

export async function updateLockoutConfig(
  data: LockoutConfigPayload
): Promise<LockoutConfig> {
  const response = await put<LockoutConfigResponse>(
    API_ENDPOINTS.LOCKOUT_CONFIG,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update lockout config')
  }

  return response.data
}
