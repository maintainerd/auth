/**
 * Authentication Service
 * Handles authentication API calls and storage operations
 */

import axios from 'axios'
import { post, get } from '../client'
import { API_CONFIG, API_ENDPOINTS } from '../config'
import { clearOAuthSession, getStoredOAuthSession, getSessionIdTokenHint } from '../oauth-session'
import type { ApiResponse } from '../types'
import type { ProfileEntity, AccountEntity, CreateProfileRequest, CreateProfileResponse, ProfileResponse } from './types'

type AccountResponse = ApiResponse<AccountEntity>

/**
 * Local-only logout: clears the console's stored OAuth session marker and performs a
 * full-document navigation to `to` (defaults to the public `/login` page).
 *
 * This does NOT end the hosted-identity SSO session — use it for flows that want
 * to re-authenticate immediately (e.g. switching tenants). For a real "sign me
 * out" use `logoutViaIdentity()`.
 *
 * The hard navigation is deliberate: a React Router `navigate` would flip
 * `isAuthenticated` to false while still on a protected route, and the route
 * guard would momentarily see "unauthenticated on a protected route" and fire
 * the hosted-identity OAuth redirect — bouncing the user straight back in.
 */
export function logoutAndRedirect(to: string = '/login'): void {
  clearOAuthSession()
  window.location.replace(to)
}

/**
 * Full logout: ends BOTH the console's own backend session and the hosted-identity
 * SSO session.
 *
 * The console holds its own `__Host-*` session cookies — issued by the private
 * API during the OAuth authorization-code exchange — that are scoped to the
 * private-API host and are independent of identity's SSO cookies (scoped to the
 * public-API host). Ending only the SSO session leaves the console's own session
 * cookie alive, so `initializeAuth` (GET /account) re-authenticates the user on
 * the next load and logout appears not to stick.
 *
 * So we log out in two steps:
 *   1. POST /logout on the console's API → revokes the refresh token and clears
 *      the console's `__Host-*` cookies (best-effort).
 *   2. Redirect through identity's RP-initiated logout (OIDC `end_session`)
 *      carrying `id_token_hint`, `client_id`, and a registered
 *      `post_logout_redirect_uri`, which clears identity's SSO cookies and
 *      redirects back to the console `/logout` route.
 *
 * The post_logout_redirect_uri must be a URI registered for the console client
 * (the client_uri seed registers `<console-origin>/logout`).
 */
export async function logoutViaIdentity(): Promise<void> {
  const session = getStoredOAuthSession()
  // Best-effort id_token_hint: the id_token now lives in an httpOnly cookie and
  // is only readable from JS for the current page session. Logout still works
  // without it — client_id plus the backend session cookie identify the session.
  const idTokenHint = getSessionIdTokenHint()

  // 1. End the console's own backend session (revoke refresh token + clear the
  //    private-API cookies). Best-effort: proceed to SSO logout even if this
  //    fails (e.g. the session was already gone, as after account deletion).
  try {
    await post(API_ENDPOINTS.AUTH.LOGOUT)
  } catch {
    /* ignore — still end the SSO session below */
  }

  clearOAuthSession()

  const params = new URLSearchParams()
  if (idTokenHint) params.set('id_token_hint', idTokenHint)
  if (session?.clientId) params.set('client_id', session.clientId)
  params.set('post_logout_redirect_uri', `${window.location.origin}/logout`)

  window.location.replace(`${API_CONFIG.IDENTITY_BASE_URL}/oauth/end_session?${params.toString()}`)
}

/**
 * App-only sign-out (the default logout for the console).
 *
 * Ends ONLY the console's own session — it does NOT touch identity's SSO, so
 * other apps stay signed in and this app doesn't perform a global logout:
 *   1. POST /logout on the console's API → revokes the refresh token and clears
 *      the console's session cookies. Best-effort.
 *   2. clear the client-side OAuth marker.
 *   3. hard-navigate to the public `/login` page.
 *
 * Crucially it does NOT redirect through identity's `end_session`, so nothing
 * auto-reauthenticates — the user lands on the login page and must click
 * "Sign in" to start OAuth again (which is a one-click SSO if still signed into
 * identity). The hard navigation forces a fresh bootstrap that reads the now
 * cleared session, so logout sticks instead of the guard bouncing back in.
 */
