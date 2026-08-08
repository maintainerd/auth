import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchSessionSettings, updateSessionSettings } from '@/services/api/session-settings'
import type { SessionSettingsPayload } from '@/services/api/session-settings/types'

export const sessionSettingsKeys = {
  all: ['sessionSettings'] as const,
  detail: () => [...sessionSettingsKeys.all, 'detail'] as const,
}

export function useSessionSettings() {
  return useQuery({
    queryKey: sessionSettingsKeys.detail(),
    queryFn: fetchSessionSettings,
  })
}

export function useUpdateSessionSettings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: SessionSettingsPayload) => updateSessionSettings(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: sessionSettingsKeys.detail() })
    },
  })
}
