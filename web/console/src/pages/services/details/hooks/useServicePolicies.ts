/**
 * Custom hook for fetching policies assigned to a service
 */

import { usePolicies } from '@/hooks/usePolicies'
import type { PolicyQueryParams } from '@/services'

interface UseServicePoliciesParams {
  serviceId: string
  page?: number
  limit?: number
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
  name?: string
  description?: string
}

export function useServicePolicies({
  serviceId,
  page = 1,
  limit = 10,
  sortBy = 'name',
  sortOrder = 'asc',
  name,
  description
}: UseServicePoliciesParams) {
  const params: PolicyQueryParams = {
    service_id: serviceId,
    page,
    limit,
    sort_by: sortBy,
    sort_order: sortOrder,
    name,
    description,
  }

  return usePolicies(params)
}

