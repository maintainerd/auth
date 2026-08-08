/**
 * Permission API Types
 * Type definitions for permission-related API requests and responses
 */

import type { Api } from '../api/types'
import type { Status } from '@/types/status'

/**
 * Permission status type
 */
export type PermissionStatus = Extract<Status, 'active' | 'inactive'>

/**
 * Permission entity from API
 */
export interface PermissionEntity {
  permission_id: string
  name: string
  description: string
  api: Api
  status: PermissionStatus
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Permission query parameters
 */
export interface PermissionQueryParams {
  name?: string
  description?: string
  api_id?: string
  role_id?: string
  client_id?: string
  // `status` and `is_system` are what PermissionFilterDTO actually reads
  // (internal/iam/handler_permission.go). There is no `is_active` query param on
  // the permission list endpoint; the old key here had no consumers and no
  // server-side effect.
  status?: string
  is_system?: boolean
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Permission list response
 */
export interface PermissionListResponse {
  rows: PermissionEntity[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Permission response (single permission)
 */
export interface PermissionResponse {
  permission: PermissionEntity
}

/**
 * Create permission request
 */
export interface CreatePermissionRequest {
  name: string
  description: string
  api_id: string
  status?: PermissionStatus
}

/**
 * Update permission request
 */
export interface UpdatePermissionRequest {
  name?: string
  description?: string
  status?: PermissionStatus
}

/**
 * Update permission status request
 */
export interface UpdatePermissionStatusRequest {
  status: PermissionStatus
}
