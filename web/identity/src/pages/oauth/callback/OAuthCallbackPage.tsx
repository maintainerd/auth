/**
 * Landing page for the first-party federated login round trip.
 *
 * The backend's broker callback redirects here with `?code=…&state=…` after the
 * upstream identity provider authenticates the user. This page validates the
 * state it stashed before the redirect, exchanges the code for an httpOnly
 * cookie session, then hands off to the shared post-auth routing rule so a
 * federated sign-in lands exactly where a password sign-in would.
 *
 * Only reachable for flows this app started (see utils/oauthFlow); a third-party
 * authorize request redirects to its own client's redirect_uri, not here.
 */
import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { AlertCircle, Loader2 } from 'lucide-react'
import LoginLayout from '@/components/layout/LoginLayout'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { exchangeAuthorizationCode } from '@/services/api/oauth'
import { finishAuthStep } from '@/utils/oauthContinuation'
import { clearPendingOAuthFlow, consumePendingOAuthFlow } from '@/utils/oauthFlow'
import { safeAuthorizeContinuation } from '@/utils/oauthRedirect'
import AuthPageHeading from '@/components/auth/AuthPageHeading'
import { useLoginPageCopy } from '@/hooks/useLoginPageCopy'

export default function OAuthCallbackPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { refreshAccount } = useAuth()
  const { currentTenant } = useTenant()
  const [error, setError] = useState<string | null>(null)
  const loadingCopy = useLoginPageCopy('oauth-authorize-loading')
  const errorCopy = useLoginPageCopy('oauth-authorize-error')
  // The exchange is single-use: React 18 StrictMode double-invokes effects in
  // development, and replaying the code would revoke the session it just issued.
  const startedRef = useRef(false)

  useEffect(() => {
    if (startedRef.current) return
    startedRef.current = true

    async function run() {
      const code = searchParams.get('code')
      const state = searchParams.get('state')

      // The provider can also report failure on the redirect rather than in the
      // exchange — surface its reason instead of a generic message.
      const providerError = searchParams.get('error')
      if (providerError) {
        clearPendingOAuthFlow()
        setError(searchParams.get('error_description') || 'The identity provider declined the sign-in request.')
        return
      }

      if (!code || !state) {
        clearPendingOAuthFlow()
        setError('This sign-in link is incomplete. Please start again.')
        return
      }

      // Consuming on a state match is the app's only CSRF defence on this leg:
      // a code injected by a third party has no matching stashed state.
      const flow = consumePendingOAuthFlow(state)
      if (!flow) {
        setError('This sign-in request has expired or was already used. Please start again.')
        return
      }

      await exchangeAuthorizationCode({
        clientId: flow.clientId,
        code,
        redirectUri: flow.redirectUri,
        codeVerifier: flow.codeVerifier,
      })

      // The session now lives in httpOnly cookies; re-sync so routing sees the
      // real email_verified / profile state.
      const account = await refreshAccount()

      // Downstream-app broker login: a console (or other) authorize request sent
      // the user here to log into the identity app via an external provider. The
      // IdP session now exists on this host, so resume that ORIGINAL authorize —
      // it will issue the downstream code from the session (no second external
      // round-trip). Only a same-origin OAuth-interaction path is honored, so a
      // tampered value can never redirect off-origin.
      const continueTo = safeAuthorizeContinuation(flow.continueTo)
      if (continueTo) {
        navigate(continueTo, { replace: true })
        return
      }

      // Direct identity-app login: apply the shared post-auth routing rule.
      finishAuthStep({ account, tenant: currentTenant, navigate })
    }

    run().catch((err: unknown) => {
      setError(err instanceof Error ? err.message : 'Sign-in could not be completed.')
    })
  }, [currentTenant, navigate, refreshAccount, searchParams])

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="space-y-6" role={error ? 'alert' : 'status'} aria-live="polite">
        <div className="space-y-3">
          <div className="flex justify-center">
            <div className={`flex size-14 items-center justify-center rounded-full ${error ? 'bg-destructive/10' : 'bg-primary/10'}`}>
              {error ? (
                <AlertCircle className="size-7 text-destructive" />
              ) : (
                <Loader2 className="size-7 animate-spin text-primary" />
              )}
            </div>
          </div>
          {/* Shares the authorize round-trip's branded copy — this page is the
              tail of that same flow. The server's reason wins on failure. */}
          <AuthPageHeading
            title={error ? errorCopy.title : loadingCopy.title}
            subtitle={error || loadingCopy.subtitle}
          />
        </div>

        {error && (
          <div className="space-y-4">
            <Button className="w-full" onClick={() => navigate('/login', { replace: true })}>
              Back to sign in
            </Button>
          </div>
        )}
      </div>
    </LoginLayout>
  )
}
