import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useProviderConfig } from './useProviderConfig'

describe('useProviderConfig — scopes', () => {
  it('seeds the provider default scopes on create', () => {
    const { result } = renderHook(() => useProviderConfig('google'))
    expect(result.current.values.scopes).toBe('openid, profile, email')
  })

  it('serializes scopes to a string array in buildConfig', () => {
    const { result } = renderHook(() => useProviderConfig('google'))
    expect(result.current.buildConfig().scopes).toEqual(['openid', 'profile', 'email'])
  })

  it('rejects rubbish scope tokens on validate', () => {
    const { result } = renderHook(() => useProviderConfig('google'))
    act(() => result.current.setFieldValue('scopes', 'openid, bad scope!'))
    let ok = true
    act(() => { ok = result.current.validate() })
    expect(ok).toBe(false)
    expect(result.current.errors.scopes).toMatch(/invalid scope/i)
  })

  it('accepts valid scopes including namespaced and URL scopes', () => {
    const { result } = renderHook(() => useProviderConfig('google'))
    act(() =>
      result.current.setFieldValue(
        'scopes',
        'openid, read:user, https://www.googleapis.com/auth/userinfo.email',
      ),
    )
    let ok = false
    act(() => { ok = result.current.validate() })
    expect(ok).toBe(true)
    expect(result.current.errors.scopes ?? '').toBe('')
  })

  it('does not seed defaults over an existing config on load (edit)', () => {
    const { result } = renderHook(() => useProviderConfig('google'))
    act(() => result.current.load({ scopes: ['openid'] }, 'google'))
    expect(result.current.values.scopes).toBe('openid')
  })

  it('drops unknown keys from a stored config on load (backend rejects them)', () => {
    // A legacy/stray key must never be surfaced or re-emitted — the backend now
    // rejects unknown config keys.
    const { result } = renderHook(() => useProviderConfig('google'))
    act(() => result.current.load({ scopes: ['openid'], hosted_domain: 'example.com' }, 'google'))
    expect(result.current.values.hosted_domain).toBeUndefined()
    expect(result.current.buildConfig().hosted_domain).toBeUndefined()
  })
})

describe('useProviderConfig — OAuth2-only providers', () => {
  it('seeds the endpoint and scope defaults on create (github)', () => {
    const { result } = renderHook(() => useProviderConfig('github'))
    expect(result.current.values.scopes).toBe('read:user, user:email')
    expect(result.current.values.authorization_endpoint).toBe('https://github.com/login/oauth/authorize')
    expect(result.current.values.token_endpoint).toBe('https://github.com/login/oauth/access_token')
    expect(result.current.values.userinfo_endpoint).toBe('https://api.github.com/user')
  })

  it('emits the seeded endpoints in buildConfig (github)', () => {
    const { result } = renderHook(() => useProviderConfig('github'))
    const config = result.current.buildConfig()
    expect(config.authorization_endpoint).toBe('https://github.com/login/oauth/authorize')
    expect(config.token_endpoint).toBe('https://github.com/login/oauth/access_token')
    expect(config.userinfo_endpoint).toBe('https://api.github.com/user')
    expect(config.scopes).toEqual(['read:user', 'user:email'])
  })

  it('fails validate when a required endpoint is cleared (github)', () => {
    const { result } = renderHook(() => useProviderConfig('github'))
    act(() => result.current.setFieldValue('token_endpoint', ''))
    let ok = true
    act(() => { ok = result.current.validate() })
    expect(ok).toBe(false)
    expect(result.current.errors.token_endpoint).toMatch(/required/i)
  })
})

describe('useProviderConfig — URL fields enforce https (parity with backend)', () => {
  it('rejects a plain http:// endpoint (github)', () => {
    const { result } = renderHook(() => useProviderConfig('github'))
    act(() => result.current.setFieldValue('token_endpoint', 'http://provider.com/token'))
    let ok = true
    act(() => { ok = result.current.validate() })
    expect(ok).toBe(false)
    expect(result.current.errors.token_endpoint).toMatch(/https/i)
  })

  it('accepts https:// endpoints (github seeded defaults)', () => {
    const { result } = renderHook(() => useProviderConfig('github'))
    let ok = false
    act(() => { ok = result.current.validate() })
    expect(ok).toBe(true)
    expect(result.current.errors.token_endpoint ?? '').toBe('')
  })

  it('rejects a github endpoint on a non-github host (host binding)', () => {
    // OAuth2-only endpoints are host-bound: a bogus token_endpoint would
    // exfiltrate the client secret, so it must be a github.com / api.github.com host.
    const { result } = renderHook(() => useProviderConfig('github'))
    act(() => result.current.setFieldValue('token_endpoint', 'https://evil.com/token'))
    let ok = true
    act(() => { ok = result.current.validate() })
    expect(ok).toBe(false)
    expect(result.current.errors.token_endpoint).toMatch(/not permitted|github/i)
  })
})

