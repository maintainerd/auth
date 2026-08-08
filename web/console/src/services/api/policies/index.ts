/**
 * Policy API
 * Handles policy-related API calls
 */

import { get, post, put, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type {
  PolicyListResponse,
  PolicyQueryParams,
  PolicyResponse,
  PolicyDetail,
  CreatePolicyRequest,
  UpdatePolicyRequest,
  UpdatePolicyStatusRequest,
  PolicyVersionHistory,
  PolicyVersionHistoryListResponse,
} from './types'
import type { ServiceListResponse, ServiceQueryParams } from '../services/types'

/**
 * Fetch policies with optional filters and pagination
 */
export async function fetchPolicies(params?: PolicyQueryParams): Promise<PolicyListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.POLICY}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<PolicyListResponse>>(endpoint)

  if (response.success && response.data) {
    return {
      ...response.data,
      rows: response.data.rows ?? [],
    }
  }

  throw new Error(response.message || 'Failed to fetch policies')
}

/**
 * Fetch a single policy by ID
 */
export async function fetchPolicyById(policyId: string): Promise<PolicyDetail> {
  const endpoint = `${API_ENDPOINTS.POLICY}/${policyId}`
  const response = await get<ApiResponse<PolicyDetail>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch policy')
}

/**
 * Create a new policy
 */
export async function createPolicy(data: CreatePolicyRequest): Promise<PolicyResponse> {
  const endpoint = API_ENDPOINTS.POLICY
  const response = await post<ApiResponse<PolicyResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to create policy')
}

/**
 * Update an existing policy
 */
export async function updatePolicy(policyId: string, data: UpdatePolicyRequest): Promise<PolicyResponse> {
  const endpoint = `${API_ENDPOINTS.POLICY}/${policyId}`
  const response = await put<ApiResponse<PolicyResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update policy')
}

/**
 * Delete a policy
 */
export async function deletePolicy(policyId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.POLICY}/${policyId}`
  const response = await deleteRequest<ApiResponse>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete policy')
  }
}

/**
 * Update policy status
 */
export async function updatePolicyStatus(policyId: string, data: UpdatePolicyStatusRequest): Promise<PolicyResponse> {
  const endpoint = `${API_ENDPOINTS.POLICY}/${policyId}/status`
  const response = await put<ApiResponse<PolicyResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update policy status')
}

/**
 * Fetch services that use a specific policy
 */
export async function fetchServicesByPolicy(policyId: string, params?: ServiceQueryParams): Promise<ServiceListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.POLICY}/${policyId}/services${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<ServiceListResponse>>(endpoint)

  if (response.success && response.data) {
    return {
      ...response.data,
      rows: response.data.rows ?? [],
    }
  }

  throw new Error(response.message || 'Failed to fetch services')
}

/**
 * Fetch the version history of a policy
 */
export async function fetchPolicyHistory(policyId: string): Promise<PolicyVersionHistoryListResponse> {
  const endpoint = `${API_ENDPOINTS.POLICY_HISTORY(policyId)}`
  const response = await get<ApiResponse<PolicyVersionHistoryListResponse>>(endpoint)
  if (response.success && response.data) return response.data
  throw new Error(response.message || 'Failed to fetch policy history')
}

/**
 * Fetch a specific version of a policy
 */
export async function fetchPolicyHistoryVersion(policyId: string, version: number): Promise<PolicyVersionHistory> {
  const endpoint = `${API_ENDPOINTS.POLICY_HISTORY(policyId)}/${version}`
  const response = await get<ApiResponse<PolicyVersionHistory>>(endpoint)
  if (response.success && response.data) return response.data
  throw new Error(response.message || 'Failed to fetch policy version')
}

// Export as policy object
export const policyService = {
  fetchPolicies,
  fetchPolicyById,
  createPolicy,
  updatePolicy,
  deletePolicy,
  updatePolicyStatus,
  fetchServicesByPolicy,
}
