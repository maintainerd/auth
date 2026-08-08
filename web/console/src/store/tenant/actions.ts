import { createAsyncThunk } from '@reduxjs/toolkit'
import {
  fetchTenant,
  fetchDefaultTenant,
  fetchTenantByIdentifier
} from '@/services'
import { fetchTenantBootstrap } from '@/services/api/tenants'

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

export const fetchTenantByIdentifierAsync = createAsyncThunk(
  'tenant/fetchByIdentifier',
  async (identifier: string) => {
    return await fetchTenantByIdentifier(identifier)
  }
)

export const initializeTenantAsync = createAsyncThunk(
  'tenant/initialize',
  async (identifier?: string) => {
    return await fetchTenant(identifier)
  }
)

// Resolve the tenant for the current host via the backend bootstrap endpoint.
// The backend resolves the tenant from the FULL host — the console passes the
// whole host and never parses a slug itself.
export const bootstrapTenantAsync = createAsyncThunk(
  'tenant/bootstrap',
  async (domain: string) => {
    return await fetchTenantBootstrap(domain)
  }
)
