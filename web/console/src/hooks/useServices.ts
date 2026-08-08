/**
 * Services Hook
 * Custom hook for fetching services using TanStack Query
 */

import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { fetchServices, fetchServiceById, createService, updateService, deleteService, updateServiceStatus } from '@/services/api/services'
import type {
  ServiceQueryParams,
  CreateServiceRequest,
  UpdateServiceRequest,
  UpdateServiceStatusRequest
} from '@/services/api/services/types'

/**
 * Query key factory for services
 */
export const serviceKeys = {
  all: ['services'] as const,
  lists: () => [...serviceKeys.all, 'list'] as const,
  list: (params?: ServiceQueryParams) => [...serviceKeys.lists(), params] as const,
  details: () => [...serviceKeys.all, 'detail'] as const,
  detail: (id: string) => [...serviceKeys.details(), id] as const,
}

/**
 * Hook to fetch services with optional filters and pagination
 */
export function useServices(params?: ServiceQueryParams) {
  return useQuery({
    queryKey: serviceKeys.list(params),
    queryFn: () => fetchServices(params),
    placeholderData: keepPreviousData,
  })
}

/**
 * Hook to fetch services for the standard listing page.
 * The shared listing filter uses human labels for system/regular, while the
 * service API expects an `is_system` boolean.
 */
export function useServicesList(params: Record<string, unknown>) {
  const { is_system, ...rest } = params
  const queryParams: ServiceQueryParams = {
    ...rest as ServiceQueryParams,
  }

  if (typeof is_system === 'string') {
    if (is_system === 'system') queryParams.is_system = true
    else if (is_system === 'regular') queryParams.is_system = false
  }

  return useServices(queryParams)
}

/**
 * Hook to fetch a single service by ID
 */
export function useService(serviceId: string) {
  return useQuery({
    queryKey: serviceKeys.detail(serviceId),
    queryFn: () => fetchServiceById(serviceId),
    enabled: !!serviceId,
  })
}

/**
 * Hook to create a new service
 */
export function useCreateService() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateServiceRequest) => createService(data),
    onSuccess: () => {
      // Invalidate services list to refetch
      queryClient.invalidateQueries({ queryKey: serviceKeys.lists() })
    },
  })
}

/**
 * Hook to update an existing service
 */
export function useUpdateService() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ serviceId, data }: { serviceId: string; data: UpdateServiceRequest }) =>
      updateService(serviceId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific service and the services list
      queryClient.invalidateQueries({ queryKey: serviceKeys.detail(variables.serviceId) })
      queryClient.invalidateQueries({ queryKey: serviceKeys.lists() })
    },
  })
}

/**
 * Hook to delete a service
 */
export function useDeleteService() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (serviceId: string) => deleteService(serviceId),
    onSuccess: () => {
      // Invalidate services list to refetch
      queryClient.invalidateQueries({ queryKey: serviceKeys.lists() })
    },
  })
}

/**
 * Hook to update service status
 */
export function useUpdateServiceStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ serviceId, data }: { serviceId: string; data: UpdateServiceStatusRequest }) =>
      updateServiceStatus(serviceId, data),
    onSuccess: (_, variables) => {
      // Invalidate both the specific service and the services list
      queryClient.invalidateQueries({ queryKey: serviceKeys.detail(variables.serviceId) })
      queryClient.invalidateQueries({ queryKey: serviceKeys.lists() })
    },
  })
}
