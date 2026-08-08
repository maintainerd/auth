import { get, put } from '@/services'
import type { AuditConfig, AuditConfigPayload, AuditConfigResponse } from './types'

export async function fetchAuditConfig(): Promise<AuditConfig> {
  const response = await get<AuditConfigResponse>('/tenant-settings/audit')
  if (!response.success) throw new Error(response.message || 'Failed to fetch audit config')
  return response.data
}

export async function updateAuditConfig(data: AuditConfigPayload): Promise<AuditConfig> {
  const response = await put<AuditConfigResponse>('/tenant-settings/audit', data)
  if (!response.success) throw new Error(response.message || 'Failed to update audit config')
  return response.data
}
