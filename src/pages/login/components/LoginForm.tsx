import { useEffect, useMemo, useState } from "react"
import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { AlertCircle, CheckCircle2, KeyRound, Loader2, Mail } from "lucide-react"
import { FieldGroup, Field, FieldLabel } from "@/components/ui/field"
import { FormPasswordField, FormSubmitButton } from "@/components/form"
import { FormEmailField } from "@/components/inputs"
import { buildLoginSchema, type LoginFormData } from "@/lib/validations"
import { useToast } from "@/hooks/useToast"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { useAuth } from "@/hooks/useAuth"
import { useTenant } from "@/hooks/useTenant"
import { LoginMFAStep } from "./LoginMFAStep"
import type { AccountEntity } from '@/services/api/auth/types'
import { sendMagicLink } from '@/services/api/auth'
import { Button } from '@/components/ui/button'
import {
  normalizeOAuthAuthorizeSearch,
  getRequestId,
  safeOAuthReturnTo,
} from '@/utils/oauthRedirect'
import { finishAuthStep } from '@/utils/oauthContinuation'
import { fetchOAuthConnections } from '@/services/api/oauth'
import type { OAuthConnection, OAuthConnections } from '@/services/api/oauth/types'

type OAuthAuthorizeTarget = {
  pathname: string
  searchParams: URLSearchParams
}

function oauthAuthorizeTargetFromLoginParams(searchParams: URLSearchParams): OAuthAuthorizeTarget | null {
  const returnTo = safeOAuthReturnTo(searchParams.get('return_to'))
  if (!returnTo) {
    return null
  }

  const url = new URL(returnTo, window.location.origin)
  return {
    pathname: url.pathname,
    searchParams: new URLSearchParams(url.search),
  }
}

function providerButtonLabel(connection: OAuthConnection): string {
  const name = connection.display_name || connection.provider || connection.identifier
  return `Continue with ${name}`
}

