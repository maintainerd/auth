import { beforeEach, describe, expect, it } from 'vitest'
import {
  brokerHintFromParams,
  consumeInviteCallback,
  consumeOAuthReturnTo,
  hasPendingInviteCallback,
  isBrokerAuthorizeRoute,
  isOAuthInteractionRoute,
  normalizeOAuthAuthorizeSearch,
  oauthLoginRoute,
  rememberInviteCallback,
  rememberOAuthReturnTo,
  safeInviteCallback,
  safeOAuthReturnTo,
} from './oauthRedirect'

describe('oauthRedirect', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('recognizes short and namespaced OAuth interaction routes', () => {
    expect(isOAuthInteractionRoute('/authorize')).toBe(true)
    expect(isOAuthInteractionRoute('/oauth/authorize')).toBe(true)
    expect(isOAuthInteractionRoute('/oauth/consent/challenge-1')).toBe(true)
    expect(isOAuthInteractionRoute('/oauth/end_session')).toBe(true)
    expect(isOAuthInteractionRoute('/login')).toBe(false)
  })

  it('recognizes and normalizes broker authorize hints', () => {
    expect(isBrokerAuthorizeRoute('/authorize', '?client_id=external&idp_hint=google')).toBe(true)
    expect(isBrokerAuthorizeRoute('/oauth/authorize', '?client_id=external&provider_hint=google')).toBe(true)
    expect(isBrokerAuthorizeRoute('/oauth/consent/challenge-1', '?idp_hint=google')).toBe(false)
    expect(brokerHintFromParams(new URLSearchParams('identity_provider=okta'))).toBe('okta')

    const normalized = new URLSearchParams(normalizeOAuthAuthorizeSearch('client_id=external&provider_hint=google'))
    expect(normalized.get('idp_hint')).toBe('google')
    expect(normalized.has('provider_hint')).toBe(false)
  })

  it('accepts only same-origin OAuth return paths', () => {
    expect(safeOAuthReturnTo('/authorize?client_id=external')).toBe('/authorize?client_id=external')
    expect(safeOAuthReturnTo('/oauth/consent/challenge-1')).toBe('/oauth/consent/challenge-1')
    expect(safeOAuthReturnTo('/dashboard')).toBeNull()
    expect(safeOAuthReturnTo('//evil.example/authorize')).toBeNull()
    expect(safeOAuthReturnTo('https://evil.example/authorize')).toBeNull()
  })

  it('stores and consumes OAuth return targets once', () => {
    expect(rememberOAuthReturnTo('/device?user_code=ABCD-EFGH')).toBe('/device?user_code=ABCD-EFGH')
    expect(consumeOAuthReturnTo()).toBe('/device?user_code=ABCD-EFGH')
    expect(consumeOAuthReturnTo()).toBeNull()
  })

  it('builds a login URL that preserves OAuth query and stores return_to', () => {
    const route = oauthLoginRoute('/authorize', '?client_id=external&scope=openid')
    const url = new URL(route, window.location.origin)

    expect(url.pathname).toBe('/login')
    expect(url.searchParams.get('client_id')).toBe('external')
    expect(url.searchParams.get('tenant_id')).toBeNull()
    expect(url.searchParams.get('return_to')).toBe('/authorize?client_id=external&scope=openid')
    expect(consumeOAuthReturnTo()).toBe('/authorize?client_id=external&scope=openid')
  })

  describe('invite callback guard', () => {
    it('accepts a server-validated https callback and hands back the parsed URL', () => {
      expect(safeInviteCallback('https://app.example.com/welcome?ref=invite'))
        .toBe('https://app.example.com/welcome?ref=invite')
      // Returned normalized, so the string that reaches location.assign is
      // exactly the origin the guard inspected.
      expect(safeInviteCallback('HTTPS://app.example.com')).toBe('https://app.example.com/')
    })

    it('rejects everything that is not an unambiguous https origin', () => {
      expect(safeInviteCallback(null)).toBeNull()
      expect(safeInviteCallback('')).toBeNull()
      expect(safeInviteCallback('http://app.example.com/welcome')).toBeNull()
      expect(safeInviteCallback('javascript:alert(1)')).toBeNull()
      expect(safeInviteCallback('//evil.test/welcome')).toBeNull()
      expect(safeInviteCallback('/welcome')).toBeNull()
      expect(safeInviteCallback('https:/\\evil.test/welcome')).toBeNull()
      expect(safeInviteCallback('https://')).toBeNull()
    })

    // Credentials in the authority make the URL bar read as the tenant's own
    // host while the browser navigates to the attacker's. The guard used to
    // accept every https URL, so this passed straight through to
    // location.assign.
    it('rejects a callback that hides an attacker host behind embedded credentials', () => {
      expect(safeInviteCallback('https://login.tenant.example@evil.test/steal')).toBeNull()
      expect(safeInviteCallback('https://login.tenant.example:pw@evil.test/steal')).toBeNull()
    })

    it('stores and consumes an invite callback once, re-validating on the way out', () => {
      expect(rememberInviteCallback('https://app.example.com/welcome')).toBe('https://app.example.com/welcome')
      expect(hasPendingInviteCallback()).toBe(true)
      expect(consumeInviteCallback()).toBe('https://app.example.com/welcome')
      expect(consumeInviteCallback()).toBeNull()
      expect(hasPendingInviteCallback()).toBe(false)
    })

    it('never stores or resumes a callback that fails the guard', () => {
      expect(rememberInviteCallback('http://evil.test/steal')).toBeNull()
      expect(hasPendingInviteCallback()).toBe(false)

      sessionStorage.setItem('maintainerd_auth_invite_callback', 'javascript:alert(1)')
      expect(hasPendingInviteCallback()).toBe(false)
      expect(consumeInviteCallback()).toBeNull()
    })
  })

  it('never injects tenant context — the tenant is resolved from the subdomain', () => {
    const route = oauthLoginRoute('/device', '?user_code=ABCD-EFGH')
    const url = new URL(route, window.location.origin)

    expect(url.pathname).toBe('/login')
    expect(url.searchParams.get('tenant_id')).toBeNull()
    expect(url.searchParams.get('return_to')).toBe('/device?user_code=ABCD-EFGH')
  })
})
