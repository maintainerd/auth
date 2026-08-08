/**
 * OAuth session marker.
 *
 * Access, refresh, and id tokens are delivered as httpOnly cookies via
 * `X-Token-Delivery: cookie` (see `exchangeAuthorizationCode` and the client's
 * refresh flow) and are NEVER persisted in localStorage — that keeps them out
 * of reach of any XSS running in the console. Only this minimal, non-secret
 * marker is persisted, so the app can remember which OAuth client the session
 * belongs to (needed for RP-initiated logout) across reloads.
 */
export interface OAuthSessionMarker {
  clientId: string
}

const STORAGE_KEY = 'maintainerd.auth.console.oauth.session'

export function getStoredOAuthSession(): OAuthSessionMarker | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const marker = JSON.parse(raw) as OAuthSessionMarker
    if (!marker.clientId) return null
    return marker
  } catch {
    return null
  }
}

export function storeOAuthSession(marker: OAuthSessionMarker) {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(marker))
}

export function clearOAuthSession() {
  window.localStorage.removeItem(STORAGE_KEY)
  inMemoryIdTokenHint = null
}

// The id_token is used only as an OIDC `id_token_hint` for RP-initiated logout.
// It is delivered as an httpOnly cookie and cannot be read from JS, so we keep
// a best-effort in-memory copy for the lifetime of the page only — never
// persisted. Logout still works without it: client_id plus the backend session
// cookie identify the session; the hint just improves the end_session request.
let inMemoryIdTokenHint: string | null = null

export function setSessionIdTokenHint(idToken: string | null | undefined) {
  inMemoryIdTokenHint = idToken ?? null
}

export function getSessionIdTokenHint(): string | null {
  return inMemoryIdTokenHint
}
