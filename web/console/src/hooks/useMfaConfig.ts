import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchMfaConfig,
  updateMfaConfig
} from '@/services/api/mfa-config'
import type { MfaConfigPayload } from '@/services/api/mfa-config/types'

export const mfaConfigKeys = {
  all: ['mfaConfig'] as const,
  detail: () => [...mfaConfigKeys.all, 'detail'] as const,
}

export function useMfaConfig() {
  return useQuery({
    queryKey: mfaConfigKeys.detail(),
    queryFn: fetchMfaConfig,
  })
}

export function useUpdateMfaConfig() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: MfaConfigPayload) => updateMfaConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mfaConfigKeys.all })
    },
  })
}
