import { get, put } from '@/services'
import type {
  ThreatDetectionSettings,
  ThreatDetectionSettingsPayload,
  ThreatDetectionSettingsResponse,
} from './types'

const API_ENDPOINTS = {
  THREAT_SETTINGS: '/security-settings/threat',
}

export async function fetchThreatDetectionSettings(): Promise<ThreatDetectionSettings> {
  const response = await get<ThreatDetectionSettingsResponse>(API_ENDPOINTS.THREAT_SETTINGS)

  if (!response.success) {
    throw new Error(response.message || 'Failed to fetch threat detection settings')
  }

  return response.data
}

export async function updateThreatDetectionSettings(
  data: ThreatDetectionSettingsPayload
): Promise<ThreatDetectionSettings> {
  const response = await put<ThreatDetectionSettingsResponse>(
    API_ENDPOINTS.THREAT_SETTINGS,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to update threat detection settings')
  }

  return response.data
}
