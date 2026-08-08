import { get, put } from '@/services'
import type {
  SessionSettings,
  SessionSettingsPayload,
  SessionSettingsResponse,
} from './types'

const API_ENDPOINTS = {
  SESSION_SETTINGS: '/security-settings/session',
}

export async function fetchSessionSettings(): Promise<SessionSettings> {
  const response = await get<SessionSettingsResponse>(API_ENDPOINTS.SESSION_SETTINGS)

  if (!response.success) {
    throw new Error(response.message || 'Failed to fetch session settings')
  }

  return response.data
}

export async function updateSessionSettings(
  data: SessionSettingsPayload
): Promise<SessionSettings> {
  const response = await put<SessionSettingsResponse>(
    API_ENDPOINTS.SESSION_SETTINGS,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update session settings')
  }

  return response.data
}
