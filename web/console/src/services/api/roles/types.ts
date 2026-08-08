/**
 * Role API Types
 */

import type { Status } from '@/types/status'

/**
 * Role status type - defines valid statuses for Roles only
 */
export type RoleStatus = Extract<Status, 'active' | 'inactive'>

/**
 * Role type
 */
export type Role = {
  role_id: string
  name: string
  description: string
  is_default: boolean
  is_system: boolean
  status: RoleStatus
  created_at: string
  updated_at: string
}

/**
 * Role list query parameters interface
 */
export interface RoleQueryParams {
  search?: string
  name?: string
  description?: string
  is_default?: boolean
  is_system?: boolean
  status?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Paginated Role list response interface
 */
export interface RoleListResponse {
  rows: Role[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Single Role response interface
 */
export interface RoleResponse {
  role_id: string
  name: string
  description: string
  is_default: boolean
  is_system: boolean
  status: RoleStatus
  created_at: string
  updated_at: string
}

/**
 * Create Role request interface
 */
export interface CreateRoleRequest {
  name: string
  description: string
  status: RoleStatus
}

/**
 * Update Role request interface
 */
export interface UpdateRoleRequest {
  name: string
  description: string
  status: RoleStatus
}

/**
 * Update Role status request interface
 */
export interface UpdateRoleStatusRequest {
  status: RoleStatus
}

/**
 * Role permissions list response interface
 */
export interface RolePermissionsListResponse {
  rows: RolePermissionEntity[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Role permission entity
 */
export interface RolePermissionEntity {
  permission_id: string
  name: string
  description: string
  api?: {
    api_id: string
    name: string
    display_name: string
    description: string
    identifier: string
    status: string
    is_default: boolean
    is_system: boolean
    created_at: string
    updated_at: string
  }
  status: string
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Add permissions to role request interface
 */
export interface AddRolePermissionsRequest {
  permissions: string[]
}
