/**
 * Registration Flow API (backend resource: /registration_flows)
 * Handles registration-flow-related API calls.
 *
 * The response types are declared truthfully (see ./types.ts) instead of being
 * re-mapped from `Record<string, unknown>`: the hand-written mappers this file
 * used to carry cast every payload through `as`, which silently hid the fact
 * that POST /{uuid}/roles returns a bare array rather than a paginated envelope.
 * The only normalization left is filling in the two collections the backend may
 * omit (`rows`, `required_fields`).
 */

import { get, post, put, patch, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type {
  RegistrationFlow,
  RegistrationFlowDetail,
  RegistrationFlowListResponse,
  RegistrationFlowQueryParams,
  RegistrationFlowRole,
  RegistrationFlowRolesQueryParams,
  RegistrationFlowRolesResponse,
  CreateRegistrationFlowRequest,
  UpdateRegistrationFlowRequest,
  UpdateRegistrationFlowStatusRequest
} from './types'

/** `required_fields` is JSON on the backend and may arrive as null. */
function withRequiredFields(flow: RegistrationFlowDetail): RegistrationFlowDetail {
  return {
    ...flow,
    required_fields: flow.required_fields ?? [],
  }
}

function buildQuery(params?: object): string {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  return queryParams.toString() ? `?${queryParams.toString()}` : ''
}

/**
 * Fetch registration flows with optional filters and pagination.
 * List rows are the lean projection — no `required_fields`, no nested client.
 */
export async function fetchRegistrationFlows(
  params?: RegistrationFlowQueryParams
): Promise<RegistrationFlowListResponse> {
  const endpoint = `${API_ENDPOINTS.REGISTRATION_FLOW}${buildQuery(params)}`
  const response = await get<ApiResponse<RegistrationFlowListResponse>>(endpoint)

  if (response.success && response.data) {
    return {
      ...response.data,
      rows: response.data.rows ?? [],
    }
  }

  throw new Error(response.message || 'Failed to fetch registration flows')
}

/**
 * Fetch a single registration flow by UUID (detail projection).
 */
export async function fetchRegistrationFlow(registrationFlowId: string): Promise<RegistrationFlowDetail> {
  const endpoint = `${API_ENDPOINTS.REGISTRATION_FLOW}/${registrationFlowId}`
  const response = await get<ApiResponse<RegistrationFlowDetail>>(endpoint)

  if (response.success && response.data) {
    return withRequiredFields(response.data)
  }

  throw new Error(response.message || 'Failed to fetch registration flow')
}

/**
 * Create a new registration flow. The response carries the server-derived,
 * immutable `identifier`.
 */
export async function createRegistrationFlow(
  data: CreateRegistrationFlowRequest
): Promise<RegistrationFlowDetail> {
  const endpoint = API_ENDPOINTS.REGISTRATION_FLOW
  const response = await post<ApiResponse<RegistrationFlowDetail>>(endpoint, data)

  if (response.success && response.data) {
    return withRequiredFields(response.data)
  }

  throw new Error(response.message || 'Failed to create registration flow')
}

/**
 * Update an existing registration flow. Omitted fields are left unchanged.
 */
export async function updateRegistrationFlow(
  registrationFlowId: string,
  data: UpdateRegistrationFlowRequest
): Promise<RegistrationFlowDetail> {
  const endpoint = `${API_ENDPOINTS.REGISTRATION_FLOW}/${registrationFlowId}`
  const response = await put<ApiResponse<RegistrationFlowDetail>>(endpoint, data)

  if (response.success && response.data) {
    return withRequiredFields(response.data)
  }

  throw new Error(response.message || 'Failed to update registration flow')
}

/**
 * Delete a registration flow
 */
export async function deleteRegistrationFlow(registrationFlowId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.REGISTRATION_FLOW}/${registrationFlowId}`
  const response = await deleteRequest<ApiResponse<void>>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete registration flow')
  }
}

/**
 * Update a registration flow's status. Deactivating a flow is the kill switch
 * for its published registration link.
 */
export async function updateRegistrationFlowStatus(
  registrationFlowId: string,
  data: UpdateRegistrationFlowStatusRequest
): Promise<RegistrationFlowDetail> {
  const endpoint = `${API_ENDPOINTS.REGISTRATION_FLOW}/${registrationFlowId}/status`
  const response = await patch<ApiResponse<RegistrationFlowDetail>>(endpoint, data)

  if (response.success && response.data) {
    return withRequiredFields(response.data)
  }

  throw new Error(response.message || 'Failed to update registration flow status')
}

/**
 * List roles assigned to a registration flow (paginated envelope).
 */
export async function fetchRegistrationFlowRoles(
  registrationFlowId: string,
  params?: RegistrationFlowRolesQueryParams
): Promise<RegistrationFlowRolesResponse> {
  const endpoint = `${API_ENDPOINTS.REGISTRATION_FLOW}/${registrationFlowId}/roles${buildQuery(params)}`
  const response = await get<ApiResponse<RegistrationFlowRolesResponse>>(endpoint)

  if (response.success && response.data) {
    return {
      ...response.data,
      rows: response.data.rows ?? [],
    }
  }

  throw new Error(response.message || 'Failed to fetch registration flow roles')
}

/**
 * Assign roles to a registration flow (full replacement of the flow's role set).
 *
 * NOTE: this endpoint returns a BARE ARRAY of the resulting roles, not the
 * paginated `{rows,total,...}` envelope that GET /{uuid}/roles returns. Treating
 * it as an envelope made a successful assignment surface as an error.
 */
export async function assignRegistrationFlowRoles(
  registrationFlowId: string,
  roleUuids: string[]
): Promise<RegistrationFlowRole[]> {
  const endpoint = `${API_ENDPOINTS.REGISTRATION_FLOW}/${registrationFlowId}/roles`
  const response = await post<ApiResponse<RegistrationFlowRole[]>>(endpoint, {
    role_uuids: roleUuids,
  })

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to assign roles to registration flow')
}

/**
 * Remove a single role from a registration flow. Returns the updated flow.
 */
export async function removeRegistrationFlowRole(
  registrationFlowId: string,
  roleUuid: string
): Promise<RegistrationFlowDetail> {
  const endpoint = `${API_ENDPOINTS.REGISTRATION_FLOW}/${registrationFlowId}/roles/${roleUuid}`
  const response = await deleteRequest<ApiResponse<RegistrationFlowDetail>>(endpoint)

  if (response.success && response.data) {
    return withRequiredFields(response.data)
  }

  throw new Error(response.message || 'Failed to remove role from registration flow')
}

// Export as registration flow object
export const registrationFlowService = {
  fetchRegistrationFlows,
  fetchRegistrationFlow,
  createRegistrationFlow,
  updateRegistrationFlow,
  deleteRegistrationFlow,
  updateRegistrationFlowStatus,
  fetchRegistrationFlowRoles,
  assignRegistrationFlowRoles,
  removeRegistrationFlowRole,
}

export type { RegistrationFlow, RegistrationFlowDetail }
