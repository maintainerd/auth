import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import QRCode from "qrcode"
import { Smartphone, ShieldCheck, RefreshCw, Copy, Check, Trash2, Download, EyeOff } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { FormCodeField } from "@/components/inputs"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { ListingItemIcon } from "@/components/details/ListingItemCard"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import { useToast } from "@/hooks/useToast"
import {
  beginTOTPEnrollment, finishTOTPEnrollment, disableTOTP, regenerateBackupCodes, fetchMFAStatus,
} from "@/services/api/mfa"
import { MFASetupShell, ConfirmRemoveDialog, MfaSetupSkeleton, MFA_HUB_ROUTE } from "./MfaShell"

/**
 * Renders the enrolment QR from the otpauth:// URI the server returns.
 *
 * The URI is a DEEP LINK, not an image: `qr_code_url` carries
 * `otpauth://totp/...`, which a browser cannot load as an <img> source — it
 * silently renders a broken image, leaving manual key entry as the only way to
 * enrol. The QR has to be drawn client-side from that URI, which is what the
 * console has always done.
 */
function QRCodeImage({ url }: { url: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!canvasRef.current || !url) return
    setFailed(false)
    QRCode.toCanvas(canvasRef.current, url, { width: 200, margin: 2 }, (err) => {
      if (err) {
        // Say so rather than leaving a blank square: the manual key below still
        // works, and a user staring at nothing has no way to know that.
        console.warn("QR render failed:", err)
        setFailed(true)
      }
    })
  }, [url])

  if (failed) {
    return (
      <p className="text-sm text-muted-foreground">
        Could not draw the QR code — use the manual entry key below.
      </p>
    )
  }
  return <canvas ref={canvasRef} aria-label="Authenticator QR code" role="img" />
}

