/**
 * Tenant Extra Reducers
 * Async thunk reducers for tenant slice
 */

import type { ActionReducerMapBuilder } from '@reduxjs/toolkit'
import { bootstrapToTenantEntity } from '@/services'
import {
  fetchTenantAsync,
  fetchDefaultTenantAsync,
  fetchTenantBySlugAsync,
  initializeTenantAsync
} from './actions'
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
    // Fetch tenant by slug
    .addCase(fetchTenantBySlugAsync.pending, (state) => {
      state.isLoading = true
      state.error = null
    })
    .addCase(fetchTenantBySlugAsync.fulfilled, (state, action) => {
      state.isLoading = false
      state.currentTenant = action.payload
      state.error = null
    })
    .addCase(fetchTenantBySlugAsync.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message || 'Failed to fetch tenant by slug'
    })
    // Initialize tenant (domain bootstrap)
    .addCase(initializeTenantAsync.pending, (state) => {
      state.isLoading = true
      state.error = null
    })
    .addCase(initializeTenantAsync.fulfilled, (state, action) => {
      state.isLoading = false
      state.bootstrap = action.payload
      state.currentTenant = bootstrapToTenantEntity(action.payload)
      state.error = null
    })
    .addCase(initializeTenantAsync.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message || 'Failed to initialize tenant'
    })
}
