/**
 * Tenant Extra Reducers
 * Async thunk reducers for tenant slice
 */

import type { ActionReducerMapBuilder } from '@reduxjs/toolkit'
import {
  fetchTenantAsync,
  fetchDefaultTenantAsync,
  fetchTenantByIdentifierAsync,
  initializeTenantAsync,
  bootstrapTenantAsync
} from './actions'
import type { TenantEntity } from '@/services/api/tenants/types'
import type { TenantState } from './types'

export const tenantExtraReducers = (builder: ActionReducerMapBuilder<TenantState>) => {
  builder
    // Fetch tenant
    .addCase(fetchTenantAsync.pending, (state) => {
      state.isLoading = true
      state.error = null
    })
    .addCase(fetchTenantAsync.fulfilled, (state, action) => {
      state.isLoading = false
      state.currentTenant = action.payload
      state.error = null
    })
    .addCase(fetchTenantAsync.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message || 'Failed to fetch tenant'
    })
    // Fetch default tenant
    .addCase(fetchDefaultTenantAsync.pending, (state) => {
      state.isLoading = true
      state.error = null
    })
    .addCase(fetchDefaultTenantAsync.fulfilled, (state, action) => {
      state.isLoading = false
      state.currentTenant = action.payload
      state.error = null
    })
    .addCase(fetchDefaultTenantAsync.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message || 'Failed to fetch default tenant'
    })
    // Fetch tenant by identifier
    .addCase(fetchTenantByIdentifierAsync.pending, (state) => {
      state.isLoading = true
      state.error = null
    })
    .addCase(fetchTenantByIdentifierAsync.fulfilled, (state, action) => {
      state.isLoading = false
      state.currentTenant = action.payload
      state.error = null
    })
    .addCase(fetchTenantByIdentifierAsync.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message || 'Failed to fetch tenant by identifier'
    })
    // Initialize tenant
    .addCase(initializeTenantAsync.pending, (state) => {
      state.isLoading = true
      state.error = null
    })
    .addCase(initializeTenantAsync.fulfilled, (state, action) => {
      state.isLoading = false
      state.currentTenant = action.payload
      state.error = null
    })
    .addCase(initializeTenantAsync.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message || 'Failed to initialize tenant'
    })
    // Bootstrap tenant from host (new backend tenant-bootstrap endpoint).
    .addCase(bootstrapTenantAsync.pending, (state) => {
      state.isLoading = true
      state.error = null
    })
    .addCase(bootstrapTenantAsync.fulfilled, (state, action) => {
      const data = action.payload
      const t = data.tenant
      const normalized: TenantEntity = {
        tenant_id: t.tenant_uuid,
        name: t.name,
        display_name: t.display_name,
        description: t.description,
        status: t.status,
        is_system: t.is_system,
        created_at: '',
        updated_at: '',
        branding: data.branding,
        password_config: data.password_config,
        registration_config: data.registration_config,
      }
      state.isLoading = false
      state.currentTenant = normalized
      state.surface = data.surface
      state.identityUrl = data.identity_url
      state.consoleUrl = data.console_url
      state.consoleClient = data.client ?? null
      state.error = null
    })
    .addCase(bootstrapTenantAsync.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message || 'Failed to bootstrap tenant'
    })
}
