import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import LoginForm from './LoginForm'

const { sendMagicLinkMock, fetchOAuthConnectionsMock, loginMock, navigateMock } = vi.hoisted(() => ({
  sendMagicLinkMock: vi.fn(),
  fetchOAuthConnectionsMock: vi.fn(),
  loginMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock('@/services/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/auth')>('@/services/api/auth')
  return { ...actual, sendMagicLink: sendMagicLinkMock }
})

vi.mock('@/services/api/oauth', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/oauth')>('@/services/api/oauth')
  return { ...actual, fetchOAuthConnections: fetchOAuthConnectionsMock }
})

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ login: loginMock }),
}))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({
    getCurrentTenant: () => ({
      identifier: 'acme',
      registration_config: { self_registration_enabled: true },
    }),
    // This suite exercises magic-link sign-in, which is opt-in per client — so
    // the surface under test has it turned on.
    bootstrap: { magic_link_enabled: true },
  }),
}))

vi.mock('@/hooks/useToast', () => ({
  useToast: () => ({ showSuccess: vi.fn() }),
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => navigateMock }
})

describe('LoginForm passwordless sign-in', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
  })

  it('requires a valid email without requiring a password', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginForm />, { route: '/login?client_id=identity', path: '/login' })

    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }))

    expect(await screen.findByText('Email is required')).toBeInTheDocument()
    expect(sendMagicLinkMock).not.toHaveBeenCalled()
  })

  it('shows progress and a check-email state for a client-scoped request', async () => {
    const user = userEvent.setup()
    let finishSending!: () => void
    sendMagicLinkMock.mockReturnValueOnce(new Promise<void>((resolve) => { finishSending = resolve }))
    renderWithProviders(<LoginForm />, { route: '/login?client_id=identity', path: '/login' })

    await user.type(screen.getByRole('textbox', { name: /email/i }), 'user@example.com')
    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }))

    expect(await screen.findByRole('button', { name: 'Sending sign-in link...' })).toBeDisabled()
    expect(sendMagicLinkMock).toHaveBeenCalledWith('user@example.com', {
      clientId: 'identity',
    })

    finishSending()
    expect(await screen.findByRole('heading', { name: 'Check your email' })).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Back to password sign in' })).toBeEnabled())
  })

  it('shows the backend error when no account exists for the email', async () => {
    const user = userEvent.setup()
    sendMagicLinkMock.mockRejectedValueOnce(new Error('no account found with that email address'))
    renderWithProviders(<LoginForm />, { route: '/login?client_id=identity', path: '/login' })

    await user.type(screen.getByRole('textbox', { name: /email/i }), 'missing@example.com')
    await user.click(screen.getByRole('button', { name: 'Email me a sign-in link' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('no account found with that email address')
    expect(screen.queryByRole('heading', { name: 'Check your email' })).not.toBeInTheDocument()
  })

  it('renders connected broker providers for an OAuth authorize request', async () => {
    const user = userEvent.setup()
    fetchOAuthConnectionsMock.mockResolvedValueOnce({
      password_enabled: false,
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
    })
    renderWithProviders(
      <LoginForm />,
      {
        route: '/login?client_id=external&response_type=code&redirect_uri=https%3A%2F%2Fapp.example%2Fcb&state=abc&return_to=%2Fauthorize%3Fclient_id%3Dexternal%26response_type%3Dcode%26redirect_uri%3Dhttps%253A%252F%252Fapp.example%252Fcb%26state%3Dabc',
        path: '/login',
      },
    )

    await user.click(await screen.findByRole('button', { name: 'Continue with Google' }))

    expect(fetchOAuthConnectionsMock).toHaveBeenCalledWith('external')
    expect(navigateMock).toHaveBeenCalledWith(
      '/authorize?client_id=external&response_type=code&redirect_uri=https%3A%2F%2Fapp.example%2Fcb&state=abc&idp_hint=google',
      { replace: true },
    )
  })

  it('does not treat raw OAuth params on /login as an OAuth continuation', () => {
    renderWithProviders(
      <LoginForm />,
      {
        route: '/login?client_id=external&response_type=code&redirect_uri=https%3A%2F%2Fapp.example%2Fcb&state=abc',
        path: '/login',
      },
    )

    expect(fetchOAuthConnectionsMock).not.toHaveBeenCalled()
    expect(screen.queryByRole('button', { name: /Continue with/i })).not.toBeInTheDocument()
  })

  it('does not resume a stale OAuth return after direct login', async () => {
    const user = userEvent.setup()
    sessionStorage.setItem('maintainerd_auth_oauth_return_to', '/authorize?client_id=old-client&response_type=code')
    loginMock.mockResolvedValueOnce({
      mfaRequired: false,
      account: {
        user_id: 'user-1',
        email: 'user@example.com',
        phone: '',
        email_verified: true,
        phone_verified: false,
        profiles: [{ profile_id: 'profile-1', first_name: 'Test', display_name: 'Test', default: true }],
        roles: [],
        permissions: [],
        tenant: { tenant_id: 'tenant-1', name: 'acme', display_name: 'Acme' },
      },
    })
    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    await user.type(screen.getByRole('textbox', { name: /email/i }), 'user@example.com')
    await user.type(screen.getByLabelText(/password/i), 'Password123!')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))

    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith('/login-success', { replace: true }))
    expect(navigateMock).not.toHaveBeenCalledWith(expect.stringContaining('/authorize'), expect.anything())
    expect(sessionStorage.getItem('maintainerd_auth_oauth_return_to')).toBeNull()
  })
})
