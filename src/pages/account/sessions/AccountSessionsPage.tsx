import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Globe, Monitor, Smartphone, Trash2 } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { ListingItemCard, ListingItemMeta } from '@/components/details'
import { Button } from '@/components/ui/button'
import { useToast } from '@/hooks/useToast'
import {
  fetchSessions,
  revokeAllSessions,
  revokeSession,
  type UserSession,
} from '@/services/api/account'

function deviceLabel(userAgent?: string): { label: string; icon: typeof Monitor } {
  if (!userAgent) return { label: 'Unknown device', icon: Globe }

  const value = userAgent.toLowerCase()
  const mobile = /iphone|android|mobile|ipad/.test(value)
  let os = 'Unknown OS'
  if (value.includes('windows')) os = 'Windows'
  else if (value.includes('mac os') || value.includes('macintosh')) os = 'macOS'
  else if (value.includes('iphone') || value.includes('ipad')) os = 'iOS'
  else if (value.includes('android')) os = 'Android'
  else if (value.includes('linux')) os = 'Linux'

  let browser = ''
  if (value.includes('edg')) browser = 'Edge'
  else if (value.includes('chrome')) browser = 'Chrome'
  else if (value.includes('firefox')) browser = 'Firefox'
  else if (value.includes('safari')) browser = 'Safari'

  return { label: browser ? `${browser} on ${os}` : os, icon: mobile ? Smartphone : Monitor }
}

function fmt(value?: string | null) {
  if (!value) return '—'
  try {
    return new Date(value).toLocaleDateString()
  } catch {
    return '—'
  }
}

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
  const { label, icon: Icon } = deviceLabel(session.user_agent)

  return (
    <ListingItemCard
      icon={Icon}
      className="items-center p-3"
      contentClassName="items-center"
      action={!session.is_current && (
        <div className="flex shrink-0 items-center gap-2">
          {confirming ? (
            <>
              <Button
                size="sm"
                variant="destructive"
                className="h-7 text-xs"
                disabled={revoking}
                onClick={() => {
                  onRevoke(session.session_uuid)
                  setConfirming(false)
                }}
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
    >
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <p className="truncate text-sm font-medium">{label}</p>
          {session.is_current && (
            <span className="rounded bg-emerald-500/10 px-1.5 py-0.5 text-xs text-emerald-700 dark:text-emerald-400">
              Current
            </span>
          )}
        </div>
        <ListingItemMeta>
          <span>{session.ip_address ?? 'Unknown IP'}</span>
          {session.created_at && <span>Signed in {fmt(session.created_at)}</span>}
          {session.last_active_at && <span>Last active {fmt(session.last_active_at)}</span>}
        </ListingItemMeta>
      </div>
    </ListingItemCard>
  )
}

export default function AccountSessionsPage() {
  const queryClient = useQueryClient()
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
      queryClient.invalidateQueries({ queryKey: ['account', 'sessions'] })
    },
    onError: (err) => showError(err, 'Could not revoke session'),
  })

  const revokeAllMutation = useMutation({
    mutationFn: revokeAllSessions,
    onSuccess: () => {
      showSuccess('All other sessions revoked')
      setConfirmingAll(false)
      queryClient.invalidateQueries({ queryKey: ['account', 'sessions'] })
    },
    onError: (err) => showError(err, 'Could not revoke all sessions'),
  })

  const otherSessions = sessions.filter((session: UserSession) => !session.is_current)
  const revokeAllAction = otherSessions.length > 0 ? (
    <div className="flex w-full justify-end gap-2 sm:w-auto">
      {confirmingAll ? (
        <>
          <Button
            size="sm"
            variant="destructive"
            className="h-7 flex-1 text-xs sm:flex-none"
            disabled={revokeAllMutation.isPending}
            onClick={() => revokeAllMutation.mutate()}
          >
            Confirm revoke all
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="h-7 flex-1 text-xs sm:flex-none"
            onClick={() => setConfirmingAll(false)}
          >
            Cancel
          </Button>
        </>
      ) : (
        <Button
          size="sm"
          variant="outline"
          className="h-7 w-full text-xs text-destructive hover:text-destructive sm:w-auto"
          onClick={() => setConfirmingAll(true)}
        >
          Revoke all others
        </Button>
      )}
    </div>
  ) : undefined

  return (
    <AccountLayout title="Sessions">
      <SettingsCard
        title="Active sign-ins"
        description="Devices and browsers currently signed in to your account."
        icon={Monitor}
        action={revokeAllAction}
      >
        <div className="space-y-4">
          {isLoading && <p className="text-sm text-muted-foreground">Loading sessions…</p>}
          {!isLoading && sessions.length === 0 && (
            <div className="flex flex-col items-center gap-3 py-8 text-center">
              <Monitor className="size-8 text-muted-foreground/50" />
              <p className="text-sm text-muted-foreground">No active sessions found.</p>
            </div>
          )}
          {!isLoading && sessions.length > 0 && (
            <div className="space-y-2">
              {sessions.map((session: UserSession) => (
                <SessionRow
                  key={session.session_uuid}
                  session={session}
                  onRevoke={(uuid) => revokeMutation.mutate(uuid)}
                  revoking={revokeMutation.isPending}
                />
              ))}
            </div>
          )}
        </div>
      </SettingsCard>
    </AccountLayout>
  )
}
