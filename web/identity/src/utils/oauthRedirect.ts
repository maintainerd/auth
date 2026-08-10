const OAUTH_RETURN_KEY = 'maintainerd_auth_oauth_return_to'
const INVITE_CALLBACK_KEY = 'maintainerd_auth_invite_callback'
const AUTHORIZE_ROUTES = ['/authorize', '/oauth/authorize']
const BROKER_HINT_PARAMS = ['idp_hint', 'provider_hint', 'identity_provider', 'connection']

/**
 * The opaque, single-use, short-lived server handle for a pending authorize
 * request (industry standard: Ory `login_challenge`, Keycloak auth-session,
 * Auth0 transaction). It is carried through each interactive step in the URL and
 * resumed via `/oauth/authorize/continue`. It is NOT a token/secret/PII — it is
 * safe to place in the URL.
 */
export const REQUEST_ID_PARAM = 'request_id'

/** Read the pending authorize handle from a query string or params. */
export function getRequestId(search: string | URLSearchParams): string | undefined {
  const params = typeof search === 'string' ? new URLSearchParams(search) : search
  return params.get(REQUEST_ID_PARAM)?.trim() || undefined
}

/**
 * Append the request_id handle to an internal app path (preserving any existing
 * query), so the handle survives each interactive step / guard redirect. No-op
 * when there is no handle.
 */
export function withRequestId(path: string, requestId?: string): string {
  if (!requestId) return path
  const [pathname, existing = ''] = path.split('?')
  const params = new URLSearchParams(existing)
  params.set(REQUEST_ID_PARAM, requestId)
  return `${pathname}?${params.toString()}`
}

const OAUTH_INTERACTION_ROUTES = [
  ...AUTHORIZE_ROUTES,
  '/device',
  '/ciba',
  '/oauth/consent',
  '/oauth/device',
  '/oauth/ciba',
  '/oauth/grants',
  '/oauth/end_session',
  '/oauth/end-session',
]

export function isOAuthInteractionRoute(pathname: string): boolean {
  return OAUTH_INTERACTION_ROUTES.some((route) => pathname === route || pathname.startsWith(`${route}/`))
}

export function isOAuthAuthorizeRoute(pathname: string): boolean {
  return AUTHORIZE_ROUTES.includes(pathname)
}

export function brokerHintFromParams(params: URLSearchParams): string | null {
  for (const key of BROKER_HINT_PARAMS) {
    const value = params.get(key)?.trim()
    if (value) return value
  }
  return null
}

export function isBrokerAuthorizeRoute(pathname: string, search: string): boolean {
  if (!isOAuthAuthorizeRoute(pathname)) return false
  return brokerHintFromParams(new URLSearchParams(search)) !== null
}

export function normalizeOAuthAuthorizeSearch(search: string): string {
  const params = new URLSearchParams(search)
  const hint = brokerHintFromParams(params)
  if (hint) {
    params.set('idp_hint', hint)
    for (const key of BROKER_HINT_PARAMS) {
      if (key !== 'idp_hint') params.delete(key)
    }
  }
  return params.toString()
}

export function safeOAuthReturnTo(value: string | null | undefined): string | null {
  if (!value || !value.startsWith('/') || value.startsWith('//')) return null
  try {
    const url = new URL(value, window.location.origin)
    if (url.origin !== window.location.origin) return null
    if (!isOAuthInteractionRoute(url.pathname)) return null
    return `${url.pathname}${url.search}${url.hash}`
  } catch {
    return null
  }
}

/**
 * Guard for the downstream-app broker `continueTo` (see PendingOAuthFlow): the
 * SAME-ORIGIN `/oauth/authorize` URL to resume after the identity session is
 * established. Deliberately NARROWER than safeOAuthReturnTo — it accepts only
 * the authorize route and REJECTS any residual broker hint, so navigating to it
 * cannot (a) redirect off-origin, (b) land on a non-authorize interaction route,
 * or (c) restart a broker leg (which would loop / spam broker sessions). Returns
 * a path-only value for react-router navigation — never pass it to
 * location.assign, and never construct it from a raw caller-supplied parameter.
 */
export function safeAuthorizeContinuation(value: string | null | undefined): string | null {
  if (!value || !value.startsWith('/') || value.startsWith('//')) return null
  try {
    const url = new URL(value, window.location.origin)
    if (url.origin !== window.location.origin) return null
    if (!isOAuthAuthorizeRoute(url.pathname)) return null
    if (brokerHintFromParams(new URLSearchParams(url.search)) !== null) return null
    return `${url.pathname}${url.search}`
  } catch {
    return null
  }
}

const ACCOUNT_LINK_PATH = '/account-link'

