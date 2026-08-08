/**
 * Policies Hook
 * Custom hook for fetching policies using TanStack Query
 */

import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { fetchPolicies, fetchPolicyById, createPolicy, updatePolicy, deletePolicy, updatePolicyStatus, fetchPolicyHistory } from '@/services/api/policies'
import type {
  PolicyQueryParams,
  CreatePolicyRequest,
  UpdatePolicyRequest,
  UpdatePolicyStatusRequest
} from '@/services/api/policies/types'

/**
 * Query key factory for policies
 */
export const policyKeys = {
  all: ['policies'] as const,
  lists: () => [...policyKeys.all, 'list'] as const,
  list: (params?: PolicyQueryParams) => [...policyKeys.lists(), params] as const,
  details: () => [...policyKeys.all, 'detail'] as const,
  detail: (id: string) => [...policyKeys.details(), id] as const,
  history: (id: string) => [...policyKeys.all, 'history', id] as const,
}

/**
 * Hook to fetch policies with optional filters and pagination
 */
export function usePolicies(params?: PolicyQueryParams, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: policyKeys.list(params),
    queryFn: () => fetchPolicies(params),
    placeholderData: keepPreviousData,
    enabled: options?.enabled,
  })
}

/**
 * Hook to fetch policies for the standard listing page.
 * The shared listing filter uses human labels for system/regular, while the
 * policy API expects an `is_system` boolean.
 */
export function usePoliciesList(params: Record<string, unknown>) {
  const { is_system, ...rest } = params
  const queryParams: PolicyQueryParams = {
    ...rest as PolicyQueryParams,
  }

  if (typeof is_system === 'string') {
    if (is_system === 'system') queryParams.is_system = true
    else if (is_system === 'regular') queryParams.is_system = false
  }

  return usePolicies(queryParams)
}

/**
 * Hook to fetch a single policy by ID
 */
export function usePolicy(policyId: string) {
  return useQuery({
    queryKey: policyKeys.detail(policyId),
    queryFn: () => fetchPolicyById(policyId),
    enabled: !!policyId,
  })
}

/**
 * Hook to create a new policy
 */
export function useCreatePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreatePolicyRequest) => createPolicy(data),
    onSuccess: () => {
      // Invalidate policies list to refetch
      queryClient.invalidateQueries({ queryKey: policyKeys.lists() })
    },
  })
}

/**
 * Hook to update an existing policy
 */
export function useUpdatePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ policyId, data }: { policyId: string; data: UpdatePolicyRequest }) =>
      updatePolicy(policyId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific policy and the policies list
      queryClient.invalidateQueries({ queryKey: policyKeys.detail(variables.policyId) })
      queryClient.invalidateQueries({ queryKey: policyKeys.lists() })
    },
  })
}

/**
 * Hook to delete a policy
 */
export function useDeletePolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (policyId: string) => deletePolicy(policyId),
    onSuccess: () => {
      // Invalidate policies list to refetch
      queryClient.invalidateQueries({ queryKey: policyKeys.lists() })
    },
  })
}

/**
 * Hook to fetch the version history of a policy
 */
export function usePolicyHistory(policyId: string) {
  return useQuery({
    queryKey: policyKeys.history(policyId),
    queryFn: () => fetchPolicyHistory(policyId),
    enabled: !!policyId,
  })
}

/**
 * Hook to update policy status
 */
export function useUpdatePolicyStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ policyId, data }: { policyId: string; data: UpdatePolicyStatusRequest }) =>
      updatePolicyStatus(policyId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific policy and the policies list
      queryClient.invalidateQueries({ queryKey: policyKeys.detail(variables.policyId) })
      queryClient.invalidateQueries({ queryKey: policyKeys.lists() })
    },
  })
}
