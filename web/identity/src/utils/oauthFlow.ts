/**
 * First-party federated login flow (PKCE).
 *
 * Used when the login page is visited directly — i.e. there is NO in-flight
 * third-party authorize request to piggyback on — and the user picks a
 * federated provider. The identity app then acts as its own OAuth client
 * against its seeded surface client:
 *
 *   /login  ──click──>  /authorize?client_id=<surface client>&idp_hint=<idp>
 *                          &redirect_uri=<origin>/callback&code_challenge=…
 *        ──> backend /oauth/authorize returns the upstream IdP URL
 *        ──> upstream IdP (e.g. Cognito) authenticates the user
 *        ──> backend /api/v1/oauth/callback/{idp} issues a maintainerd code
 *        ──> <origin>/callback?code=…&state=…  (OAuthCallbackPage)
 *        ──> POST /oauth/token exchanges the code for an httpOnly cookie session
 *
 * When the login page IS mid-authorize (a `return_to` pointing at an OAuth
 * interaction route), that request owns the flow and LoginForm forwards the
 * `idp_hint` to it instead — this module is not involved.
 *
 * `state` is the only CSRF defence on the app's leg of the round trip: it is
 * generated here, stashed in sessionStorage alongside the PKCE verifier, and
 * must match on the way back before the code is exchanged.
 */

/** Path the backend redirects to after a federated login completes. Must match
 *  a `redirect_uri` registered on the surface client. */
export const OAUTH_CALLBACK_ROUTE = '/callback'

const PENDING_KEY = 'maintainerd.auth.identity.oauth.pending'

export interface PendingOAuthFlow {
  state: string
  codeVerifier: string
  clientId: string
  redirectUri: string
  /** Provider identifier, kept for error messaging on the callback page. */
  idpHint: string
}

function base64Url(bytes: Uint8Array): string {
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return window.btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

/** Cryptographically random, URL-safe value. The backend appends `state` to the
 *  redirect without escaping it, so base64url (never raw bytes) matters here. */
export function randomOAuthValue(byteLength = 32): string {
  const bytes = new Uint8Array(byteLength)
  window.crypto.getRandomValues(bytes)
  return base64Url(bytes)
}

export async function pkceChallenge(verifier: string): Promise<string> {
  const digest = await window.crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier))
  return base64Url(new Uint8Array(digest))
}

/** The app's own callback URL. Must byte-for-byte match the value replayed to
 *  the token endpoint, which the backend compares against the stored code. */
export function identityRedirectUri(): string {
  return `${window.location.origin}${OAUTH_CALLBACK_ROUTE}`
}

export function savePendingOAuthFlow(flow: PendingOAuthFlow): void {
  window.sessionStorage.setItem(PENDING_KEY, JSON.stringify(flow))
}

/**
 * Single-use retrieval: returns the pending flow only when `state` matches, and
 * clears it either way so a code can never be replayed through this page.
 */
export function consumePendingOAuthFlow(state: string): PendingOAuthFlow | null {
  try {
    const raw = window.sessionStorage.getItem(PENDING_KEY)
    if (!raw) return null
    window.sessionStorage.removeItem(PENDING_KEY)
    const flow = JSON.parse(raw) as PendingOAuthFlow
    return flow.state === state ? flow : null
  } catch {
    window.sessionStorage.removeItem(PENDING_KEY)
    return null
  }
}

export function clearPendingOAuthFlow(): void {
  window.sessionStorage.removeItem(PENDING_KEY)
}

/**
 * Mint a PKCE + state pair, stash it, and build the in-app `/authorize` URL that
 * starts the broker leg for `idpHint`. Returns a path+query (same origin), so
 * the caller navigates with the router rather than a full page load.
 */
export async function buildFirstPartyBrokerAuthorizeUrl(params: {
  clientId: string
  idpHint: string
  scope?: string
}): Promise<string> {
  const state = randomOAuthValue()
  const codeVerifier = randomOAuthValue(48)
  const redirectUri = identityRedirectUri()
  const codeChallenge = await pkceChallenge(codeVerifier)

  savePendingOAuthFlow({
    state,
    codeVerifier,
    clientId: params.clientId,
    redirectUri,
    idpHint: params.idpHint,
  })

  const query = new URLSearchParams({
    response_type: 'code',
    client_id: params.clientId,
    redirect_uri: redirectUri,
    scope: params.scope ?? 'openid profile email',
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
    idp_hint: params.idpHint,
  })

  return `/authorize?${query.toString()}`
}
