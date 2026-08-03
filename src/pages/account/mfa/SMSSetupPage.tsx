import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { MessageSquare, Phone, ShieldCheck, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { ListingItemIcon } from "@/components/details/ListingItemCard"
import { useToast } from "@/hooks/useToast"
import { beginSMSEnrollment, verifySMSEnrollment, disableSMS, fetchMFAStatus } from "@/services/api/mfa"
import { MFASetupShell, ConfirmRemoveDialog, MfaSetupSkeleton, MFA_HUB_ROUTE } from "./MfaShell"

export default function SMSSetupPage() {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({ queryKey: ["mfa", "status"], queryFn: fetchMFAStatus, retry: false })
  const enabled = data?.is_sms_available ?? false

  const [step, setStep] = useState<"idle" | "verify">("idle")
  const [phone, setPhone] = useState("")
  const [code, setCode] = useState("")
  const [showDisable, setShowDisable] = useState(false)

  const sendMutation = useMutation({
    mutationFn: (p: string) => beginSMSEnrollment(p),
    onSuccess: () => { setStep("verify"); showSuccess("Verification code sent to your phone") },
    onError: (e) => showError(e),
  })

  const verifyMutation = useMutation({
    mutationFn: (vars: { phone: string; code: string }) => verifySMSEnrollment(vars.phone, vars.code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
      showSuccess("SMS authentication enabled")
      navigate(MFA_HUB_ROUTE)
    },
    onError: (e) => showError(e),
  })

  const disableMutation = useMutation({
    mutationFn: disableSMS,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
      showSuccess("SMS authentication removed")
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
              <ListingItemIcon className="size-10 text-emerald-600">
                <ShieldCheck className="size-5" />
              </ListingItemIcon>
              <div>
                <CardTitle className="text-base">SMS authentication is active</CardTitle>
                <CardDescription>One-time codes are sent to your verified phone at sign-in.</CardDescription>
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
              <Trash2 className="mr-2 size-4" /> Remove SMS authentication
            </Button>
          </CardContent>
        </Card>

        <ConfirmRemoveDialog
          open={showDisable}
          onOpenChange={setShowDisable}
          onConfirm={() => disableMutation.mutate()}
          title="Remove SMS authentication"
          description="You'll no longer receive sign-in codes by text message. You can set it up again at any time."
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
              <ListingItemIcon className="size-10">
                <MessageSquare className="size-5 text-muted-foreground" />
              </ListingItemIcon>
              <div>
                <CardTitle className="text-base">Set up text message authentication</CardTitle>
                <CardDescription>Verify your phone number to receive sign-in codes by SMS.</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="phone">Phone number</Label>
              <Input id="phone" type="tel" placeholder="+1 555 123 4567" value={phone} onChange={(e) => setPhone(e.target.value)} />
              <p className="text-xs text-muted-foreground">We&apos;ll send a 6-digit verification code to this number.</p>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => navigate(MFA_HUB_ROUTE)}>Cancel</Button>
              <Button onClick={() => sendMutation.mutate(phone.trim())} disabled={!phone.trim() || sendMutation.isPending}>
                <Phone className="mr-2 size-4" />
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
            <CardDescription>Enter the 6-digit code sent to {phone}.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="sms-code">Verification code</Label>
              <Input
                id="sms-code"
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
              <Button disabled={code.length !== 6 || verifyMutation.isPending} onClick={() => verifyMutation.mutate({ phone, code })}>
                {verifyMutation.isPending ? "Verifying…" : "Verify & enable"}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </MFASetupShell>
  )
}
