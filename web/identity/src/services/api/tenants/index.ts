/**
 * Tenant Service
 * Handles the public tenant-lookup API calls used by the identity app.
 *
 * NOTE: administrative tenant CRUD (list/create/update/delete/status) lives on
 * the INTERNAL API (:8080) and is exposed through the admin console, not this
 * public app — so only the read-only tenant lookups below are defined here.
 */

import { get } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type { TenantBootstrap, TenantEntity, TenantResponse } from './types'

/**
 * Resolve the current host to its tenant via the domain-bootstrap endpoint.
 *
 * The FULL host is sent as `?domain=` and the backend resolves the tenant,
 * surface, canonical URLs, public branding, and the tenant's default client for
 * that surface. The frontend does NOT parse the host — this is the single source
 * of truth for tenant-context resolution.
 *
 * @param host - the full window host (e.g. `acme.auth.maintainerd.local`).
 * @param clientId - optional OAuth client identifier used only for branding selection.
 */
export async function fetchTenantBootstrap(host: string, clientId?: string): Promise<TenantBootstrap> {
  const params = new URLSearchParams({ domain: host })
  const normalizedClientId = clientId?.trim()
  if (normalizedClientId) {
    params.set('client_id', normalizedClientId)
  }

  const response = await get<ApiResponse<TenantBootstrap>>(`${API_ENDPOINTS.TENANT}?${params.toString()}`)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error('Failed to bootstrap tenant from domain')
}

/**
 * Map the domain-bootstrap payload onto the TenantEntity shape the app consumes
 * (tenant identity, slug, password/registration policy, branding). The
 * password_config / registration_config are carried through in the SAME shape
 * the public-tenant endpoint produced, so existing consumers enforce the tenant's
 * actual policy client-side (the backend stays the enforcing authority).
 */
export function bootstrapToTenantEntity(bootstrap: TenantBootstrap): TenantEntity {
  return {
    tenant_id: bootstrap.tenant.tenant_id,
    name: bootstrap.tenant.name,
    display_name: bootstrap.tenant.display_name,
    description: bootstrap.tenant.description,
    status: bootstrap.tenant.status,
    is_default: bootstrap.tenant.is_system,
    is_system: bootstrap.tenant.is_system,
    created_at: '',
    updated_at: '',
    password_config: bootstrap.password_config,
    registration_config: bootstrap.registration_config,
    branding: bootstrap.branding ?? undefined,
  }
}

/**
 * Fetch default tenant
 * @returns Promise<TenantEntity>
 */
export async function fetchDefaultTenant(): Promise<TenantEntity> {
  const response = await get<TenantResponse>(API_ENDPOINTS.TENANT)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error('Failed to fetch default tenant')
}

/**
 * Fetch a tenant by its slug (the subdomain-derived tenant name).
 * @param slug - Tenant slug (tenant `name`)
 * @returns Promise<TenantEntity>
 */
export async function fetchTenantBySlug(slug: string): Promise<TenantEntity> {
  const response = await get<TenantResponse>(`${API_ENDPOINTS.TENANT}/${encodeURIComponent(slug)}`)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(`Failed to fetch tenant with slug: ${slug}`)
}

/**
 * Fetch a tenant by slug, or the default (system) tenant when no slug is given.
 * @param slug - Optional tenant slug (absent for the system tenant)
 * @returns Promise<TenantEntity>
 */
export async function fetchTenant(slug?: string): Promise<TenantEntity> {
  if (slug) {
    return await fetchTenantBySlug(slug)
  } else {
    return await fetchDefaultTenant()
  }
}
