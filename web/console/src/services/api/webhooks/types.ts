/**
 * Webhook API Types
 * Types for the webhook-endpoints feature (event delivery registrations).
 */

// The backend allows `active`/`inactive` via the API; `quarantined` is set by the
// system after repeated delivery failures and is read-only from the UI's view.
export type WebhookStatus = 'active' | 'inactive' | 'quarantined'

// A registered webhook endpoint. Note: the API returns the UUID in the
// `webhook_endpoint_id` field, which is the id used in all path params.
export type Webhook = {
  webhook_endpoint_id: string
  url: string
  /** Plaintext signing secret — returned ONLY once, on create or secret rotation. */
  signing_secret?: string
  subscribe_all: boolean
  max_retries: number
  timeout_seconds: number
  status: WebhookStatus
  description: string
  last_triggered_at?: string | null
  created_at: string
  updated_at: string
}

export interface WebhookQueryParams {
  status?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface WebhookListResponse {
  rows: Webhook[]
  total: number
  page: number
  limit: number
  total_pages: number
}

export interface CreateWebhookRequest {
  url: string
  subscribe_all: boolean
  description?: string
  max_retries?: number
  timeout_seconds?: number
  status?: Exclude<WebhookStatus, 'quarantined'>
}

export interface UpdateWebhookRequest {
  url: string
  subscribe_all: boolean
  /** When true, the backend generates and returns a fresh signing secret. */
  rotate_secret?: boolean
  description?: string
  max_retries?: number
  timeout_seconds?: number
  status?: Exclude<WebhookStatus, 'quarantined'>
}

export interface UpdateWebhookStatusRequest {
  status: Exclude<WebhookStatus, 'quarantined'>
}

export interface WebhookSubscription {
  event_type_uuid: string
  event_type_key: string
}

export interface SetWebhookSubscriptionRequest {
  event_type_uuid: string
}
