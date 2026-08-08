/**
 * Tenant Store Types
 * Redux-specific types for tenant state management
 */

import type { TenantEntity, TenantBootstrapClient } from '@/services/api/tenants/types'

export type TenantSurface = 'identity' | 'console'

export interface TenantState {
  /** Tenant resolved from the host by the backend bootstrap endpoint. */
  currentTenant: TenantEntity | null
  /** Which surface this host serves (`console` for the admin console host). */
  surface: TenantSurface | null
  /** Per-tenant hosted-identity origin — the real source for OAuth authorize. */
  identityUrl: string | null
  /** Per-tenant console origin. */
  consoleUrl: string | null
  /** Tenant's console OAuth client, used to start the login flow. */
  consoleClient: TenantBootstrapClient | null
  isLoading: boolean
  error: string | null
}
