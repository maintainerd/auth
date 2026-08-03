import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { AlertCircle, Loader2 } from 'lucide-react'
import LoginLayout from '@/components/layout/LoginLayout'
import { Button } from '@/components/ui/button'
import { authorizeOAuth } from '@/services/api/oauth'
import { useTenant } from '@/hooks/useTenant'
import { normalizeOAuthAuthorizeSearch, oauthLoginRoute, withRequestId } from '@/utils/oauthRedirect'
import { ApiError } from '@/services/api/client'

export default function OAuthAuthorizePage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { currentTenant } = useTenant()
  const [error, setError] = useState<string | null>(null)
  const startedRef = useRef(false)

  useEffect(() => {
    if (startedRef.current) return
    startedRef.current = true

    // Defined inside the effect so it closes over the current searchParams
    // without needing to be an effect dependency (the effect runs once).
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

    async function run() {
      try {
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
  }, [navigate, searchParams])

  return (
    <LoginLayout branding={currentTenant?.branding}>
      <div className="flex flex-col gap-8 text-center" role={error ? 'alert' : 'status'} aria-live="polite">
        <div className="flex flex-col items-center gap-3">
          <div className={`flex size-14 items-center justify-center rounded-full ${error ? 'bg-destructive/10' : 'bg-primary/10'}`}>
            {error ? (
              <AlertCircle className="size-7 text-destructive" />
            ) : (
              <Loader2 className="size-7 animate-spin text-primary" />
            )}
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">
            {error ? 'Authorization unavailable' : 'Authorizing'}
          </h1>
          <p className="max-w-xs text-sm text-muted-foreground">
            {error || 'Preparing the secure redirect.'}
          </p>
        </div>

        {error && (
          <Button className="w-full" onClick={() => navigate('/login', { replace: true })}>
            Back to sign in
          </Button>
        )}
      </div>
    </LoginLayout>
  )
}
