/**
 * Service API Types
 */

import type { Status } from '@/types/status'

/**
 * Service status type - defines valid statuses for services only
 */
export type ServiceStatus = Extract<Status, 'active' | 'maintenance' | 'deprecated' | 'inactive'>

/**
 * Service type
 */
export type Service = {
  service_id: string
  name: string
  display_name: string
  description: string
  version: string
  status: ServiceStatus
  is_system: boolean
  api_count: number
  policy_count: number
  created_at: string
  updated_at: string
}

/**
 * Service list query parameters interface
 */
export interface ServiceQueryParams {
  name?: string
  display_name?: string
  description?: string
  version?: string
  status?: string
  is_system?: boolean
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Paginated service list response interface
 */
export interface ServiceListResponse {
  rows: Service[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Single service response interface
 */
export interface ServiceResponse {
  service_id: string
  name: string
  display_name: string
  description: string
  version: string
  status: ServiceStatus
  is_system: boolean
  api_count: number
  policy_count: number
  created_at: string
  updated_at: string
}

/**
 * Create service request interface
 */
export interface CreateServiceRequest {
  name: string
  display_name: string
  description: string
  version: string
  status: ServiceStatus
}

/**
 * Update service request interface
 */
export interface UpdateServiceRequest {
  name: string
  display_name: string
  description: string
  version: string
  status: ServiceStatus
}

/**
 * Update service status request interface
 */
export interface UpdateServiceStatusRequest {
  status: ServiceStatus
}
