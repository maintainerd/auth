/**
 * Registration Flows Hooks
 * TanStack Query hooks for the registration-flow resource.
 */

import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import {
  fetchRegistrationFlows,
  fetchRegistrationFlow,
  fetchRegistrationFlowRoles,
  createRegistrationFlow,
  updateRegistrationFlow,
  deleteRegistrationFlow,
  updateRegistrationFlowStatus,
  assignRegistrationFlowRoles,
  removeRegistrationFlowRole,
} from '@/services/api/registration-flows'
import type {
  RegistrationFlowQueryParams,
  RegistrationFlowRolesQueryParams,
  CreateRegistrationFlowRequest,
  UpdateRegistrationFlowRequest,
  UpdateRegistrationFlowStatusRequest
} from '@/services/api/registration-flows/types'

/**
 * Query key factory for registration flows
 */
export const registrationFlowKeys = {
  all: ['registrationFlows'] as const,
  lists: () => [...registrationFlowKeys.all, 'list'] as const,
  list: (params?: RegistrationFlowQueryParams) => [...registrationFlowKeys.lists(), params] as const,
  details: () => [...registrationFlowKeys.all, 'detail'] as const,
  detail: (id: string) => [...registrationFlowKeys.details(), id] as const,
  rolesList: (id: string) => [...registrationFlowKeys.detail(id), 'roles'] as const,
  roles: (id: string, params?: RegistrationFlowRolesQueryParams) =>
    [...registrationFlowKeys.rolesList(id), params] as const,
}

/**
 * Hook to fetch registration flows for the listing page.
 * Maps the shared listing's human-labelled is_system filter ("system"/"regular")
 * onto the boolean the backend expects — same shape as useIdentityProvidersList.
 */
export function useRegistrationFlowsList(params: Record<string, unknown>) {
  const { is_system, ...rest } = params
  const queryParams: RegistrationFlowQueryParams = {
    ...rest as RegistrationFlowQueryParams,
  }

  if (typeof is_system === 'string') {
    if (is_system === 'system') queryParams.is_system = true
    else if (is_system === 'regular') queryParams.is_system = false
  }

  return useRegistrationFlows(queryParams)
}

/**
 * Hook to fetch registration flows with optional filters and pagination.
 * `keepPreviousData` keeps the current page visible while the next one loads, so
 * the table doesn't blank out on every page/filter change.
 */
export function useRegistrationFlows(params?: RegistrationFlowQueryParams) {
  return useQuery({
    queryKey: registrationFlowKeys.list(params),
    queryFn: () => fetchRegistrationFlows(params),
    placeholderData: keepPreviousData,
  })
}

/**
 * Hook to fetch a single registration flow by ID (detail projection)
 */
export function useRegistrationFlow(registrationFlowId: string) {
  return useQuery({
    queryKey: registrationFlowKeys.detail(registrationFlowId),
    queryFn: () => fetchRegistrationFlow(registrationFlowId),
    enabled: !!registrationFlowId,
  })
}

/**
 * Hook to create a new registration flow
 */
export function useCreateRegistrationFlow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateRegistrationFlowRequest) => createRegistrationFlow(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.lists() })
    },
  })
}

/**
 * Hook to update an existing registration flow.
 * A successful update may have replaced the flow's role set (role_ids), so the
 * nested roles listing is invalidated alongside the flow itself.
 */
export function useUpdateRegistrationFlow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ registrationFlowId, data }: { registrationFlowId: string; data: UpdateRegistrationFlowRequest }) =>
      updateRegistrationFlow(registrationFlowId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.detail(variables.registrationFlowId) })
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.lists() })
    },
  })
}

/**
 * Hook to delete a registration flow
 */
export function useDeleteRegistrationFlow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (registrationFlowId: string) => deleteRegistrationFlow(registrationFlowId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.lists() })
    },
  })
}

/**
 * Hook to update registration flow status
 */
export function useUpdateRegistrationFlowStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ registrationFlowId, data }: { registrationFlowId: string; data: UpdateRegistrationFlowStatusRequest }) =>
      updateRegistrationFlowStatus(registrationFlowId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.detail(variables.registrationFlowId) })
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.lists() })
    },
  })
}

/**
 * Hook to fetch roles associated with a registration flow.
 * `options.enabled` lets a caller (e.g. a dialog, or the create form) avoid
 * fetching until the data is actually needed.
 */
export function useRegistrationFlowRoles(
  registrationFlowId: string,
  params?: RegistrationFlowRolesQueryParams,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: registrationFlowKeys.roles(registrationFlowId, params),
    queryFn: () => fetchRegistrationFlowRoles(registrationFlowId, params),
    placeholderData: keepPreviousData,
    enabled: !!registrationFlowId && (options?.enabled ?? true),
  })
}

/**
 * Hook to assign roles to a registration flow.
 * Assignment replaces the flow's role set, so both the roles listing and the
 * flow detail are invalidated.
 */
export function useAssignRegistrationFlowRoles() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ registrationFlowId, data }: { registrationFlowId: string; data: { role_uuids: string[] } }) =>
      assignRegistrationFlowRoles(registrationFlowId, data.role_uuids),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.rolesList(variables.registrationFlowId) })
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.detail(variables.registrationFlowId) })
    },
  })
}

/**
 * Hook to detach a single role from a registration flow
 */
export function useRemoveRegistrationFlowRole() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ registrationFlowId, roleId }: { registrationFlowId: string; roleId: string }) =>
      removeRegistrationFlowRole(registrationFlowId, roleId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.rolesList(variables.registrationFlowId) })
      queryClient.invalidateQueries({ queryKey: registrationFlowKeys.detail(variables.registrationFlowId) })
    },
  })
}
