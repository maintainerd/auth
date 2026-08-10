import { beforeEach, describe, expect, it, vi } from 'vitest'
import { buildFirstPartyBrokerAuthorizeUrl, consumePendingOAuthFlow } from './oauthFlow'

// jsdom provides window.crypto.getRandomValues; stub subtle.digest for PKCE so the
// test is deterministic and doesn't depend on a real SHA-256 implementation.
beforeEach(() => {
  vi.spyOn(window.crypto.subtle, 'digest').mockResolvedValue(new Uint8Array(32).buffer)
  window.sessionStorage.clear()
})

describe('buildFirstPartyBrokerAuthorizeUrl', () => {
  it('includes every parameter the authorize endpoint requires', async () => {
    const url = await buildFirstPartyBrokerAuthorizeUrl({
      clientId: 'surface-client',
      idpHint: 'idp-abc',
      continueTo: '/oauth/authorize?client_id=console',
    })
    const params = new URLSearchParams(url.split('?')[1])

    expect(url.startsWith('/authorize?')).toBe(true)
    expect(params.get('response_type')).toBe('code')
    expect(params.get('client_id')).toBe('surface-client')
    // redirect_uri is REQUIRED — its omission 400'd every provider login. Guard it.
    expect(params.get('redirect_uri')).toBe(`${window.location.origin}/callback`)
    expect(params.get('code_challenge')).toBeTruthy()
    expect(params.get('code_challenge_method')).toBe('S256')
    expect(params.get('state')).toBeTruthy()
    expect(params.get('idp_hint')).toBe('idp-abc')
    // offline_access → a refresh token so the federated session renews.
    expect(params.get('scope')).toContain('offline_access')
  })

  it('stashes the PKCE verifier + state + continueTo so /callback can complete', async () => {
    const url = await buildFirstPartyBrokerAuthorizeUrl({
      clientId: 'surface-client',
      idpHint: 'idp-abc',
      continueTo: '/oauth/authorize?client_id=console',
    })
    const state = new URLSearchParams(url.split('?')[1]).get('state') as string
    const flow = consumePendingOAuthFlow(state)
    expect(flow).not.toBeNull()
    expect(flow?.clientId).toBe('surface-client')
    expect(flow?.redirectUri).toBe(`${window.location.origin}/callback`)
    expect(flow?.continueTo).toBe('/oauth/authorize?client_id=console')
    expect(flow?.codeVerifier).toBeTruthy()
  })
})
