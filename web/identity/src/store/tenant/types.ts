/**
 * Tenant Store Types
 * Redux-specific types for tenant state management
 */

import type { TenantBootstrap, TenantEntity } from '@/services/api/tenants/types'

export interface TenantState {
  currentTenant: TenantEntity | null
  /**
   * The full domain-bootstrap payload (tenant + branding + client + surface +
   * canonical URLs). `currentTenant` is derived from this for the many
   * branding/config consumers; the bootstrap keeps the extra fields (surface,
   * identity_url, console_url, default client) available.
   */
  bootstrap: TenantBootstrap | null
  isLoading: boolean
  error: string | null
}
