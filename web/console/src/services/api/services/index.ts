/**
 * Service API
 * Handles service-related API calls
 */

import { get, post, put, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type {
  ServiceListResponse,
  ServiceQueryParams,
  ServiceResponse,
  CreateServiceRequest,
  UpdateServiceRequest,
  UpdateServiceStatusRequest
} from './types'

/**
 * Fetch services with optional filters and pagination
 */
export async function fetchServices(params?: ServiceQueryParams): Promise<ServiceListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.SERVICE}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<ServiceListResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch services')
}

/**
 * Fetch a single service by ID
 */
export async function fetchServiceById(serviceId: string): Promise<ServiceResponse> {
  const endpoint = `${API_ENDPOINTS.SERVICE}/${serviceId}`
  const response = await get<ApiResponse<ServiceResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch service')
}

/**
 * Create a new service
 */
export async function createService(data: CreateServiceRequest): Promise<ServiceResponse> {
  const endpoint = API_ENDPOINTS.SERVICE
  const response = await post<ApiResponse<ServiceResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to create service')
}

/**
 * Update an existing service
 */
export async function updateService(serviceId: string, data: UpdateServiceRequest): Promise<ServiceResponse> {
  const endpoint = `${API_ENDPOINTS.SERVICE}/${serviceId}`
  const response = await put<ApiResponse<ServiceResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update service')
}

/**
 * Delete a service
 */
export async function deleteService(serviceId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.SERVICE}/${serviceId}`
  const response = await deleteRequest<ApiResponse<void>>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete service')
  }
}

/**
 * Update service status
 */
export async function updateServiceStatus(serviceId: string, data: UpdateServiceStatusRequest): Promise<ServiceResponse> {
  const endpoint = `${API_ENDPOINTS.SERVICE}/${serviceId}/status`
  const response = await put<ApiResponse<ServiceResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update service status')
}

/**
 * Assign a policy to a service
 */
export async function assignPolicyToService(serviceId: string, policyId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.SERVICE}/${serviceId}/policies/${policyId}`
  const response = await post<ApiResponse>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to assign policy to service')
  }
}

/**
 * Remove a policy from a service
 */
export async function removePolicyFromService(serviceId: string, policyId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.SERVICE}/${serviceId}/policies/${policyId}`
  const response = await deleteRequest<ApiResponse>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to remove policy from service')
  }
}

// Export as service object
export const serviceService = {
  fetchServices,
  fetchServiceById,
  createService,
  updateService,
  deleteService,
  updateServiceStatus,
  assignPolicyToService,
  removePolicyFromService,
}
