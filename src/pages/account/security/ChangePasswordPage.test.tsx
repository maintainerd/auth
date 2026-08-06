import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import ChangePasswordPage from './ChangePasswordPage'

const { fetchAccountInfoMock, changePasswordMock, forgotPasswordMock } = vi.hoisted(() => ({
  fetchAccountInfoMock: vi.fn(),
  changePasswordMock: vi.fn(),
  forgotPasswordMock: vi.fn(),
}))

vi.mock('@/services/api/account', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/account')>('@/services/api/account')
  return { ...actual, fetchAccountInfo: fetchAccountInfoMock, changePassword: changePasswordMock }
})

vi.mock('@/services/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/auth')>('@/services/api/auth')
  return { ...actual, forgotPassword: forgotPasswordMock }
})

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ logout: vi.fn().mockResolvedValue(undefined) }),
}))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({ currentTenant: null, getCurrentTenant: () => null }),
}))

vi.mock('@/hooks/useToast', () => ({
  useToast: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}))

function renderPage() {
  return renderWithProviders(<ChangePasswordPage />, {
    route: '/account/security/password',
    path: '/account/security/password',
  })
}

describe('ChangePasswordPage reset-link fallback', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // The button used to be silently disabled whenever the account lookup failed,
  // because the failure was swallowed into "no account data".
  it('explains why the reset link is unavailable and offers a retry', async () => {
    const user = userEvent.setup()
    fetchAccountInfoMock.mockRejectedValue(new Error('Too many requests'))
    renderPage()

    await user.click(await screen.findByRole('button', { name: 'Forgot your current password?' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't load your email address",
    )
    expect(screen.getByRole('button', { name: 'Try again' })).toBeEnabled()
    expect(screen.queryByRole('button', { name: 'Send password reset link' })).not.toBeInTheDocument()
  })

  it('offers the reset link once the account resolves', async () => {
    const user = userEvent.setup()
    fetchAccountInfoMock.mockResolvedValue({
      user_id: 'user-1',
      email: 'ada@example.com',
      phone: '',
      email_verified: true,
      phone_verified: false,
      profiles: [],
      roles: [],
      permissions: [],
      tenant: { tenant_id: 'tenant-1', name: 'acme', display_name: 'Acme' },
    })
    renderPage()

    await user.click(await screen.findByRole('button', { name: 'Forgot your current password?' }))

    expect(await screen.findByRole('button', { name: 'Send password reset link' })).toBeEnabled()
  })
})
