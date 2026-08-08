import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchAuditConfig, updateAuditConfig } from '@/services/api/audit-config'
import type { AuditConfigPayload } from '@/services/api/audit-config/types'

export const auditConfigKeys = {
  all: ['auditConfig'] as const,
  detail: () => [...auditConfigKeys.all, 'detail'] as const,
}

export function useAuditConfig() {
  return useQuery({
    queryKey: auditConfigKeys.detail(),
    queryFn: fetchAuditConfig,
  })
}

export function useUpdateAuditConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: AuditConfigPayload) => updateAuditConfig(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: auditConfigKeys.all }),
  })
}
