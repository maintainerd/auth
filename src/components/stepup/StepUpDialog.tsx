import { useEffect, useState } from "react"
import { ShieldCheck } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { cn } from "@/lib/utils"
import { getAssertion } from "@/lib/webauthn"
import { MFA_METHOD_META as METHOD_META, extractMFACode } from "@/lib/mfaMethods"
import { useToast } from "@/hooks/useToast"
import { beginWebAuthnAuthentication, issueStepUpChallenge, sendStepUpSMS, sendStepUpEmailOtp, verifyStepUp } from "@/services/api/mfa"
import { useMutation } from "@tanstack/react-query"

interface StepUpDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Called with an elevated (acr=2) access token once verification succeeds. */
  onVerified: (accessToken: string) => void
  /** Called when the user dismisses the dialog without verifying. */
  onCancel?: () => void
  title?: string
  description?: string
}

/**
 * Prompts the user to re-confirm their identity with a second factor and returns
 * an elevated access token. Gates sensitive actions the backend protects with
 * step-up (acr=2) — both self-service (delete account, revoke sessions) and, via
 * the global step-up interceptor, admin actions (assign role, delete user, …).
 */
export function StepUpDialog({ open, onOpenChange, onVerified, onCancel, title, description }: StepUpDialogProps) {
  const { showError } = useToast()
  const [challengeToken, setChallengeToken] = useState("")
  const [methods, setMethods] = useState<string[]>([])
  const [method, setMethod] = useState("")
  const [code, setCode] = useState("")
  const [smsSent, setSmsSent] = useState(false)
  const [emailOtpSent, setEmailOtpSent] = useState(false)

  const challengeMutation = useMutation({
    mutationFn: issueStepUpChallenge,
    onSuccess: (res) => {
      const allowed = res.allowed_methods?.filter((m) => METHOD_META[m]) ?? []
      setChallengeToken(res.challenge_token)
      setMethods(allowed)
      setMethod(allowed[0] ?? "")
    },
    onError: (e) => { showError(e); onOpenChange(false) },
  })

  const smsMutation = useMutation({
    mutationFn: sendStepUpSMS,
    onSuccess: () => setSmsSent(true),
    onError: (e) => showError(e),
  })

  const emailOtpMutation = useMutation({
    mutationFn: sendStepUpEmailOtp,
    onSuccess: () => setEmailOtpSent(true),
    onError: (e) => showError(e),
  })

  const verifyMutation = useMutation({
    mutationFn: async () => {
      if (METHOD_META[method]?.webauthn) {
        const options = await beginWebAuthnAuthentication()
        const assertion = await getAssertion(options)
        return verifyStepUp(challengeToken, method, { assertion })
      }
      return verifyStepUp(challengeToken, method, { code: extractMFACode(method, code) })
    },
    onSuccess: (res) => { onVerified(res.access_token); onOpenChange(false) },
    onError: (e) => showError(e),
  })

  // Reset and request a fresh challenge each time the dialog opens.
  useEffect(() => {
    if (open) {
      setChallengeToken(""); setMethods([]); setMethod(""); setCode(""); setSmsSent(false); setEmailOtpSent(false)
      challengeMutation.mutate()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  // Treat any close-without-verify as a cancellation so callers (and the global
  // step-up bridge) can settle their pending promise.
  const handleOpenChange = (next: boolean) => {
    if (!next && !verifyMutation.isSuccess) onCancel?.()
    onOpenChange(next)
  }

  const meta = METHOD_META[method]
  const isWebAuthn = meta?.webauthn ?? false
  const numeric = meta?.numeric ?? false
  const canSubmit = isWebAuthn
    ? !verifyMutation.isPending
    : Boolean(method) && Boolean(code.trim()) && !verifyMutation.isPending && (!numeric || code.length === 6)

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldCheck className="size-5 text-muted-foreground" />
            {title ?? "Confirm it's you"}
          </DialogTitle>
          <DialogDescription>
            {description ?? "This is a sensitive action. Verify with a second factor to continue."}
          </DialogDescription>
        </DialogHeader>

        {challengeMutation.isPending ? (
          <p className="py-8 text-center text-sm text-muted-foreground">Preparing verification…</p>
        ) : methods.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            No second factor is available on your account. Set up MFA first to perform this action.
          </p>
        ) : (
          <div className="space-y-4">
            {methods.length > 1 && (
              <div className="space-y-2">
                <Label>Verification method</Label>
                <div className="grid gap-2">
                  {methods.map((m) => {
                    const Icon = METHOD_META[m].icon
                    const selected = m === method
                    return (
                      <button
                        key={m}
                        type="button"
                        onClick={() => { setMethod(m); setCode(""); setSmsSent(false); setEmailOtpSent(false) }}
                        className={cn(
                          "flex items-center gap-3 rounded-lg border p-3 text-left text-sm transition-colors",
                          selected ? "border-primary bg-accent" : "hover:bg-accent/50",
                        )}
                      >
                        <Icon className="size-4 text-muted-foreground" />
                        <span className="font-medium">{METHOD_META[m].label}</span>
                      </button>
                    )
                  })}
                </div>
              </div>
            )}

            {isWebAuthn ? (
              <p className="auth-security-panel rounded-lg border p-3 text-sm text-muted-foreground">
                Use Face ID, Touch ID, Windows Hello, or your security key to confirm.
              </p>
            ) : (
              <>
                {method === "sms" && (
                  <Button variant="outline" size="sm" onClick={() => smsMutation.mutate()} disabled={smsMutation.isPending}>
                    {smsMutation.isPending ? "Sending…" : smsSent ? "Resend code" : "Send code to my phone"}
                  </Button>
                )}
                {method === "email_otp" && (
                  <Button variant="outline" size="sm" onClick={() => emailOtpMutation.mutate()} disabled={emailOtpMutation.isPending}>
                    {emailOtpMutation.isPending ? "Sending…" : emailOtpSent ? "Resend code" : "Send code to my email"}
                  </Button>
                )}

                <div className="space-y-2">
                  <Label htmlFor="stepup-code">{numeric ? `${meta?.label} code` : (meta?.label ?? "Code")}</Label>
                  <Input
                    id="stepup-code"
                    inputMode={numeric ? "numeric" : "text"}
                    autoComplete="one-time-code"
                    placeholder={numeric ? "000000" : "Enter one backup code"}
                    className={numeric ? "font-mono tracking-[0.4em] text-center" : "font-mono"}
                    value={code}
                    onChange={(e) => setCode(numeric ? e.target.value.replace(/\D/g, "").slice(0, 6) : e.target.value)}
                  />
                  {method === "backup_code" && (
                    <p className="text-xs text-muted-foreground">
                      Enter a single code from your saved list — each one works only once.
                    </p>
                  )}
                </div>
              </>
            )}
          </div>
        )}

        <DialogFooter>
          <Button variant="ghost" onClick={() => handleOpenChange(false)}>Cancel</Button>
          <Button onClick={() => verifyMutation.mutate()} disabled={!canSubmit}>
            {verifyMutation.isPending
              ? (isWebAuthn ? "Waiting for device…" : "Verifying…")
              : (isWebAuthn ? "Use passkey" : "Verify")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
