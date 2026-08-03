import { useState } from "react"
import { Monitor, Smartphone, Globe, LogOut } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { SettingsCard } from "@/components/card"
import { ConfirmationDialog } from "@/components/dialog"
import { ListingItemCard } from "@/components/details"
import { useToast } from "@/hooks/useToast"
import { fetchAccountSessions, revokeAccountSession, revokeAllAccountSessions, type AccountSession } from "@/services/api/account"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { StepUpDialog } from "@/components/stepup/StepUpDialog"

function deviceLabel(ua?: string): { label: string; icon: typeof Monitor } {
  if (!ua) return { label: "Unknown device", icon: Globe }
  const s = ua.toLowerCase()
  const mobile = /iphone|android|mobile|ipad/.test(s)
  let os = "Unknown OS"
  if (s.includes("windows")) os = "Windows"
  else if (s.includes("mac os") || s.includes("macintosh")) os = "macOS"
  else if (s.includes("iphone") || s.includes("ipad")) os = "iOS"
  else if (s.includes("android")) os = "Android"
  else if (s.includes("linux")) os = "Linux"
  let browser = ""
  if (s.includes("edg")) browser = "Edge"
  else if (s.includes("chrome")) browser = "Chrome"
  else if (s.includes("firefox")) browser = "Firefox"
  else if (s.includes("safari")) browser = "Safari"
  return { label: browser ? `${browser} on ${os}` : os, icon: mobile ? Smartphone : Monitor }
}

function timeAgo(value?: string | null) {
  if (!value) return "—"
  try { return formatDistanceToNow(new Date(value), { addSuffix: true }) } catch { return "—" }
}

export function SecuritySessions() {
  const { showSuccess, showError } = useToast()
  const queryClient = useQueryClient()
  const [showRevokeAll, setShowRevokeAll] = useState(false)
  const [stepUpOpen, setStepUpOpen] = useState(false)

  const { data, isLoading, isError } = useQuery({ queryKey: ["account", "sessions"], queryFn: fetchAccountSessions, retry: false })

  const revokeOne = useMutation({
    mutationFn: revokeAccountSession,
    onSuccess: () => { showSuccess("Session revoked"); queryClient.invalidateQueries({ queryKey: ["account", "sessions"] }) },
    onError: (e) => showError(e),
  })

  const revokeAll = useMutation({
    mutationFn: (token: string) => revokeAllAccountSessions(token),
    onSuccess: () => { showSuccess("All other sessions revoked"); queryClient.invalidateQueries({ queryKey: ["account", "sessions"] }) },
    onError: (e) => showError(e),
  })

  const sessions = data ?? []

  return (
    <SettingsCard
      title="Active sessions"
      description="Devices and browsers currently signed in to your account."
      icon={Monitor}
    >
      {isLoading ? (
        <div className="space-y-2">{[0, 1].map((i) => <Skeleton key={i} className="h-16 w-full rounded-lg" />)}</div>
      ) : isError ? (
        <p className="py-6 text-center text-sm text-destructive">Failed to load sessions.</p>
      ) : sessions.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">No active sessions found.</p>
      ) : (
        <>
          <div className="space-y-2">
            {sessions.map((s: AccountSession) => {
              const { label, icon: Icon } = deviceLabel(s.user_agent)
              return (
                <ListingItemCard
                  key={s.session_id}
                  icon={Icon}
                  className="items-center p-3"
                  action={
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => revokeOne.mutate(s.session_id)}
                      disabled={revokeOne.isPending}
                    >
                      Revoke
                    </Button>
                  }
                >
                  <p className="truncate text-sm font-medium">{label}</p>
                  <p className="truncate text-xs text-muted-foreground">
                    {s.ip_address || "Unknown IP"} · Last active {timeAgo(s.last_used_at ?? s.created_at)}
                  </p>
                </ListingItemCard>
              )
            })}
          </div>
          <div className="mt-4">
            <Button variant="outline" className="text-destructive hover:text-destructive" onClick={() => setShowRevokeAll(true)}>
              <LogOut className="size-4 mr-2" />
              Sign out everywhere else
            </Button>
          </div>
        </>
      )}

      <ConfirmationDialog
        open={showRevokeAll}
        onOpenChange={setShowRevokeAll}
        onConfirm={() => { setShowRevokeAll(false); setStepUpOpen(true) }}
        title="Sign out everywhere else"
        description="This revokes every other active session. You'll need to sign in again on those devices. We'll ask you to verify your identity first."
        confirmText="Continue"
        variant="destructive"
      />

      <StepUpDialog
        open={stepUpOpen}
        onOpenChange={setStepUpOpen}
        onVerified={(token) => revokeAll.mutate(token)}
        title="Verify to sign out sessions"
        description="Confirm your identity with a second factor to revoke all other sessions."
      />
    </SettingsCard>
  )
}
