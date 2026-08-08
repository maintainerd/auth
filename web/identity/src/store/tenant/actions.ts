import { createAsyncThunk } from '@reduxjs/toolkit'
import {
  fetchTenant,
  fetchDefaultTenant,
  fetchTenantBySlug,
  fetchTenantBootstrap
} from '@/services'
import { setTenantBootstrapContext } from '@/utils/clientContext'

type InitializeTenantArgs = {
  clientId?: string
}

export const fetchTenantAsync = createAsyncThunk(
  'tenant/fetch',
  async (identifier?: string) => {
    return await fetchTenant(identifier)
  }
)

export const fetchDefaultTenantAsync = createAsyncThunk(
  'tenant/fetchDefault',
  async () => {
    return await fetchDefaultTenant()
  }
)

export const fetchTenantBySlugAsync = createAsyncThunk(
  'tenant/fetchBySlug',
  async (slug: string) => {
    return await fetchTenantBySlug(slug)
  }
)

/**
 * Initialize the tenant from the current host via the domain-bootstrap endpoint.
 * The full window host is sent as `?domain=` and the backend resolves the tenant
 * — the frontend no longer parses the host. The resolved tenant slug and default
 * identity client are published to the client-context registry so the public API
 * layer can attach the right context to subsequent calls.
 */
export const initializeTenantAsync = createAsyncThunk(
  'tenant/initialize',
  async (args?: InitializeTenantArgs) => {
    const host = typeof window !== 'undefined' ? window.location.host : ''
    const bootstrap = await fetchTenantBootstrap(host, args?.clientId)
    setTenantBootstrapContext({
      tenantSlug: bootstrap.tenant.name,
      defaultClientId: bootstrap.client?.client_id,
    })
    return bootstrap
  }
)
