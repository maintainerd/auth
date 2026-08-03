/**
 * Post-authentication routing
 *
 * Single source of truth for "where should this user go now?" once we have a
 * session (after login, MFA, registration, or email verification). All callers
 * — LoginForm, RegisterForm, VerifyEmailPage, RegisterProfilePage —
 * must use this so the registration/verification flow stays consistent.
 *
 * Decision order (mirrors the backend gating):
 *   1. tenant requires email verification AND email not verified → /email-verification
 *   2. no profile yet                                            → /register/profile
 *   3. otherwise                                                 → /login-success
 */

import type { AccountEntity } from '@/services/api/auth/types'
import type { TenantEntity } from '@/services/api/tenants/types'
import { getRequestId, isBrokerAuthorizeRoute, isOAuthInteractionRoute, oauthLoginRoute, withRequestId } from '@/utils/oauthRedirect'

export const VERIFY_EMAIL_ROUTE = '/email-verification'
export const REGISTER_PROFILE_ROUTE = '/register/profile'
export const REGISTER_ROUTE = '/register'
export const REGISTER_INVITE_ROUTE = '/register/invite'
export const LOGIN_ROUTE = '/login'
export const MAGIC_LINK_ROUTE = '/magic-link'
export const NO_ACCESS_ROUTE = '/no-access'
export const SERVICE_UNAVAILABLE_ROUTE = '/service-unavailable'
export const LOGIN_SUCCESS_ROUTE = '/login-success'
// The self-service account profile — where a fully-registered, authenticated
// user lands when there is no OAuth redirect (or invite callback) to continue.
export const ACCOUNT_ROUTE = '/account/profile'

// Public auth pages an authenticated, fully-registered user should never sit on.
// NOTE: /register/invite is deliberately NOT here — an invite is addressed to a
// specific (possibly different) identity, so it must render regardless of who is
// currently signed in. It is handled by an explicit early return in
// resolveGuardRedirect instead of the "bounce authenticated users home" rule.
const AUTH_PAGES = [
  LOGIN_ROUTE,
  REGISTER_ROUTE,
  '/forgot-password',
  '/reset-password',
  MAGIC_LINK_ROUTE,
]

function isAuthPage(pathname: string): boolean {
  if (AUTH_PAGES.includes(pathname)) return true
  return /^\/[^/]+\/login$/.test(pathname)
}

export function loginSuccessRoute(): string {
  return LOGIN_SUCCESS_ROUTE
}

export function resolvePostAuthRoute(
  account: AccountEntity | null | undefined,
  tenant?: TenantEntity | null,
  registration?: { verificationRequired?: boolean },
): string {
  if (!account) {
    return REGISTER_PROFILE_ROUTE
  }

  if ((tenant?.registration_config?.require_email_verification || registration?.verificationRequired) && !account.email_verified) {
    return VERIFY_EMAIL_ROUTE
  }

  if (!account.profiles?.length) {
    return REGISTER_PROFILE_ROUTE
  }

  return LOGIN_SUCCESS_ROUTE
}

export interface GuardContext {
  pathname: string
  search?: string
  isAuthenticated: boolean
  account: AccountEntity | null | undefined
  tenant: TenantEntity | null | undefined
  registrationEnabled?: boolean
  verificationRequired?: boolean
  /**
   * Whether there is a pending continuation (an OAuth authorize return-to or an
   * invite callback) waiting to be resumed. Computed by the caller from a
   * non-consuming peek so this function stays pure. When true, a finished user
   * on a registration detour is routed to /login-success (which consumes the
   * continuation and forwards to the callback) instead of the account profile.
   */
  pendingContinuation?: boolean
}

/**
 * The single source of truth for "should this URL be allowed to render, and if
 * not, where should we send the user?" — given the current path and session.
 *
 * Returns a path to redirect to, or `null` to render the requested route.
 * Used by the app bootstrap gate on first load/reload and by the runtime route
 * guard, so all of login / register / email-verification / profile / success
 * gating lives here instead of being scattered across pages.
 *
 * Callers must only invoke this once auth+tenant initialization has completed
 * (otherwise account/tenant are not yet known).
 */
