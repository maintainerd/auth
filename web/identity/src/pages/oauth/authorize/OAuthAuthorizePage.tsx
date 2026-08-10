import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { AlertCircle, Loader2 } from 'lucide-react'
import LoginLayout from '@/components/layout/LoginLayout'
import { Button } from '@/components/ui/button'
import { authorizeOAuth } from '@/services/api/oauth'
import { useTenant } from '@/hooks/useTenant'
import { brokerHintFromParams, normalizeOAuthAuthorizeSearch, oauthLoginRoute, withRequestId } from '@/utils/oauthRedirect'
import { buildFirstPartyBrokerAuthorizeUrl } from '@/utils/oauthFlow'
import { ApiError } from '@/services/api/client'
import AuthPageHeading from '@/components/auth/AuthPageHeading'
import { useLoginPageCopy } from '@/hooks/useLoginPageCopy'

// Query keys carrying an external-provider hint; stripped when we continue the
// downstream authorize after the identity session exists (so it is not re-brokered).
const BROKER_HINT_PARAMS = ['idp_hint', 'provider_hint', 'identity_provider', 'connection']

export default function OAuthAuthorizePage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { currentTenant, defaultClient, isLoading: tenantLoading } = useTenant()
  const [error, setError] = useState<string | null>(null)
  // Guard keyed on the processed query, NOT a one-shot boolean. This page can
  // navigate to ITSELF with a new query (the downstream broker orchestration
  // redirects /oauth/authorize?client_id=console&idp_hint=… →
  // /authorize?client_id=<surface>&idp_hint=…), and React keeps the same
  // component instance across that same-route navigation. A one-shot boolean
  // would run only for the first query and leave the second stuck on the
  // spinner. Keying on the query re-runs for a genuinely new request while
  // still de-duping React 18 StrictMode's double-invoke (identical query).
  const processedSearchRef = useRef<string | null>(null)
  const loadingCopy = useLoginPageCopy('oauth-authorize-loading')
  const errorCopy = useLoginPageCopy('oauth-authorize-error')

  useEffect(() => {
    // Wait for the tenant bootstrap before running: we must know our own surface
    // client id to tell a downstream broker request (console → identity) apart
    // from a first-party one, and acting before it resolves would misroute the
    // very case this page needs to special-case.
    if (tenantLoading) return
    const currentSearch = searchParams.toString()
    if (processedSearchRef.current === currentSearch) return
    processedSearchRef.current = currentSearch

    // Defined inside the effect so it closes over the query for THIS run (the
    // effect may run again if the page navigates to itself with a new query).
    const postSilentResult = (message: { redirect_uri?: string; error?: string }): boolean => {
      if (searchParams.get('prompt') !== 'none' || window.parent === window) return false
      const redirectURI = searchParams.get('redirect_uri')
      if (!redirectURI) return false
      try {
        const targetOrigin = new URL(redirectURI).origin
        window.parent.postMessage({
          type: 'maintainerd:oauth:silent',
          state: searchParams.get('state') || '',
          ...message,
        }, targetOrigin)
        return true
      } catch {
        return false
      }
    }

    // Strip external-provider hints from a query string. Used to continue the
    // downstream authorize once the identity session exists, so the backend
    // issues the code from that session instead of starting a fresh broker leg.
    const withoutBrokerHints = (search: string): string => {
      const params = new URLSearchParams(search)
      for (const key of BROKER_HINT_PARAMS) params.delete(key)
      return params.toString()
    }

    async function run() {
      try {
        const brokerHint = brokerHintFromParams(searchParams)
        const requestedClient = searchParams.get('client_id') || ''
        const surfaceClient = defaultClient?.client_id || ''
        // prompt=none is a SILENT check (hidden iframe): it must never trigger
        // interactive UI. The normal path below already handles idp_hint+prompt=none
        // correctly (backend returns interaction_required, delivered to the parent
        // via postSilentResult), so the orchestration — which would redirect the
        // iframe to the external provider — must not run for it.
        const isSilent = searchParams.get('prompt') === 'none'
        // A DOWNSTREAM app (e.g. the console) sent the user here to authenticate
        // via an external provider. Brokering directly for that client only
        // authenticates the downstream app — the identity app (the IdP) is left
        // without a session on its own host, so SSO to the next app would prompt
        // again. Instead: ensure the identity session first (via our OWN
        // first-party broker login), THEN resume this authorize from that session
        // — mirroring how password login already establishes the IdP session
        // before issuing the downstream code. Flow B (this IS the surface client)
        // and non-broker requests fall through to the normal path unchanged.
        if (!isSilent && brokerHint && surfaceClient && requestedClient && requestedClient !== surfaceClient) {
          // Same-origin authorize URL to resume afterwards, with the hint removed.
          const continueTo = `${window.location.pathname}?${withoutBrokerHints(window.location.search)}`
          // If the user already has an identity session, the hint-stripped
          // authorize issues the downstream code straight away (no external
          // round-trip). Otherwise it reports login_required and we broker.
          try {
            const direct = await authorizeOAuth(withoutBrokerHints(searchParams.toString()))
            if (direct.redirect_uri) {
              if (postSilentResult({ redirect_uri: direct.redirect_uri })) return
              window.location.assign(direct.redirect_uri)
              return
            }
            if (direct.consent_challenge) {
              if (postSilentResult({ error: 'consent_required' })) return
              navigate(`/oauth/consent/${encodeURIComponent(direct.consent_challenge)}`, { replace: true })
              return
            }
          } catch (probeErr) {
            if (!(probeErr instanceof Error) || probeErr.message !== 'login_required') throw probeErr
            const authorizeURL = await buildFirstPartyBrokerAuthorizeUrl({
              clientId: surfaceClient,
              idpHint: brokerHint,
              continueTo,
            })
            navigate(authorizeURL, { replace: true })
            return
          }
        }

        const result = await authorizeOAuth(normalizeOAuthAuthorizeSearch(searchParams.toString()))
        if (result.redirect_uri) {
          if (postSilentResult({ redirect_uri: result.redirect_uri })) return
          window.location.assign(result.redirect_uri)
          return
        }
        if (result.consent_challenge) {
          if (postSilentResult({ error: 'consent_required' })) return
          navigate(`/oauth/consent/${encodeURIComponent(result.consent_challenge)}`, { replace: true })
          return
        }
        setError('Authorization could not continue.')
      } catch (err) {
        const errorCode = err instanceof Error ? err.message : 'authorization_failed'
        if (postSilentResult({ error: errorCode })) return
        if (err instanceof Error && err.message === 'login_required') {
          // Industry-standard server-handle continuation: the backend persists
          // the authorize request and returns an opaque, single-use request_id.
          // Carry it into the interactive-step URL for ALL paths (signup AND
          // login) so the flow resumes via /oauth/authorize/continue. The legacy
          // sessionStorage return-to (set by oauthLoginRoute) remains as a
          // defensive fallback when the backend returns no handle.
          const requestId = err instanceof ApiError ? err.requestId : undefined
          const screenHint = searchParams.get('screen_hint')
          if (screenHint === 'signup' && requestId) {
            const params = new URLSearchParams(searchParams.toString())
            params.set('request_id', requestId)
            navigate({ pathname: '/register', search: params.toString() }, { replace: true })
            return
          }
          const loginRoute = oauthLoginRoute(window.location.pathname, window.location.search)
          navigate(withRequestId(loginRoute, requestId), { replace: true })
          return
        }
        setError(err instanceof Error ? err.message : 'Authorization failed.')
      }
    }

    run()
  }, [navigate, searchParams, tenantLoading, defaultClient])

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
          {/* The server's failure reason is more useful than the branded
              subtitle, so it takes precedence when present. */}
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
