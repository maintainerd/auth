import { get } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type { AuthEventQueryParams, AuthEventListResponse, AuthEvent } from './types'

export async function exportAuthEvents(
  format: 'csv' | 'json',
  params?: Omit<AuthEventQueryParams, 'page' | 'limit'>
): Promise<void> {
  const queryParams = new URLSearchParams({ format })
  if (params?.event_type) queryParams.set('event_type', params.event_type)
  if (params?.category) queryParams.set('category', params.category)
  if (params?.date_from) queryParams.set('date_from', params.date_from)
  if (params?.date_to) queryParams.set('date_to', params.date_to)
  if (params?.result) queryParams.set('result', params.result)
  if (params?.severity) queryParams.set('severity', params.severity)

  // Use window.open to trigger download since get() parses JSON
  const url = `${API_ENDPOINTS.AUTH_EVENTS_EXPORT}?${queryParams.toString()}`
  window.open(url, '_blank')
}

export async function fetchAuthEventCount(eventType: string): Promise<number> {
  const response = await get<ApiResponse<{ count: number }>>(
    `${API_ENDPOINTS.AUTH_EVENTS_COUNT}?event_type=${encodeURIComponent(eventType)}`
  )
  if (response.success && response.data) {
    return response.data.count ?? 0
  }
  return 0
}

export async function fetchAuthEvents(params?: AuthEventQueryParams): Promise<AuthEventListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.AUTH_EVENTS}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<AuthEventListResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch auth events')
}

export async function fetchAuthEventById(eventId: string): Promise<AuthEvent> {
  const endpoint = `${API_ENDPOINTS.AUTH_EVENTS}/${eventId}`
  const response = await get<ApiResponse<AuthEvent>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch auth event')
}
