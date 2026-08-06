import { useEffect, useMemo, useState } from "react"
import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { useNavigate, useSearchParams, Link } from "react-router-dom"
import { AlertCircle, Loader2, Mail } from "lucide-react"
import { FormSubmitButton, FormPasswordField, PasswordRequirements, FormConsentCheckbox } from "@/components/form"
import { Field, FieldLabel } from "@/components/ui/field"
import { FieldGroup } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { useAuth } from "@/hooks/useAuth"
import { useTenant } from "@/hooks/useTenant"
import { useToast } from "@/hooks/useToast"
import { resolvePostAuthRoute, loginSuccessRoute } from "@/utils/postAuthRoute"
import { rememberOAuthReturnTo, clearOAuthReturnTo, rememberInviteCallback } from "@/utils/oauthRedirect"
import { fetchInviteContext } from "@/services/api/auth"
import * as yup from "yup"
import { buildPasswordValidation, acceptTermsValidation } from "@/lib/validations/authSchema"
import type { PasswordConfigPublic } from "@/services/api/tenants/types"

interface InviteFormData {
  password: string
  confirmPassword: string
  acceptTerms: boolean
}

function buildInviteSchema(cfg?: PasswordConfigPublic) {
  return yup.object({
    // Reuse the shared tenant password policy so invite registration enforces
    // exactly the same rules as standard registration and reset-password.
    password: buildPasswordValidation(cfg),
    confirmPassword: yup
      .string()
      .required('Please confirm your password')
      .oneOf([yup.ref('password')], 'Passwords must match'),
    acceptTerms: acceptTermsValidation(),
  })
}

