/**
 * Tenant Hook
 * Custom hook that uses Redux for tenant state management and initialization
 */

import { useCallback } from 'react'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { useToast } from '@/hooks/useToast'
import {
  fetchTenantAsync,
  fetchDefaultTenantAsync,
  fetchTenantBySlugAsync,
  initializeTenantAsync,
  clearTenant as clearTenantAction,
  clearError
} from '@/store/tenant/reducers'

export function useTenant() {
  const dispatch = useAppDispatch()
  const { showError } = useToast()
  const { currentTenant, bootstrap, isLoading, error } = useAppSelector((state) => state.tenant)

  // Resolve the tenant from the current host via the backend domain bootstrap.
  const initializeFromLocation = useCallback(async () => {
    try {
      const clientId = typeof window !== 'undefined'
        ? new URLSearchParams(window.location.search).get('client_id')?.trim() || undefined
        : undefined
      const result = await dispatch(initializeTenantAsync({ clientId })).unwrap()
      return result
    } catch (error) {
      showError('Failed to initialize tenant')
      throw error
    }
  }, [dispatch, showError])

  const getCurrentTenant = useCallback(() => {
    return currentTenant
  }, [currentTenant])

  const clearTenant = useCallback(() => {
    dispatch(clearTenantAction())
  }, [dispatch])

  const fetchTenant = useCallback(async (identifier?: string) => {
    const result = await dispatch(fetchTenantAsync(identifier)).unwrap()
    return result
  }, [dispatch])

  const fetchDefault = useCallback(async () => {
    const result = await dispatch(fetchDefaultTenantAsync()).unwrap()
    return result
  }, [dispatch])

  const fetchBySlug = useCallback(async (slug: string) => {
    const result = await dispatch(fetchTenantBySlugAsync(slug)).unwrap()
    return result
  }, [dispatch])

  const initializeTenant = useCallback(async (clientId?: string) => {
    const result = await dispatch(initializeTenantAsync({ clientId })).unwrap()
    return result
  }, [dispatch])

  const clearTenantError = useCallback(() => {
    dispatch(clearError())
  }, [dispatch])

  return {
    // State
    currentTenant,
    // Full domain-bootstrap payload: surface, identity_url, console_url, and the
    // tenant's default client for this surface (in addition to tenant + branding).
    bootstrap,
    surface: bootstrap?.surface,
    identityUrl: bootstrap?.identity_url,
    consoleUrl: bootstrap?.console_url,
    defaultClient: bootstrap?.client ?? null,
    isLoading,
    error,
    // Actions
    getCurrentTenant,
    clearTenant,
    fetchTenant,
    fetchDefault,
    fetchBySlug,
    initializeTenant,
    initializeFromLocation,
    clearTenantError
  }
}
