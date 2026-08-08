import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchTokenConfig,
  updateTokenConfig
} from '@/services/api/token-config'
import type { TokenConfigPayload } from '@/services/api/token-config/types'

export const tokenConfigKeys = {
  all: ['tokenConfig'] as const,
  detail: () => [...tokenConfigKeys.all, 'detail'] as const,
}

export function useTokenConfig() {
  return useQuery({
    queryKey: tokenConfigKeys.detail(),
    queryFn: fetchTokenConfig,
  })
}

export function useUpdateTokenConfig() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: TokenConfigPayload) => updateTokenConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tokenConfigKeys.all })
    },
  })
}
