/**
 * Webhook API Service
 * Service for managing webhook-endpoint API calls.
 */

import { get, post, put, patch, deleteRequest } from '../client'
import { API_ENDPOINTS } from '../config'
import type { ApiResponse } from '../types'
import type {
  WebhookQueryParams,
  WebhookListResponse,
  Webhook,
  CreateWebhookRequest,
  UpdateWebhookRequest,
  UpdateWebhookStatusRequest,
  WebhookSubscription,
  SetWebhookSubscriptionRequest,
} from './types'

/**
 * Fetch webhook endpoints with optional filters and pagination.
 */
export async function fetchWebhooks(params?: WebhookQueryParams): Promise<WebhookListResponse> {
  const queryParams = new URLSearchParams()

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value))
      }
    })
  }

  const endpoint = `${API_ENDPOINTS.WEBHOOK_ENDPOINT}${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  const response = await get<ApiResponse<WebhookListResponse>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch webhooks')
}

/**
 * Fetch a single webhook endpoint by id.
 */
export async function fetchWebhookById(webhookId: string): Promise<Webhook> {
  const endpoint = `${API_ENDPOINTS.WEBHOOK_ENDPOINT}/${webhookId}`
  const response = await get<ApiResponse<Webhook>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch webhook')
}

/**
 * Create a new webhook endpoint. The signing secret is returned exactly once.
 */
export async function createWebhook(data: CreateWebhookRequest): Promise<Webhook> {
  const endpoint = API_ENDPOINTS.WEBHOOK_ENDPOINT
  const response = await post<ApiResponse<Webhook>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to create webhook')
}

/**
 * Update an existing webhook endpoint. When `rotate_secret` is set, the new
 * signing secret is returned exactly once.
 */
export async function updateWebhook(webhookId: string, data: UpdateWebhookRequest): Promise<Webhook> {
  const endpoint = `${API_ENDPOINTS.WEBHOOK_ENDPOINT}/${webhookId}`
  const response = await put<ApiResponse<Webhook>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update webhook')
}

/**
 * Delete a webhook endpoint.
 */
export async function deleteWebhook(webhookId: string): Promise<void> {
  const endpoint = `${API_ENDPOINTS.WEBHOOK_ENDPOINT}/${webhookId}`
  const response = await deleteRequest<ApiResponse<void>>(endpoint)

  if (!response.success) {
    throw new Error(response.message || 'Failed to delete webhook')
  }
}

/**
 * Update a webhook endpoint's status (enable/disable). Uses PATCH.
 */
export async function updateWebhookStatus(
  webhookId: string,
  data: UpdateWebhookStatusRequest,
): Promise<Webhook> {
  const endpoint = `${API_ENDPOINTS.WEBHOOK_ENDPOINT}/${webhookId}/status`
  const response = await patch<ApiResponse<Webhook>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to update webhook status')
}

export const webhookService = {
  fetchWebhooks,
  fetchWebhookById,
  createWebhook,
  updateWebhook,
  deleteWebhook,
  updateWebhookStatus,
}

/**
 * Fetch subscriptions (event type UUIDs) for a webhook endpoint.
 */
export async function fetchWebhookSubscriptions(webhookId: string): Promise<WebhookSubscription[]> {
  const endpoint = `${API_ENDPOINTS.WEBHOOK_ENDPOINT}/${webhookId}/subscriptions`
  const response = await get<ApiResponse<WebhookSubscription[]>>(endpoint)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to fetch webhook subscriptions')
}

/**
 * Add an event type subscription to a webhook endpoint.
 */
export async function addWebhookSubscription(
  webhookId: string,
  data: SetWebhookSubscriptionRequest,
): Promise<WebhookSubscription> {
  const endpoint = `${API_ENDPOINTS.WEBHOOK_ENDPOINT}/${webhookId}/subscriptions`
  const response = await post<ApiResponse<WebhookSubscription>>(endpoint, data)

  if (response.success && response.data) {
    return response.data
  }

  throw new Error(response.message || 'Failed to add subscription')
}

/**
 * Remove an event type subscription from a webhook endpoint.
 */
export async function removeWebhookSubscription(
  webhookId: string,
  data: SetWebhookSubscriptionRequest,
): Promise<void> {
  const endpoint = `${API_ENDPOINTS.WEBHOOK_ENDPOINT}/${webhookId}/subscriptions`
  const response = await deleteRequest<ApiResponse<void>>(endpoint, { data })

  if (!response.success) {
    throw new Error(response.message || 'Failed to remove subscription')
  }
}

// Delivery lifecycle states emitted by the backend: `pending` (queued/awaiting
// retry), `success`, or `dead_letter` (retries exhausted or endpoint quarantined).
export type DeliveryFinalStatus = 'pending' | 'success' | 'dead_letter'

export interface DeliveryHistoryItem {
  delivery_history_id: string
  /** The outbox event this delivery carried — the id POST /webhook-replay replays. */
  event_id: string
  event_type: string
  attempt_count: number
  final_status: DeliveryFinalStatus
  response_status?: number | null
  response_summary?: string
  is_replay: boolean
  created_at: string
}

export async function fetchDeliveryHistory(webhookId: string): Promise<DeliveryHistoryItem[]> {
  const response = await get<ApiResponse<DeliveryHistoryItem[]>>(
    `${API_ENDPOINTS.WEBHOOK_ENDPOINT}/${webhookId}/deliveries`
  )
  return response.data ?? []
}

// The backend replays an EVENT, not a delivery row: its replayRequestDTO reads
// `event_id` and nothing else. This used to post `delivery_history_id`, which
// left event_id empty and failed uuid.Parse("") — so every Replay click 400'd.
export async function replayDelivery(eventId: string): Promise<{ message: string }> {
  const response = await post<ApiResponse<{ message: string }>>(API_ENDPOINTS.WEBHOOK_REPLAY, {
    event_id: eventId,
  })
  return response.data ?? { message: response.message ?? "Replay initiated" }
}