/**
 * Shape guard for the account-link continuation. The broker's email-collision
 * page sends an unauthenticated user to /login with `return_to` pointing back
 * at /account-link — its confirmation token and broker_session live ONLY in
 * that query string, so dropping the return_to (as safeOAuthReturnTo does:
 * /account-link is not an OAuth interaction route) strands the link flow on
 * the account profile and the accounts are never linked. Same-origin and
 * path-anchored, mirroring safeOAuthReturnTo.
 */
export function safeAccountLinkReturnTo(value: string | null | undefined): string | null {
  if (!value || !value.startsWith('/') || value.startsWith('//')) return null
  try {
    const url = new URL(value, window.location.origin)
    if (url.origin !== window.location.origin) return null
    if (url.pathname !== ACCOUNT_LINK_PATH) return null
    return `${url.pathname}${url.search}${url.hash}`
  } catch {
    return null
  }
}

export function rememberOAuthReturnTo(value: string | null | undefined): string | null {
  const safe = safeOAuthReturnTo(value)
  if (safe) sessionStorage.setItem(OAUTH_RETURN_KEY, safe)
  return safe
}

export function consumeOAuthReturnTo(): string | null {
  const safe = safeOAuthReturnTo(sessionStorage.getItem(OAUTH_RETURN_KEY))
  sessionStorage.removeItem(OAUTH_RETURN_KEY)
  return safe
}

/**
 * Non-consuming peek: is there a valid, pending OAuth return-to stored?
 * Used by the route guard to decide whether a finished user should continue the
 * OAuth authorize flow (via /login-success, which consumes it) instead of being
 * coerced to the account profile. Does NOT remove the stored value.
 */
export function hasPendingOAuthReturnTo(): boolean {
  return safeOAuthReturnTo(sessionStorage.getItem(OAUTH_RETURN_KEY)) !== null
}

export function clearOAuthReturnTo(): void {
  sessionStorage.removeItem(OAUTH_RETURN_KEY)
}

export function oauthLoginRoute(pathname: string, search: string): string {
  const current = `${pathname}${search}`
  const params = new URLSearchParams(search)
  params.set('return_to', current)
  // The tenant is resolved from the subdomain (utils/tenant), so it is never
  // injected into the login URL. Any OAuth `client_id` already present in the
  // authorize search is preserved as-is.
  rememberOAuthReturnTo(current)
  const query = params.toString()
  return query ? `/login?${query}` : '/login'
}

/**
 * Shape guard for the one cross-origin redirect this app performs: the invite
 * callback.
 *
 * AUTHORIZATION is the server's job and is NOT re-derivable here — the backend
 * validates an invite's callback against the inviting client's registered
 * redirect URIs before it stores the invite, so the only value allowed to reach
 * this function is the server-validated one returned by the invite-context
 * probe. Never pass it a raw `callback_url` query parameter: this app cannot
 * check the invite URL's signature, so a rewritten link would otherwise bounce a
 * freshly registered user to an attacker origin from the tenant's own identity
 * domain.
 *
 * This guard is the second line of defence and rejects, beyond the plain https
 * requirement:
 *  - embedded credentials (`https://login.tenant.example@evil.test/`), which
 *    read as the tenant's own host while navigating to the attacker's;
 *  - scheme-relative / non-absolute / opaque values, which inherit whatever
 *    origin they are resolved against.
 * It returns the parsed, normalized URL rather than the caller's string so the
 * value handed to `location.assign` is exactly the origin that was inspected —
 * backslash and whitespace tricks cannot smuggle a different one past it.
 */
export function safeInviteCallback(url: string | null | undefined): string | null {
  if (!url || !/^https:\/\//i.test(url)) return null
  try {
    const parsed = new URL(url)
    if (parsed.protocol !== 'https:') return null
    if (parsed.username || parsed.password) return null
    if (!parsed.hostname) return null
    return parsed.toString()
  } catch {
    return null
  }
}

export function rememberInviteCallback(url: string | null | undefined): string | null {
  const safe = safeInviteCallback(url)
  if (safe) sessionStorage.setItem(INVITE_CALLBACK_KEY, safe)
  return safe
}

export function consumeInviteCallback(): string | null {
  const value = sessionStorage.getItem(INVITE_CALLBACK_KEY)
  sessionStorage.removeItem(INVITE_CALLBACK_KEY)
  return safeInviteCallback(value)
}

/**
 * Non-consuming peek: is there a valid, pending invite callback stored? Like
 * hasPendingOAuthReturnTo, this lets the guard route a finished user to
 * /login-success (which consumes it and continues to the callback) rather than
 * the dashboard. Does NOT remove the stored value.
 */
export function hasPendingInviteCallback(): boolean {
  return safeInviteCallback(sessionStorage.getItem(INVITE_CALLBACK_KEY)) !== null
}

export function clearInviteCallback(): void {
  sessionStorage.removeItem(INVITE_CALLBACK_KEY)
}
