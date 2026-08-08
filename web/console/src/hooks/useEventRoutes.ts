import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchEventRoutes,
  fetchEventRouteById,
  createEventRoute,
  updateEventRoute,
  deleteEventRoute,
} from '@/services/api/event-routes'
import type {
  CreateEventRouteRequest,
  UpdateEventRouteRequest,
  EventRoute,
} from '@/services/api/event-routes/types'

export const eventRouteKeys = {
  all: ['event-routes'] as const,
  list: () => [...eventRouteKeys.all, 'list'] as const,
}

export function useEventRoutes() {
  return useQuery({
    queryKey: eventRouteKeys.list(),
    queryFn: fetchEventRoutes,
  })
}

export function useEventRoute(routeId: string | undefined) {
  return useQuery({
    queryKey: ['event-routes', routeId],
    queryFn: () => fetchEventRouteById(routeId!),
    enabled: !!routeId,
  })
}

export function useCreateEventRoute() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateEventRouteRequest) => createEventRoute(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventRouteKeys.list() })
    },
  })
}

export function useUpdateEventRoute() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ uuid, data }: { uuid: string; data: UpdateEventRouteRequest }) =>
      updateEventRoute(uuid, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventRouteKeys.list() })
    },
  })
}

export function useDeleteEventRoute() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (uuid: string) => deleteEventRoute(uuid),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventRouteKeys.list() })
    },
  })
}

/**
 * Toggle an event route on/off for the current tenant + channel.
 * Creates a new route when none exists, enables/disables an existing one.
 */
export function useToggleEventRoute() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      eventTypeUuid,
      enabled,
    }: {
      eventTypeUuid: string
      enabled: boolean
    }) => {
      const routes = queryClient.getQueryData<EventRoute[]>(eventRouteKeys.list()) ?? []
      const existing = routes.find((r) => r.event_type_uuid === eventTypeUuid)

      if (enabled) {
        if (existing) {
          return updateEventRoute(existing.uuid, { enabled: true })
        }
        return createEventRoute({ event_type_uuid: eventTypeUuid })
      }
      if (existing) {
        return updateEventRoute(existing.uuid, { enabled: false })
      }
      return existing
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: eventRouteKeys.list() })
    },
  })
}
