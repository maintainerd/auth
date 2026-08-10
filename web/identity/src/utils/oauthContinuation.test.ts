import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { AccountEntity } from '@/services/api/auth/types'
import { finishAuthStep, resumeOAuthContinuation } from './oauthContinuation'
import { rememberOAuthReturnTo } from './oauthRedirect'

const { continueOAuthMock } = vi.hoisted(() => ({ continueOAuthMock: vi.fn() }))

vi.mock('@/services/api/oauth', () => ({ continueOAuth: continueOAuthMock }))

const complete = {
  email_verified: true,
  profiles: [{ profile_id: 'p1', display_name: 'Test', default: true }],
} as unknown as AccountEntity

const needsProfile = { email_verified: true, profiles: [] } as unknown as AccountEntity
const needsVerify = { email_verified: false, profiles: [] } as unknown as AccountEntity

let assignMock: ReturnType<typeof vi.fn>
const originalLocation = window.location
const testOrigin = window.location.origin

beforeEach(() => {
  sessionStorage.clear()
  continueOAuthMock.mockReset()
  assignMock = vi.fn()
  // jsdom's location.assign is non-configurable, so replace the whole location
  // object with a configurable stub, preserving origin for safeOAuthReturnTo.
  Object.defineProperty(window, 'location', {
    configurable: true,
    value: { origin: testOrigin, href: `${testOrigin}/login-success`, assign: assignMock },
  })
})

afterEach(() => {
  Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
  vi.restoreAllMocks()
  sessionStorage.clear()
})

describe('finishAuthStep', () => {
  it('routes an incomplete account (no profile) to the profile step, threading request_id', () => {
    const navigate = vi.fn()
    const outcome = finishAuthStep({ account: needsProfile, requestId: 'abc', navigate })
    expect(outcome).toBe('detour')
    expect(navigate).toHaveBeenCalledWith('/register/profile?request_id=abc', { replace: true })
  })

  it('routes an unverified account to email verification, threading request_id', () => {
    const navigate = vi.fn()
    const outcome = finishAuthStep({ account: needsVerify, requestId: 'abc', verificationRequired: true, navigate })
    expect(outcome).toBe('detour')
    expect(navigate).toHaveBeenCalledWith('/email-verification?request_id=abc', { replace: true })
  })

  it('converges a fully-registered account with a request_id on login-success (threaded)', () => {
    const navigate = vi.fn()
    const outcome = finishAuthStep({ account: complete, requestId: 'abc', navigate })
    expect(outcome).toBe('continued')
    expect(navigate).toHaveBeenCalledWith('/login-success?request_id=abc', { replace: true })
  })

  // The console (and every external RP) sends the browser to /authorize. The
  // guard bounces an anonymous user to /login BEFORE the backend is called, so a
  // request_id can never exist on this path — the return_to is the only handle.
  // Clearing it here stranded every SSO login on the identity account page.
  it('resumes a guard-bounced authorize using return_to when no request_id exists', () => {
    const navigate = vi.fn()
    const outcome = finishAuthStep({
      account: complete,
      returnTo: '/authorize?client_id=console&redirect_uri=https%3A%2F%2Fconsole.example%2Fauth%2Fcallback',
      navigate,
    })

    expect(outcome).toBe('continued')
    expect(navigate).toHaveBeenCalledWith('/login-success', { replace: true })
    // Left armed for LoginSuccessPage to consume and send the user back.
    expect(sessionStorage.getItem('maintainerd_auth_oauth_return_to')).toContain('/authorize')
  })

  // The broker's email-collision page bounces an anonymous user to /login with
  // return_to pointing back at /account-link — its confirmation token and
  // broker_session live only in that query string. /account-link is not an
  // OAuth interaction route, so without the dedicated branch the return_to was
  // dropped and the accounts were never linked.
  it('returns to a pending /account-link confirmation after sign-in', () => {
    const navigate = vi.fn()
    const returnTo = '/account-link?token=tok-1&provider=cognito&email=a%40b.c&broker_session=bs-1'
    const outcome = finishAuthStep({ account: complete, returnTo, navigate })

    expect(outcome).toBe('continued')
    expect(navigate).toHaveBeenCalledWith(returnTo, { replace: true })
  })

  it('ignores an unsafe return_to rather than following it', () => {
    const navigate = vi.fn()
    const outcome = finishAuthStep({
      account: complete,
      returnTo: 'https://evil.example/steal',
      navigate,
    })

    expect(outcome).toBe('dashboard')
    expect(sessionStorage.getItem('maintainerd_auth_oauth_return_to')).toBeNull()
  })

  it('sends a fully-registered account with NO request_id to the dashboard landing and clears stale markers', () => {
    rememberOAuthReturnTo('/authorize?client_id=stale')
    const navigate = vi.fn()
    const outcome = finishAuthStep({ account: complete, navigate })
    expect(outcome).toBe('dashboard')
    expect(navigate).toHaveBeenCalledWith('/login-success', { replace: true })
    // Stale return-to cleared so it can't hijack the direct login at login-success.
    expect(sessionStorage.getItem('maintainerd_auth_oauth_return_to')).toBeNull()
  })
})

describe('resumeOAuthContinuation', () => {
  it('resumes the authorize request and redirects to the callback (request_id primary)', async () => {
    continueOAuthMock.mockResolvedValueOnce({ redirect_uri: 'https://app.example/cb?code=xyz' })
    const navigate = vi.fn()
    const handled = await resumeOAuthContinuation('abc', navigate)
    expect(continueOAuthMock).toHaveBeenCalledWith('abc')
    expect(assignMock).toHaveBeenCalledWith('https://app.example/cb?code=xyz')
    expect(handled).toBe(true)
  })

  it('navigates to the consent screen when the authorize needs consent', async () => {
    continueOAuthMock.mockResolvedValueOnce({ consent_challenge: 'chal-1' })
    const navigate = vi.fn()
    const handled = await resumeOAuthContinuation('abc', navigate)
    expect(navigate).toHaveBeenCalledWith('/oauth/consent/chal-1', { replace: true })
    expect(handled).toBe(true)
  })

  it('falls back to the legacy sessionStorage return-to when there is no request_id', async () => {
    rememberOAuthReturnTo('/authorize?client_id=external')
    const navigate = vi.fn()
    const handled = await resumeOAuthContinuation(undefined, navigate)
    expect(continueOAuthMock).not.toHaveBeenCalled()
    expect(navigate).toHaveBeenCalledWith('/authorize?client_id=external', { replace: true })
    expect(handled).toBe(true)
  })

  it('returns false (→ dashboard) when there is nothing to resume', async () => {
    const navigate = vi.fn()
    const handled = await resumeOAuthContinuation(undefined, navigate)
    expect(handled).toBe(false)
    expect(navigate).not.toHaveBeenCalled()
  })

  it('falls back to the return-to when continueOAuth fails, else reports nothing to resume', async () => {
    continueOAuthMock.mockRejectedValueOnce(new Error('expired'))
    const navigate = vi.fn()
    const handled = await resumeOAuthContinuation('abc', navigate)
    expect(handled).toBe(false)
    expect(navigate).not.toHaveBeenCalled()
  })
})
