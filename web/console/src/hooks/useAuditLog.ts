import { useQuery } from '@tanstack/react-query'
import { fetchAuditLog, fetchAuditLogEntry } from '@/services/api/audit-log'
import type { AuditLogQueryParams } from '@/services/api/audit-log/types'

export const auditLogKeys = {
  all: ['audit-log'] as const,
  lists: () => [...auditLogKeys.all, 'list'] as const,
  list: (params?: AuditLogQueryParams) => [...auditLogKeys.lists(), params] as const,
  details: () => [...auditLogKeys.all, 'detail'] as const,
  detail: (uuid: string) => [...auditLogKeys.details(), uuid] as const,
}

export function useAuditLog(params?: AuditLogQueryParams) {
  return useQuery({
    queryKey: auditLogKeys.list(params),
    queryFn: () => fetchAuditLog(params),
  })
}

export function useAuditLogList(params: Record<string, unknown>) {
  return useAuditLog(params as AuditLogQueryParams)
}

export function useAuditLogEntry(uuid: string) {
  return useQuery({
    queryKey: auditLogKeys.detail(uuid),
    queryFn: () => fetchAuditLogEntry(uuid),
    enabled: !!uuid,
  })
}
