/**
 * Event Types hook (TanStack Query)
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchEventTypes,
  fetchTenantEventTypes,
  setTenantEventType,
} from '@/services/api/event-types'
import type {
  SetTenantEventTypeRequest,
  TenantEventTypeConfig,
} from '@/services/api/event-types/types'

export const eventTypeKeys = {
  all: ['event-types'] as const,
  list: () => [...eventTypeKeys.all, 'list'] as const,
}

export const tenantEventTypeKeys = {
  all: ['tenant-event-types'] as const,
  list: () => [...tenantEventTypeKeys.all, 'list'] as const,
}

export function useEventTypes() {
  return useQuery({
    queryKey: eventTypeKeys.list(),
    queryFn: fetchEventTypes,
  })
}

/** The tenant's event-type overrides (only events with an explicit on/off). */
export function useTenantEventTypes() {
  return useQuery({
    queryKey: tenantEventTypeKeys.list(),
    queryFn: fetchTenantEventTypes,
  })
}

/**
 * Toggle an event type on/off for the tenant, with an optimistic cache update
 * so the switch responds instantly and rolls back on error.
 */
export function useSetTenantEventType() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: SetTenantEventTypeRequest) => setTenantEventType(data),
    onMutate: async (vars) => {
      await queryClient.cancelQueries({ queryKey: tenantEventTypeKeys.list() })
      const previous = queryClient.getQueryData<TenantEventTypeConfig[]>(tenantEventTypeKeys.list())

      queryClient.setQueryData<TenantEventTypeConfig[]>(tenantEventTypeKeys.list(), (old) => {
        const list = old ?? []
        const idx = list.findIndex((c) => c.event_type_uuid === vars.event_type_uuid)
        if (idx >= 0) {
          const next = list.slice()
          next[idx] = { ...next[idx], enabled: vars.enabled }
          return next
        }
        return [
          ...list,
          {
            tenant_event_type_uuid: '',
            tenant_uuid: '',
            event_type_uuid: vars.event_type_uuid,
            event_type_key: '',
            enabled: vars.enabled,
          },
        ]
      })

      return { previous }
    },
    onSuccess: (result) => {
      queryClient.setQueryData<TenantEventTypeConfig[]>(tenantEventTypeKeys.list(), (old) => {
        const list = old ?? []
        const idx = list.findIndex((c) => c.event_type_uuid === result.event_type_uuid)
        if (idx >= 0) {
          const next = list.slice()
          next[idx] = result
          return next
        }
        return [...list, result]
      })
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(tenantEventTypeKeys.list(), context.previous)
      }
    },
  })
}
