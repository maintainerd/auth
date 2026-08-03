import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import AppLoadingScreen from '@/components/layout/AppLoadingScreen'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { exchangeAuthorizationCode } from '@/services/api/oauth'
import { consumePendingOAuthFlow } from '@/utils/oauthFlow'
import { resolvePostAuthRoute } from '@/utils/postAuthRoute'

const OAuthCallbackPage = () => {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { refreshAccount } = useAuth()
  const { currentTenant } = useTenant()
  const [failed, setFailed] = useState(false)
  // The exchange is single-use: consumePendingOAuthFlow removes the stashed
  // flow, so a second invocation (StrictMode, or any dep change) finds nothing
  // and reports failure — which flashed /no-access while the first exchange was
  // still in flight. Identity's callback already guards this way.
  const startedRef = useRef(false)

  useEffect(() => {
    if (startedRef.current) return
    startedRef.current = true

    const run = async () => {
      const code = searchParams.get('code')
      const state = searchParams.get('state')
      if (!code || !state) {
        setFailed(true)
        return
      }

      const flow = consumePendingOAuthFlow(state)
      if (!flow) {
        setFailed(true)
        return
      }

      await exchangeAuthorizationCode({
        clientId: flow.clientId,
        code,
        redirectUri: flow.redirectUri,
        codeVerifier: flow.codeVerifier,
      })

      await refreshAccount()
      navigate(flow.returnTo || resolvePostAuthRoute(), { replace: true })
    }

    run().catch((error) => {
      console.error('[console-oauth] failed to exchange authorization code for token', error)
      setFailed(true)
    })
  }, [currentTenant, navigate, refreshAccount, searchParams])

  useEffect(() => {
    if (failed) {
      navigate('/no-access', { replace: true })
    }
  }, [failed, navigate])

  return <AppLoadingScreen branding={currentTenant?.branding} />
}

export default OAuthCallbackPage

