/**
 * Custom hook for fetching services that use a specific policy
 */

import { useQuery } from '@tanstack/react-query'
import { fetchServicesByPolicy } from '@/services'
import type { ServiceQueryParams } from '@/services/api/services/types'

interface UseServicesByPolicyParams extends ServiceQueryParams {
  policyId: string
}

/**
 * Query key factory for the services-that-use-a-policy listing.
 *
 * Exported so mutations that change a service↔policy link can invalidate this
 * cache by reference. Duplicating the raw array literal in the mutation file is
 * how the policy's Services tab silently went stale after an assignment made
 * from the service side.
 */
export const policyServicesKeys = {
  all: (policyId: string) => ['policy', policyId, 'services'] as const,
  list: (policyId: string, params?: ServiceQueryParams) =>
    [...policyServicesKeys.all(policyId), params] as const,
}

export function useServicesByPolicy({
  policyId,
  page = 1,
  limit = 10,
  sort_by = 'name',
  sort_order = 'asc',
  name,
  display_name,
  description,
}: UseServicesByPolicyParams) {
  const params: ServiceQueryParams = {
    page,
    limit,
    sort_by,
    sort_order,
    name,
    display_name,
    description,
  }

  return useQuery({
    queryKey: policyServicesKeys.list(policyId, params),
    queryFn: () => fetchServicesByPolicy(policyId, params),
    enabled: !!policyId,
  })
}

