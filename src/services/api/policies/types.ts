/**
 * Policy API Types
 */

import type { Status } from '@/types/status'

/**
 * Policy status type - defines valid statuses for policies only
 */
export type PolicyStatus = Extract<Status, 'active' | 'inactive'>

/**
 * Policy statement type
 */
export type PolicyStatement = {
  effect: 'allow' | 'deny'
  action: string[]
  resource: string[]
}

/**
 * Policy document type
 */
export type PolicyDocument = {
  version: string
  statement: PolicyStatement[]
}

/**
 * Policy type (for list view)
 */
export type Policy = {
  policy_id: string
  name: string
  description: string | null
  version: string
  status: PolicyStatus
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Policy detail type (includes document)
 */
export type PolicyDetail = {
  policy_id: string
  name: string
  description: string | null
  document: PolicyDocument
  version: string
  status: PolicyStatus
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Policy query parameters interface
 */
export interface PolicyQueryParams {
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  name?: string
  description?: string
  version?: string
  status?: string
  is_system?: boolean
  service_id?: string
}

/**
 * Policy list response interface
 */
export interface PolicyListResponse {
  rows: Policy[]
  total: number
  page: number
  limit: number
  total_pages: number
}

/**
 * Single policy response interface
 */
export interface PolicyResponse {
  policy_id: string
  name: string
  description: string | null
  version: string
  status: PolicyStatus
  is_system: boolean
  created_at: string
  updated_at: string
}

/**
 * Create policy request interface
 */
export interface CreatePolicyRequest {
  name: string
  description: string
  version: string
  status: PolicyStatus
  document: PolicyDocument
}

/**
 * Update policy request interface
 */
export interface UpdatePolicyRequest {
  name: string
  description: string
  version: string
  status: PolicyStatus
  document: PolicyDocument
  /**
   * Recorded on the policy's version-history row. Optional, but it is what turns
   * "the document changed" into an answerable audit trail — the history tab shows
   * the before-snapshot, and without a reason nobody can tell why it changed.
   */
  change_reason?: string
}

/**
 * Update policy status request interface
 */
export interface UpdatePolicyStatusRequest {
  status: PolicyStatus
}

export interface PolicyVersionHistory {
  version_number: number
  document: PolicyDocument
  snapshot_at: string
  changed_by_user_id?: string | null
}

export interface PolicyVersionHistoryListResponse {
  rows: PolicyVersionHistory[]
  total: number
  page: number
  limit: number
  total_pages: number
}
