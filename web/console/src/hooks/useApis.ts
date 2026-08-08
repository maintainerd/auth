/**
 * APIs Hook
 * Custom hook for fetching APIs using TanStack Query
 */

import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { fetchApis, fetchApiById, createApi, updateApi, deleteApi, updateApiStatus } from '@/services/api/api'
import type {
  ApiQueryParams,
  CreateApiRequest,
  UpdateApiRequest,
  UpdateApiStatusRequest
} from '@/services/api/api/types'

/**
 * Query key factory for APIs
 */
export const apiKeys = {
  all: ['apis'] as const,
  lists: () => [...apiKeys.all, 'list'] as const,
  list: (params?: ApiQueryParams) => [...apiKeys.lists(), params] as const,
  details: () => [...apiKeys.all, 'detail'] as const,
  detail: (id: string) => [...apiKeys.details(), id] as const,
}

/**
 * Hook to fetch APIs with optional filters and pagination.
 *
 * `options.enabled` lets a caller hold the request until it is actually wanted
 * (a closed dialog, for instance). Without it, callers had to conditionally
 * mount the whole component to avoid fetching on page load.
 *
 * `keepPreviousData` keeps the current rows on screen while a new `name` /
 * `display_name` search resolves, so a server-side search does not blank the
 * list on every keystroke.
 */
export function useApis(params?: ApiQueryParams, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: apiKeys.list(params),
    queryFn: () => fetchApis(params),
    placeholderData: keepPreviousData,
    enabled: options?.enabled,
  })
}

/**
 * Hook to fetch a single API by ID
 */
export function useApi(apiId: string) {
  return useQuery({
    queryKey: apiKeys.detail(apiId),
    queryFn: () => fetchApiById(apiId),
    enabled: !!apiId,
  })
}

/**
 * Hook to fetch APIs for the standard listing page.
 * The shared listing filter uses human labels for system/regular, while the
 * API service expects an `is_system` boolean.
 */
export function useApisList(params: Record<string, unknown>) {
  const { is_system, ...rest } = params
  const queryParams: ApiQueryParams = {
    ...rest as ApiQueryParams,
  }

  if (typeof is_system === 'string') {
    if (is_system === 'system') queryParams.is_system = true
    else if (is_system === 'regular') queryParams.is_system = false
  }

  return useApis(queryParams)
}

/**
 * Hook to create a new API
 */
export function useCreateApi() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateApiRequest) => createApi(data),
    onSuccess: () => {
      // Invalidate APIs list to refetch
      queryClient.invalidateQueries({ queryKey: apiKeys.lists() })
    },
  })
}

/**
 * Hook to update an existing API
 */
export function useUpdateApi() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ apiId, data }: { apiId: string; data: UpdateApiRequest }) =>
      updateApi(apiId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific API and the APIs list
      queryClient.invalidateQueries({ queryKey: apiKeys.detail(variables.apiId) })
      queryClient.invalidateQueries({ queryKey: apiKeys.lists() })
    },
  })
}

/**
 * Hook to delete an API
 */
export function useDeleteApi() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (apiId: string) => deleteApi(apiId),
    onSuccess: () => {
      // Invalidate APIs list to refetch
      queryClient.invalidateQueries({ queryKey: apiKeys.lists() })
    },
  })
}

/**
 * Hook to update API status
 */
export function useUpdateApiStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ apiId, data }: { apiId: string; data: UpdateApiStatusRequest }) =>
      updateApiStatus(apiId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific API and the APIs list
      queryClient.invalidateQueries({ queryKey: apiKeys.detail(variables.apiId) })
      queryClient.invalidateQueries({ queryKey: apiKeys.lists() })
    },
  })
}
