import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { resolveGuardRedirect } from '@/utils/postAuthRoute'
import { getRequestId, hasPendingOAuthReturnTo, hasPendingInviteCallback } from '@/utils/oauthRedirect'
import AppLoadingScreen from '@/components/layout/AppLoadingScreen'
import { useOAuthConnections } from '@/hooks/useOAuthConnections'

/**
 * Single runtime redirect authority. Wraps the whole route tree and, on every
 * navigation, asks resolveGuardRedirect() whether the current path is allowed
 * for the current session — redirecting if not. Replaces the per-page guards
 * (RedirectIfAuthenticated / ProtectedRoute / inline page checks).
 *
 * It runs only after AppBootstrap has finished initialization, so account and
 * tenant are known and decisions are flash-free (computed at render time).
 */
export function RouteGuard({ children }: { children: ReactNode }) {
  const location = useLocation()
  const { isAuthenticated, account, status } = useAuth()
  const { currentTenant } = useTenant()
  const registrationEntry = !isAuthenticated && location.pathname === '/register'
  const connections = useOAuthConnections(registrationEntry)

  if (registrationEntry && connections.isPending && connections.fetchStatus !== 'idle') {
    return <AppLoadingScreen branding={currentTenant?.branding} />
  }

  // status === 'error' means we could NOT determine auth (a transient 429/5xx on
  // /account), NOT that the user is logged out (that would be 'anonymous' from a
  // 401/403). Treating it as anonymous here bounced a still-valid session to
  // /login and fed the logout/login churn. Hold the current route on a transient
  // error; the next successful /account resolves the real state and re-runs the
  // guard. A definitive 'anonymous' still falls through to the redirect below.
  if (status === 'error') {
    return <>{children}</>
  }

  // A pending continuation means a finished user finishing a registration detour
  // should continue the OAuth flow (via /login-success) rather than being coerced
  // to the account profile. Primary signal is the request_id handle in the URL;
  // the legacy sessionStorage return-to / invite callback remain as a defensive
  // fallback. Peeked non-consumingly so the guard function stays pure and the
  // handle is only consumed by LoginSuccessPage.
  const pendingContinuation =
    getRequestId(location.search) !== undefined ||
    hasPendingOAuthReturnTo() ||
    hasPendingInviteCallback()

  const target = resolveGuardRedirect({
    pathname: location.pathname,
    search: location.search,
    isAuthenticated,
    account,
    tenant: currentTenant,
    registrationEnabled: connections.data?.registration_enabled,
    pendingContinuation,
  })

  if (target && target !== location.pathname) {
    return <Navigate to={target} replace />
  }

  return <>{children}</>
}

export default RouteGuard