export function resolveGuardRedirect(ctx: GuardContext): string | null {
  const { pathname, search = '', isAuthenticated, account, tenant, registrationEnabled, verificationRequired, pendingContinuation } = ctx

  if (pathname === NO_ACCESS_ROUTE || pathname === SERVICE_UNAVAILABLE_ROUTE) {
    return null
  }

  // An invite link is an explicit onboarding action addressed to a specific
  // email, which may differ from whoever is currently signed in. Always let it
  // render — anonymous invitees complete registration here, and a signed-in user
  // opening an invite for someone else is handled by the form (sign-out prompt)
  // rather than being silently bounced to their own account profile.
  if (pathname === REGISTER_INVITE_ROUTE) {
    return null
  }

  // `home` is the sentinel used to detect incomplete-registration detours
  // (verify email / create profile). A fully-registered session resolves to
  // LOGIN_SUCCESS_ROUTE.
  const home = isAuthenticated ? resolvePostAuthRoute(account, tenant, { verificationRequired }) : LOGIN_ROUTE

  // The opaque authorize handle rides through every guard redirect so it survives
  // each interactive-step hop (login → verify → profile → login-success) and the
  // final continuation resumes the server-side authorize request.
  const requestId = getRequestId(search)

  // Where a fully-registered, authenticated user actually lands. Normally the
  // account profile — but when a continuation is pending (a request_id handle,
  // or the legacy OAuth return-to / invite callback fallback), route to
  // /login-success instead: that page resumes the authorize request and forwards
  // to the caller's callback URL. Without this, finishing a registration detour
  // (create profile / verify email) during an OAuth flow would be coerced to the
  // profile page before the continuation ran. Detour homes also carry the handle.
  const authenticatedHome = home === LOGIN_SUCCESS_ROUTE
    ? (pendingContinuation ? withRequestId(LOGIN_SUCCESS_ROUTE, requestId) : ACCOUNT_ROUTE)
    : withRequestId(home, requestId)

  if (pathname === '/') {
    return authenticatedHome
  }

  if (
    pathname === REGISTER_ROUTE &&
    !isAuthenticated &&
    (registrationEnabled === false || tenant?.registration_config?.self_registration_enabled === false)
  ) {
    return LOGIN_ROUTE
  }

  if (isAuthPage(pathname)) {
    return isAuthenticated ? authenticatedHome : null
  }

  if (pathname === VERIFY_EMAIL_ROUTE) {
    if (!isAuthenticated) return null
    return home === VERIFY_EMAIL_ROUTE ? null : authenticatedHome
  }

  if (pathname === REGISTER_PROFILE_ROUTE) {
    if (!isAuthenticated) return LOGIN_ROUTE
    return home === REGISTER_PROFILE_ROUTE ? null : authenticatedHome
  }

  if (pathname === LOGIN_SUCCESS_ROUTE) {
    if (!isAuthenticated) return LOGIN_ROUTE
    return null
  }

  if (!isAuthenticated) {
    if (isBrokerAuthorizeRoute(pathname, search)) {
      return null
    }
    if (isOAuthInteractionRoute(pathname)) {
      return oauthLoginRoute(pathname, search)
    }
    return LOGIN_ROUTE
  }

  // Authenticated but mid-registration (verify email / create profile) — the
  // detour takes precedence over whatever route was requested.
  if (home !== LOGIN_SUCCESS_ROUTE) return home

  if (pathname === '/account') {
    return ACCOUNT_ROUTE
  }

  // Fully-registered authenticated user on a non-auth route: OAuth interaction
  // routes (authorize / consent / device / …) render as-is so the OAuth2
  // redirect continues; every other valid account route renders too.
  // The identity app uses flat routes with no /:tenantId/ prefix, so there is
  // no URL-level tenant to check.
  return null
}
