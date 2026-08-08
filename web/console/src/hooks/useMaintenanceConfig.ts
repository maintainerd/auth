import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchMaintenanceConfig, updateMaintenanceConfig } from '@/services/api/maintenance-config'
import type { MaintenanceConfigPayload } from '@/services/api/maintenance-config/types'

export const maintenanceConfigKeys = {
  all: ['maintenanceConfig'] as const,
  detail: () => [...maintenanceConfigKeys.all, 'detail'] as const,
}

export function useMaintenanceConfig() {
  return useQuery({
    queryKey: maintenanceConfigKeys.detail(),
    queryFn: fetchMaintenanceConfig,
  })
}

export function useUpdateMaintenanceConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: MaintenanceConfigPayload) => updateMaintenanceConfig(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: maintenanceConfigKeys.all }),
  })
}
