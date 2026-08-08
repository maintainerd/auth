/**
 * Threat Detection Settings Hooks
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchThreatDetectionSettings, updateThreatDetectionSettings } from '@/services/api/threat-detection-settings'
import type { ThreatDetectionSettingsPayload } from '@/services/api/threat-detection-settings/types'

/**
 * Query keys for threat detection settings
 */
export const threatDetectionSettingsKeys = {
  all: ['threat-detection-settings'] as const,
  detail: () => [...threatDetectionSettingsKeys.all, 'detail'] as const,
}

/**
 * Fetch threat detection settings
 */
export function useThreatDetectionSettings() {
  return useQuery({
    queryKey: threatDetectionSettingsKeys.detail(),
    queryFn: fetchThreatDetectionSettings,
  })
}

/**
 * Update threat detection settings
 */
export function useUpdateThreatDetectionSettings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: ThreatDetectionSettingsPayload) => updateThreatDetectionSettings(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: threatDetectionSettingsKeys.all })
    },
  })
}
