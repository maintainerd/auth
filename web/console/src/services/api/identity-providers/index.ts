/**
 * Identity Provider API
 * Handles identity provider-related API calls
 */

import axios from 'axios'
import { get, post, put, deleteRequest } from '../client'
import { API_CONFIG, API_ENDPOINTS } from '../config'
import {
  parseSamlServiceProviderMetadata,
  type SamlServiceProviderMetadata,
} from '@/utils/samlMetadata'
import type { ApiResponse } from '../types'
import type {
  IdentityProviderListResponse,
  IdentityProviderQueryParams,
  IdentityProviderResponse,
  IdentityProviderConfig,
  IdentityProviderDetailResponse,
  CreateIdentityProviderRequest,
  UpdateIdentityProviderRequest,
  UpdateIdentityProviderStatusRequest
} from './types'

function normalizeConfig(config: unknown): IdentityProviderConfig | null {
  if (config === null || config === undefined) return null
  if (typeof config === 'object' && !Array.isArray(config)) {
    return config as IdentityProviderConfig
  }
  return {}
}

function normalizeIdentityProviderDetail(provider: IdentityProviderDetailResponse): IdentityProviderDetailResponse {
  return {
    ...provider,
    allowed_audiences: provider.allowed_audiences ?? [],
    email_domains: provider.email_domains ?? [],
    config: normalizeConfig(provider.config),
  }
}

/**
 * Fetch identity providers with optional filters and pagination
 */
export async function fetchIdentityProviders(params?: IdentityProviderQueryParams): Promise<IdentityProviderListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.IDENTITY_PROVIDER}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<IdentityProviderListResponse>>(endpoint)

  if (response.success && response.data) {
    return {
      ...response.data,
      rows: response.data.rows ?? [],
    }
  }

  throw new Error(response.message || 'Failed to fetch identity providers')
}

/**
 * Fetch a single identity provider by ID
 */
export async function fetchIdentityProviderById(identityProviderId: string): Promise<IdentityProviderDetailResponse> {
  const endpoint = `${API_ENDPOINTS.IDENTITY_PROVIDER}/${identityProviderId}`
  const response = await get<ApiResponse<IdentityProviderDetailResponse>>(endpoint)

  if (response.success && response.data) {
    return normalizeIdentityProviderDetail(response.data)
  }

  throw new Error(response.message || 'Failed to fetch identity provider')
}

/**
 * Create a new identity provider
 */
export async function createIdentityProvider(data: CreateIdentityProviderRequest): Promise<IdentityProviderResponse> {
  const endpoint = API_ENDPOINTS.IDENTITY_PROVIDER
  const response = await post<ApiResponse<IdentityProviderResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to create identity provider')
}

/**
 * Update an existing identity provider
 */
export async function updateIdentityProvider(identityProviderId: string, data: UpdateIdentityProviderRequest): Promise<IdentityProviderResponse> {
  const endpoint = `${API_ENDPOINTS.IDENTITY_PROVIDER}/${identityProviderId}`
  const response = await put<ApiResponse<IdentityProviderResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update identity provider')
}

/**
 * Delete an identity provider
 */
export async function deleteIdentityProvider(identityProviderId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.IDENTITY_PROVIDER}/${identityProviderId}`
  const response = await deleteRequest<ApiResponse<void>>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete identity provider')
  }
}

/**
 * Update identity provider status
 */
export async function updateIdentityProviderStatus(identityProviderId: string, data: UpdateIdentityProviderStatusRequest): Promise<IdentityProviderResponse> {
  const endpoint = `${API_ENDPOINTS.IDENTITY_PROVIDER}/${identityProviderId}/status`
  const response = await put<ApiResponse<IdentityProviderResponse>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update identity provider status')
}

// Mirrors the backend TestConnectionResultDTO / TestCheckDTO
// (internal/idp/types.go): each check carries { step, ok, error?, url? }.
// `message` is only used for a client-side network/error fallback.
export interface TestConnectionCheck {
  step: string
  ok: boolean
  error?: string
  url?: string
}

export interface TestConnectionResult {
  success: boolean
  checks?: TestConnectionCheck[]
  message?: string
}

export async function testIdentityProviderConnection(data: Record<string, unknown>): Promise<TestConnectionResult> {
  const response = await post<ApiResponse<TestConnectionResult>>(API_ENDPOINTS.IDENTITY_PROVIDER_TEST, data)
  return (response.data ?? { success: false, message: response.message ?? "Unknown error" }) as TestConnectionResult
}

export interface SamlServiceProviderDetails extends SamlServiceProviderMetadata {
  /** The metadata document verbatim, for IdPs that import a file. */
  xml: string
}

/**
 * Fetch the SAML service-provider metadata Maintainerd publishes for a provider.
 *
 * This is the ONLY source for the entity ID / ACS URL an admin must give the
 * upstream IdP: those values are composed from the backend's public hostname,
 * which the console does not know. Deriving them here would risk advertising an
 * endpoint that never receives the assertion.
 *
 * The endpoint lives on the PUBLIC (data) plane and is unauthenticated — it is
 * fetched with plain axios rather than the control-plane client so that a 404
 * (e.g. the provider was renamed) cannot trip the session-recovery interceptor.
 */
export async function fetchSamlServiceProviderMetadata(identifier: string): Promise<SamlServiceProviderDetails> {
  const url = `${API_CONFIG.PUBLIC_BASE_URL}${API_ENDPOINTS.SAML_SP_METADATA(identifier)}`
  const response = await axios.get<string>(url, {
    // Axios sniffs XML-ish payloads into a Document under jsdom; force text so
    // the parser below always receives the raw bytes the IdP would also read.
    responseType: 'text',
    transformResponse: [(data: unknown) => data],
    headers: { Accept: 'application/samlmetadata+xml, application/xml, text/xml' },
    timeout: API_CONFIG.TIMEOUT,
  })

  const xml = typeof response.data === 'string' ? response.data : String(response.data ?? '')
  return { ...parseSamlServiceProviderMetadata(xml), xml }
}

// Export as identity provider object
export const identityProviderService = {
  fetchIdentityProviders,
  fetchIdentityProviderById,
  createIdentityProvider,
  updateIdentityProvider,
  deleteIdentityProvider,
  updateIdentityProviderStatus,
}
