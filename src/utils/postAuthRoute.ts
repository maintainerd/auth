/**
 * Post-authentication routing
 *
 * Single source of truth for console routing after a hosted identity OAuth
 * login. Registration, email verification, and profile completion now belong
 * to the public identity surface, so console users land on their tenant
 * dashboard once an account session is available.
 *
 * The tenant is identified by the host subdomain (resolved by the backend
 * tenant-bootstrap endpoint into `currentTenant`), never by a URL path segment —
 * so console routes are flat (`/dashboard`, `/users`, …) and the same path means
 * "this tenant's page" on every tenant subdomain.
 */

import type { AccountEntity } from '@/services/api/auth/types'
import type { TenantEntity } from '@/services/api/tenants/types'
import { OAUTH_CALLBACK_ROUTE } from '@/utils/oauthFlow'

export const LOGIN_ROUTE = '/login'
export const NO_ACCESS_ROUTE = '/no-access'
export const SERVICE_UNAVAILABLE_ROUTE = '/service-unavailable'
// Post-logout landing. The hosted-identity RP-initiated logout redirects here
// (a registered post_logout_redirect_uri); the route just bounces to /login.
export const LOGOUT_ROUTE = '/logout'
// Public landing page. Unauthenticated users rest here (instead of being
// auto-redirected into the hosted identity OAuth flow) so that logging out
// doesn't immediately bounce back through SSO and re-authenticate.
export const LANDING_ROUTE = '/'

export const DASHBOARD_ROUTE = '/dashboard'

/**
 * Public (unprotected) console routes. Unauthenticated users may rest on these
 * WITHOUT being auto-redirected into the hosted-identity OAuth flow:
 *
 *  - the landing / login page (`/`, `/login`) — shows a Sign in button
 *  - the first-run setup wizard (`/setup/*`)
 *  - the no-access / service-unavailable / OAuth callback pages
 *
 * Any other route is protected: an unauthenticated visit starts the OAuth flow.
 *
 * This is the single source of truth shared by the bootstrap gate (loading
 * screen) and the runtime route guard, so they always agree on what is public
 * and logout reliably lands on the login page instead of looping through SSO.
 */
export function isPublicConsoleRoute(pathname: string): boolean {
  return (
    pathname === LANDING_ROUTE ||
    pathname === LOGIN_ROUTE ||
    pathname === LOGOUT_ROUTE ||
    pathname === NO_ACCESS_ROUTE ||
    pathname === SERVICE_UNAVAILABLE_ROUTE ||
    pathname === OAUTH_CALLBACK_ROUTE ||
    pathname.startsWith('/setup')
  )
}

// The login page is an auth page an authenticated user should never sit on —
// bounce them to their dashboard.
function isAuthPage(pathname: string): boolean {
  return pathname === LOGIN_ROUTE
}

export function dashboardRoute(): string {
  return DASHBOARD_ROUTE
}

export function resolvePostAuthRoute(): string {
  // The tenant is fixed by the host subdomain, so every tenant's post-auth home
  // is simply the (flat) dashboard route on the current host.
  return dashboardRoute()
}

export interface GuardContext {
  pathname: string
  isAuthenticated: boolean
  account: AccountEntity | null | undefined
  tenant: TenantEntity | null | undefined
}

/**
 * The single source of truth for "should this URL be allowed to render, and if
 * not, where should we send the user?" — given the current path and session.
 *
 * Returns a path to redirect to, or `null` to render the requested route.
 * Used by the app bootstrap gate on first load/reload and by the runtime route
 * guard, so login and dashboard gating lives here instead of being scattered
 * across pages. (Registration, email verification, and profile completion
 * belong to the hosted identity surface and no longer render here.)
 *
 * Callers must only invoke this once auth+tenant initialization has completed
 * (otherwise account/tenant are not yet known).
 */
export function resolveGuardRedirect(ctx: GuardContext): string | null {
  const { pathname, isAuthenticated, account, tenant } = ctx

  // First-run setup and the no-access page are never gated.
  if (pathname === NO_ACCESS_ROUTE || pathname === SERVICE_UNAVAILABLE_ROUTE || pathname.startsWith('/setup')) {
    return null
  }

  // The OAuth callback owns its own routing: it exchanges the code, refreshes
  // the account, then navigates to the caller's return_to. Evaluating the
  // tenant-isolation rule here mid-exchange preempted that with /no-access.
  if (pathname === OAUTH_CALLBACK_ROUTE) {
    return null
  }

  const home = resolvePostAuthRoute()

  // Root → resolved home. Unauthenticated routing is handled by RouteGuard,
  // which starts the hosted identity OAuth flow.
  if (pathname === '/') {
    return isAuthenticated ? home : null
  }

  // The login page is off-limits to authenticated users — send them home.
  // Unauthenticated users are picked up by RouteGuard's OAuth redirector.
  if (isAuthPage(pathname)) {
    return isAuthenticated ? home : null
  }

  // Everything else is a tenant-scoped protected page.
  if (!isAuthenticated) return null

  // Tenant isolation (defense-in-depth): the host's resolved tenant must match
  // the signed-in user's tenant. Both are keyed by the tenant `name`. The host's
  // tenant is the one the backend resolved into `currentTenant`; when it is
  // unknown, we do not block.
  const hostTenant = tenant?.name
  const ownTenant = account?.tenant?.name
  if (hostTenant && ownTenant && hostTenant !== ownTenant) return NO_ACCESS_ROUTE

  return null
}
