import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { LucideIcon } from "lucide-react"
import {
  ShieldCheck, ShieldAlert, ShieldOff, Smartphone, MessageSquare, Mail, Key, KeyRound, ChevronRight,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { ListingItemIcon } from "@/components/details/ListingItemCard"
import { Separator } from "@/components/ui/separator"
import AccountLayout from "@/components/layout/AccountLayout"
import { useToast } from "@/hooks/useToast"
import { cn } from "@/lib/utils"
import { fetchMFAStatus, resetAllMFA } from "@/services/api/mfa"
import { ConfirmRemoveDialog } from "./MfaShell"

export default function MFAPage() {
  const navigate = useNavigate()
  const { data, isLoading } = useQuery({ queryKey: ["mfa", "status"], queryFn: fetchMFAStatus })

  if (isLoading) {
    return (
      <AccountLayout title="Two-Factor Authentication">
        <MFAHubSkeleton />
      </AccountLayout>
    )
  }

  const totpOn = data?.is_totp_enabled ?? false
  const smsOn = data?.is_sms_available ?? false
  const emailOn = data?.is_email_otp_available ?? false
  const passkeyCount = data?.webauthn_keys?.length ?? 0
  const passkeyOn = (data?.is_webauthn_enabled ?? false) || passkeyCount > 0
  const activeCount = [totpOn, smsOn, emailOn, passkeyOn].filter(Boolean).length
  const isProtected = activeCount > 0
  const backupCount = data?.backup_codes_count ?? 0

  return (
    <AccountLayout title="Two-Factor Authentication">
      <p className="-mt-4 mb-6 text-muted-foreground">
        Add an extra layer of security to your account by requiring a second step at sign-in.
      </p>

      <div className="space-y-6">
        {/* Security status */}
        <Card data-md-listing-item className="auth-security-panel">
          <CardContent className="flex items-center gap-4">
            <ListingItemIcon className="size-11 rounded-full">
              {isProtected
                ? <ShieldCheck className="size-6 text-emerald-600" />
                : <ShieldAlert className="size-6 text-amber-600" />}
            </ListingItemIcon>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-semibold">
                {isProtected ? "Your account is protected" : "Your account is not protected"}
              </p>
              <p className="mt-0.5 text-sm text-muted-foreground">
                {isProtected
                  ? `${activeCount} authentication ${activeCount === 1 ? "method is" : "methods are"} active.`
                  : "Enable at least one authentication method to secure your account."}
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Authentication methods */}
        <div>
          <h2 className="mb-2 px-1 text-sm font-medium text-muted-foreground">Authentication methods</h2>
          <Card className="overflow-hidden py-0">
            <CardContent className="p-0">
              <MethodRow
                icon={Smartphone}
                title="Authenticator app"
                description="Use Google Authenticator, Authy, or 1Password to generate codes."
                active={totpOn}
                onClick={() => navigate("/account/mfa/totp")}
              />
              <Separator />
              <MethodRow
                icon={MessageSquare}
                title="Text message (SMS)"
                description="Receive one-time codes on your verified phone number."
                active={smsOn}
                onClick={() => navigate("/account/mfa/sms")}
              />
              <Separator />
              <MethodRow
                icon={Mail}
                title="Email OTP"
                description="Receive one-time codes at your verified email address."
                active={emailOn}
                onClick={() => navigate("/account/mfa/email-otp")}
              />
              <Separator />
              <MethodRow
                icon={Key}
                title="Passkeys"
                description="Sign in with Face ID, Touch ID, Windows Hello, or a security key."
                active={passkeyOn}
                activeLabel={passkeyOn ? `${passkeyCount} registered` : undefined}
                onClick={() => navigate("/account/mfa/passkeys")}
              />
            </CardContent>
          </Card>
        </div>

        {/* Recovery */}
        {totpOn && (
          <div>
            <h2 className="mb-2 px-1 text-sm font-medium text-muted-foreground">Recovery</h2>
            <Card className="overflow-hidden py-0">
              <CardContent className="p-0">
                <MethodRow
                  icon={KeyRound}
                  title="Backup codes"
                  description="One-time codes to use if you lose access to your other methods."
                  active={backupCount > 0}
                  activeLabel={backupCount > 0 ? `${backupCount} remaining` : undefined}
                  onClick={() => navigate("/account/mfa/totp")}
                />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Reset (self-service) */}
        {isProtected && <ResetAllMFA />}
      </div>
    </AccountLayout>
  )
}

// Lets the account owner clear all of their own MFA at once. Self-scoped on the
// server (no target param), so this only ever resets the signed-in user's MFA.
function ResetAllMFA() {
  const queryClient = useQueryClient()
  const { showSuccess, showError } = useToast()
  const [open, setOpen] = useState(false)

  const mutation = useMutation({
    mutationFn: resetAllMFA,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
      showSuccess("All MFA methods have been removed")
      setOpen(false)
    },
    onError: (e) => { showError(e); setOpen(false) },
  })

  return (
    <div>
      <h2 className="mb-2 px-1 text-sm font-medium text-muted-foreground">Reset</h2>
      <Card className="border-destructive/30">
        <CardContent className="flex items-center justify-between gap-4">
          <div className="min-w-0">
            <p className="text-sm font-medium">Reset all methods</p>
            <p className="mt-0.5 text-sm text-muted-foreground">
              Remove every MFA method from your account. You can set them up again at any time.
            </p>
          </div>
          <Button variant="outline" className="shrink-0" onClick={() => setOpen(true)}>
            <ShieldOff className="mr-2 size-4" /> Reset MFA
          </Button>
        </CardContent>
      </Card>

      <ConfirmRemoveDialog
        open={open}
        onOpenChange={setOpen}
        onConfirm={() => mutation.mutate()}
        title="Reset your MFA"
        description="This removes all of your multi-factor authentication methods — authenticator app, SMS, passkeys, and backup codes. You'll be asked to set up MFA again on your next sign-in. Continue?"
        confirmText="Reset MFA"
        loadingText="Resetting…"
        isLoading={mutation.isPending}
      />
    </div>
  )
}