const RegisterInviteForm = () => {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { registerInvite, refreshAccount, isAuthenticated, account, logout } = useAuth()
  const { getCurrentTenant } = useTenant()
  const { showSuccess } = useToast()
  const [registerError, setRegisterError] = useState<string | null>(null)
  const [signingOut, setSigningOut] = useState(false)

  const urlEmail = searchParams.get('email') || ''
  const inviteToken = searchParams.get('invite_token') || ''

  // The invite is checked BEFORE the form renders. The probe used to run only
  // when callback_url was missing, kept nothing but callback_url, and dropped
  // every error — so a revoked or expired invite (410 Gone) looked exactly like
  // a valid one until the user had filled in the form, chosen a password and
  // pressed submit. Verifying up front costs one request and turns that dead end
  // into something the user can act on.
  const [inviteCheck, setInviteCheck] = useState<
    { state: 'checking' } | { state: 'valid'; email?: string; callbackUrl?: string | null } | { state: 'invalid'; message: string }
  >(inviteToken ? { state: 'checking' } : { state: 'valid' })

  useEffect(() => {
    if (!inviteToken) {
      setInviteCheck({ state: 'valid' })
      return
    }
    let cancelled = false
    setInviteCheck({ state: 'checking' })
    fetchInviteContext(inviteToken)
      .then((ctx) => {
        if (cancelled) return
        setInviteCheck({ state: 'valid', email: ctx.email, callbackUrl: ctx.callback_url })
      })
      .catch((error: unknown) => {
        if (cancelled) return
        setInviteCheck({
          state: 'invalid',
          message: error instanceof Error && error.message
            ? error.message
            : 'This invitation is no longer valid.',
        })
      })
    return () => { cancelled = true }
  }, [inviteToken])

  // The signed URL's email is authoritative — it is covered by the signature —
  // but the probe's copy is a valid fallback for a link that omitted it, which
  // previously dead-ended on "missing the email parameter".
  const invitedEmail = urlEmail || (inviteCheck.state === 'valid' ? inviteCheck.email ?? '' : '')

  // The post-registration callback comes ONLY from the invite-context probe. The
  // backend validated that value against the inviting client's registered
  // redirect URIs before storing the invite; the link's own `callback_url`
  // parameter carries no guarantee this app can check, because the signature
  // covering it is only ever verified server-side. Preferring the parameter — as
  // this did — meant a rewritten invite link could bounce the newly registered
  // user to an attacker origin, launched from the tenant's own identity domain
  // and from a page they had every reason to trust. No probe value means no
  // external redirect at all.
  const inviteCallback = inviteCheck.state === 'valid' ? inviteCheck.callbackUrl ?? null : null

  const passwordConfig = getCurrentTenant()?.password_config
  const inviteSchema = useMemo(() => buildInviteSchema(passwordConfig), [passwordConfig])

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting }
  } = useForm<InviteFormData>({
    resolver: yupResolver(inviteSchema),
    defaultValues: {
      password: "",
      confirmPassword: "",
      acceptTerms: false
    },
    mode: 'onSubmit',
    reValidateMode: 'onSubmit'
  })

  const passwordValue = watch("password") || ""

  const [passwordTyped, setPasswordTyped] = useState(false)
  useEffect(() => {
    if (passwordValue.length > 0) setPasswordTyped(true)
  }, [passwordValue])

  const onSubmit = async (data: InviteFormData) => {
    setRegisterError(null)
    try {
      await registerInvite(invitedEmail, data.password)

      if (invitedEmail) {
        sessionStorage.setItem('register_email', invitedEmail)
      }
      showSuccess('Account created successfully!')

      // Store the guard's own output, never the caller's string, so the value a
      // registration detour later resumes is byte-for-byte the one that passed
      // the check. A rejected callback is simply not remembered.
      const safeCallback = rememberInviteCallback(inviteCallback)

      const account = await refreshAccount()
      const dest = resolvePostAuthRoute(account, getCurrentTenant())
      if (dest === loginSuccessRoute() && safeCallback) {
        window.location.assign(safeCallback)
        return
      }
      const oauthReturnTo = dest === loginSuccessRoute()
        ? rememberOAuthReturnTo(searchParams.get('return_to'))
        : null
      if (dest === loginSuccessRoute() && !oauthReturnTo) {
        clearOAuthReturnTo()
      }
      navigate(oauthReturnTo || dest, { replace: true })
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : "Registration failed. Please try again."
      setRegisterError(errorMessage)
    }
  }

  if (inviteCheck.state === 'checking') {
    return (
      <div className="flex flex-col items-center gap-3 py-10 text-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
        <p className="text-sm text-muted-foreground">Checking your invitation…</p>
      </div>
    )
  }

  if (inviteCheck.state === 'invalid') {
    return (
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-destructive/10">
            <AlertCircle className="size-7 text-destructive" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">This invitation can't be used</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            {inviteCheck.message} Ask whoever invited you to send a new invitation.
          </p>
        </div>

        <Button asChild className="w-full">
          <Link to="/login">Back to sign in</Link>
        </Button>
      </div>
    )
  }

  // An invite is addressed to a specific email. If a different account is already
  // signed in on this browser, accepting the invite would silently swap sessions —
  // so make the user sign out first and offer a one-click way to do it.
  const mismatchedSession = isAuthenticated && !!account && account.email !== invitedEmail

  if (mismatchedSession && invitedEmail) {
    const handleSignOut = async () => {
      setSigningOut(true)
      try {
        await logout()
      } catch {
        setSigningOut(false)
      }
    }
    return (
      <div className="flex flex-col gap-6 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-primary/10">
            <Mail className="size-7 text-primary" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Accept your invitation</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            You're signed in as <span className="font-medium text-foreground">{account?.email}</span>.
            This invitation is for <span className="font-medium text-foreground">{invitedEmail}</span>.
            Sign out to accept it.
          </p>
        </div>

        <Button className="w-full" onClick={handleSignOut} disabled={signingOut}>
          {signingOut ? 'Signing out…' : 'Sign out to continue'}
        </Button>
      </div>
    )
  }

  if (!invitedEmail) {
    return (
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-destructive/10">
            <AlertCircle className="size-7 text-destructive" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Invalid invite link</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            This invite link is missing the email parameter. Please request a new invitation.
          </p>
        </div>

        <Button asChild className="w-full">
          <Link to="/login">Back to sign in</Link>
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col items-center gap-1 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Accept your invitation</h1>
        <p className="text-sm text-muted-foreground">
          Set up your password to complete registration.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        <FieldGroup>
          {registerError && (
            <div
              role="alert"
              className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"
            >
              <AlertCircle className="mt-0.5 size-4 shrink-0" />
              <span>{registerError}</span>
            </div>
          )}

          {/* Read-only email: uses the shared Field scaffold so its label sits
              the same distance above its control as every other field. */}
          <Field>
            <FieldLabel>Email</FieldLabel>
            <div className="flex h-9 items-center gap-2 rounded-md border border-input bg-muted/50 px-3 text-sm text-muted-foreground">
              <Mail className="size-4 shrink-0" />
              <span>{invitedEmail}</span>
            </div>
          </Field>

          <FormPasswordField
            label="Password"
            placeholder="Enter a strong password"
            autoComplete="new-password"
            disabled={isSubmitting}
            error={errors.password?.message}
            required
            footer={
              passwordTyped ? (
                <PasswordRequirements password={passwordValue} config={passwordConfig} />
              ) : undefined
            }
            {...register("password")}
          />
          <FormPasswordField
            label="Confirm password"
            placeholder="Re-enter your password"
            autoComplete="new-password"
            disabled={isSubmitting}
            error={errors.confirmPassword?.message}
            required
            {...register("confirmPassword")}
          />
          <FormConsentCheckbox
            error={errors.acceptTerms?.message}
            termsUrl={getCurrentTenant()?.branding?.terms_of_service_url}
            privacyUrl={getCurrentTenant()?.branding?.privacy_policy_url}
            {...register("acceptTerms")}
          />
          <FormSubmitButton
            isSubmitting={isSubmitting}
            submitText="Create account"
            submittingText="Creating account..."
            className="mt-1 w-full"
          />
        </FieldGroup>
      </form>

      <div className="text-center text-sm text-muted-foreground">
        Already have an account?{" "}
        <Link to="/login" className="font-medium text-primary underline-offset-4 hover:underline">
          Sign in
        </Link>
      </div>
    </div>
  )
}

export default RegisterInviteForm
