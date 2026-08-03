import { useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { useForm } from "react-hook-form"
import * as yup from 'yup'
import { yupResolver } from "@hookform/resolvers/yup"
import { AlertCircle, Mail, Loader2, CheckCircle } from "lucide-react"
import { FieldGroup } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { FormSubmitButton } from "@/components/form"
import { FormCodeField } from "@/components/inputs"
import { post } from "@/services/api/client"
import { publicAuthQuery } from "@/utils/clientContext"
import LoginLayout from "@/components/layout/LoginLayout"
import { useTenant } from "@/hooks/useTenant"
import { useAuth } from "@/hooks/useAuth"
import { useToast } from "@/hooks/useToast"
import { getRequestId } from "@/utils/oauthRedirect"
import { finishAuthStep } from "@/utils/oauthContinuation"

const schema = yup.object({
  code: yup.string().required('Code is required').min(6).max(6),
})

type FormData = yup.InferType<typeof schema>

export default function VerifyEmailPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { currentTenant } = useTenant()
  const { isAuthenticated, refreshAccount } = useAuth()
  const { showSuccess, showError } = useToast()
  const [error, setError] = useState<string | null>(null)
  const [resending, setResending] = useState(false)
  const [verified, setVerified] = useState(false)

  const email = sessionStorage.getItem('register_email') || ''

  // Note: the already-verified / wrong-step redirect is handled centrally by
  // RouteGuard before this page renders.

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting }
  } = useForm<FormData>({
    resolver: yupResolver(schema),
    defaultValues: { code: "" },
  })

  const onSubmit = async (data: FormData) => {
    setError(null)
    try {
      await post(`/email-verification/verify?${publicAuthQuery()}`, { email, otp: data.code })

      // Active sign-up session: apply the shared continuation rule — continue the
      // pending OAuth authorize (request_id) once fully registered, else route to
      // the next detour step (profile) threading the handle. Otherwise (verified
      // from a login redirect with no session yet) show the success screen and
      // send them to sign in.
      if (isAuthenticated) {
        sessionStorage.removeItem('register_email')
        const fresh = await refreshAccount()
        finishAuthStep({
          account: fresh,
          tenant: currentTenant,
          requestId: getRequestId(searchParams),
          navigate,
        })
        return
      }
      sessionStorage.removeItem('register_email')
      setVerified(true)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Verification failed'
      setError(msg)
    }
  }

  const handleResend = async () => {
    setResending(true)
    setError(null)
    try {
      await post(`/email-verification/send?${publicAuthQuery()}`, { email })
      showSuccess('A new verification code has been sent to your email.')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Could not resend the verification code. Please try again.'
      showError(msg)
      setError(msg)
    } finally {
      setResending(false)
    }
  }

  if (verified) {
    return (
      <LoginLayout branding={currentTenant?.branding}>
        <div className="flex flex-col gap-8 text-center">
          <div className="flex flex-col items-center gap-3">
            <div className="flex size-14 items-center justify-center rounded-full bg-emerald-500/10">
              <CheckCircle className="size-7 text-emerald-600" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight">Email verified</h1>
            <p className="text-sm text-muted-foreground max-w-xs">
              Your email has been confirmed. You can now sign in to your account.
            </p>
          </div>
          <Button className="w-full" onClick={() => navigate('/login', { replace: true })}>
            Sign in
          </Button>
        </div>
      </LoginLayout>
    )
  }

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8">
        <div className="flex flex-col items-center gap-3 text-center">
          <div className="flex size-14 items-center justify-center rounded-full bg-primary/10">
            <Mail className="size-7 text-primary" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Verify your email</h1>
          <p className="text-sm text-muted-foreground">
            Enter the code sent to <strong>{email || 'your email'}</strong>.
          </p>
        </div>

        {error && (
          <div role="alert" className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
            <AlertCircle className="mt-0.5 size-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit(onSubmit)}>
          <FieldGroup>
            <FormCodeField
              label="Verification code"
              placeholder="000000"
              numeric
              disabled={isSubmitting}
              error={errors.code?.message}
              required
              {...register("code")}
            />
            <FormSubmitButton
              isSubmitting={isSubmitting}
              submitText="Verify email"
              submittingText="Verifying..."
              className="w-full"
            />
          </FieldGroup>
        </form>

        <div className="flex flex-col items-center gap-1">
          <Button variant="link" size="sm" onClick={handleResend} disabled={resending} className="text-muted-foreground">
            {resending ? <Loader2 className="mr-1 size-3 animate-spin" /> : null}
            Didn't receive a code? Resend
          </Button>
          <Button
            type="button"
            variant="link"
            size="sm"
            onClick={() => navigate('/login')}
            className="text-muted-foreground"
          >
            Back to login
          </Button>
        </div>
      </div>
    </LoginLayout>
  )
}
