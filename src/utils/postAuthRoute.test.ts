import { describe, expect, it } from 'vitest'
import { resolveGuardRedirect } from './postAuthRoute'

describe('resolveGuardRedirect OAuth broker routing', () => {
  const unauthenticated = {
    isAuthenticated: false,
    account: null,
    tenant: null,
  }
  const authenticated = {
    isAuthenticated: true,
    account: {
      user_id: 'user-1',
      email: 'user@example.com',
      phone: '',
      email_verified: true,
      phone_verified: false,
      profiles: [{ profile_id: 'profile-1', first_name: 'Test', last_name: null, display_name: 'Test', default: true }],
      roles: [],
      permissions: [],
      tenant: { tenant_id: 'tenant-1', name: 'system', display_name: 'System', identifier: 'system' },
    },
    tenant: {
      tenant_id: 'tenant-1',
      name: 'system',
      display_name: 'System',
      description: '',
      identifier: 'system',
      status: 'active' as const,
      is_default: true,
      is_system: true,
      created_at: '',
      updated_at: '',
    },
  }

  it('allows broker authorize requests with a hint to reach the backend before login', () => {
    expect(resolveGuardRedirect({
      ...unauthenticated,
      pathname: '/authorize',
      search: '?client_id=external&response_type=code&idp_hint=google',
    })).toBeNull()
  })

  it('routes regular authorize requests through the built-in login flow', () => {
    const redirect = resolveGuardRedirect({
      ...unauthenticated,
      pathname: '/authorize',
      search: '?client_id=external&response_type=code',
    })

    expect(redirect).toBe('/login?client_id=external&response_type=code&return_to=%2Fauthorize%3Fclient_id%3Dexternal%26response_type%3Dcode')
  })

  it('allows authenticated users to resume authorize instead of no-access', () => {
    expect(resolveGuardRedirect({
      ...authenticated,
      pathname: '/authorize',
      search: '?client_id=external&response_type=code',
    })).toBeNull()
  })

  it('allows authenticated users to complete OAuth consent instead of no-access', () => {
    expect(resolveGuardRedirect({
      ...authenticated,
      pathname: '/oauth/consent/challenge-1',
      search: '',
    })).toBeNull()
  })

  it('blocks direct registration when the client registration gate is disabled', () => {
    expect(resolveGuardRedirect({
      ...unauthenticated,
      pathname: '/register',
      registrationEnabled: false,
    })).toBe('/login')
  })

  it('routes unverified accounts through verification when the client flow requires it', () => {
    expect(resolveGuardRedirect({
      ...authenticated,
      account: { ...authenticated.account, email_verified: false },
      pathname: '/register/profile',
      verificationRequired: true,
    })).toBe('/email-verification')
  })

  it('sends an already-authenticated user landing on / to the account dashboard', () => {
    expect(resolveGuardRedirect({ ...authenticated, pathname: '/' })).toBe('/account')
  })

  it('sends an already-authenticated user on the login page to the account dashboard', () => {
    expect(resolveGuardRedirect({ ...authenticated, pathname: '/login' })).toBe('/account')
  })

  it('lets an authenticated user render the account dashboard without redirecting', () => {
    expect(resolveGuardRedirect({ ...authenticated, pathname: '/account' })).toBeNull()
  })

  it('still renders login-success (it performs OAuth/invite continuation itself)', () => {
    expect(resolveGuardRedirect({ ...authenticated, pathname: '/login-success' })).toBeNull()
  })

  it('lets an anonymous invitee render the invite registration page', () => {
    expect(resolveGuardRedirect({
      ...unauthenticated,
      pathname: '/register/invite',
      search: '?client_id=abc&email=invitee%40example.com',
    })).toBeNull()
  })

  it('renders the invite page even for an already-signed-in user (does not bounce to the dashboard)', () => {
    expect(resolveGuardRedirect({
      ...authenticated,
      pathname: '/register/invite',
      search: '?client_id=abc&email=invitee%40example.com',
    })).toBeNull()
  })

  describe('finishing a registration detour with a pending continuation', () => {
    it('routes a completed user off /register/profile to /login-success when a continuation is pending', () => {
      expect(resolveGuardRedirect({
        ...authenticated,
        pathname: '/register/profile',
        pendingContinuation: true,
      })).toBe('/login-success')
    })

    it('routes a completed user off /register/profile to /account when no continuation is pending', () => {
      expect(resolveGuardRedirect({
        ...authenticated,
        pathname: '/register/profile',
        pendingContinuation: false,
      })).toBe('/account')
    })

    it('defaults to /account off /register/profile when pendingContinuation is unset', () => {
      expect(resolveGuardRedirect({
        ...authenticated,
        pathname: '/register/profile',
      })).toBe('/account')
    })

    it('routes a completed user off /email-verification to /login-success when a continuation is pending', () => {
      expect(resolveGuardRedirect({
        ...authenticated,
        pathname: '/email-verification',
        pendingContinuation: true,
      })).toBe('/login-success')
    })

    it('routes a completed user off /email-verification to /account when no continuation is pending', () => {
      expect(resolveGuardRedirect({
        ...authenticated,
        pathname: '/email-verification',
        pendingContinuation: false,
      })).toBe('/account')
    })
  })

  describe('threads the request_id handle through guard redirects', () => {
    it('carries request_id onto the login-success convergence for a completed detour', () => {
      expect(resolveGuardRedirect({
        ...authenticated,
        pathname: '/register/profile',
        search: '?request_id=abc',
        pendingContinuation: true,
      })).toBe('/login-success?request_id=abc')
    })

    it('carries request_id onto the next detour hop (verify → profile)', () => {
      expect(resolveGuardRedirect({
        ...authenticated,
        account: { ...authenticated.account, profiles: [] },
        pathname: '/email-verification',
        search: '?request_id=abc',
        pendingContinuation: true,
      })).toBe('/register/profile?request_id=abc')
    })

    it('does not thread request_id onto the dashboard (no pending continuation)', () => {
      expect(resolveGuardRedirect({
        ...authenticated,
        pathname: '/register/profile',
        search: '?request_id=abc',
        pendingContinuation: false,
      })).toBe('/account')
    })
  })
})
