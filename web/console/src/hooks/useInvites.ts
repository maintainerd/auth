/**
 * Invitations hooks (TanStack Query)
 */

import { useMemo } from 'react'
import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { fetchInvites, sendInvite, resendInvite, revokeInvite, fetchInviteById } from '@/services/api/invites'
import type { Invite, SendInviteRequest } from '@/services/api/invites/types'
import type { ServerListResult } from '@/components/data-table'

export const inviteKeys = {
  all: ['invites'] as const,
  lists: () => [...inviteKeys.all, 'list'] as const,
  details: () => [...inviteKeys.all, 'detail'] as const,
  detail: (id: string) => [...inviteKeys.details(), id] as const,
}

export function useInvites() {
  return useQuery({
    queryKey: inviteKeys.lists(),
    queryFn: fetchInvites,
    placeholderData: keepPreviousData,
  })
}

// Single invitation fetched from the dedicated GET-by-id endpoint.
export function useInvite(inviteId: string | undefined) {
  return useQuery({
    queryKey: inviteKeys.detail(inviteId ?? ''),
    queryFn: () => fetchInviteById(inviteId!),
    enabled: !!inviteId,
  })
}

/**
 * Adapter that presents the (un-paginated) invitations array through the
 * standard `ServerListResult` shape `ResourceListing`/`useServerDataTable`
 * expects — applying search, the status filter, sorting, and pagination
 * client-side. Invitations is tabular operational data, so it uses the same
 * DataTable standard as users/roles.
 */
export function useInvitesList(params: Record<string, unknown>) {
  const { data: all, isLoading, error } = useInvites()

  const data: ServerListResult<Invite> = useMemo(() => {
    let rows = [...(all ?? [])]

    const term = String(params.invited_email ?? '').toLowerCase().trim()
    if (term) rows = rows.filter((i) => i.invited_email?.toLowerCase().includes(term))

    const status = String(params.status ?? '')
    if (status) {
      const wanted = status.split(',').filter(Boolean)
      rows = rows.filter((i) => wanted.includes(i.status))
    }

    const sortKey = String(params.sort_by ?? 'created_at').toLowerCase()
    const getValue = (i: Invite): string => {
      switch (sortKey) {
        case 'email':
        case 'invited_email':
          return i.invited_email?.toLowerCase() ?? ''
        case 'status':
          return i.status
        case 'expires':
        case 'expires_at':
          return i.expires_at ?? ''
        default:
          return i.created_at ?? ''
      }
    }
    const desc = String(params.sort_order ?? 'desc') === 'desc'
    rows.sort((a, b) => {
      const av = getValue(a)
      const bv = getValue(b)
      if (av < bv) return desc ? 1 : -1
      if (av > bv) return desc ? -1 : 1
      return 0
    })

    const total = rows.length
    const page = Math.max(1, Number(params.page ?? 1))
    const limit = Math.max(1, Number(params.limit ?? 10))
    const start = (page - 1) * limit
    return { rows: rows.slice(start, start + limit), total }
  }, [all, params])

  return { data, isLoading, error: (error as Error) ?? null }
}

export function useSendInvite() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: SendInviteRequest) => sendInvite(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: inviteKeys.lists() })
    },
  })
}

export function useResendInvite() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (inviteId: string) => resendInvite(inviteId),
    onSuccess: (_data, inviteId) => {
      queryClient.invalidateQueries({ queryKey: inviteKeys.lists() })
      queryClient.invalidateQueries({ queryKey: inviteKeys.detail(inviteId) })
    },
  })
}

export function useRevokeInvite() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (inviteId: string) => revokeInvite(inviteId),
    onSuccess: (_data, inviteId) => {
      queryClient.invalidateQueries({ queryKey: inviteKeys.lists() })
      queryClient.invalidateQueries({ queryKey: inviteKeys.detail(inviteId) })
    },
  })
}
