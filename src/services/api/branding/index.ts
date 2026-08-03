/**
 * Branding API
 * Admin endpoints for managing per-tenant branding themes (list, create,
 * update, activate, restore, delete). The active theme is the loaded style.
 */

import { get, post, put, patch, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import { unwrap, assertSuccess } from '../_lib/unwrap'
import type { Branding, BrandingRequest } from './types'

// List all branding themes for the current tenant (system themes first).
export async function fetchBrandings(): Promise<Branding[]> {
  const r = await get<ApiResponse<Branding[]>>(API_ENDPOINTS.BRANDING)
  return unwrap(r, 'fetch brandings')
}

// Create a new custom branding theme. Never active on creation — activate it
// explicitly afterwards.
export async function createBranding(data: BrandingRequest): Promise<Branding> {
  const r = await post<ApiResponse<Branding>>(API_ENDPOINTS.BRANDING, data)
  return unwrap(r, 'create branding')
}

// Update an existing branding theme (system themes may be edited, not deleted).
export async function updateBranding(brandingId: string, data: BrandingRequest): Promise<Branding> {
  const r = await put<ApiResponse<Branding>>(`${API_ENDPOINTS.BRANDING}/${brandingId}`, data)
  return unwrap(r, 'update branding')
}

// Set a branding theme as the active (loaded) style; deactivates the rest.
export async function activateBranding(brandingId: string): Promise<Branding> {
  const r = await patch<ApiResponse<Branding>>(`${API_ENDPOINTS.BRANDING}/${brandingId}/activate`)
  return unwrap(r, 'activate branding')
}

// Restore a system branding theme to the seeded defaults.
export async function restoreBranding(brandingId: string): Promise<Branding> {
  const r = await patch<ApiResponse<Branding>>(`${API_ENDPOINTS.BRANDING}/${brandingId}/restore`)
  return unwrap(r, 'restore branding')
}

// Delete a custom branding theme. System themes cannot be deleted.
export async function deleteBranding(brandingId: string): Promise<void> {
  const r = await deleteRequest<ApiResponse<void>>(`${API_ENDPOINTS.BRANDING}/${brandingId}`)
  assertSuccess(r, 'delete branding')
}
