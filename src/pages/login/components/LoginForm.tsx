import { useEffect, useMemo, useState } from "react"
import { useForm } from "react-hook-form"
import { yupResolver } from "@hookform/resolvers/yup"
import { AlertCircle, CheckCircle2, KeyRound, Loader2, Mail } from "lucide-react"
import { FormInputField, FormPasswordField, FormSubmitButton } from "@/components/form"
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
import { buildFirstPartyBrokerAuthorizeUrl } from '@/utils/oauthFlow'
import AuthDivider from '@/components/auth/AuthDivider'
import AuthPageHeading from '@/components/auth/AuthPageHeading'
import { useLoginPageCopy } from '@/hooks/useLoginPageCopy'

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
  const { getCurrentTenant, bootstrap } = useTenant()
  const { showSuccess } = useToast()
  const [loginError, setLoginError] = useState<string | null>(null)
  const [mfaChallenge, setMfaChallenge] = useState<{ token: string; methods: string[] } | null>(null)
  const [isSendingMagicLink, setIsSendingMagicLink] = useState(false)
  const [magicLinkSent, setMagicLinkSent] = useState(false)
  const [connections, setConnections] = useState<OAuthConnections | null>(null)
  const [connectionsError, setConnectionsError] = useState<string | null>(null)
  const [startingProvider, setStartingProvider] = useState<string | null>(null)

  // Page copy is tenant-configurable in the console branding editor; these
  // resolve to the same defaults its preview shows when unset.
  const pageCopy = useLoginPageCopy('login')
  const mfaCopy = useLoginPageCopy('login-mfa-code')
  const magicLinkCopy = useLoginPageCopy('login-magic-link-sent')
  const unavailableCopy = useLoginPageCopy('login-methods-unavailable')

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
  // Two sources, one shape. Mid-authorize, the in-flight request's client owns
  // the provider list and we fetch it for that client_id. Visited directly, the
  // tenant bootstrap already carried the surface client's providers, so they
  // render on first paint with no extra round trip.
  const providerConnections = shouldLoadConnections
    ? connections?.connections ?? []
    : bootstrap?.connections ?? []
  // Passwordless email sign-in is opt-in per client and off by default, so an
  // absent value means "don't offer it" — same source split as the providers.
  const magicLinkEnabled = shouldLoadConnections
    ? connections?.magic_link_enabled === true
    : bootstrap?.magic_link_enabled === true
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
    setError,
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
      // Present when the guard bounced an authorize request here to sign in;
      // it is what sends the user back to the calling app afterwards.
      returnTo: searchParams.get('return_to'),
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
    const identifierIsValid = await trigger('email')
    if (!identifierIsValid) return

    // The shared field accepts a username OR an email because that is what
    // password sign-in supports. A magic link has to be delivered somewhere, so
    // this path needs an actual address — checked here rather than in the schema
    // so it does not leak back onto the password path.
    const identifier = getValues('email').trim()
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(identifier)) {
      setError('email', { message: 'Enter an email address to receive a sign-in link.' })
      return
    }

    setLoginError(null)
    setIsSendingMagicLink(true)
    try {
      await sendMagicLink(identifier, {
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

  const handleBrokerLogin = async (connection: OAuthConnection) => {
    setLoginError(null)
    setStartingProvider(connection.identifier)

    // Mid-authorize: the in-flight request owns the flow and already carries the
    // caller's client_id, redirect_uri and PKCE — just add the provider hint and
    // let it resume. It redirects to the original app, not back here.
    if (oauthAuthorizeTarget) {
      const params = new URLSearchParams(oauthAuthorizeTarget.searchParams)
      params.set('idp_hint', connection.identifier)
      const query = normalizeOAuthAuthorizeSearch(params.toString())
      navigate(`${oauthAuthorizeTarget.pathname}?${query}`, { replace: true })
      return
    }

    // Visited directly: there is no authorize request to join, so this app
    // starts one of its own against the tenant's surface client and lands the
    // result on /callback.
    const surfaceClientId = bootstrap?.client?.client_id
    if (!surfaceClientId) {
      setStartingProvider(null)
      setLoginError('Sign-in with this provider is unavailable right now. Please use your email and password.')
      return
    }

    try {
      const authorizeUrl = await buildFirstPartyBrokerAuthorizeUrl({
        clientId: surfaceClientId,
        idpHint: connection.identifier,
      })
      navigate(authorizeUrl, { replace: true })
    } catch {
      setStartingProvider(null)
      setLoginError('Could not start sign-in with this provider. Please try again.')
    }
  }

  if (mfaChallenge) {
    return (
      <div className="space-y-6">
        <AuthPageHeading title={mfaCopy.title} subtitle={mfaCopy.subtitle} />
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
      <div className="space-y-6">
        <div className="space-y-3">
          <div className="flex justify-center">
            <div className="flex size-14 items-center justify-center rounded-full bg-emerald-500/10">
              <CheckCircle2 className="size-7 text-emerald-600" />
            </div>
          </div>
          <AuthPageHeading title={magicLinkCopy.title} subtitle={magicLinkCopy.subtitle} />
        </div>

        <div className="space-y-4">
          <Button type="button" variant="outline" className="w-full" onClick={() => setMagicLinkSent(false)}>
            Back to password sign in
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <AuthPageHeading title={pageCopy.title} subtitle={pageCopy.subtitle} />

      {/* One uniform element stack, ordered to match the console's login-template
          preview: identity providers first, then the email/password form, then
          the secondary actions. */}
      <div className="space-y-4">
        {connectionsError && (
          <div
            role="alert"
            className="flex items-start gap-2.5 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"
          >
            <AlertCircle className="mt-0.5 size-4 shrink-0" />
            <span>{connectionsError}</span>
          </div>
        )}

        {isLoadingConnections && (
          <div className="auth-progress-panel flex items-center justify-center gap-2 rounded-md border p-3 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            <span>Loading sign-in methods...</span>
          </div>
        )}

        {providerConnections.length > 0 && !connectionsError && (
          <>
            {providerConnections.map((connection) => (
              <Button
                key={connection.identifier}
                type="button"
                variant="outline"
                className="w-full"
                disabled={isSubmitting || isSendingMagicLink || startingProvider !== null}
                onClick={() => void handleBrokerLogin(connection)}
              >
                {startingProvider === connection.identifier ? (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                ) : (
                  <KeyRound className="mr-2 size-4" />
                )}
                {startingProvider === connection.identifier ? 'Redirecting...' : providerButtonLabel(connection)}
              </Button>
            ))}

            {passwordEnabled && <AuthDivider label="or continue with email" />}
          </>
        )}

        {passwordEnabled && !connectionsError && (
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault()
              handleSubmit(onSubmit)(e)
            }}
          >
            {loginError && (
              <div
                role="alert"
                className="flex items-start gap-2.5 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"
              >
                <AlertCircle className="mt-0.5 size-4 shrink-0" />
                <span>{loginError}</span>
              </div>
            )}

            {/* FormInputField, not FormEmailField: sign-in accepts a username
                or an email, so type="email" would let the browser block a valid
                username before submit. autoComplete="username" is the correct
                token for a sign-in identifier even when it holds an email —
                it is what pairs with current-password for password managers. */}
            <FormInputField
              label="Email or username"
              placeholder="you@company.com"
              autoComplete="username"
              disabled={isSubmitting || isLoadingConnections}
              error={errors.email?.message}
              required
              {...register("email")}
            />

            <FormPasswordField
              id="password"
              label="Password"
              placeholder="Enter your password"
              autoComplete="current-password"
              disabled={isSubmitting || isLoadingConnections}
              error={errors.password?.message}
              required
              labelAction={
                <Link
                  to="/forgot-password"
                  className="text-sm font-medium text-primary underline-offset-4 hover:underline"
                >
                  Forgot password?
                </Link>
              }
              {...register("password")}
            />

            <FormSubmitButton
              isSubmitting={isSubmitting}
              submitText="Sign in"
              submittingText="Signing in..."
              className="w-full"
            />

            {magicLinkEnabled && (
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
            )}
          </form>
        )}

        {!passwordEnabled && !isLoadingConnections && providerConnections.length === 0 && !connectionsError && (
          <div className="rounded-md border p-3 text-center text-sm text-muted-foreground">
            {unavailableCopy.subtitle}
          </div>
        )}

        {/* Mirrors the register page's "Already have an account? Sign in" — the
            prompt reads as a sentence, with only the action as the link. */}
        {showSignUp && (
          <div className="text-center text-sm text-muted-foreground">
            Don&apos;t have an account?{" "}
            <Link
              to={{ pathname: "/register", search: searchParams.toString() }}
              className="font-medium text-primary underline-offset-4 hover:underline"
            >
              Sign up
            </Link>
          </div>
        )}
      </div>
    </div>
  )
}

export default LoginForm
