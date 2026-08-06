import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Globe, Monitor, Smartphone, Trash2 } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { SettingsCard } from '@/components/card'
import { ListingItemCard, ListingItemMeta } from '@/components/details'
import { Button } from '@/components/ui/button'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from '@/components/ui/dialog'
import { useAuth } from '@/hooks/useAuth'
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
      action={(
        <div className="flex shrink-0 items-center gap-2">
          {confirming ? (
            <>
              <Button
                size="sm"
                variant="destructive"
                className="h-9 sm:h-7 text-xs"
                disabled={revoking}
                onClick={() => {
                  onRevoke(session.session_id)
                  setConfirming(false)
                }}
              >
                Confirm revoke
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="h-9 sm:h-7 text-xs"
                onClick={() => setConfirming(false)}
              >
                Cancel
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              variant="ghost"
              className="h-9 sm:h-7 gap-1 text-xs text-destructive hover:text-destructive"
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
        {/* No "Current" badge: /account/sessions does not return an
            is_current flag, and the previous truthiness check on a
            non-existent field marked nothing while implying it did. */}
        <div className="flex flex-wrap items-center gap-2">
          <p className="truncate text-sm font-medium">{label}</p>
        </div>
        <ListingItemMeta>
          <span className="break-all">{session.ip_address ?? 'Unknown IP'}</span>
          {session.created_at && <span>Signed in {fmt(session.created_at)}</span>}
          {session.last_used_at && <span>Last active {fmt(session.last_used_at)}</span>}
        </ListingItemMeta>
      </div>
    </ListingItemCard>
  )
}

export default function AccountSessionsPage() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { logout } = useAuth()
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

  // DELETE /account/sessions calls RevokeAllSessions, which revokes EVERY
  // session for the user — the caller's included — plus every OAuth refresh
  // token. The old label ("Revoke all others") and toast ("All other sessions
  // revoked") described a sign-out-others action the endpoint does not perform,
  // so the user was signed out of the browser in front of them while being told
  // it had been spared. The copy now states what happens, and this browser
  // follows the server: local auth state is cleared and we land on /login
  // instead of leaving a dead session rendering account pages.
  const revokeAllMutation = useMutation({
    mutationFn: revokeAllSessions,
    onSuccess: async () => {
      showSuccess('Signed out on every device')
      setConfirmingAll(false)
      queryClient.removeQueries({ queryKey: ['account'] })
      try {
        await logout()
      } catch {
        // The session backing this browser is already gone, so /logout answering
        // 401 is expected — the store clears auth state either way.
      }
      navigate('/login', { replace: true })
    },
    onError: (err) => showError(err, 'Could not sign out everywhere'),
  })

  const revokeAllAction = sessions.length > 0 ? (
    <div className="flex w-full justify-end sm:w-auto">
      <Button
        size="sm"
        variant="outline"
        className="h-9 sm:h-7 w-full text-xs text-destructive hover:text-destructive sm:w-auto"
        onClick={() => setConfirmingAll(true)}
      >
        Sign out everywhere
      </Button>
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
                  key={session.session_id}
                  session={session}
                  onRevoke={(uuid) => revokeMutation.mutate(uuid)}
                  revoking={revokeMutation.isPending}
                />
              ))}
            </div>
          )}
        </div>
      </SettingsCard>

      {/* A dialog rather than a pair of inline buttons: this action ends the
          session the user is reading the page in, and that needs a sentence to
          say so, not a header button with no room for one. */}
      <Dialog open={confirmingAll} onOpenChange={(open) => { if (!open) setConfirmingAll(false) }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Sign out everywhere?</DialogTitle>
            <DialogDescription>
              This ends every session on your account — including this one, on this device.
              Apps you have connected will also need you to sign in again. You will be returned
              to the sign-in page.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setConfirmingAll(false)}>Cancel</Button>
            <Button
              variant="destructive"
              disabled={revokeAllMutation.isPending}
              onClick={() => revokeAllMutation.mutate()}
            >
              {revokeAllMutation.isPending ? 'Signing out…' : 'Sign out everywhere'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </AccountLayout>
  )
}
