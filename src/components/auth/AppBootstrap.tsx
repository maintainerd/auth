import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { useTenant } from '@/hooks/useTenant'
import { SERVICE_UNAVAILABLE_ROUTE } from '@/utils/postAuthRoute'
import AppLoadingScreen from '@/components/layout/AppLoadingScreen'
import { RouteGuard } from './RouteGuard'
import { rememberPublicAuthContext } from '@/utils/clientContext'
import { isOAuthInteractionRoute } from '@/utils/oauthRedirect'
import { applyBranding, getBrandingBackground } from '@/utils/branding'
import { fetchOAuthConnections } from '@/services/api/oauth'
import { setLimitRedirectHandler } from '@/services/api/client'
import type { BrandingPublic } from '@/services/api/tenants/types'

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
  const navigate = useNavigate()
  const { initializeAuth, isInitialized } = useAuth()
  const { initializeTenant, currentTenant, error: tenantError } = useTenant()

  const authStartedRef = useRef(false)
  const tenantStartedRef = useRef(false)
  const [tenantSettled, setTenantSettled] = useState(false)
  const [clientBranding, setClientBranding] = useState<BrandingPublic | null>(null)

  // Tenant branding is a document-level concern: every auth route consumes the
  // same semantic CSS tokens, and cleanup prevents one tenant's theme leaking
  // into the next tenant after an in-app context switch.
  useLayoutEffect(() => {
    const resolvedBranding = clientBranding ?? currentTenant?.branding
    const metadata = resolvedBranding?.metadata
    return applyBranding(
      metadata?.colors,
      metadata?.font,
      getBrandingBackground(metadata),
    )
  }, [clientBranding, currentTenant?.branding])

  // D5: send the user to the lockout / rate-limit screen when the backend
  // returns 423 / 429 on any request. Registered here (inside the router) so the
  // module-level axios interceptor can trigger a navigation.
  useEffect(() => {
    setLimitRedirectHandler((kind, retryAfterSeconds) => {
      const path =
        kind === 'locked'
          ? '/account-locked'
          : kind === 'maintenance'
            ? '/service-unavailable'
            : '/too-many-requests'
      navigate(path, {
        replace: true,
        state: retryAfterSeconds !== undefined ? { retryAfter: retryAfterSeconds } : undefined,
      })
    })
    return () => setLimitRedirectHandler(null)
  }, [navigate])

  // Initialize auth once on mount (fetches /account if a session cookie exists).
  useEffect(() => {
    if (authStartedRef.current) return
    authStartedRef.current = true
    initializeAuth().catch(() => {
      /* handled inside initializeAuth */
    })
  }, [initializeAuth])

  // Resolve the tenant from the full host via the backend domain bootstrap and
  // settle the splash. The tenant is NEVER parsed from the host client-side, nor
  // taken from a query param / client_id — the backend resolves it from the full
  // host (`GET /tenant?domain=<host>`). This holds for both direct navigation and
  // the OAuth2 authorize flow (the calling app redirects to the correct tenant
  // subdomain). An explicit OAuth `client_id` in the URL, when present, is used
  // only to overlay per-client branding — not to resolve the tenant, and the
  // tenant's own default client does not trigger an overlay.
  useEffect(() => {
    const run = async () => {
      const urlClientId = new URLSearchParams(location.search).get('client_id')?.trim() || undefined
      // Persist the URL client_id (side-effect) for later same-session calls.
      rememberPublicAuthContext(location.search)
      setClientBranding(null)
      try {
        if (!tenantStartedRef.current) {
          tenantStartedRef.current = true
          await initializeTenant()
        }
        if (urlClientId) {
          try {
            const connections = await fetchOAuthConnections(urlClientId)
            setClientBranding(connections.branding
              ? { ...connections.branding, layout: connections.branding.layout as BrandingPublic['layout'] }
              : null)
          } catch {
            setClientBranding(null)
          }
        }
      } catch {
        /* tenant initialization or branding lookup failed */
      } finally {
        setTenantSettled(true)
      }
    }
    run()
  }, [location.search, initializeTenant])

  const ready = isInitialized && tenantSettled
  if (!ready) {
    return <AppLoadingScreen branding={currentTenant?.branding} />
  }

  const isExemptFromServiceCheck =
    location.pathname === SERVICE_UNAVAILABLE_ROUTE ||
    location.pathname === '/login' ||
    location.pathname === '/magic-link' ||
    location.pathname === '/reset-password' ||
    isOAuthInteractionRoute(location.pathname) ||
    location.pathname.startsWith('/register') ||
    /^\/[^/]+\/login$/.test(location.pathname)

  if (!currentTenant && tenantError && !isExemptFromServiceCheck) {
    return <Navigate to={SERVICE_UNAVAILABLE_ROUTE} replace />
  }

  return <RouteGuard>{children}</RouteGuard>
}

export default AppBootstrap
