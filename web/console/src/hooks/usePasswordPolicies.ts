import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchPasswordPolicies,
  updatePasswordPolicies
} from '@/services/api/password-policies'
import type { PasswordPoliciesPayload } from '@/services/api/password-policies/types'

export const passwordPoliciesKeys = {
  all: ['passwordPolicies'] as const,
  detail: () => [...passwordPoliciesKeys.all, 'detail'] as const,
}

export function usePasswordPolicies() {
  return useQuery({
    queryKey: passwordPoliciesKeys.detail(),
    queryFn: fetchPasswordPolicies,
  })
}

export function useUpdatePasswordPolicies() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: PasswordPoliciesPayload) => updatePasswordPolicies(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: passwordPoliciesKeys.detail() })
    },
  })
}
