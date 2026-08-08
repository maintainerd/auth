import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { SERVICE_UNAVAILABLE_ROUTE, isPublicConsoleRoute } from '@/utils/postAuthRoute'
import AppLoadingScreen from '@/components/layout/AppLoadingScreen'
import { setBootstrapSettled } from '@/services/api/client'
import { RouteGuard } from './RouteGuard'

/**
 * App initialization gate.
 *
 * Runs once on first load / reload / direct-URL entry: kicks off auth and tenant
 * initialization and shows a single full-screen loading splash until both have
 * settled. Only then does it render the route tree (wrapped in RouteGuard, which
 * decides where the user actually belongs).
 *
 * It is NOT re-shown during in-app navigation — once the app has booted, the
 * splash never returns; runtime gating is handled instantly by RouteGuard.
 */
export function AppBootstrap({ children }: { children: ReactNode }) {
  const location = useLocation()
  const { initializeAuth, isInitialized, status: authStatus } = useAuth()
  const { initializeFromHost, currentTenant, error: tenantError } = useTenant()

  const authStartedRef = useRef(false)
  const tenantStartedRef = useRef(false)
  const [tenantSettled, setTenantSettled] = useState(false)

  // Initialize auth once on mount (fetches /account if a session cookie exists).
  useEffect(() => {
    if (authStartedRef.current) return
    authStartedRef.current = true
    initializeAuth().catch(() => {
      /* handled inside initializeAuth */
    })
  }, [initializeAuth])

  // Initialize the tenant from the host subdomain. The subdomain is fixed for
  // the lifetime of the page, so this runs exactly once — switching tenants is a
  // full cross-subdomain navigation, which reloads the app.
  useEffect(() => {
    if (tenantStartedRef.current) return
    tenantStartedRef.current = true

    const run = async () => {
      // Setup routes have no tenant yet — skip initialization entirely.
      if (location.pathname.startsWith('/setup')) {
        setTenantSettled(true)
        return
      }

      try {
        await initializeFromHost()
      } catch {
        /* error already surfaced in initializeFromHost */
      } finally {
        setTenantSettled(true)
      }
    }
    run()
  }, [location.pathname, initializeFromHost])

  // Nothing renders until the session verdict is IN. `isInitialized` alone was
  // not enough: it flips true even when the account fetch merely failed, and
  // downstream code then read isAuthenticated=false as "anonymous" and bounced a
  // signed-in user through SSO. Waiting on a resolved status is what removes the
  // flash of the login / no-access page.
  const authResolved = isInitialized && authStatus !== 'unknown'

  // Tell the API layer the verdict is in. Until this fires the axios interceptor
  // must not navigate on a 401 — during boot a 401 from the session probe is the
  // expected answer for a logged-out visitor, not a session to recover. It used
  // to act on that immediately, racing this very component: the tenant lookup
  // resolved 200 while the probe resolved 401, and whichever landed last decided
  // whether the reload guard survived. When it did not, the page reloaded again,
  // and the URL flickered between / and /login until the timing happened to
  // break the cycle.
  useEffect(() => {
    if (!authResolved || !tenantSettled) return
    setBootstrapSettled(authStatus === 'authenticated')
  }, [authResolved, tenantSettled, authStatus])

  if (!authResolved || !tenantSettled) {
    return <AppLoadingScreen branding={currentTenant?.branding} />
  }

  // We reached the app but could not determine the session. Treating this as
  // anonymous would silently sign the user out over a blip, so it is surfaced as
  // an outage on protected routes instead.
  if (authStatus === 'error' && !isPublicConsoleRoute(location.pathname)) {
    return <Navigate to={SERVICE_UNAVAILABLE_ROUTE} replace />
  }

  // Public routes (login / landing, setup wizard, error / callback pages) must
  // render even when the tenant couldn't be resolved — otherwise logging out
  // could bounce to service-unavailable instead of the login page. Only
  // protected routes fall back to the service-unavailable screen.
  if (!currentTenant && tenantError && !isPublicConsoleRoute(location.pathname)) {
    return <Navigate to={SERVICE_UNAVAILABLE_ROUTE} replace />
  }

  return <RouteGuard>{children}</RouteGuard>
}