const LoginForm = () => {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { login } = useAuth()
  const { getCurrentTenant } = useTenant()
  const { showSuccess } = useToast()
  const [loginError, setLoginError] = useState<string | null>(null)
  const [mfaChallenge, setMfaChallenge] = useState<{ token: string; methods: string[] } | null>(null)
  const [isSendingMagicLink, setIsSendingMagicLink] = useState(false)
  const [magicLinkSent, setMagicLinkSent] = useState(false)
  const [connections, setConnections] = useState<OAuthConnections | null>(null)
  const [connectionsError, setConnectionsError] = useState<string | null>(null)
  const [startingProvider, setStartingProvider] = useState<string | null>(null)

  const currentTenant = getCurrentTenant()
  const clientId = searchParams.get('client_id') || undefined
  // Tenant comes from the domain bootstrap (its slug), never from a query param.
  const tenantId = currentTenant?.name ?? undefined
  const screenHint = searchParams.get('screen_hint') || undefined
  const oauthAuthorizeTarget = useMemo(() => oauthAuthorizeTargetFromLoginParams(searchParams), [searchParams])
  const shouldLoadConnections = Boolean(clientId && oauthAuthorizeTarget)
  const loginSchema = buildLoginSchema()
  const showSignUp = shouldLoadConnections ? connections?.registration_enabled !== false : currentTenant?.registration_config?.self_registration_enabled !== false
  const passwordEnabled = shouldLoadConnections ? connections?.password_enabled === true : true
  const providerConnections = shouldLoadConnections ? connections?.connections ?? [] : []
  const isLoadingConnections = shouldLoadConnections && !connections && !connectionsError

  useEffect(() => {
    let cancelled = false
    setConnections(null)
    setConnectionsError(null)

    if (!clientId || !shouldLoadConnections) return

    fetchOAuthConnections(clientId)
      .then((result) => {
        if (!cancelled) setConnections(result)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        setConnectionsError(err instanceof Error ? err.message : 'Failed to load sign-in methods.')
      })

    return () => {
      cancelled = true
    }
  }, [clientId, shouldLoadConnections])

  // screen_hint=signup: redirect to the registration screen, carrying the same
  // OAuth authorize params so the sign-up flow can resume the authorize request
  // via /oauth/authorize/continue with the server-persisted request_id.
  useEffect(() => {
    if (screenHint === 'signup' && showSignUp) {
      const requestId = searchParams.get('request_id')
      if (requestId) {
        const params = new URLSearchParams(searchParams.toString())
        params.set('request_id', requestId)
        navigate({ pathname: '/register', search: params.toString() }, { replace: true })
      } else {
        navigate({ pathname: '/register', search: searchParams.toString() }, { replace: true })
      }
    }
  }, [screenHint, showSignUp, navigate, searchParams])

  const {
    register,
    handleSubmit,
    getValues,
    trigger,
    formState: { errors, isSubmitting }
  } = useForm<LoginFormData>({
    resolver: yupResolver(loginSchema),
    defaultValues: {
      email: "",
      password: ""
    },
    mode: 'onSubmit',
    reValidateMode: 'onSubmit'
  })

  const finishLogin = (account: AccountEntity | null | undefined) => {
    // Single shared continuation rule: fully registered → continue the pending
    // OAuth authorize (request_id) or the dashboard; mid-registration → the next
    // detour step, threading request_id through the URL.
    const outcome = finishAuthStep({
      account,
      tenant: currentTenant,
      requestId: getRequestId(searchParams),
      navigate,
    })
    // Only celebrate a completed direct sign-in; the OAuth continuation and the
    // verify/profile detours are not a finished first-party login.
    if (outcome === 'dashboard') {
      showSuccess('Login successful!')
    }
  }

  const onSubmit = async (data: LoginFormData) => {
    setLoginError(null)
    try {
      const response = await login(data.email, data.password)
      // MFA enrolled — show the second step; the session is issued there.
      if (response.mfaRequired) {
        setMfaChallenge({ token: response.challengeToken ?? '', methods: response.allowedMethods ?? [] })
        return
      }
      finishLogin(response.account)
    } catch (err: unknown) {
      const errorMessage = (err instanceof Error ? err.message : (err as { message?: string })?.message) || "Invalid email or password"

      if (errorMessage === 'email is not verified') {
        sessionStorage.setItem('register_email', data.email)
        navigate('/email-verification', { replace: true })
        return
      }

      // require_phone_verification is enabled for this tenant and the account's
      // phone is not yet verified. SMS OTP login IS the verification path: a
      // successful code proves phone possession, marks it verified server-side,
      // and issues the session — so route there rather than dead-ending. Preserve
      // the auth context (client_id) so the public SMS flow resolves its client.
      if (errorMessage === 'phone is not verified') {
        sessionStorage.setItem('phone_verification_required', '1')
        navigate({ pathname: '/sms-login', search: searchParams.toString() }, { replace: true })
        return
      }

      setLoginError(errorMessage)
    }
  }

  const handleMagicLink = async () => {
    const emailIsValid = await trigger('email')
    if (!emailIsValid) return

    setLoginError(null)
    setIsSendingMagicLink(true)
    try {
      await sendMagicLink(getValues('email'), {
        clientId,
        tenantId,
      })
      setMagicLinkSent(true)
    } catch (err: unknown) {
      setLoginError(err instanceof Error ? err.message : 'Failed to send sign-in link. Please try again.')
    } finally {
      setIsSendingMagicLink(false)
    }
  }

  const handleBrokerLogin = (connection: OAuthConnection) => {
    if (!oauthAuthorizeTarget) return

    setLoginError(null)
    setStartingProvider(connection.identifier)
    const params = new URLSearchParams(oauthAuthorizeTarget.searchParams)
    params.set('idp_hint', connection.identifier)
    const query = normalizeOAuthAuthorizeSearch(params.toString())
    navigate(`${oauthAuthorizeTarget.pathname}?${query}`, { replace: true })
  }

  if (mfaChallenge) {
    return (
      <div className="flex flex-col gap-8">
        <div className="flex flex-col items-center gap-2 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">Two-step verification</h1>
          <p className="text-sm text-muted-foreground">
            Confirm your second factor to finish signing in.
          </p>
        </div>
        <LoginMFAStep
          challengeToken={mfaChallenge.token}
          allowedMethods={mfaChallenge.methods}
          clientId={clientId}
          tenantId={tenantId}
          onVerified={(result) => finishLogin(result.account)}
          onCancel={() => setMfaChallenge(null)}
        />
      </div>
    )
  }

  if (magicLinkSent) {
    return (
      <div className="flex flex-col gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <div className="flex size-14 items-center justify-center rounded-full bg-emerald-500/10">
            <CheckCircle2 className="size-7 text-emerald-600" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Check your email</h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            A secure sign-in link will arrive shortly.
          </p>
        </div>

        <Button type="button" variant="outline" className="w-full" onClick={() => setMagicLinkSent(false)}>
          Back to password sign in
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight">Welcome back</h1>
        <p className="text-sm text-muted-foreground">
          Sign in to your account to continue.
        </p>
      </div>

      {connectionsError && (
        <div
          role="alert"
          className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"
        >
          <AlertCircle className="mt-0.5 size-4 shrink-0" />
          <span>{connectionsError}</span>
        </div>
      )}

      {isLoadingConnections && (
        <div className="auth-progress-panel flex items-center justify-center gap-2 rounded-lg border p-3 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          <span>Loading sign-in methods...</span>
        </div>
      )}

      {passwordEnabled && !connectionsError && (
        <form
          onSubmit={(e) => {
            e.preventDefault()
            handleSubmit(onSubmit)(e)
          }}
        >
          <FieldGroup>
            {loginError && (
              <div
                role="alert"
                className="flex items-start gap-2.5 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"
              >
                <AlertCircle className="mt-0.5 size-4 shrink-0" />
                <span>{loginError}</span>
              </div>
            )}
            <FormEmailField
              label="Email"
              placeholder="you@company.com"
              disabled={isSubmitting || isLoadingConnections}
              error={errors.email?.message}
              required
              {...register("email")}
            />
            <Field>
              <div className="flex items-center">
                <FieldLabel htmlFor="password">
                  Password
                  <span className="text-red-500 ml-1">*</span>
                </FieldLabel>
                <Link
                  to="/forgot-password"
                  className="ml-auto text-sm font-medium text-primary underline-offset-4 hover:underline"
                >
                  Forgot password?
                </Link>
              </div>
              <FormPasswordField
                id="password"
                label=""
                placeholder="Enter your password"
                autoComplete="current-password"
                disabled={isSubmitting || isLoadingConnections}
                error={errors.password?.message}
                containerClassName="space-y-0"
                labelClassName="sr-only"
                required
                {...register("password")}
              />
            </Field>
            <FormSubmitButton
              isSubmitting={isSubmitting}
              submitText="Sign in"
              submittingText="Signing in..."
              className="mt-1 w-full"
            />

            <div className="relative py-1">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-card px-2 text-muted-foreground">Or continue with</span>
              </div>
            </div>

            <Button
              type="button"
              variant="outline"
              className="w-full"
              disabled={isSubmitting || isSendingMagicLink || isLoadingConnections}
              onClick={handleMagicLink}
            >
              {isSendingMagicLink ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <Mail className="mr-2 size-4" />
              )}
              {isSendingMagicLink ? 'Sending sign-in link...' : 'Email me a sign-in link'}
            </Button>
          </FieldGroup>
        </form>
      )}

      {providerConnections.length > 0 && !connectionsError && (
        <div className="flex flex-col gap-3">
          {passwordEnabled && (
            <div className="relative py-1">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-card px-2 text-muted-foreground">Or use an identity provider</span>
              </div>
            </div>
          )}

          {providerConnections.map((connection) => (
            <Button
              key={connection.identifier}
              type="button"
              variant="outline"
              className="w-full"
              disabled={isSubmitting || isSendingMagicLink || startingProvider !== null}
              onClick={() => handleBrokerLogin(connection)}
            >
              {startingProvider === connection.identifier ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <KeyRound className="mr-2 size-4" />
              )}
              {startingProvider === connection.identifier ? 'Redirecting...' : providerButtonLabel(connection)}
            </Button>
          ))}
        </div>
      )}

      {!passwordEnabled && !isLoadingConnections && providerConnections.length === 0 && !connectionsError && (
        <div className="rounded-lg border p-3 text-center text-sm text-muted-foreground">
          No sign-in methods are available for this application.
        </div>
      )}

      {showSignUp && (
      <div className="text-center text-sm text-muted-foreground">
        Don&apos;t have an account?{" "}
        <Link to={{ pathname: "/register", search: searchParams.toString() }} className="font-medium text-primary underline-offset-4 hover:underline">
          Sign up
        </Link>
      </div>
      )}
    </div>
  )
}

export default LoginForm
