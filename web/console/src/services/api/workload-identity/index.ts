/**
 * Workload Identity Federation API
 * Handles workload identity federation API calls
 */

import { get, post, put, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import { unwrap, assertSuccess } from '../_lib/unwrap'
import type { ApiResponse } from '../types'
import type {
  WorkloadIdentityFederation,
  WorkloadIdentityListResponse,
  WorkloadIdentityQueryParams,
  CreateWorkloadIdentityRequest,
  UpdateWorkloadIdentityRequest,
} from './types'

const BASE = API_ENDPOINTS.WORKLOAD_IDENTITY_FEDERATIONS

/**
 * Fetch workload identity federations with optional filters and pagination
 */
export async function fetchWorkloadIdentities(
  params?: WorkloadIdentityQueryParams,
): Promise<WorkloadIdentityListResponse> {
  const queryParams = new URLSearchParams()
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }
  const endpoint = `${BASE}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const r = await get<ApiResponse<WorkloadIdentityListResponse>>(endpoint)
  return unwrap(r, 'fetch workload identity federations')
}

/**
 * Fetch a single workload identity federation by UUID
 */
export async function fetchWorkloadIdentityById(
  federationId: string,
): Promise<WorkloadIdentityFederation> {
  const r = await get<ApiResponse<WorkloadIdentityFederation>>(`${BASE}/${federationId}`)
  return unwrap(r, 'fetch workload identity federation')
}

/**
 * Create a new workload identity federation
 */
export async function createWorkloadIdentity(
  data: CreateWorkloadIdentityRequest,
): Promise<WorkloadIdentityFederation> {
  const r = await post<ApiResponse<WorkloadIdentityFederation>>(BASE, data)
  return unwrap(r, 'create workload identity federation')
}

/**
 * Update an existing workload identity federation
 */
export async function updateWorkloadIdentity(
  federationId: string,
  data: UpdateWorkloadIdentityRequest,
): Promise<WorkloadIdentityFederation> {
  const r = await put<ApiResponse<WorkloadIdentityFederation>>(`${BASE}/${federationId}`, data)
  return unwrap(r, 'update workload identity federation')
}

/**
 * Delete a workload identity federation
 */
export async function deleteWorkloadIdentity(federationId: string): Promise<void> {
  const r = await deleteRequest<ApiResponse<void>>(`${BASE}/${federationId}`)
  assertSuccess(r, 'delete workload identity federation')
}
