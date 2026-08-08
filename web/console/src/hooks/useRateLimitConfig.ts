import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchRateLimitConfig, updateRateLimitConfig } from '@/services/api/rate-limit-config'
import type { RateLimitConfigPayload } from '@/services/api/rate-limit-config/types'

export const rateLimitConfigKeys = {
  all: ['rateLimitConfig'] as const,
  detail: () => [...rateLimitConfigKeys.all, 'detail'] as const,
}

export function useRateLimitConfig() {
  return useQuery({
    queryKey: rateLimitConfigKeys.detail(),
    queryFn: fetchRateLimitConfig,
  })
}

export function useUpdateRateLimitConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: RateLimitConfigPayload) => updateRateLimitConfig(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: rateLimitConfigKeys.all }),
  })
}
