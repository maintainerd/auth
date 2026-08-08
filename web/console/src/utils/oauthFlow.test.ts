import { beforeEach, describe, expect, it } from 'vitest'
import {
  buildConsoleAuthorizeUrl,
  consumePendingOAuthFlow,
  discardPendingOAuthFlow,
} from './oauthFlow'

describe('console OAuth flow', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('builds a prompt=none PKCE request and stores the matching verifier', async () => {
    const authorizeURL = await buildConsoleAuthorizeUrl({
      clientId: 'console-client',
      tenantId: 'tenant-a',
      returnTo: '/tenant-a/dashboard',
      prompt: 'none',
    })

    const url = new URL(authorizeURL)
    expect(url.searchParams.get('prompt')).toBe('none')
    expect(url.searchParams.get('code_challenge_method')).toBe('S256')
    const state = url.searchParams.get('state')!
    const pending = consumePendingOAuthFlow(state)
    expect(pending).toMatchObject({
      state,
      clientId: 'console-client',
      tenantId: 'tenant-a',
      returnTo: '/tenant-a/dashboard',
    })
    expect(pending?.codeVerifier).toBeTruthy()
  })

  it('uses the per-tenant identity origin when one is provided', async () => {
    const authorizeURL = await buildConsoleAuthorizeUrl({
      clientId: 'console-client',
      tenantId: 'acme',
      returnTo: '',
      identityBaseUrl: 'https://acme.auth.maintainerd.local/',
    })

    const url = new URL(authorizeURL)
    expect(url.origin).toBe('https://acme.auth.maintainerd.local')
    expect(url.pathname).toBe('/authorize')
  })

  it('only discards the pending request for the matching state', async () => {
    const authorizeURL = await buildConsoleAuthorizeUrl({
      clientId: 'console-client',
      tenantId: 'tenant-a',
      returnTo: '',
    })
    const state = new URL(authorizeURL).searchParams.get('state')!

    discardPendingOAuthFlow('different-state')
    expect(consumePendingOAuthFlow(state)).not.toBeNull()

    const nextURL = await buildConsoleAuthorizeUrl({
      clientId: 'console-client',
      tenantId: 'tenant-a',
      returnTo: '',
    })
    const nextState = new URL(nextURL).searchParams.get('state')!
    discardPendingOAuthFlow(nextState)
    expect(consumePendingOAuthFlow(nextState)).toBeNull()
  })
})
