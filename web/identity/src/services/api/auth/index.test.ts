import { beforeEach, describe, expect, it, vi } from 'vitest'
import { post } from '../client'
import { TOKEN_DELIVERY_HEADER } from '../config'
import {
  beginMFALoginWebAuthn,
  sendMagicLink,
  sendMFALoginEmailOtp,
  sendMFALoginSMS,
  verifyMagicLink,
  verifyMFALogin,
} from './index'

vi.mock('../client', () => ({
  get: vi.fn(),
  post: vi.fn(),
}))

describe('verifyMagicLink', () => {
  beforeEach(() => {
    vi.mocked(post).mockReset()
  })

  it('forwards the complete signed query to the verification endpoint', async () => {
    const signedQuery = 'token=opaque&client_id=identity&expires=1234567890&sig=signed-value%3D'
    vi.mocked(post).mockResolvedValue({ success: true })

    await verifyMagicLink(signedQuery)

    expect(post).toHaveBeenCalledWith(
      `/magic-link/verify?${signedQuery}`,
      {},
      { headers: TOKEN_DELIVERY_HEADER },
    )
  })
})

describe('sendMagicLink', () => {
  beforeEach(() => {
    vi.mocked(post).mockReset()
    vi.mocked(post).mockResolvedValue({ success: true })
  })

  it('uses only the public client context', async () => {
    await sendMagicLink('user@example.com', { clientId: 'identity' })

    expect(post).toHaveBeenCalledWith('/magic-link/send?client_id=identity', {
      email: 'user@example.com',
    })
  })

  it('uses tenant context for first-party identity links', async () => {
    await sendMagicLink('user@example.com', { tenantId: 'acme' })

    expect(post).toHaveBeenCalledWith('/magic-link/send?tenant_id=acme', {
      email: 'user@example.com',
    })
  })
})

describe('verifyMFALogin', () => {
  beforeEach(() => {
    vi.mocked(post).mockReset()
  })

  it('preserves client context for a magic-link MFA challenge', async () => {
    vi.mocked(post).mockResolvedValueOnce({ success: true })
    const request = { mfa_challenge_token: 'challenge', method: 'totp', code: '123456' }

    await verifyMFALogin(request, undefined, 'identity-client')

    expect(post).toHaveBeenCalledWith(
      '/login/mfa/verify?client_id=identity-client',
      request,
      { headers: TOKEN_DELIVERY_HEADER },
    )
  })

  it('preserves tenant context for a first-party MFA challenge', async () => {
    vi.mocked(post).mockResolvedValueOnce({ success: true })
    const request = { mfa_challenge_token: 'challenge', method: 'totp', code: '123456' }

    await verifyMFALogin(request, 'acme')

    expect(post).toHaveBeenCalledWith(
      '/login/mfa/verify?tenant_id=acme',
      request,
      { headers: TOKEN_DELIVERY_HEADER },
    )
  })
})

describe('login MFA challenge helpers', () => {
  beforeEach(() => {
    vi.mocked(post).mockReset()
    vi.mocked(post).mockResolvedValue({ success: true, data: { publicKey: {} } })
  })

  it('sends SMS with tenant context', async () => {
    await sendMFALoginSMS('challenge', { tenantId: 'acme' })

    expect(post).toHaveBeenCalledWith('/login/mfa/send-sms?tenant_id=acme', {
      mfa_challenge_token: 'challenge',
    })
  })

  it('sends email OTP with client context', async () => {
    await sendMFALoginEmailOtp('challenge', { clientId: 'identity-client' })

    expect(post).toHaveBeenCalledWith(
      '/login/mfa/send-email-otp?client_id=identity-client',
      { mfa_challenge_token: 'challenge' },
      { headers: TOKEN_DELIVERY_HEADER },
    )
  })

  it('begins WebAuthn with tenant context', async () => {
    await beginMFALoginWebAuthn('challenge', { tenantId: 'acme' })

    expect(post).toHaveBeenCalledWith('/login/mfa/webauthn/begin?tenant_id=acme', {
      mfa_challenge_token: 'challenge',
    })
  })
})