export default function TOTPSetupPage() {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({ queryKey: ["mfa", "status"], queryFn: fetchMFAStatus, retry: false })
  const enabled = data?.is_totp_enabled ?? false
  const backupCount = data?.backup_codes_count ?? 0

  const [step, setStep] = useState<"idle" | "scan" | "verify">("idle")
  const [secret, setSecret] = useState("")
  const [qrUrl, setQrUrl] = useState("")
  // Sized from the server, never assumed. A tenant on totp_digits=8 gets an
  // authenticator showing 8 digits; hardcoding 6 caps the input below the code
  // length and makes enrolment impossible to complete.
  const [digits, setDigits] = useState(6)
  const [code, setCode] = useState("")
  const [backupCodes, setBackupCodes] = useState<string[]>([])
  // Codes start hidden. They are shown on a page the user may have open in a
  // shared or screen-shared window, and unlike a password they cannot be
  // re-displayed later — so revealing is a deliberate act, matching the console.
  const [codesRevealed, setCodesRevealed] = useState(false)
  const [showCodes, setShowCodes] = useState(false)
  const [returnToHub, setReturnToHub] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showDisable, setShowDisable] = useState(false)

  const enrollMutation = useMutation({
    mutationFn: beginTOTPEnrollment,
    onSuccess: (res) => {
      setSecret(res.secret)
      setQrUrl(res.qr_code_url)
      setDigits(res.digits ?? 6)
      setStep("scan")
    },
    onError: (e) => showError(e),
  })

  const verifyMutation = useMutation({
    mutationFn: (c: string) => finishTOTPEnrollment({ code: c }),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
      if (res.codes?.length) {
        setBackupCodes(res.codes)
        setCodesRevealed(false)
        setReturnToHub(true)
        setShowCodes(true)
      } else {
        showSuccess("Authenticator app enabled")
        navigate(MFA_HUB_ROUTE)
      }
    },
    onError: (e) => showError(e),
  })

  const disableMutation = useMutation({
    mutationFn: disableTOTP,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
      showSuccess("Authenticator app removed")
      navigate(MFA_HUB_ROUTE)
    },
    onError: (e) => showError(e),
  })

  const regenerateMutation = useMutation({
    mutationFn: regenerateBackupCodes,
    onSuccess: (res) => {
      setBackupCodes(res.codes)
      // Regenerated codes are just as sensitive as the first set.
      setCodesRevealed(false)
      setReturnToHub(false)
      setShowCodes(true)
      queryClient.invalidateQueries({ queryKey: ["mfa", "status"] })
    },
    onError: (e) => showError(e),
  })

  const copyCodes = async () => {
    try {
      await navigator.clipboard.writeText(backupCodes.join("\n"))
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      showError(new Error("Couldn't copy codes to clipboard"))
    }
  }

  const downloadCodes = () => {
    const blob = new Blob([backupCodes.join("\n")], { type: "text/plain" })
    const href = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = href
    a.download = "backup-codes.txt"
    a.click()
    // Release the blob: without this the codes stay reachable through a live
    // object URL for the lifetime of the document.
    URL.revokeObjectURL(href)
  }

  const closeCodes = (open: boolean) => {
    setShowCodes(open)
    if (!open && returnToHub) {
      showSuccess("Authenticator app enabled")
      navigate(MFA_HUB_ROUTE)
    }
  }

  const backupCodesDialog = (
    <Dialog open={showCodes} onOpenChange={closeCodes}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Save your backup codes</DialogTitle>
          <DialogDescription>
            Store these somewhere safe. Each code works once, and you won&apos;t be able to see them again.
          </DialogDescription>
        </DialogHeader>
        {!codesRevealed ? (
          <div className="flex flex-col items-center gap-4 py-6">
            <EyeOff className="size-12 text-muted-foreground" />
            <p className="text-center text-sm text-muted-foreground">
              Backup codes are hidden for security. Reveal them only when you&apos;re ready to save them.
            </p>
            <Button onClick={() => setCodesRevealed(true)}>Reveal codes</Button>
          </div>
        ) : (
          <div className="grid max-h-48 grid-cols-2 gap-2 overflow-y-auto rounded-lg bg-muted p-4 font-mono text-sm">
            {backupCodes.map((c, i) => <span key={i}>{c}</span>)}
          </div>
        )}
        {codesRevealed && (
          <DialogFooter className="flex-col gap-2 sm:flex-row sm:justify-between">
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={copyCodes}>
                {copied ? <Check className="mr-2 size-4" /> : <Copy className="mr-2 size-4" />}
                {copied ? "Copied!" : "Copy all"}
              </Button>
              <Button variant="outline" size="sm" onClick={downloadCodes}>
                <Download className="mr-2 size-4" />Download
              </Button>
            </div>
            <Button size="sm" onClick={() => closeCodes(false)}>Done</Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  )

  if (isLoading) {
    return <MFASetupShell><MfaSetupSkeleton /></MFASetupShell>
  }

  // ── Manage (already enabled) ──────────────────────────────────────────────
  if (enabled) {
    return (
      <MFASetupShell>
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <div className="flex items-center gap-3">
                <ListingItemIcon className="size-10 text-emerald-600">
                  <ShieldCheck className="size-5" />
                </ListingItemIcon>
                <div>
                  <CardTitle className="text-base">Authenticator app is active</CardTitle>
                  <CardDescription>Your authenticator app generates sign-in codes for this account.</CardDescription>
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
                <Trash2 className="mr-2 size-4" /> Remove authenticator app
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <ListingItemIcon className="size-10">
                    <RefreshCw className="size-5 text-muted-foreground" />
                  </ListingItemIcon>
                  <div>
                    <CardTitle className="text-base">Backup codes</CardTitle>
                    <CardDescription>
                      {backupCount > 0 ? `${backupCount} unused code${backupCount !== 1 ? "s" : ""} remaining` : "No backup codes available"}
                    </CardDescription>
                  </div>
                </div>
                <Button variant="outline" size="sm" onClick={() => regenerateMutation.mutate()} disabled={regenerateMutation.isPending}>
                  {regenerateMutation.isPending ? "Generating…" : "Regenerate"}
                </Button>
              </div>
            </CardHeader>
          </Card>
        </div>

        <ConfirmRemoveDialog
          open={showDisable}
          onOpenChange={setShowDisable}
          onConfirm={() => disableMutation.mutate()}
          title="Remove authenticator app"
          description="You'll no longer be asked for a code from your authenticator app. Your backup codes will also stop working."
          isLoading={disableMutation.isPending}
        />
        {backupCodesDialog}
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
                <Smartphone className="size-5 text-muted-foreground" />
              </ListingItemIcon>
              <div>
                <CardTitle className="text-base">Set up authenticator app</CardTitle>
                <CardDescription>Works with Google Authenticator, Authy, 1Password, or any TOTP app.</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <ol className="list-decimal space-y-2 pl-5 text-sm text-muted-foreground">
              <li>Install an authenticator app on your phone if you don&apos;t have one.</li>
              <li>Scan the QR code we&apos;ll show you on the next step.</li>
              <li>Enter the {digits}-digit code from the app to confirm.</li>
            </ol>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => navigate(MFA_HUB_ROUTE)}>Cancel</Button>
              <Button onClick={() => enrollMutation.mutate()} disabled={enrollMutation.isPending}>
                {enrollMutation.isPending ? "Preparing…" : "Begin setup"}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {step === "scan" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Scan the QR code</CardTitle>
            <CardDescription>Open your authenticator app and scan this code, or enter the key manually.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex justify-center">
              <div className="rounded-lg border bg-white p-4">
                <QRCodeImage url={qrUrl} />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Manual entry key</Label>
              <div className="select-all break-all rounded-md bg-muted px-3 py-2 font-mono text-sm">{secret}</div>
            </div>
            <div className="flex justify-between gap-2">
              <Button variant="ghost" onClick={() => setStep("idle")}>Back</Button>
              <Button onClick={() => setStep("verify")}>I&apos;ve scanned the code</Button>
            </div>
          </CardContent>
        </Card>
      )}

      {step === "verify" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Enter verification code</CardTitle>
          <CardDescription>Enter the {digits}-digit code from your authenticator app.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormCodeField
              id="totp-code"
              label="Verification code"
              numeric
              placeholder="000000"
              maxLength={digits}
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
            />
            <div className="flex justify-between gap-2">
              <Button variant="ghost" onClick={() => { setStep("scan"); setCode("") }}>Back</Button>
              <Button disabled={code.length !== digits || verifyMutation.isPending} onClick={() => verifyMutation.mutate(code)}>
                {verifyMutation.isPending ? "Verifying…" : "Verify & enable"}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {backupCodesDialog}
    </MFASetupShell>
  )
}
