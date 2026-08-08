import { useState, useMemo } from "react"
import { Monitor, Globe, Calendar, Clock, Trash2 } from "lucide-react"
import { safeFormat } from "@/lib/formatDate"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { DataTablePagination, usePaginationTable, RowActions, type RowActionItem } from "@/components/data-table"
import { useUserSessions, useRevokeUserSession, useRevokeAllUserSessions } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import type { UserSession } from "@/services/api/users/types"
import { type PaginationState } from "@tanstack/react-table"

interface UserSessionsProps {
  userId: string
}

function formatDate(value?: string | null) {
  return safeFormat(value, "PPp")
}

export function UserSessions({ userId }: UserSessionsProps) {
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const { data, isLoading, isError } = useUserSessions(userId)
  const { showSuccess, showError } = useToast()
  const revokeMutation = useRevokeUserSession()
  const revokeAllMutation = useRevokeAllUserSessions()

  const sessions = useMemo(() => data ?? [], [data])

  const handleRevokeAll = async () => {
    try {
      await revokeAllMutation.mutateAsync(userId)
      showSuccess("All sessions revoked")
    } catch (error) {
      showError(error)
    }
  }

  const revokeAllActions: RowActionItem[] = [
    {
      key: "revoke-all",
      label: "Revoke all sessions",
      icon: Trash2,
      destructive: true,
      onSelect: handleRevokeAll,
      confirm: {
        title: "Revoke all sessions?",
        description:
          "This signs the user out of every device immediately. They will need to sign in again.",
        confirmText: "Revoke all",
      },
    },
  ]

  const paginatedSessions = useMemo(() => {
    const start = pagination.pageIndex * pagination.pageSize
    return sessions.slice(start, start + pagination.pageSize)
  }, [sessions, pagination])

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: Math.max(1, Math.ceil(sessions.length / pagination.pageSize)),
  })

  const revokeSession = async (session: UserSession) => {
    try {
      await revokeMutation.mutateAsync({ userId, sessionId: session.session_id })
      showSuccess("Session revoked")
    } catch (error) {
      showError(error)
    }
  }

  const sessionActions = (session: UserSession): RowActionItem[] => [
    {
      key: "revoke",
      label: "Revoke session",
      icon: Trash2,
      destructive: true,
      onSelect: () => revokeSession(session),
      confirm: {
        title: "Revoke session",
        description: "This signs the user out of this session immediately. Continue?",
        confirmText: "Revoke",
      },
    },
  ]

  return (
    <InformationCard
      title="Sessions"
      description="Active sign-in sessions for this user. Revoke any to sign them out of that device."
      icon={Monitor}
      action={sessions.length > 0 ? <RowActions items={revokeAllActions} variant="header" /> : undefined}
    >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

        {isError && <p className="py-8 text-center text-sm text-destructive">Failed to load sessions</p>}

        {data && data.length === 0 && (
          <EmptyState
            icon={Monitor}
            title="No active sessions"
            description="This user has no active sign-in sessions right now."
          />
        )}

        {paginatedSessions.length > 0 && (
          <div className="space-y-3">
            {paginatedSessions.map((session) => (
              <div
                data-md-listing-item
                key={session.session_id}
                className="flex items-start justify-between gap-3 rounded-lg border p-4"
              >
                <div className="flex min-w-0 items-start gap-3">
                  <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Monitor className="size-4" />
                  </div>
                  <div className="min-w-0 space-y-1">
                    <p className="break-words text-sm font-medium">{session.user_agent || "Unknown device"}</p>
                    <div data-md-listing-meta className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      {session.ip_address && (
                        <span className="inline-flex items-center gap-1 font-mono">
                          <Globe className="size-3" />
                          {session.ip_address}
                        </span>
                      )}
                      <span className="inline-flex items-center gap-1">
                        <Clock className="size-3" />
                        Last used {formatDate(session.last_used_at)}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        <Calendar className="size-3" />
                        Started {formatDate(session.created_at)}
                      </span>
                    </div>
                  </div>
                </div>
                <RowActions items={sessionActions(session)} />
              </div>
            ))}
          </div>
        )}

        {sessions.length > 0 && (
          <div className="border-t pt-4">
            <DataTablePagination table={table} rowCount={sessions.length} />
          </div>
        )}
      </div>
    </InformationCard>
  )
}
