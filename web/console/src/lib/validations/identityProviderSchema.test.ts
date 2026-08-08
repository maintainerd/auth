import { describe, it, expect } from 'vitest'
import { identityProviderSchema } from './identityProviderSchema'

// These tests lock the frontend rules to the backend
// (internal/idp/validation_provider.go). If a field is required/format-checked
// there, it must be here too — keep the two in lock-step.

// A valid active external (Google) provider: issuer + client id are required
// once such a provider is active.
function validExternal(overrides: Record<string, unknown> = {}) {
  return {
    name: 'my-idp',
    displayName: 'My Identity Provider',
    provider: 'google',
    status: 'active',
    issuer: 'https://accounts.google.com',
    clientId: 'test-client',
    ...overrides,
  }
}

const isValid = (obj: Record<string, unknown>) => identityProviderSchema.isValid(obj)

describe('identityProviderSchema — backend parity', () => {
  it('accepts a valid active external provider', async () => {
    expect(await isValid(validExternal())).toBe(true)
  })

  describe('name (mirrors idpNamePattern + length 3–50)', () => {
    it('rejects uppercase', async () => {
      expect(await isValid(validExternal({ name: 'MyIdp' }))).toBe(false)
    })
    it('rejects colon (allowed for roles, not IdP)', async () => {
      expect(await isValid(validExternal({ name: 'my:idp' }))).toBe(false)
    })
    it('rejects a leading or trailing hyphen', async () => {
      expect(await isValid(validExternal({ name: '-myidp' }))).toBe(false)
      expect(await isValid(validExternal({ name: 'myidp-' }))).toBe(false)
    })
    it('rejects fewer than 3 characters', async () => {
      expect(await isValid(validExternal({ name: 'ab' }))).toBe(false)
    })
    it('accepts lowercase, digits and interior hyphens', async () => {
      expect(await isValid(validExternal({ name: 'my-idp-2' }))).toBe(true)
    })
  })

  describe('displayName (length 2–100)', () => {
    it('rejects fewer than 2 characters', async () => {
      expect(await isValid(validExternal({ displayName: 'a' }))).toBe(false)
    })
    it('rejects more than 100 characters', async () => {
      expect(await isValid(validExternal({ displayName: 'x'.repeat(101) }))).toBe(false)
    })
    it('accepts a short brand name like "Okta"', async () => {
      expect(await isValid(validExternal({ displayName: 'Okta' }))).toBe(true)
    })
  })

  describe('issuer / clientId — required for active external providers', () => {
    it('rejects a missing issuer', async () => {
      expect(await isValid(validExternal({ issuer: '' }))).toBe(false)
    })
    it('rejects a non-URL issuer', async () => {
      expect(await isValid(validExternal({ issuer: 'not-a-url' }))).toBe(false)
    })
    it('rejects a plain http:// issuer (HTTPS enforcement)', async () => {
      expect(await isValid(validExternal({ issuer: 'http://provider.com' }))).toBe(false)
    })
    it('accepts an https:// issuer', async () => {
      // gitlab is a variable-domain OIDC provider (self-managed), so its issuer
      // host is not bound — the right vehicle for a generic https issuer check.
      expect(await isValid(validExternal({ provider: 'gitlab', issuer: 'https://provider.com' }))).toBe(true)
    })
    it('accepts http://localhost for local dev', async () => {
      expect(await isValid(validExternal({ provider: 'gitlab', issuer: 'http://localhost:8080' }))).toBe(true)
    })
    it('rejects a missing client id', async () => {
      expect(await isValid(validExternal({ clientId: '' }))).toBe(false)
    })
    it('allows a draft (inactive) external provider without creds', async () => {
      expect(await isValid(validExternal({ status: 'inactive', issuer: '', clientId: '' }))).toBe(true)
    })
  })

  describe('per-provider host binding (mirrors backend requireAllowedHost)', () => {
    it('rejects a google provider whose issuer is not accounts.google.com', async () => {
      expect(await isValid(validExternal({ provider: 'google', issuer: 'https://evil.com' }))).toBe(false)
    })
    it('accepts a google provider with its official issuer', async () => {
      expect(await isValid(validExternal({ provider: 'google', issuer: 'https://accounts.google.com' }))).toBe(true)
    })
    it('accepts a cognito provider with a regional issuer host', async () => {
      expect(await isValid(validExternal({
        provider: 'cognito',
        issuer: 'https://cognito-idp.us-east-1.amazonaws.com/us-east-1_AbC123',
      }))).toBe(true)
    })
    it('rejects a cognito provider with a bogus issuer host', async () => {
      expect(await isValid(validExternal({ provider: 'cognito', issuer: 'https://evil.com/us-east-1_AbC123' }))).toBe(false)
    })
    it('does not bind the host for variable-domain providers (auth0 custom domain)', async () => {
      expect(await isValid(validExternal({ provider: 'auth0', issuer: 'https://login.acme.example' }))).toBe(true)
    })
  })

  describe('issuer requiredness follows provider class (OIDC vs OAuth2-only)', () => {
    it('requires issuer for an active OIDC provider (google)', async () => {
      expect(await isValid(validExternal({ provider: 'google', issuer: '' }))).toBe(false)
    })
    it('does NOT require issuer for an active OAuth2-only provider (github)', async () => {
      // github has no OIDC discovery; its OAuth2 endpoints live in config, so the
      // top-level issuer is optional. Client id is still required.
      expect(
        await isValid({
          name: 'my-github',
          displayName: 'My GitHub',
          provider: 'github',
          status: 'active',
          clientId: 'gh-client',
        }),
      ).toBe(true)
    })
    it('still requires client id for an active OAuth2-only provider (github)', async () => {
      expect(
        await isValid({
          name: 'my-github',
          displayName: 'My GitHub',
          provider: 'github',
          status: 'active',
          clientId: '',
        }),
      ).toBe(false)
    })
    it('applies the same rule to facebook and twitter (issuer optional)', async () => {
      for (const provider of ['facebook', 'twitter']) {
        expect(
          await isValid({
            name: `my-${provider}`,
            displayName: `My ${provider}`,
            provider,
            status: 'active',
            clientId: `${provider}-client`,
          }),
        ).toBe(true)
      }
    })
  })

  describe('token federation — issuer + audiences required when active', () => {
    it('requires at least one audience', async () => {
      expect(
        await isValid(validExternal({ allowTokenFederation: true, allowedAudiences: '' })),
      ).toBe(false)
    })
    it('accepts token federation with an audience', async () => {
      expect(
        await isValid(validExternal({ allowTokenFederation: true, allowedAudiences: 'app-1' })),
      ).toBe(true)
    })
  })

  describe('email domains (mirrors backend is.Domain)', () => {
    it('rejects an invalid domain', async () => {
      expect(await isValid(validExternal({ emailDomains: 'bad domain!' }))).toBe(false)
    })
    it('accepts valid comma/space separated domains', async () => {
      expect(await isValid(validExternal({ emailDomains: 'example.com, sub.example.org' }))).toBe(true)
    })
    it('accepts space-only separated domains (tokenizer parity with submit)', async () => {
      // Previously the yup tokenizer split on whitespace but the form's submit
      // tokenizer did not, so this validated on the FE and then submitted as one
      // invalid token. Both now split on /[\s,\n]+/, so it is genuinely valid.
      expect(await isValid(validExternal({ emailDomains: 'example.com sub.example.org' }))).toBe(true)
    })
  })

  describe('a user-created maintainerd provider is external OIDC federation', () => {
    it('requires issuer + client id for an active maintainerd provider', async () => {
      expect(await isValid({
        name: 'acme-maintainerd',
        displayName: 'Acme Maintainerd',
        provider: 'maintainerd',
        status: 'active',
      })).toBe(false)
    })
    it('accepts an active maintainerd provider with issuer + client id', async () => {
      expect(await isValid({
        name: 'acme-maintainerd',
        displayName: 'Acme Maintainerd',
        provider: 'maintainerd',
        status: 'active',
        issuer: 'https://auth.acme.example',
        clientId: 'acme-client',
      })).toBe(true)
    })
  })

  describe('SAML is not an OIDC external provider', () => {
    it('does not require issuer/clientId for an active SAML provider', async () => {
      expect(await isValid({
        name: 'corp-saml',
        displayName: 'Corporate SAML',
        provider: 'saml',
        status: 'active',
      })).toBe(true)
    })
  })
})
