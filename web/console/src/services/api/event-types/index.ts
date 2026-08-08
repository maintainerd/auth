/**
 * Event Types API
 * Read-only catalog of event types available for webhook subscription.
 */

import { get, put } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import { unwrap } from '../_lib/unwrap'
import type { EventType, TenantEventTypeConfig, SetTenantEventTypeRequest } from './types'

// List all active event types for the current tenant.
export async function fetchEventTypes(): Promise<EventType[]> {
  const r = await get<ApiResponse<EventType[]>>(API_ENDPOINTS.EVENT_TYPE)
  return unwrap(r, 'fetch event types')
}

// List the tenant's event-type overrides (only events with an explicit on/off).
export async function fetchTenantEventTypes(): Promise<TenantEventTypeConfig[]> {
  const r = await get<ApiResponse<TenantEventTypeConfig[]>>(API_ENDPOINTS.TENANT_EVENT_TYPE)
  return unwrap(r, 'fetch tenant event types')
}

// Enable or disable an event type for the current tenant (the master switch).
export async function setTenantEventType(
  data: SetTenantEventTypeRequest,
): Promise<TenantEventTypeConfig> {
  const r = await put<ApiResponse<TenantEventTypeConfig>>(API_ENDPOINTS.TENANT_EVENT_TYPE, data)
  return unwrap(r, 'update tenant event type')
}