const VALID_PEM_CERT =
  '-----BEGIN CERTIFICATE-----\nMIIBfakeBase64Data123==\n-----END CERTIFICATE-----'

describe('useProviderConfig — SAML certificate PEM shape check', () => {
  it('rejects a certificate that is not a PEM block', () => {
    const { result } = renderHook(() => useProviderConfig('saml'))
    act(() => {
      result.current.setFieldValue('sso_url', 'https://idp.example.com/sso')
      result.current.setFieldValue('entity_id', 'https://idp.example.com/metadata')
      result.current.setFieldValue('certificate', 'not-a-certificate')
    })
    let ok = true
    act(() => { ok = result.current.validate() })
    expect(ok).toBe(false)
    expect(result.current.errors.certificate).toMatch(/pem/i)
  })

  it('accepts a well-formed PEM certificate', () => {
    const { result } = renderHook(() => useProviderConfig('saml'))
    act(() => {
      result.current.setFieldValue('sso_url', 'https://idp.example.com/sso')
      result.current.setFieldValue('entity_id', 'https://idp.example.com/metadata')
      result.current.setFieldValue('certificate', VALID_PEM_CERT)
    })
    let ok = false
    act(() => { ok = result.current.validate() })
    expect(ok).toBe(true)
    expect(result.current.errors.certificate ?? '').toBe('')
  })
})

describe('useProviderConfig — inactive gating (presence checks only when active)', () => {
  it('allows a blank required config field when saving inactive (saml)', () => {
    // SAML sso_url/entity_id/certificate are required, but a draft (inactive)
    // provider may be saved with them blank — matching the backend.
    const { result } = renderHook(() => useProviderConfig('saml'))
    let ok = false
    act(() => { ok = result.current.validate('inactive') })
    expect(ok).toBe(true)
  })

  it('still fails an active provider with those fields blank (saml)', () => {
    const { result } = renderHook(() => useProviderConfig('saml'))
    let ok = true
    act(() => { ok = result.current.validate('active') })
    expect(ok).toBe(false)
    expect(result.current.errors.sso_url).toMatch(/required/i)
  })

  it('still format-checks a present value even when inactive (https)', () => {
    // Format checks are status-independent: an http:// endpoint is rejected even
    // for a draft provider.
    const { result } = renderHook(() => useProviderConfig('github'))
    act(() => result.current.setFieldValue('token_endpoint', 'http://provider.com/token'))
    let ok = true
    act(() => { ok = result.current.validate('inactive') })
    expect(ok).toBe(false)
    expect(result.current.errors.token_endpoint).toMatch(/https/i)
  })
})

describe('useProviderConfig — list tokenizer round-trips space-separated values', () => {
  it('splits whitespace/comma/newline into separate tokens on buildConfig (scopes)', () => {
    // The parseList tokenizer (/[\s,\n]+/) is shared with the form submit and the
    // yup emailDomains tokenizer, so a space-separated value round-trips to
    // multiple tokens rather than one invalid token.
    const { result } = renderHook(() => useProviderConfig('google'))
    act(() => result.current.setFieldValue('scopes', 'openid profile\nemail, offline_access'))
    expect(result.current.buildConfig().scopes).toEqual([
      'openid',
      'profile',
      'email',
      'offline_access',
    ])
  })
})

describe('useProviderConfig — switching providers', () => {
  it('carries customized scopes across two providers that both define them', () => {
    const { result, rerender } = renderHook(({ p }) => useProviderConfig(p), {
      initialProps: { p: 'google' },
    })
    act(() => result.current.setFieldValue('scopes', 'openid, custom:scope'))
    rerender({ p: 'auth0' })
    expect(result.current.values.scopes).toBe('openid, custom:scope')
  })

  it('drops non-shared keys the target provider does not define', () => {
    // SAML defines sso_url; Google does not — switching must drop it and seed
    // Google's own defaults, never leaving stray keys behind.
    const { result, rerender } = renderHook(({ p }) => useProviderConfig(p), {
      initialProps: { p: 'saml' },
    })
    act(() => result.current.setFieldValue('sso_url', 'https://idp.example.com/sso'))
    rerender({ p: 'google' })
    expect(result.current.values.sso_url).toBeUndefined()
    // Shared/default scopes for the new provider are seeded.
    expect(result.current.values.scopes).toBe('openid, profile, email')
  })
})
