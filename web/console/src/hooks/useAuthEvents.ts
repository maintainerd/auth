import { useQuery } from '@tanstack/react-query'
import { fetchAuthEvents, fetchAuthEventById, fetchAuthEventCount } from '@/services/api/auth-events'
import type { AuthEventQueryParams } from '@/services/api/auth-events/types'

export const authEventKeys = {
  all: ['auth-events'] as const,
  lists: () => [...authEventKeys.all, 'list'] as const,
  list: (params?: AuthEventQueryParams) => [...authEventKeys.lists(), params] as const,
  details: () => [...authEventKeys.all, 'detail'] as const,
  detail: (id: string) => [...authEventKeys.details(), id] as const,
}

export function useAuthEvents(params?: AuthEventQueryParams) {
  return useQuery({
    queryKey: authEventKeys.list(params),
    queryFn: () => fetchAuthEvents(params),
  })
}

export function useAuthEventsList(params: Record<string, unknown>) {
  return useAuthEvents(params as AuthEventQueryParams)
}

export function useAuthEvent(eventId: string) {
  return useQuery({
    queryKey: authEventKeys.detail(eventId),
    queryFn: () => fetchAuthEventById(eventId),
    enabled: !!eventId,
  })
}

export function useAuthEventCount(eventType: string) {
  return useQuery({
    queryKey: ['auth-events', 'count', eventType],
    queryFn: () => fetchAuthEventCount(eventType),
    enabled: !!eventType,
  })
}
