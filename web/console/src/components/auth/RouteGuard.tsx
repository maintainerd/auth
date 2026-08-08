import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { resolveGuardRedirect, isPublicConsoleRoute } from '@/utils/postAuthRoute'
import { ConsoleOAuthRedirect } from './ConsoleOAuthRedirect'

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
  const { isAuthenticated, account } = useAuth()
  const { currentTenant } = useTenant()
  const pathname = location.pathname
  const returnTo = `${location.pathname}${location.search}${location.hash}`

  // Unauthenticated visits to a PROTECTED route start the hosted-identity OAuth
  // flow (deep links are preserved via returnTo). Public routes — the login /
  // landing page, setup wizard, and error/callback pages — render normally, so
  // logging out lands on the login page instead of looping back through SSO.
  if (!isAuthenticated && !isPublicConsoleRoute(pathname)) {
    return <ConsoleOAuthRedirect returnTo={returnTo === '/' ? '' : returnTo} />
  }

  const target = resolveGuardRedirect({
    pathname,
    isAuthenticated,
    account,
    tenant: currentTenant,
  })

  if (target && target !== location.pathname) {
    return <Navigate to={target} replace />
  }

  return <>{children}</>
}
