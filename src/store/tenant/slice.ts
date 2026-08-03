/**
 * Tenant Slice
 * Redux slice for tenant state management
 */

import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { tenantExtraReducers } from './extra-reducers'
import type { TenantState } from './types'
import type { BrandingPublic } from '@/services/api/tenants/types'

const initialState: TenantState = {
  currentTenant: null,
  surface: null,
  identityUrl: null,
  consoleUrl: null,
  consoleClient: null,
  isLoading: false,
  error: null
}

const tenantSlice = createSlice({
  name: 'tenant',
  initialState,
  reducers: {
    clearError: (state: TenantState) => {
      state.error = null
    },
    clearTenant: (state: TenantState) => {
      state.currentTenant = null
      state.surface = null
      state.identityUrl = null
      state.consoleUrl = null
      state.consoleClient = null
      state.error = null
    },
    setTenantBranding: (state: TenantState, action: PayloadAction<BrandingPublic | null>) => {
      if (!state.currentTenant) return
      if (action.payload) {
        state.currentTenant.branding = action.payload
      } else {
        delete state.currentTenant.branding
      }
    }
  },
  extraReducers: tenantExtraReducers
})

export const { clearError, clearTenant, setTenantBranding } = tenantSlice.actions
export default tenantSlice.reducer
