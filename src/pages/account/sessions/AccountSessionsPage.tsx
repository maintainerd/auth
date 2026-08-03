import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Monitor, Trash2 } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useToast } from '@/hooks/useToast'
import {
  fetchSessions,
  revokeSession,
  revokeAllSessions,
  type UserSession,
} from '@/services/api/account'

function SessionRow({
  session,
  onRevoke,
  revoking,
}: {
  session: UserSession
  onRevoke: (uuid: string) => void
  revoking: boolean
}) {
  const [confirming, setConfirming] = useState(false)
  const ua = session.user_agent ? session.user_agent.slice(0, 60) : 'Unknown device'

  return (
    <li className="flex items-start justify-between gap-4 py-3">
      <div className="flex min-w-0 items-center gap-3">
        <Monitor className="size-5 shrink-0 text-muted-foreground" />
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">
            {ua}
            {session.is_current && (
              <span className="ml-2 rounded bg-emerald-500/10 px-1.5 py-0.5 text-xs text-emerald-700 dark:text-emerald-400">
                Current
              </span>
            )}
          </p>
          <p className="text-xs text-muted-foreground">
            {session.ip_address ?? 'Unknown IP'}
            {session.created_at && <> &middot; signed in {new Date(session.created_at).toLocaleDateString()}</>}
            {session.last_active_at && <> &middot; last active {new Date(session.last_active_at).toLocaleDateString()}</>}
          </p>
        </div>
      </div>

      {!session.is_current && (
        <div className="flex shrink-0 items-center gap-2">
          {confirming ? (
            <>
              <Button
                size="sm"
                variant="destructive"
                className="h-7 text-xs"
                disabled={revoking}
                onClick={() => { onRevoke(session.session_uuid); setConfirming(false) }}
              >
                Confirm revoke
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="h-7 text-xs"
                onClick={() => setConfirming(false)}
              >
                Cancel
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              variant="ghost"
              className="h-7 gap-1 text-xs text-destructive hover:text-destructive"
              disabled={revoking}
              onClick={() => setConfirming(true)}
            >
              <Trash2 className="size-3" />
              Revoke
            </Button>
          )}
        </div>
      )}
    </li>
  )
}

export default function AccountSessionsPage() {
  const qc = useQueryClient()
  const { showError, showSuccess } = useToast()
  const [confirmingAll, setConfirmingAll] = useState(false)

  const { data: sessions = [], isLoading } = useQuery({
    queryKey: ['account', 'sessions'],
    queryFn: fetchSessions,
  })

  const revokeMutation = useMutation({
    mutationFn: (uuid: string) => revokeSession(uuid),
    onSuccess: () => {
      showSuccess('Session revoked')
      qc.invalidateQueries({ queryKey: ['account', 'sessions'] })
    },
    onError: (err) => showError(err, 'Could not revoke session'),
  })

  const revokeAllMutation = useMutation({
    mutationFn: revokeAllSessions,
    onSuccess: () => {
      showSuccess('All other sessions revoked')
      setConfirmingAll(false)
      qc.invalidateQueries({ queryKey: ['account', 'sessions'] })
    },
    onError: (err) => showError(err, 'Could not revoke all sessions'),
  })

  const otherSessions = sessions.filter((s: UserSession) => !s.is_current)

  return (
    <AccountLayout title="Sessions">
      <div className="space-y-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-base">Active sign-ins</CardTitle>
            {otherSessions.length > 0 && (
              <div className="flex gap-2">
                {confirmingAll ? (
                  <>
                    <Button
                      size="sm"
                      variant="destructive"
                      className="h-7 text-xs"
                      disabled={revokeAllMutation.isPending}
                      onClick={() => revokeAllMutation.mutate()}
                    >
                      Confirm revoke all
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-7 text-xs"
                      onClick={() => setConfirmingAll(false)}
                    >
                      Cancel
                    </Button>
                  </>
                ) : (
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-7 text-xs text-destructive hover:text-destructive"
                    onClick={() => setConfirmingAll(true)}
                  >
                    Revoke all others
                  </Button>
                )}
              </div>
            )}
          </CardHeader>
          <CardContent>
            {isLoading && (
              <p className="text-sm text-muted-foreground">Loading sessions…</p>
            )}
            {!isLoading && sessions.length === 0 && (
              <div className="flex flex-col items-center gap-3 py-8 text-center">
                <Monitor className="size-8 text-muted-foreground/50" />
                <p className="text-sm text-muted-foreground">No active sessions found.</p>
              </div>
            )}
            <ul className="divide-y">
              {sessions.map((session: UserSession) => (
                <SessionRow
                  key={session.session_uuid}
                  session={session}
                  onRevoke={(uuid) => revokeMutation.mutate(uuid)}
                  revoking={revokeMutation.isPending}
                />
              ))}
            </ul>
          </CardContent>
        </Card>
      </div>
    </AccountLayout>
  )
}
