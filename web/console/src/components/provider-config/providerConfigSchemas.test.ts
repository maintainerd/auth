import { describe, it, expect } from 'vitest'
import {
  getProviderFieldKeys,
  isOAuth2OnlyProvider,
} from './providerConfigSchemas'

// These schemas must stay aligned with the strict, generic OIDC/OAuth2 backend:
// only OAuth scopes and OAuth2 endpoints are valid config keys. Per-provider
// "directory" fields the backend never reads have been removed, and the backend
// now rejects unknown config keys — so their presence here would break saves.

describe('providerConfigSchemas — dead config fields removed', () => {
  const DEAD_FIELDS: Record<string, string[]> = {
    cognito: ['region', 'user_pool_id', 'domain'],
    auth0: ['domain', 'audience', 'connection'],
    microsoft: ['tenant'],
    google: ['hosted_domain'],
    github: ['enterprise_base_url'],
    gitlab: ['base_url'],
    facebook: ['api_version'],
    twitter: ['api_version'],
  }

  for (const [provider, dead] of Object.entries(DEAD_FIELDS)) {
    it(`drops [${dead.join(', ')}] from ${provider}`, () => {
      const keys = getProviderFieldKeys(provider)
      for (const field of dead) {
        expect(keys).not.toContain(field)
      }
    })
  }

  it('cognito is OIDC — scopes only, no endpoint overrides (backend rejects them)', () => {
    const keys = getProviderFieldKeys('cognito')
    expect(keys).not.toContain('region')
    // OIDC providers derive endpoints from discovery; the backend rejects any
    // endpoint override, so the form must not collect them (nor the dead jwks_uri).
    expect(keys).not.toContain('jwks_uri')
    expect(keys).not.toContain('authorization_endpoint')
    expect(keys).not.toContain('token_endpoint')
    expect(keys).not.toContain('userinfo_endpoint')
    expect(keys).toEqual(['scopes'])
  })
})

describe('providerConfigSchemas — OAuth2-only providers require endpoints', () => {
  for (const provider of ['github', 'facebook', 'twitter']) {
    it(`${provider} is OAuth2-only and its three endpoints are required`, () => {
      expect(isOAuth2OnlyProvider(provider)).toBe(true)
      const keys = getProviderFieldKeys(provider)
      expect(keys).toEqual(['scopes', 'authorization_endpoint', 'token_endpoint', 'userinfo_endpoint'])
    })
  }

  it('OIDC providers are not OAuth2-only', () => {
    for (const provider of ['google', 'microsoft', 'cognito', 'auth0', 'gitlab', 'linkedin', 'maintainerd']) {
      expect(isOAuth2OnlyProvider(provider)).toBe(false)
    }
  })
})
