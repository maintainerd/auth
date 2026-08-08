/**
 * Event Types API types
 *
 * The catalog of event types the system can emit. Read-only — used to discover
 * which events a webhook endpoint can subscribe to. The list endpoint returns a
 * plain array of the currently-active event types.
 */
export interface EventType {
  /** Catalog UUID — the id used to toggle tenant config / webhook subscriptions. */
  uuid: string
  /** Stable event key, e.g. `user_created`. */
  key: string
  /** Grouping category, e.g. `USER`, `IAM`, `SESSION`. */
  category: string
  description: string
  version: number
  is_active: boolean
}

/**
 * A per-tenant event-type override. The tenant config endpoint only returns
 * rows that exist (explicit overrides) — absence means the event is enabled
 * (default-on). This is the master switch: disabling stops the event from
 * propagating to ANY webhook or message broker for the tenant.
 */
export interface TenantEventTypeConfig {
  tenant_event_type_uuid: string
  tenant_uuid: string
  event_type_uuid: string
  event_type_key: string
  enabled: boolean
}

export interface SetTenantEventTypeRequest {
  event_type_uuid: string
  enabled: boolean
}
