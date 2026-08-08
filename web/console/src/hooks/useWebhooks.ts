/**
 * Webhooks Hook
 * Custom hooks for fetching/mutating webhook endpoints using TanStack Query.
 */

import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import {
  fetchWebhooks,
  fetchWebhookById,
  createWebhook,
  updateWebhook,
  deleteWebhook,
  updateWebhookStatus,
  fetchWebhookSubscriptions,
  addWebhookSubscription,
  removeWebhookSubscription,
} from '@/services/api/webhooks'
import type {
  WebhookQueryParams,
  CreateWebhookRequest,
  UpdateWebhookRequest,
  UpdateWebhookStatusRequest,
  SetWebhookSubscriptionRequest,
} from '@/services/api/webhooks/types'

/**
 * Query key factory for webhooks
 */
export const webhookKeys = {
  all: ['webhooks'] as const,
  lists: () => [...webhookKeys.all, 'list'] as const,
  list: (params?: WebhookQueryParams) => [...webhookKeys.lists(), params] as const,
  details: () => [...webhookKeys.all, 'detail'] as const,
  detail: (id: string) => [...webhookKeys.details(), id] as const,
  subscriptions: (id: string) => [...webhookKeys.all, 'subscriptions', id] as const,
}

/**
 * Hook to fetch webhooks with optional filters and pagination
 */
export function useWebhooks(params?: WebhookQueryParams) {
  return useQuery({
    queryKey: webhookKeys.list(params),
    queryFn: () => fetchWebhooks(params),
    placeholderData: keepPreviousData,
  })
}

/**
 * Hook to fetch webhooks for the standard listing page. The shared listing
 * passes free-text search + filter values; the API only filters by `status`,
 * so other keys are forwarded as-is and ignored server-side.
 */
export function useWebhooksList(params: Record<string, unknown>) {
  const queryParams: WebhookQueryParams = {
    ...(params as WebhookQueryParams),
  }
  return useWebhooks(queryParams)
}

/**
 * Hook to fetch a single webhook by id
 */
export function useWebhook(webhookId: string) {
  return useQuery({
    queryKey: webhookKeys.detail(webhookId),
    queryFn: () => fetchWebhookById(webhookId),
    enabled: !!webhookId,
  })
}

/**
 * Hook to create a new webhook. The returned signing secret is shown only once.
 */
export function useCreateWebhook() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateWebhookRequest) => createWebhook(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.lists() })
    },
  })
}

/**
 * Hook to update an existing webhook
 */
export function useUpdateWebhook() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ webhookId, data }: { webhookId: string; data: UpdateWebhookRequest }) =>
      updateWebhook(webhookId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.detail(variables.webhookId) })
      queryClient.invalidateQueries({ queryKey: webhookKeys.lists() })
    },
  })
}

/**
 * Hook to delete a webhook
 */
export function useDeleteWebhook() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (webhookId: string) => deleteWebhook(webhookId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.lists() })
    },
  })
}

/**
 * Hook to update a webhook's status (enable/disable)
 */
export function useUpdateWebhookStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ webhookId, data }: { webhookId: string; data: UpdateWebhookStatusRequest }) =>
      updateWebhookStatus(webhookId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.detail(variables.webhookId) })
      queryClient.invalidateQueries({ queryKey: webhookKeys.lists() })
    },
  })
}

/**
 * Hook to fetch subscriptions for a webhook endpoint.
 */
export function useWebhookSubscriptions(webhookId: string) {
  return useQuery({
    queryKey: webhookKeys.subscriptions(webhookId),
    queryFn: () => fetchWebhookSubscriptions(webhookId),
    enabled: !!webhookId,
  })
}

/**
 * Hook to add a webhook event subscription.
 */
export function useAddWebhookSubscription() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ webhookId, data }: { webhookId: string; data: SetWebhookSubscriptionRequest }) =>
      addWebhookSubscription(webhookId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.subscriptions(variables.webhookId) })
    },
  })
}

/**
 * Hook to remove a webhook event subscription.
 */
export function useRemoveWebhookSubscription() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ webhookId, data }: { webhookId: string; data: SetWebhookSubscriptionRequest }) =>
      removeWebhookSubscription(webhookId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.subscriptions(variables.webhookId) })
    },
  })
}
