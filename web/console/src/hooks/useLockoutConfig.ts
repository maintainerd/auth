import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchLockoutConfig,
  updateLockoutConfig
} from '@/services/api/lockout-config'
import type { LockoutConfigPayload } from '@/services/api/lockout-config/types'

export const lockoutConfigKeys = {
  all: ['lockoutConfig'] as const,
  detail: () => [...lockoutConfigKeys.all, 'detail'] as const,
}

export function useLockoutConfig() {
  return useQuery({
    queryKey: lockoutConfigKeys.detail(),
    queryFn: fetchLockoutConfig,
  })
}

export function useUpdateLockoutConfig() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: LockoutConfigPayload) => updateLockoutConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: lockoutConfigKeys.all })
    },
  })
}
