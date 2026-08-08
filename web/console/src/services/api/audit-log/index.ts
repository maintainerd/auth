import { get } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type { AuditLogQueryParams, AuditLogListResponse, AuditLogEntry } from './types'

export async function fetchAuditLog(params?: AuditLogQueryParams): Promise<AuditLogListResponse> {
  const queryParams = new URLSearchParams()
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }
  const endpoint = `${API_ENDPOINTS.MANAGEMENT_AUDIT_LOG}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<AuditLogListResponse>>(endpoint)
  if (response.success && response.data) return response.data
  throw new Error(response.message || 'Failed to fetch audit log')
}

export async function fetchAuditLogEntry(uuid: string): Promise<AuditLogEntry> {
  const endpoint = `${API_ENDPOINTS.MANAGEMENT_AUDIT_LOG}/${uuid}`
  const response = await get<ApiResponse<AuditLogEntry>>(endpoint)
  if (response.success && response.data) return response.data
  throw new Error(response.message || 'Failed to fetch audit log entry')
}
