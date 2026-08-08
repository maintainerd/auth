import { API_CONFIG } from '@/services/api/config'

export const OAUTH_CALLBACK_ROUTE = '/auth/callback'

interface PendingOAuthFlow {
  state: string
  codeVerifier: string
  clientId: string
  tenantId: string
  returnTo: string
  redirectUri: string
}

const PENDING_KEY = 'maintainerd.auth.console.oauth.pending'

function base64Url(bytes: Uint8Array): string {
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return window.btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

export function randomOAuthValue(byteLength = 32): string {
  const bytes = new Uint8Array(byteLength)
  window.crypto.getRandomValues(bytes)
  return base64Url(bytes)
}

export async function pkceChallenge(verifier: string): Promise<string> {
  const digest = await window.crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier))
  return base64Url(new Uint8Array(digest))
}

export function consoleRedirectUri(): string {
  return `${window.location.origin}${OAUTH_CALLBACK_ROUTE}`
}

export function savePendingOAuthFlow(flow: PendingOAuthFlow) {
  window.sessionStorage.setItem(PENDING_KEY, JSON.stringify(flow))
}

export function consumePendingOAuthFlow(state: string): PendingOAuthFlow | null {
  try {
    const raw = window.sessionStorage.getItem(PENDING_KEY)
    if (!raw) return null
    const flow = JSON.parse(raw) as PendingOAuthFlow
    if (flow.state !== state) return null
    window.sessionStorage.removeItem(PENDING_KEY)
    return flow
  } catch {
    return null
  }
}

export function discardPendingOAuthFlow(state: string): void {
  try {
    const raw = window.sessionStorage.getItem(PENDING_KEY)
    if (!raw) return
    const flow = JSON.parse(raw) as PendingOAuthFlow
    if (flow.state === state) window.sessionStorage.removeItem(PENDING_KEY)
  } catch {
    window.sessionStorage.removeItem(PENDING_KEY)
  }
}

export async function buildConsoleAuthorizeUrl(params: {
  clientId: string
  tenantId: string
  returnTo: string
  // Per-tenant hosted-identity origin from the tenant-bootstrap response. This is
  // the real source of the identity host; `API_CONFIG.IDENTITY_BASE_URL` is only
  // a last-resort fallback when the bootstrap did not provide one.
  identityBaseUrl?: string
  prompt?: 'none'
}): Promise<string> {
  const state = randomOAuthValue()
  const codeVerifier = randomOAuthValue(48)
  const redirectUri = consoleRedirectUri()
  const codeChallenge = await pkceChallenge(codeVerifier)

  savePendingOAuthFlow({
    state,
    codeVerifier,
    clientId: params.clientId,
    tenantId: params.tenantId,
    returnTo: params.returnTo,
    redirectUri,
  })

  const query = new URLSearchParams({
    response_type: 'code',
    client_id: params.clientId,
    redirect_uri: redirectUri,
    scope: 'openid profile email',
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
  })

  if (params.prompt) query.set('prompt', params.prompt)

  const identityBase = (params.identityBaseUrl || API_CONFIG.IDENTITY_BASE_URL).replace(/\/$/, '')
  return `${identityBase}/authorize?${query.toString()}`
}
