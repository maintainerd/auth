/**
 * Client API Service
 * Service for managing client-related API calls
 */

import { get, post, put, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type { Role } from '../roles/types'
import type {
  ClientQueryParams,
  ClientListResponse,
  ClientResponse,
  ClientCreateResponse,
  CreateClientRequest,
  UpdateClientRequest,
  UpdateClientStatusRequest,
  RotateClientSecretRequest,
  RotateClientSecretResponse,
  ClientConfig,
  ClientUrisResponse,
  ClientUri,
  CreateClientUriRequest,
  UpdateClientUriRequest,
  ClientApisResponse,
  AddClientApisRequest,
  AddClientApiPermissionsRequest,
  AddClientIdentityProviderRequest,
} from './types'

/**
 * Fetch clients with optional filters and pagination
 */
export async function fetchClients(params?: ClientQueryParams): Promise<ClientListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.CLIENT}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<ClientListResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch clients')
}

/**
 * Fetch a single client by ID
 */
export async function fetchClientById(clientId: string): Promise<ClientResponse> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}`
  const response = await get<ApiResponse<ClientResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch client')
}

/**
 * Create a new client
 */
export async function createClient(data: CreateClientRequest): Promise<ClientCreateResponse> {
  const endpoint = API_ENDPOINTS.CLIENT
  const response = await post<ApiResponse<ClientCreateResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to create client')
}

/**
 * Update an existing client
 */
export async function updateClient(clientId: string, data: UpdateClientRequest): Promise<ClientResponse> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}`
  const response = await put<ApiResponse<ClientResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update client')
}

/**
 * Delete a client
 */
export async function deleteClient(clientId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}`
  const response = await deleteRequest<ApiResponse<void>>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete client')
  }
}

/**
 * Connect an identity provider to an existing client.
 */
export async function addClientIdentityProvider(
  clientId: string,
  data: AddClientIdentityProviderRequest,
): Promise<ClientResponse> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/identity_providers`
  const response = await post<ApiResponse<ClientResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to connect identity provider')
}

/**
 * Disconnect an identity provider from a client by removing the relationship row.
 */
export async function removeClientIdentityProvider(
  clientId: string,
  connectionId: string,
): Promise<ClientResponse> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/identity_providers/${connectionId}`
  const response = await deleteRequest<ApiResponse<ClientResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to disconnect identity provider')
}

export interface UpdateClientIdentityProviderRequest {
  is_default?: boolean
  enabled?: boolean
  display_order?: number
}

export async function updateClientIdentityProvider(
  clientId: string,
  connectionId: string,
  data: UpdateClientIdentityProviderRequest,
): Promise<ClientResponse> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/identity_providers/${connectionId}`
  const response = await put<ApiResponse<ClientResponse>>(endpoint, data)
  if (response.success && response.data) return response.data
  throw new Error(response.message || 'Failed to update identity provider connection')
}

/**
 * Update client status
 */
export async function updateClientStatus(clientId: string, data: UpdateClientStatusRequest): Promise<ClientResponse> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/status`
  const response = await put<ApiResponse<ClientResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update client status')
}

/**
 * Rotate client secret by ID. The plaintext secret is returned exactly once.
 */
export async function rotateClientSecret(
  clientId: string,
  data: RotateClientSecretRequest = { grace_period_hours: 24 },
): Promise<RotateClientSecretResponse> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/rotate-secret`
  const response = await post<ApiResponse<RotateClientSecretResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to rotate client secret')
}

/**
 * Fetch client config by ID
 */
export async function fetchClientConfig(clientId: string): Promise<ClientConfig> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/config`
  const response = await get<ApiResponse<ClientConfig>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch client config')
}

/**
 * Fetch client URIs by client ID
 */
export async function fetchClientUris(clientId: string): Promise<ClientUrisResponse> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/uris`
  const response = await get<ApiResponse<ClientUrisResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch client URIs')
}

/**
 * Create a new client URI
 */
