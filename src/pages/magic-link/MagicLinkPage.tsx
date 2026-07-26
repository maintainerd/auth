import { useCallback, useEffect, useState, useRef } from "react"
import { useSearchParams, useNavigate } from "react-router-dom"
import { Loader2, CheckCircle2, AlertCircle, ArrowRight, ShieldCheck } from "lucide-react"
import { Button } from "@/components/ui/button"
import { verifyMagicLink, fetchAccount } from "@/services/api/auth"
import { useTenant } from "@/hooks/useTenant"
import LoginLayout from "@/components/layout/LoginLayout"
import { LoginMFAStep } from "@/pages/login/components/LoginMFAStep"
import type { AccountEntity } from "@/services/api/auth/types"

export default function MagicLinkPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { currentTenant } = useTenant()
  const [status, setStatus] = useState<"verifying" | "mfa" | "success" | "error">("verifying")
  const [errorMessage, setErrorMessage] = useState("")
  const [successPath, setSuccessPath] = useState("")
  const [mfaChallenge, setMfaChallenge] = useState<{ token: string; methods: string[] } | null>(null)
  const verifiedRef = useRef(false)

  const finishAuthentication = useCallback((account: AccountEntity | null | undefined) => {
    const tenantSlug = account?.tenant?.name
    if (!tenantSlug) {
      setStatus("error")
      setErrorMessage("The sign-in session could not be established. Please request a new magic link.")
      return
    }
    setSuccessPath("/login-success")
    setStatus("success")
  }, [])

  useEffect(() => {
    if (verifiedRef.current) return
    const hasSingleAuthContext = searchParams.has("client_id") !== searchParams.has("tenant_id")
    const hasSignedLink = searchParams.has("expires") && searchParams.has("sig")

    if (!searchParams.get("token") || !hasSingleAuthContext || !hasSignedLink) {
      setStatus("error")
      setErrorMessage("This magic link is invalid or incomplete. Please request a new one.")
      return
    }

    verifiedRef.current = true

    async function verify() {
      try {
        const response = await verifyMagicLink(searchParams.toString())
        if (response.data?.mfa_required) {
          const challengeToken = response.data.mfa_challenge_token
          if (!challengeToken) {
            throw new Error("The MFA challenge could not be started. Please request a new magic link.")
          }
          setMfaChallenge({
            token: challengeToken,
            // Defensive filtering: the backend already excludes this redundant
            // method, but the UI must never offer the same mailbox twice.
            methods: (response.data.mfa_allowed_methods ?? []).filter((method) => method !== "email_otp"),
          })
          setStatus("mfa")
          return
        }
        const account = await fetchAccount()
        finishAuthentication(account)
      } catch (err) {
        const msg = err instanceof Error ? err.message : ""
        // Magic-link proves email but not phone. When the tenant requires phone
        // verification, route to SMS login (the verification path) instead of a
        // dead-end error, preserving the auth context.
        if (msg === "phone is not verified") {
          sessionStorage.setItem("phone_verification_required", "1")
          navigate({ pathname: "/sms-login", search: searchParams.toString() }, { replace: true })
          return
        }
        setStatus("error")
        setErrorMessage(
          msg || "The magic link is invalid or has expired. Please request a new one."
        )
      }
    }

    verify()
  }, [finishAuthentication, searchParams, navigate])

  useEffect(() => {
    if (status !== "success" || !successPath) return

    const timer = window.setTimeout(() => {
      window.location.assign(successPath)
    }, 3000)

    return () => window.clearTimeout(timer)
  }, [successPath, status])

  return (
    <LoginLayout branding={currentTenant?.branding}>
      {status === "verifying" && (
        <div className="flex flex-col gap-8 text-center" role="status" aria-live="polite">
          <div className="flex flex-col items-center gap-3">
            <div className="flex size-14 items-center justify-center rounded-full bg-blue-100">
              <Loader2 className="size-7 animate-spin text-blue-600" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight">Signing you in</h1>
            <p className="max-w-xs text-sm text-muted-foreground">
              We&apos;re securely verifying your magic link.
            </p>
          </div>
        </div>
      )}

      {status === "success" && (
        <div className="flex flex-col gap-8 text-center" role="status" aria-live="polite">
          <div className="flex flex-col items-center gap-3">
            <div className="flex size-14 items-center justify-center rounded-full bg-green-100">
              <CheckCircle2 className="size-7 text-green-600" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight">You&apos;re signed in</h1>
            <p className="max-w-xs text-sm text-muted-foreground">
              Your magic link was verified successfully. We&apos;re taking you to the app.
            </p>
          </div>

          <Button className="w-full" onClick={() => window.location.assign(successPath)}>
            Continue
            <ArrowRight className="ml-2 size-4" />
          </Button>

          <p className="text-xs text-muted-foreground">
            Redirecting automatically in a few seconds...
          </p>
        </div>
      )}

      {status === "mfa" && mfaChallenge && (
        <div className="flex flex-col gap-8">
          <div className="flex flex-col items-center gap-3 text-center">
            <div className="flex size-14 items-center justify-center rounded-full bg-blue-100">
              <ShieldCheck className="size-7 text-blue-600" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight">Two-step verification</h1>
            <p className="max-w-xs text-sm text-muted-foreground">
              Your magic link was accepted. Confirm a different factor to finish signing in.
            </p>
          </div>

          <LoginMFAStep
            challengeToken={mfaChallenge.token}
            allowedMethods={mfaChallenge.methods}
            clientId={searchParams.get("client_id") || undefined}
            tenantId={searchParams.get("tenant_id") || undefined}
            onVerified={(result) => finishAuthentication(result.account)}
            onCancel={() => navigate("/login", { replace: true })}
          />
        </div>
      )}

      {status === "error" && (
        <div className="flex flex-col gap-8 text-center" role="alert">
          <div className="flex flex-col items-center gap-3">
            <div className="flex size-14 items-center justify-center rounded-full bg-destructive/10">
              <AlertCircle className="size-7 text-destructive" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight">Magic link unavailable</h1>
            <p className="max-w-xs text-sm text-muted-foreground">{errorMessage}</p>
          </div>

          <Button className="w-full" onClick={() => navigate("/login", { replace: true })}>
            Back to sign in
          </Button>
        </div>
      )}
    </LoginLayout>
  )
}
