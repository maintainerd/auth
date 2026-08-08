import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchRegistrationConfig,
  updateRegistrationConfig
} from '@/services/api/registration-config'
import type { RegistrationConfigPayload } from '@/services/api/registration-config/types'

export const registrationConfigKeys = {
  all: ['registrationConfig'] as const,
  detail: () => [...registrationConfigKeys.all, 'detail'] as const,
}

export function useRegistrationConfig() {
  return useQuery({
    queryKey: registrationConfigKeys.detail(),
    queryFn: fetchRegistrationConfig,
  })
}

export function useUpdateRegistrationConfig() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: RegistrationConfigPayload) => updateRegistrationConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: registrationConfigKeys.all })
    },
  })
}