export async function createClientUri(clientId: string, data: CreateClientUriRequest): Promise<ClientUri> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/uris`
  const response = await post<ApiResponse<ClientUri>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to create client URI')
}

/**
 * Update an existing client URI
 */
export async function updateClientUri(clientId: string, clientUriId: string, data: UpdateClientUriRequest): Promise<ClientUri> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/uris/${clientUriId}`
  const response = await put<ApiResponse<ClientUri>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update client URI')
}

/**
 * Delete a client URI
 */
export async function deleteClientUri(clientId: string, clientUriId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.CLIENT}/${clientId}/uris/${clientUriId}`
  const response = await deleteRequest<ApiResponse<void>>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete client URI')
  }
}

/**
 * Fetch client APIs by client ID
 */
export async function fetchClientApis(clientId: string): Promise<ClientApisResponse> {
  const response = await get<ApiResponse<ClientApisResponse>>(
    `${API_ENDPOINTS.CLIENT}/${clientId}/apis`
  )

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch client APIs')
}

/**
 * Add APIs to a client
 */
export async function addClientApis(clientId: string, data: AddClientApisRequest): Promise<void> {
  const response = await post<ApiResponse<void>>(
    `${API_ENDPOINTS.CLIENT}/${clientId}/apis`,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to add APIs to client')
  }
}

/**
 * Remove an API from a client
 */
export async function removeClientApi(clientId: string, apiId: string): Promise<void> {
  const response = await deleteRequest<ApiResponse<void>>(
    `${API_ENDPOINTS.CLIENT}/${clientId}/apis/${apiId}`
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to remove API from client')
  }
}

/**
 * Add permissions to a client API
 */
export async function addClientApiPermissions(
  clientId: string,
  apiId: string,
  data: AddClientApiPermissionsRequest
): Promise<void> {
  const response = await post<ApiResponse<void>>(
    `${API_ENDPOINTS.CLIENT}/${clientId}/apis/${apiId}/permissions`,
    data
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to add permissions to client API')
  }
}

/**
 * Remove a permission from a client API
 */
export async function removeClientApiPermission(
  clientId: string,
  apiId: string,
  permissionId: string
): Promise<void> {
  const response = await deleteRequest<ApiResponse<void>>(
    `${API_ENDPOINTS.CLIENT}/${clientId}/apis/${apiId}/permissions/${permissionId}`
  )

  if (!response.success) {
    throw new Error(response.message || 'Failed to remove permission from client API')
  }
}

/**
 * Fetch roles assigned to a client
 */
export async function fetchClientRoles(clientId: string): Promise<Role[]> {
  const response = await get<ApiResponse<Role[]>>(API_ENDPOINTS.CLIENT_ROLES(clientId))
  if (response.success && response.data) return response.data
  throw new Error(response.message || 'Failed to fetch client roles')
}

/**
 * Assign a role to a client
 */
export async function addClientRole(clientId: string, roleUuid: string): Promise<void> {
  const response = await post<ApiResponse<void>>(
    API_ENDPOINTS.CLIENT_ROLES(clientId),
    { role_uuid: roleUuid },
  )
  if (!response.success) throw new Error(response.message || 'Failed to add client role')
}

/**
 * Remove a role from a client
 */
export async function removeClientRole(clientId: string, roleUuid: string): Promise<void> {
  const response = await deleteRequest<ApiResponse<void>>(
    `${API_ENDPOINTS.CLIENT_ROLES(clientId)}/${roleUuid}`,
  )
  if (!response.success) throw new Error(response.message || 'Failed to remove client role')
}

// Export as client object
export const clientService = {
  fetchClients,
  fetchClientById,
  createClient,
  updateClient,
  deleteClient,
  updateClientStatus,
  rotateClientSecret,
  fetchClientConfig,
  fetchClientUris,
  createClientUri,
  updateClientUri,
  deleteClientUri,
  fetchClientApis,
  addClientApis,
  removeClientApi,
  addClientApiPermissions,
  removeClientApiPermission,
}
