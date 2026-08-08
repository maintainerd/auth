/**
 * TanStack Query Client Configuration
 * Centralized configuration for React Query
 */

import { QueryCache, QueryClient, MutationCache } from '@tanstack/react-query'
import { toast } from 'react-toastify'

// Statuses that represent a definitive client-side outcome — retrying them just
// wastes a round-trip and delays the error the user needs to see.
const NON_RETRYABLE_STATUSES = new Set([400, 401, 403, 404, 409, 422])

function statusOf(error: unknown): number | undefined {
  return (error as { status?: number } | null | undefined)?.status
}

function messageOf(error: unknown): string {
  if (error instanceof Error && error.message) return error.message
  if (typeof error === 'string' && error) return error
  if (error && typeof error === 'object' && 'message' in error) return String((error as { message: unknown }).message)
  return 'Something went wrong. Please try again.'
}

// Module-level error surface for queries/mutations that don't handle their own
// errors. Deduped by message so a burst of identical failures shows one toast.
function showError(error: unknown): void {
  const message = messageOf(error)
  toast.error(message, { toastId: message })
}

// Retry policy: never retry a definitive client error, otherwise allow a single
// retry for transient failures (network / 5xx).
function shouldRetry(failureCount: number, error: unknown): boolean {
  const status = statusOf(error)
  if (status !== undefined && NON_RETRYABLE_STATUSES.has(status)) return false
  return failureCount < 1
}

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error) => showError(error),
  }),
  mutationCache: new MutationCache({
    onError: (error) => showError(error),
  }),
  defaultOptions: {
    queries: {
      retry: shouldRetry,
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
    mutations: {
      retry: shouldRetry,
    },
  },
})
