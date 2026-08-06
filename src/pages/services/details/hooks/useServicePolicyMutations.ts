/**
 * Custom hook for managing service policy assignments
 */

import { useMutation, useQueryClient } from '@tanstack/react-query'
import { assignPolicyToService, removePolicyFromService } from '@/services'
import { policyKeys } from '@/hooks/usePolicies'
import { serviceKeys } from '@/hooks/useServices'
import { policyServicesKeys } from '@/pages/policies/details/hooks/useServicesByPolicy'

// Success/error feedback lives with the calling component (useToast), matching
// the shared mutation-hook convention: hooks only mutate and invalidate.
export function useServicePolicyMutations(serviceId: string) {
  const queryClient = useQueryClient()

  const invalidate = (policyId: string) => {
    // The assigned-policies tab lists policies filtered by service_id.
    queryClient.invalidateQueries({ queryKey: policyKeys.all })
    // Service detail + listing carry the policy_count.
    queryClient.invalidateQueries({ queryKey: serviceKeys.detail(serviceId) })
    queryClient.invalidateQueries({ queryKey: serviceKeys.lists() })
    // The link is bidirectional: the policy's own Services tab is keyed under
    // ['policy', id, 'services'], which policyKeys.all ('policies') does not
    // match — without this it kept rendering the pre-assignment service list.
    queryClient.invalidateQueries({ queryKey: policyServicesKeys.all(policyId) })
  }

  const assignPolicy = useMutation({
    mutationFn: (policyId: string) => assignPolicyToService(serviceId, policyId),
    onSuccess: (_, policyId) => invalidate(policyId),
  })

  const removePolicy = useMutation({
    mutationFn: (policyId: string) => removePolicyFromService(serviceId, policyId),
    onSuccess: (_, policyId) => invalidate(policyId),
  })

  return {
    assignPolicy,
    removePolicy,
  }
}
