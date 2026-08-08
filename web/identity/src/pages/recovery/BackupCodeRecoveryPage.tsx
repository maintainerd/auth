import { useState } from "react"
import { useNavigate, Link, useSearchParams } from "react-router-dom"
import { AlertCircle, KeyRound, ArrowLeft } from "lucide-react"
import { Button } from "@/components/ui/button"
import { FormCodeField, FormEmailField } from "@/components/inputs"
import LoginLayout from "@/components/layout/LoginLayout"
import { useAuth } from "@/hooks/useAuth"
import { useToast } from "@/hooks/useToast"
import { post } from "@/services/api/client"
import { API_ENDPOINTS } from "@/services/api/config"
import { isRateLimitError, rateLimitMessage } from "@/services/api/rateLimit"
import { resolvePublicAuthContext } from "@/utils/clientContext"

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** Heading + explanation shared by every terminal state on this page. */
function RecoveryNotice({
  tone,
  title,
  children,
}: {
  tone: 'error' | 'success'
  title: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-2 text-center">
      <div
        className={`mx-auto flex size-12 items-center justify-center rounded-full ${
          tone === 'error' ? 'bg-destructive/10' : 'bg-emerald-500/10'
        }`}
      >
        {tone === 'error'
          ? <AlertCircle className="size-6 text-destructive" />
          : <KeyRound className="size-6 text-emerald-600" />}
      </div>
      <h1 className="text-2xl font-bold">{title}</h1>
      <p className="text-muted-foreground">{children}</p>
    </div>
  )
}

export default function BackupCodeRecoveryPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { refreshAccount } = useAuth()
  const { showError } = useToast()

  const [email, setEmail] = useState(searchParams.get("email")?.trim() ?? "")
  const [code, setCode] = useState("")
  const [emailError, setEmailError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [recoveredWithoutSession, setRecoveredWithoutSession] = useState(false)
  const [throttled, setThrottled] = useState<string | null>(null)

  // user.VerifyBackupCodeDTO requires all four of email / code / client_id /
  // provider_id, each validation.Required. The page used to send
  // { backup_code, client_id, tenant_id }: three of the four names were wrong or
  // absent, so every submission failed validation before a code was ever
  // checked, and the two fields the user has to supply — their email address and
  // the provider the account lives on — were never collected at all.
  const clientId = resolvePublicAuthContext({
    clientId: searchParams.get("client_id")?.trim() || undefined,
  }).clientId
  const providerId = searchParams.get("provider_id")?.trim() || undefined

  const handleRecover = async () => {
    const trimmedEmail = email.trim()
    if (!EMAIL_PATTERN.test(trimmedEmail)) {
      setEmailError("Enter the email address on your account.")
      return
    }
    setEmailError(null)
    setThrottled(null)
    if (!code.trim() || !clientId || !providerId) return

    setLoading(true)
    try {
      await post(API_ENDPOINTS.AUTH.RECOVERY_BACKUP_CODE, {
        email: trimmedEmail,
        code: code.trim(),
        client_id: clientId,
        provider_id: providerId,
      })

      // The handler returns tokens in the response body rather than as cookies,
      // so a 200 does not by itself mean this browser has a session. Confirm one
      // exists before routing anywhere — navigating to "/" without it just
      // bounces back to login and reads as "my code didn't work".
      const account = await refreshAccount()
      if (!account) {
        setRecoveredWithoutSession(true)
        return
      }
      navigate("/", { replace: true })
    } catch (e: unknown) {
      // The recovery throttle answers 403, not 429, so without this the user is
      // told "You don't have permission to perform this action." for a code that
      // is probably fine — and a toast disappears long before the cooldown does.
      if (isRateLimitError(e)) {
        setThrottled(rateLimitMessage(e))
        return
      }
      showError(e, "Recovery failed")
    } finally {
      setLoading(false)
    }
  }

  const backToLogin = (
    <Link to="/login" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
      <ArrowLeft className="h-3 w-3" /> Back to login
    </Link>
  )

  // Without a client and provider the request cannot pass validation, so say so
  // up front instead of letting the user type a single-use backup code into a
  // form that is guaranteed to reject it.
  if (!clientId || !providerId) {
    return (
      <LoginLayout>
        <div className="space-y-6 w-full max-w-sm">
          {backToLogin}
          <RecoveryNotice tone="error" title="This recovery link is incomplete">
            Backup-code recovery has to know which application you are signing in to. Start
            again from the sign-in page, or ask your administrator for a new recovery link.
          </RecoveryNotice>
          <Button className="w-full" onClick={() => navigate("/login", { replace: true })}>
            Go to sign in
          </Button>
        </div>
      </LoginLayout>
    )
  }

  if (recoveredWithoutSession) {
    return (
      <LoginLayout>
        <div className="space-y-6 w-full max-w-sm">
          {backToLogin}
          <RecoveryNotice tone="success" title="Backup code accepted">
            Your code was accepted and has now been used, but we could not sign you in on this
            device. Sign in with your password to continue — you will not need this code again.
          </RecoveryNotice>
          <Button className="w-full" onClick={() => navigate("/login", { replace: true })}>
            Go to sign in
          </Button>
        </div>
      </LoginLayout>
    )
  }

  return (
    <LoginLayout>
      <div className="space-y-6 w-full max-w-sm">
        {backToLogin}

        <div className="space-y-2 text-center">
          <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-muted">
            <KeyRound className="size-6" />
          </div>
          <h1 className="text-2xl font-bold">Backup Code Recovery</h1>
          <p className="text-muted-foreground">
            Enter your email address and one of your saved backup codes to regain access to your account.
          </p>
        </div>

        <div className="space-y-3">
          {throttled && (
            <p
              role="alert"
              className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            >
              {throttled}
            </p>
          )}

          <FormEmailField
            label="Email address"
            value={email}
            onChange={(e) => { setEmail(e.target.value); setEmailError(null) }}
            error={emailError ?? undefined}
            autoComplete="username"
          />

          <FormCodeField
            label="Backup code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="Enter your backup code"
            autoComplete="off"
          />

          <Button className="w-full" onClick={handleRecover} disabled={loading || !code.trim()}>
            {loading ? "Verifying..." : "Recover Account"}
          </Button>
        </div>
      </div>
    </LoginLayout>
  )
}
