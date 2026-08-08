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
  fetchTenantByIdentifierAsync,
  initializeTenantAsync,
  bootstrapTenantAsync,
  clearTenant as clearTenantAction,
  clearError
} from '@/store/tenant/reducers'

export function useTenant() {
  const dispatch = useAppDispatch()
  const { showError } = useToast()
  const {
    currentTenant,
    surface,
    identityUrl,
    consoleUrl,
    consoleClient,
    isLoading,
    error,
  } = useAppSelector((state) => state.tenant)

  // Resolve the tenant for the current host via the backend bootstrap endpoint.
  // The backend resolves the tenant from the FULL host (including any subdomain),
  // so we pass `window.location.host` verbatim and never parse a slug here.
  const initializeFromHost = useCallback(async () => {
    try {
      const result = await dispatch(bootstrapTenantAsync(window.location.host)).unwrap()
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

  const fetchByIdentifier = useCallback(async (identifier: string) => {
    const result = await dispatch(fetchTenantByIdentifierAsync(identifier)).unwrap()
    return result
  }, [dispatch])

  const initializeTenant = useCallback(async (identifier?: string) => {
    const result = await dispatch(initializeTenantAsync(identifier)).unwrap()
    return result
  }, [dispatch])

  const clearTenantError = useCallback(() => {
    dispatch(clearError())
  }, [dispatch])

  return {
    // State
    currentTenant,
    surface,
    identityUrl,
    consoleUrl,
    consoleClient,
    isLoading,
    error,
    // Actions
    getCurrentTenant,
    clearTenant,
    fetchTenant,
    fetchDefault,
    fetchByIdentifier,
    initializeTenant,
    initializeFromHost,
    clearTenantError
  }
}
