/**
 * API API Types
 */

import type { Status } from '@/types/status'
import type { Service } from '../services/types'

/**
 * API status type - defines valid statuses for APIs only
 */
export type ApiStatus = Extract<Status, 'active' | 'inactive'>

/**
 * API entity
 */
export type Api = {
  api_id: string
  name: string
  display_name: string
  description: string
  identifier: string
  /**
   * Absent when the API has no owning service. The Go DTO declares it
   * `Service *ServiceResponseDTO \`json:"service,omitempty"\`` (internal/iam/types.go),
   * and toAPIResponseDTO only fills it when the preloaded relation is non-nil,
   * so the key is dropped from the payload entirely rather than sent as null.
   */
  service?: Service
  status: ApiStatus
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * API list query parameters interface
 */
export interface ApiQueryParams {
  name?: string
  display_name?: string
  description?: string
  identifier?: string
  service_id?: string
  is_system?: boolean
  status?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Paginated API list response interface
 */
export interface ApiListResponse {
  rows: Api[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Create API request interface
 */
export interface CreateApiRequest {
  name: string
  display_name: string
  description: string
  service_id: string
  status: ApiStatus
}

/**
 * Update API request interface
 */
export interface UpdateApiRequest {
  name: string
  display_name: string
  description: string
  status: ApiStatus
  service_id: string
}

/**
 * Update API status request interface
 */
export interface UpdateApiStatusRequest {
  status: ApiStatus
}
