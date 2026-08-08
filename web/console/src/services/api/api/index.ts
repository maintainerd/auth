/**
 * API API
 * Handles API-related API calls
 */

import { get, post, put, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type {
  Api,
  ApiListResponse,
  ApiQueryParams,
  CreateApiRequest,
  UpdateApiRequest,
  UpdateApiStatusRequest
} from './types'

/**
 * Fetch APIs with optional filters and pagination
 */
export async function fetchApis(params?: ApiQueryParams): Promise<ApiListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.API}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<ApiListResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch APIs')
}

/**
 * Fetch a single API by ID
 */
export async function fetchApiById(apiId: string): Promise<Api> {
  const endpoint = `${API_ENDPOINTS.API}/${apiId}`
  const response = await get<ApiResponse<Api>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch API')
}

/**
 * Create a new API
 */
export async function createApi(data: CreateApiRequest): Promise<Api> {
  const endpoint = API_ENDPOINTS.API
  const response = await post<ApiResponse<Api>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to create API')
}

/**
 * Update an existing API
 */
export async function updateApi(apiId: string, data: UpdateApiRequest): Promise<Api> {
  const endpoint = `${API_ENDPOINTS.API}/${apiId}`
  const response = await put<ApiResponse<Api>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update API')
}

/**
 * Delete an API
 */
export async function deleteApi(apiId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.API}/${apiId}`
  const response = await deleteRequest<ApiResponse<void>>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete API')
  }
}

/**
 * Update API status
 */
export async function updateApiStatus(apiId: string, data: UpdateApiStatusRequest): Promise<Api> {
  const endpoint = `${API_ENDPOINTS.API}/${apiId}/status`
  const response = await put<ApiResponse<Api>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update API status')
}

// Export as api object
export const apiService = {
  fetchApis,
  fetchApiById,
  createApi,
  updateApi,
  deleteApi,
  updateApiStatus,
}