function StatusBadge({ active }: { active: boolean }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-md border bg-background px-2 py-0.5 text-xs font-medium">
      <span className={cn("size-1.5 rounded-full", active ? "bg-emerald-500" : "bg-muted-foreground")} />
      {active ? "Active" : "Inactive"}
    </span>
  )
}

interface MethodRowProps {
  icon: LucideIcon
  title: string
  description: string
  active: boolean
  activeLabel?: string
  onClick: () => void
}

function MethodRow({ icon: Icon, title, description, active, activeLabel, onClick }: MethodRowProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex w-full items-center gap-4 px-4 py-4 text-left transition-colors hover:bg-accent/50 focus-visible:bg-accent/50 focus-visible:outline-none first:rounded-t-xl last:rounded-b-xl"
    >
      <ListingItemIcon className="size-10">
        <Icon className="size-5" />
      </ListingItemIcon>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="text-sm font-medium">{title}</p>
          <StatusBadge active={active} />
          {active && activeLabel && <span className="text-xs text-muted-foreground">· {activeLabel}</span>}
        </div>
        <p className="mt-0.5 truncate text-sm text-muted-foreground">{description}</p>
      </div>
      <div className="flex shrink-0 items-center gap-2 text-sm text-muted-foreground">
        <span className="hidden sm:inline">{active ? "Manage" : "Set up"}</span>
        <ChevronRight className="size-4" />
      </div>
    </button>
  )
}

function MFAHubSkeleton() {
  return (
    <div className="space-y-6">
      <div className="h-20 w-full animate-pulse rounded-xl bg-muted" />
      <div className="space-y-2">
        <div className="h-4 w-40 animate-pulse rounded bg-muted" />
        <Card className="overflow-hidden py-0">
          <CardContent className="p-0">
            {[0, 1, 2].map((i) => (
              <div key={i} className="flex items-center gap-4 px-4 py-4">
                <div className="size-10 shrink-0 animate-pulse rounded-lg bg-muted" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 w-48 animate-pulse rounded bg-muted" />
                  <div className="h-3 w-64 animate-pulse rounded bg-muted" />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
