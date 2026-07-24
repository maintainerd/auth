import { useEffect, useMemo, useState } from "react"
import { useForm, type Resolver } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { useNavigate, Link, useSearchParams } from "react-router-dom"
import { AlertCircle, TriangleAlert } from "lucide-react"
import { FormSubmitButton, FormInputField, FormPasswordField, PasswordRequirements, FormConsentCheckbox } from "@/components/form"
import { FieldGroup } from "@/components/ui/field"
import { buildRegisterSchema, type RegisterFormData } from "@/lib/validations"
import { useAuth } from "@/hooks/useAuth"
import { useTenant } from "@/hooks/useTenant"
import { useToast } from "@/hooks/useToast"
import { getRequestId } from "@/utils/oauthRedirect"
import { finishAuthStep } from "@/utils/oauthContinuation"
import { useRegistrationContext } from "@/hooks/useRegistrationContext"
import { Button } from "@/components/ui/button"

// screen_hint=signup is what sent the user here, so forwarding it to /login makes
// the login page bounce straight back — a redirect loop with no way to reach the
// sign-in form. Everything else (client_id, request_id, registration_flow) must
// survive.
function withoutScreenHint(params: URLSearchParams): string {
  const next = new URLSearchParams(params)
  next.delete("screen_hint")
  return next.toString()
}

const RegisterForm = () => {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { register: registerUser, refreshAccount } = useAuth()
  const { getCurrentTenant } = useTenant()
  const { showSuccess } = useToast()
  const [registerError, setRegisterError] = useState<string | null>(null)

  // What this flow's link requires the form to collect. `none` when the link
  // names no flow (ordinary self-service signup).
  const registrationContext = useRegistrationContext()
  const requiredFields =
    registrationContext.status === "ready" ? registrationContext.context.required_fields : []
  const needsFullname = requiredFields.includes("fullname")
  const needsPhone = requiredFields.includes("phone")

  // Password rules follow the tenant policy; the extra fields follow the flow.
  const passwordConfig = getCurrentTenant()?.password_config
  const registerSchema = useMemo(
    () => buildRegisterSchema(passwordConfig, { fullname: needsFullname, phone: needsPhone }),
    [passwordConfig, needsFullname, needsPhone],
  )

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting }
  } = useForm<RegisterFormData>({
    resolver: yupResolver(registerSchema) as Resolver<RegisterFormData>,
    defaultValues: {
      fullname: "",
      phone: "",
      email: "",
      password: "",
      confirmPassword: "",
      acceptTerms: false
    },
    mode: 'onSubmit',
    reValidateMode: 'onSubmit'
  })

  const passwordValue = watch("password") || ""

  // Reveal the requirements checklist once the user starts typing a password,
  // then keep it visible even if they clear the field.
  const [passwordTyped, setPasswordTyped] = useState(false)
  useEffect(() => {
    if (passwordValue.length > 0) setPasswordTyped(true)
  }, [passwordValue])

  const onSubmit = async (data: RegisterFormData) => {
    setRegisterError(null)
    try {
      const requestId = getRequestId(searchParams)
      await registerUser({
        email: data.email,
        password: data.password,
        fullname: data.fullname,
        phone: data.phone,
      })

      sessionStorage.setItem('register_email', data.email)
      // Pre-fill the profile step from the name already collected here.
      if (data.fullname?.trim()) sessionStorage.setItem('register_fullname', data.fullname.trim())
      showSuccess('Account created successfully!')

      // Registration issues an httpOnly session cookie, so sync the auth state
      // and apply the single shared continuation rule: route to the next required
      // detour (email verification → profile) threading the request_id handle, or
      // resume the pending OAuth authorize once the account is fully registered.
      const account = await refreshAccount()
      finishAuthStep({
        account,
        tenant: getCurrentTenant(),
        requestId,
        navigate,
      })
    } catch (error: unknown) {
      // registerAsync rejects with a plain { message, status } object, so testing
      // `instanceof Error` alone would discard every actionable server message
      // (e.g. "fullname is required by the registration flow").
      const asObject = error as { message?: string; status?: number } | null
      const message =
        (error instanceof Error ? error.message : undefined) ||
        asObject?.message ||
        "Registration failed. Please try again."
      setRegisterError(message)
    }
  }

  if (registrationContext.status === "invalid") {
    return (
      <div className="flex flex-col gap-6 text-center">
        <div className="flex flex-col items-center gap-2">
          <TriangleAlert className="size-8 text-destructive" />
          <h1 className="text-2xl font-semibold tracking-tight">This sign-up link is no longer valid</h1>
          <p className="text-sm text-muted-foreground">
            It may have been renamed, deactivated, or replaced. Ask whoever sent it for an up-to-date
            link.
          </p>
        </div>
        <Button asChild variant="outline">
          <Link to={{ pathname: "/login", search: withoutScreenHint(searchParams) }}>Sign in instead</Link>
        </Button>
      </div>
    )
  }

  if (registrationContext.status === "loading") {
    return (
      <div className="flex flex-col gap-4" aria-busy="true">
        <div className="h-8 animate-pulse rounded bg-muted" />
        <div className="h-10 animate-pulse rounded bg-muted" />
        <div className="h-10 animate-pulse rounded bg-muted" />
        <div className="h-10 animate-pulse rounded bg-muted" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Create your account</h1>
        <p className="text-sm text-muted-foreground">
          Sign up to get started.
        </p>
      </div>

      {registrationContext.status === "unavailable" && (
        <div
          role="alert"
          className="flex items-start gap-2.5 rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm"
        >
          <TriangleAlert className="mt-0.5 size-4 shrink-0" />
          <span className="flex-1">
            We could not confirm this sign-up link&apos;s requirements. You can continue — anything
            still missing will be flagged when you submit.
          </span>
          <Button type="button" variant="outline" size="sm" onClick={registrationContext.retry}>
            Retry
          </Button>
        </div>
      )}

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
          {needsFullname && (
            <FormInputField
              label="Full name"
              type="text"
              placeholder="Your full name"
              autoComplete="name"
              disabled={isSubmitting}
              error={errors.fullname?.message}
              required
              {...register("fullname")}
            />
          )}
          {needsPhone && (
            <FormInputField
              label="Phone"
              type="tel"
              inputMode="tel"
              placeholder="+1 212 555 1234"
              autoComplete="tel"
              disabled={isSubmitting}
              error={errors.phone?.message}
              required
              {...register("phone")}
            />
          )}
          <FormInputField
            label="Email"
            type="email"
            placeholder="you@company.com"
            autoComplete="email"
            disabled={isSubmitting}
            error={errors.email?.message}
            required
            {...register("email")}
          />
          <div className="flex flex-col gap-2">
            <FormPasswordField
              label="Password"
              placeholder="Enter a strong password"
              autoComplete="new-password"
              disabled={isSubmitting}
              error={errors.password?.message}
              required
              {...register("password")}
            />
            {passwordTyped && <PasswordRequirements password={passwordValue} config={passwordConfig} />}
          </div>
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
        <Link to={{ pathname: "/login", search: withoutScreenHint(searchParams) }} className="font-medium text-primary underline-offset-4 hover:underline">
          Sign in
        </Link>
      </div>
    </div>
  )
}

export default RegisterForm;
