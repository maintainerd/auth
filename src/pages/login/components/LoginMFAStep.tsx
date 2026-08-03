import { useEffect, useState } from "react"
import { ShieldCheck } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"
import { getAssertion } from "@/lib/webauthn"
import { MFA_METHOD_META as METHOD_META, extractMFACode } from "@/lib/mfaMethods"
import { useToast } from "@/hooks/useToast"
import { useAuth } from "@/hooks/useAuth"
import { sendMFALoginSMS, sendMFALoginEmailOtp, beginMFALoginWebAuthn } from "@/services/api/auth"
import type { AccountEntity } from "@/services/api/auth/types"
import { useMutation } from "@tanstack/react-query"

interface LoginMFAStepProps {
  challengeToken: string
  allowedMethods: string[]
  tenantId?: string
  clientId?: string
  /** Called after the MFA step succeeds and the session is established. */
  onVerified: (result: { account: AccountEntity | null }) => void
  /** Return to the username/password form. */
  onCancel: () => void
}

/**
 * Second login step shown when the account has MFA enrolled. The user confirms
 * a factor (TOTP/SMS/backup code/passkey); on success the backend issues an
 * acr=2 session so every step-up-gated action works for the whole session.
 */
export function LoginMFAStep({ challengeToken, allowedMethods, tenantId, clientId, onVerified, onCancel }: LoginMFAStepProps) {
  const { showError } = useToast()
  const { completeMFALogin } = useAuth()

  const methods = allowedMethods.filter((m) => METHOD_META[m])
  const [method, setMethod] = useState(methods[0] ?? "")
  const [code, setCode] = useState("")
  const [smsSent, setSmsSent] = useState(false)
  const [emailOtpSent, setEmailOtpSent] = useState(false)
  const [rememberDevice, setRememberDevice] = useState(false)

  useEffect(() => {
    if (!method && methods.length > 0) setMethod(methods[0])
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allowedMethods])

  const smsMutation = useMutation({
    mutationFn: () => sendMFALoginSMS(challengeToken, { tenantId, clientId }),
    onSuccess: () => setSmsSent(true),
    onError: (e) => showError(e),
  })

  const emailOtpMutation = useMutation({
    mutationFn: () => sendMFALoginEmailOtp(challengeToken, { tenantId, clientId }),
    onSuccess: () => setEmailOtpSent(true),
    onError: (e) => showError(e),
  })

  const verifyMutation = useMutation({
    mutationFn: async () => {
      if (METHOD_META[method]?.webauthn) {
        const options = await beginMFALoginWebAuthn(challengeToken, { tenantId, clientId })
        const assertion = await getAssertion(options)
        return completeMFALogin(challengeToken, method, { assertion }, tenantId, clientId, rememberDevice)
      }
      return completeMFALogin(challengeToken, method, { code: extractMFACode(method, code) }, tenantId, clientId, rememberDevice)
    },
    onSuccess: (result) => onVerified(result),
    onError: (e) => showError(e),
  })

  const meta = METHOD_META[method]
  const isWebAuthn = meta?.webauthn ?? false
  const numeric = meta?.numeric ?? false
  const canSubmit = isWebAuthn
    ? !verifyMutation.isPending
    : Boolean(method) && Boolean(code.trim()) && !verifyMutation.isPending && (!numeric || code.length === 6)

  if (methods.length === 0) {
    return (
      <div className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          MFA is required but no supported factor is available. Contact your administrator.
        </p>
        <Button variant="ghost" onClick={onCancel}>Back to login</Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <ShieldCheck className="size-5 text-muted-foreground" />
        <div>
          <p className="font-medium">Two-step verification</p>
          <p className="text-sm text-muted-foreground">Confirm your identity with a second factor.</p>
        </div>
      </div>

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
                  data-md-option-card
                  data-state={selected ? "open" : undefined}
                  aria-pressed={selected}
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
            <Label htmlFor="login-mfa-code">{numeric ? `${meta?.label} code` : (meta?.label ?? "Code")}</Label>
            <Input
              id="login-mfa-code"
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

      <label htmlFor="login-mfa-remember" className="flex items-center gap-2 text-sm text-muted-foreground">
        <input
          id="login-mfa-remember"
          type="checkbox"
          className="size-4 rounded border-input accent-primary"
          checked={rememberDevice}
          onChange={(e) => setRememberDevice(e.target.checked)}
          disabled={verifyMutation.isPending}
        />
        Trust this device — skip verification here next time
      </label>

      <div className="flex items-center gap-2">
        <Button onClick={() => verifyMutation.mutate()} disabled={!canSubmit} className="flex-1">
          {verifyMutation.isPending
            ? (isWebAuthn ? "Waiting for device…" : "Verifying…")
            : (isWebAuthn ? "Use passkey" : "Verify")}
        </Button>
        <Button variant="ghost" onClick={onCancel} disabled={verifyMutation.isPending}>Cancel</Button>
      </div>
    </div>
  )
}
