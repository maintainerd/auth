import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Mail, ShieldCheck, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { useAuth } from "@/hooks/useAuth"
import { useToast } from "@/hooks/useToast"
import { beginEmailOtpEnrollment, verifyEmailOtpEnrollment, disableEmailOtp, fetchMFAStatus } from "@/services/api/mfa"
import { MFASetupShell, ConfirmRemoveDialog, MfaSetupSkeleton, MFA_HUB_ROUTE } from "./MfaShell"

export default function EmailOtpSetupPage() {
  const navigate = useNavigate()
  const { account } = useAuth()
  const { showSuccess, showError } = useToast()
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({ queryKey: ["mfa", "status"], queryFn: fetchMFAStatus, retry: false })
  const enabled = data?.is_email_otp_available ?? false

  const [step, setStep] = useState<"idle" | "verify">("idle")
  const [email, setEmail] = useState(account?.email ?? "")
  const [code, setCode] = useState("")
  const [showDisable, setShowDisable] = useState(false)

  const sendMutation = useMutation({
    mutationFn: (e: string) => beginEmailOtpEnrollment(e),
    onSuccess: () => { setStep("verify"); showSuccess("Verification code sent to your email") },
    onError: (e) => showError(e),
  })

  const verifyMutation = useMutation({
    mutationFn: (vars: { email: string; code: string }) => verifyEmailOtpEnrollment(vars.email, vars.code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
      showSuccess("Email OTP authentication enabled")
      navigate(MFA_HUB_ROUTE)
    },
    onError: (e) => showError(e),
  })

  const disableMutation = useMutation({
    mutationFn: disableEmailOtp,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
      showSuccess("Email OTP authentication removed")
      navigate(MFA_HUB_ROUTE)
    },
    onError: (e) => showError(e),
  })

  if (isLoading) {
    return <MFASetupShell><MfaSetupSkeleton /></MFASetupShell>
  }

  // ── Manage (already enabled) ──────────────────────────────────────────────
  if (enabled) {
    return (
      <MFASetupShell>
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div data-md-listing-icon className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-emerald-600">
                <ShieldCheck className="size-5" />
              </div>
              <div>
                <CardTitle className="text-base">Email OTP authentication is active</CardTitle>
                <CardDescription>One-time codes are sent to your verified email at sign-in.</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <Button
              variant="outline"
              className="text-destructive hover:text-destructive"
              onClick={() => setShowDisable(true)}
              disabled={disableMutation.isPending}
            >
              <Trash2 className="mr-2 size-4" /> Remove Email OTP authentication
            </Button>
          </CardContent>
        </Card>

        <ConfirmRemoveDialog
          open={showDisable}
          onOpenChange={setShowDisable}
          onConfirm={() => disableMutation.mutate()}
          title="Remove Email OTP authentication"
          description="You'll no longer receive sign-in codes by email. You can set it up again at any time."
          isLoading={disableMutation.isPending}
        />
      </MFASetupShell>
    )
  }

  // ── Setup wizard ──────────────────────────────────────────────────────────
  return (
    <MFASetupShell>
      {step === "idle" && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div data-md-listing-icon className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                <Mail className="size-5 text-muted-foreground" />
              </div>
              <div>
                <CardTitle className="text-base">Set up email OTP authentication</CardTitle>
                <CardDescription>Verify your email address to receive sign-in codes by email.</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email address</Label>
              <Input id="email" type="email" placeholder="you@example.com" value={email} onChange={(e) => setEmail(e.target.value)} />
              <p className="text-xs text-muted-foreground">We&apos;ll send a 6-digit verification code to this address.</p>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => navigate(MFA_HUB_ROUTE)}>Cancel</Button>
              <Button onClick={() => sendMutation.mutate(email.trim())} disabled={!email.trim() || sendMutation.isPending}>
                <Mail className="mr-2 size-4" />
                {sendMutation.isPending ? "Sending code…" : "Send verification code"}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {step === "verify" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Enter verification code</CardTitle>
            <CardDescription>Enter the 6-digit code sent to {email}.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email-otp-code">Verification code</Label>
              <Input
                id="email-otp-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                placeholder="000000"
                maxLength={6}
                className="text-center font-mono tracking-[0.5em]"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
              />
            </div>
            <div className="flex justify-between gap-2">
              <Button variant="ghost" onClick={() => { setStep("idle"); setCode("") }}>Back</Button>
              <Button disabled={code.length !== 6 || verifyMutation.isPending} onClick={() => verifyMutation.mutate({ email, code })}>
                {verifyMutation.isPending ? "Verifying…" : "Verify & enable"}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </MFASetupShell>
  )
}
