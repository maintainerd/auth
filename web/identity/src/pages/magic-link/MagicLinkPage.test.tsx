import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/utils'
import MagicLinkPage from './MagicLinkPage'

const { verifyMagicLinkMock, fetchAccountMock } = vi.hoisted(() => ({
  verifyMagicLinkMock: vi.fn(),
  fetchAccountMock: vi.fn(),
}))

vi.mock('@/services/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/auth')>('@/services/api/auth')
  return {
    ...actual,
    verifyMagicLink: verifyMagicLinkMock,
    fetchAccount: fetchAccountMock,
  }
})

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({ fetchDefault: vi.fn(), currentTenant: null }),
}))

vi.mock('@/pages/login/components/LoginMFAStep', () => ({
  LoginMFAStep: ({ allowedMethods, clientId }: { allowedMethods: string[]; clientId?: string }) => (
    <div>
      <span>Methods: {allowedMethods.join(',')}</span>
      <span>Client: {clientId}</span>
    </div>
  ),
}))

const callbackURL = '/magic-link?token=opaque&client_id=identity&expires=1234567890&sig=signed'

describe('MagicLinkPage MFA routing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows the MFA step without offering email OTP when challenged', async () => {
    verifyMagicLinkMock.mockResolvedValueOnce({
      success: true,
      data: {
        mfa_required: true,
        mfa_challenge_token: 'challenge-token',
        mfa_allowed_methods: ['email_otp', 'totp', 'webauthn'],
      },
    })

    renderWithProviders(<MagicLinkPage />, { route: callbackURL, path: '/magic-link' })

    expect(await screen.findByRole('heading', { name: 'Two-step verification' })).toBeInTheDocument()
    expect(screen.getByText('Methods: totp,webauthn')).toBeInTheDocument()
    expect(screen.getByText('Client: identity')).toBeInTheDocument()
    expect(fetchAccountMock).not.toHaveBeenCalled()
  })

  it('loads the account and continues to success when MFA is not required', async () => {
    verifyMagicLinkMock.mockResolvedValueOnce({ success: true, data: {} })
    fetchAccountMock.mockResolvedValueOnce({ tenant: { name: 'acme' } })

    renderWithProviders(<MagicLinkPage />, { route: callbackURL, path: '/magic-link' })

    expect(await screen.findByRole('heading', { name: "You're signed in" })).toBeInTheDocument()
    expect(fetchAccountMock).toHaveBeenCalledOnce()
  })
})
