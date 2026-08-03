import { useState } from "react"
import { useNavigate, Link, useSearchParams } from "react-router-dom"
import { MessageSquare, ArrowLeft } from "lucide-react"
import { Button } from "@/components/ui/button"
import { FormCodeField, FormPhoneField } from "@/components/inputs"
import LoginLayout from "@/components/layout/LoginLayout"
import { useTenant } from "@/hooks/useTenant"
import { useToast } from "@/hooks/useToast"
import { post } from "@/services/api/client"
import { API_ENDPOINTS } from "@/services/api/config"

export default function SMSLoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { currentTenant } = useTenant()
  const { showError, showSuccess } = useToast()

  const [phone, setPhone] = useState("")
  const [code, setCode] = useState("")
  const [otpSent, setOtpSent] = useState(false)
  const [sending, setSending] = useState(false)
  const [verifying, setVerifying] = useState(false)

  // Set when the user was routed here from login/magic-link because the tenant
  // requires phone verification. A successful SMS code verifies the phone and
  // signs them in. Read once and clear so it doesn't persist across visits.
  const [phoneVerificationRequired] = useState(() => {
    const flagged = sessionStorage.getItem("phone_verification_required") === "1"
    if (flagged) sessionStorage.removeItem("phone_verification_required")
    return flagged
  })

  // Tenant comes from the domain bootstrap (its slug); client_id (OAuth) still
  // comes from the URL.
  const tenantId = currentTenant?.name
  const clientId = searchParams.get("client_id")

  const handleSendCode = async () => {
    if (!phone) return
    setSending(true)
    try {
      const body: Record<string, string> = { phone }
      if (tenantId) body.tenant_id = tenantId
      if (clientId) body.client_id = clientId
      await post(API_ENDPOINTS.AUTH.SMS_LOGIN_SEND, body)
      setOtpSent(true)
      showSuccess("Verification code sent")
    } catch (e: unknown) {
      showError(e, "Failed to send code")
    } finally {
      setSending(false)
    }
  }

  const handleVerify = async () => {
    if (!phone || !code) return
    setVerifying(true)
    try {
      const body: Record<string, string> = { phone, code }
      if (tenantId) body.tenant_id = tenantId
      if (clientId) body.client_id = clientId
      await post(API_ENDPOINTS.AUTH.SMS_LOGIN_VERIFY, body)
      showSuccess("Signed in successfully")
      navigate("/", { replace: true })
    } catch (e: unknown) {
      showError(e, "Verification failed")
    } finally {
      setVerifying(false)
    }
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="space-y-6 w-full max-w-sm">
        <Link to="/login" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="h-3 w-3" /> Back to login
        </Link>

        {phoneVerificationRequired && (
          <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-foreground">
            Your phone must be verified before you can sign in. Enter your number
            to receive a code — verifying it completes sign-in.
          </div>
        )}

        <div className="space-y-2 text-center">
          <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-muted">
            <MessageSquare className="size-6" />
          </div>
          <h1 className="text-2xl font-bold">Sign in with SMS</h1>
          <p className="text-muted-foreground">
            {otpSent
              ? `Enter the verification code sent to ${phone}`
              : "Enter your phone number to receive a one-time code"}
          </p>
        </div>

        <div className="space-y-3">
          <FormPhoneField
            label="Phone number"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="+1234567890"
            disabled={otpSent}
          />

          {otpSent && (
            <FormCodeField
              label="Verification code"
              numeric
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
              placeholder="000000"
              maxLength={6}
            />
          )}

          {!otpSent ? (
            <Button className="w-full" onClick={handleSendCode} disabled={sending || !phone}>
              {sending ? "Sending..." : "Send Code"}
            </Button>
          ) : (
            <Button className="w-full" onClick={handleVerify} disabled={verifying || code.length < 6}>
              {verifying ? "Verifying..." : "Verify & Sign In"}
            </Button>
          )}

          {otpSent && (
            <Button variant="link" className="w-full text-sm" onClick={handleSendCode} disabled={sending}>
              Resend code
            </Button>
          )}
        </div>
      </div>
    </LoginLayout>
  )
}
