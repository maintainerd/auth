import { beforeEach, describe, expect, it, vi } from 'vitest'
import { deleteRequest, get, post } from '@/services/api/client'
import {
  approveOAuthCIBA,
  approveOAuthDevice,
  authorizeOAuth,
  fetchOAuthConnections,
  fetchOAuthConsentChallenge,
  oauthEndSessionURL,
  revokeOAuthConsentGrant,
  submitOAuthConsent,
} from './index'

vi.mock('@/services/api/client', () => ({
  deleteRequest: vi.fn(),
  get: vi.fn(),
  post: vi.fn(),
}))

describe('oauth api', () => {
  beforeEach(() => {
    vi.mocked(deleteRequest).mockReset()
    vi.mocked(get).mockReset()
    vi.mocked(post).mockReset()
  })

  it('forwards the authorize query exactly once', async () => {
    vi.mocked(get).mockResolvedValueOnce({
      success: true,
      data: { consent_challenge: 'challenge-1' },
    })

    const result = await authorizeOAuth('?client_id=external&redirect_uri=https%3A%2F%2Fapp.example%2Fcb')

    expect(result).toEqual({ consent_challenge: 'challenge-1' })
    expect(get).toHaveBeenCalledWith('/oauth/authorize?client_id=external&redirect_uri=https%3A%2F%2Fapp.example%2Fcb')
  })

  it('loads public OAuth connections for a client', async () => {
    vi.mocked(get).mockResolvedValueOnce({
      success: true,
      data: {
        password_enabled: true,
        registration_enabled: true,
        connections: [
          {
            identifier: 'google',
            display_name: 'Google',
            provider: 'google',
            provider_type: 'oidc',
            is_default: true,
            display_order: 0,
          },
        ],
      },
    })

    const result = await fetchOAuthConnections('external/client')

    expect(result.connections[0].identifier).toBe('google')
    expect(result.registration_enabled).toBe(true)
    expect(get).toHaveBeenCalledWith('/oauth/connections?client_id=external%2Fclient')
  })

  it('loads a consent challenge with an encoded path segment', async () => {
    vi.mocked(get).mockResolvedValueOnce({
      success: true,
      data: {
        challenge_id: 'challenge/1',
        client_name: 'Calendar',
        client_uuid: 'client-uuid',
        scopes: ['openid', 'profile'],
        redirect_uri: 'https://app.example/callback',
        expires_at: 123,
      },
    })

    await fetchOAuthConsentChallenge('challenge/1')

    expect(get).toHaveBeenCalledWith('/oauth/consent/challenge%2F1')
  })

  it('submits an allow or deny consent decision as JSON', async () => {
    vi.mocked(post).mockResolvedValueOnce({
      success: true,
      data: { redirect_uri: 'https://app.example/callback?code=abc' },
    })

    const result = await submitOAuthConsent('challenge-1', true)

    expect(result.redirect_uri).toContain('code=abc')
    expect(post).toHaveBeenCalledWith('/oauth/consent', {
      challenge_id: 'challenge-1',
      approved: true,
    })
  })

  it('uses form encoding for device approvals', async () => {
    vi.mocked(post).mockResolvedValueOnce({ success: true })

    await approveOAuthDevice('ABCD-EFGH')

    expect(post).toHaveBeenCalledWith(
      '/oauth/device',
      expect.any(URLSearchParams),
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } },
    )
    expect(String(vi.mocked(post).mock.calls[0][1])).toBe('user_code=ABCD-EFGH')
  })

  it('uses form encoding for CIBA approvals', async () => {
    vi.mocked(post).mockResolvedValueOnce({ success: true })

    await approveOAuthCIBA('auth-request-1')

    expect(post).toHaveBeenCalledWith(
      '/oauth/ciba/approve',
      expect.any(URLSearchParams),
      { headers: { 'Content-Type': 'application/x-www-form-urlencoded' } },
    )
    expect(String(vi.mocked(post).mock.calls[0][1])).toBe('auth_req_id=auth-request-1')
  })

  it('revokes a consent grant through the grant id path', async () => {
    vi.mocked(deleteRequest).mockResolvedValueOnce({ success: true })

    await revokeOAuthConsentGrant('grant/1')

    expect(deleteRequest).toHaveBeenCalledWith('/oauth/consent/grants/grant%2F1')
  })

  it('builds a public API end-session forwarding URL', () => {
    expect(oauthEndSessionURL('?id_token_hint=token&post_logout_redirect_uri=https%3A%2F%2Fapp.example')).toBe(
      '/api/v1/oauth/end_session?id_token_hint=token&post_logout_redirect_uri=https%3A%2F%2Fapp.example',
    )
    expect(oauthEndSessionURL('')).toBe('/api/v1/oauth/end_session')
  })
})
