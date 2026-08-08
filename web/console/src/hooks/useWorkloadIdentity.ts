/**
 * Workload Identity Federation Hooks
 * Custom hooks for managing workload identity federations using TanStack Query
 */

import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import {
  fetchWorkloadIdentities,
  fetchWorkloadIdentityById,
  createWorkloadIdentity,
  updateWorkloadIdentity,
  deleteWorkloadIdentity,
} from '@/services/api/workload-identity'
import type {
  WorkloadIdentityQueryParams,
  CreateWorkloadIdentityRequest,
  UpdateWorkloadIdentityRequest,
} from '@/services/api/workload-identity/types'

/**
 * Query key factory for workload identity federations
 */
export const workloadIdentityKeys = {
  all: ['workload-identity'] as const,
  lists: () => [...workloadIdentityKeys.all, 'list'] as const,
  list: (params?: WorkloadIdentityQueryParams) =>
    [...workloadIdentityKeys.lists(), params] as const,
  details: () => [...workloadIdentityKeys.all, 'detail'] as const,
  detail: (federationId: string) => [...workloadIdentityKeys.details(), federationId] as const,
}

/**
 * Hook to fetch workload identity federations with optional filters and pagination
 */
export function useWorkloadIdentities(params?: WorkloadIdentityQueryParams) {
  return useQuery({
    queryKey: workloadIdentityKeys.list(params),
    queryFn: () => fetchWorkloadIdentities(params),
    placeholderData: keepPreviousData,
  })
}

/**
 * Hook to fetch workload identity federations for the standard listing page
 */
export function useWorkloadIdentitiesList(params: Record<string, unknown>) {
  return useWorkloadIdentities(params as WorkloadIdentityQueryParams)
}

/**
 * Hook to fetch a single workload identity federation by UUID
 */
export function useWorkloadIdentity(federationId: string) {
  return useQuery({
    queryKey: workloadIdentityKeys.detail(federationId),
    queryFn: () => fetchWorkloadIdentityById(federationId),
    enabled: !!federationId,
  })
}

/**
 * Hook to create a new workload identity federation
 */
export function useCreateWorkloadIdentity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateWorkloadIdentityRequest) => createWorkloadIdentity(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workloadIdentityKeys.lists() })
    },
  })
}

/**
 * Hook to update an existing workload identity federation
 */
export function useUpdateWorkloadIdentity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ federationId, data }: { federationId: string; data: UpdateWorkloadIdentityRequest }) =>
      updateWorkloadIdentity(federationId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: workloadIdentityKeys.detail(variables.federationId) })
      queryClient.invalidateQueries({ queryKey: workloadIdentityKeys.lists() })
    },
  })
}

/**
 * Hook to delete a workload identity federation
 */
export function useDeleteWorkloadIdentity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (federationId: string) => deleteWorkloadIdentity(federationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workloadIdentityKeys.lists() })
    },
  })
}
