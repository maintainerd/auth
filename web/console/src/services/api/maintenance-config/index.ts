import { get, put } from '@/services'
import type { MaintenanceConfig, MaintenanceConfigPayload, MaintenanceConfigResponse } from './types'

export async function fetchMaintenanceConfig(): Promise<MaintenanceConfig> {
  const response = await get<MaintenanceConfigResponse>('/tenant-settings/maintenance')
  if (!response.success) throw new Error(response.message || 'Failed to fetch maintenance config')
  return response.data
}

export async function updateMaintenanceConfig(data: MaintenanceConfigPayload): Promise<MaintenanceConfig> {
  const response = await put<MaintenanceConfigResponse>('/tenant-settings/maintenance', data)
  if (!response.success) throw new Error(response.message || 'Failed to update maintenance config')
  return response.data
}