export async function logout(to: string = '/login'): Promise<void> {
  try {
    // The PUBLIC plane, not the control plane. Interactive authentication is
    // public-only — the internal router deliberately does not mount /logout, so
    // posting there 404'd and logout silently did nothing. The console's session
    // is minted by the public token endpoint, so it must be ended there too.
    await axios.post(`${API_CONFIG.PUBLIC_BASE_URL}${API_ENDPOINTS.AUTH.LOGOUT}`, {}, {
      withCredentials: true,
    })
  } catch {
    /* best-effort — still clear client state and land on /login */
  }
  clearOAuthSession()
  window.location.replace(to)
}

/**
 * Fetch user profile from API
 * @returns Promise<ProfileEntity | null> - Returns null if profile doesn't exist (e.g., newly registered user)
 */
export async function fetchProfile(): Promise<ProfileEntity | null> {
  try {
    const response = await get<ProfileResponse>(API_ENDPOINTS.AUTH.PROFILE)

    if (response.success && response.data) {
      return response.data
    }

    return null
  } catch (err: unknown) {
    // A missing profile (404) is expected for newly registered users — return
    // null. Any other error (401/403/5xx) is a real failure and must propagate.
    const apiErr = err as { status?: number }
    if (apiErr?.status === 404) return null
    throw err
  }
}

/**
 * Create user profile for authenticated users (requires cookie token)
 * @param data - Profile creation data
 * @returns Promise<CreateProfileResponse>
 */
export async function createUserProfile(data: CreateProfileRequest): Promise<CreateProfileResponse> {
  const response = await post<CreateProfileResponse>(
    API_ENDPOINTS.AUTH.PROFILE,
    data
  )
  return response
}

/**
 * Create profile for registered users (dedicated register flow)
 * @param data - Profile creation data
 * @returns Promise<CreateProfileResponse>
 */
export async function createRegisterProfile(data: CreateProfileRequest): Promise<CreateProfileResponse> {
  const response = await post<CreateProfileResponse>(
    API_ENDPOINTS.AUTH.PROFILE,
    data
  )
  return response
}

/**
 * Validate if the user is still authenticated with the backend
 * This checks if the backend cookie is still valid by calling the profile endpoint
 * @returns Promise<ProfileEntity | null> - Returns profile if authenticated, null otherwise
 */
export async function validateAuthentication(): Promise<AccountEntity | null> {
  try {
    const response = await get<AccountResponse>(API_ENDPOINTS.AUTH.ACCOUNT)
    if (response.success && response.data) return response.data
    return null
  } catch (err: unknown) {
    const apiErr = err as { status?: number }
    // 401/403 is the server's verdict: there is no session. Anything else — a
    // timeout, a 5xx, a network blip, a rate limit — means we do NOT know. It
    // must propagate so boot can record "unknown", not silently resolve to
    // "anonymous" and bounce an authenticated user through SSO.
    if (apiErr?.status === 401 || apiErr?.status === 403) return null
    throw err
  }
}

export async function fetchAccount(): Promise<AccountEntity | null> {
  try {
    const response = await get<AccountResponse>(API_ENDPOINTS.AUTH.ACCOUNT)
    if (response.success && response.data) return response.data
    return null
  } catch (err: unknown) {
    // Mirror validateAuthentication: a genuine auth failure (401/403) must
    // propagate so the caller can react (e.g. redirect to login); other errors
    // resolve to null.
    const apiErr = err as { status?: number }
    if (apiErr?.status === 401 || apiErr?.status === 403) throw err
    return null
  }
}

// Export functions as an object for backward compatibility
export const authService = {
  fetchProfile,
  fetchAccount,
  createUserProfile,
  createRegisterProfile,
  validateAuthentication,
}
