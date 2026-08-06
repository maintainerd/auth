import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import { ApiError } from '@/services/api/client'
import RegisterInviteForm from './RegisterInviteForm'

const { fetchInviteContextMock, registerInviteMock, navigateMock, refreshAccountMock } = vi.hoisted(() => ({
  fetchInviteContextMock: vi.fn(),
  registerInviteMock: vi.fn(),
  navigateMock: vi.fn(),
  refreshAccountMock: vi.fn(),
}))

vi.mock('@/services/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/auth')>('@/services/api/auth')
  return { ...actual, fetchInviteContext: fetchInviteContextMock }
})

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({
    registerInvite: registerInviteMock,
    refreshAccount: refreshAccountMock,
    isAuthenticated: false,
    account: null,
    logout: vi.fn(),
  }),
}))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({ getCurrentTenant: () => ({ name: 'acme' }) }),
}))

vi.mock('@/hooks/useToast', () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => navigateMock }
})

const INVITE_ROUTE = '/register/invite?invite_token=tok-1&email=ada%40example.com&expires=1&sig=abc'

function renderForm(route = INVITE_ROUTE) {
  return renderWithProviders(<RegisterInviteForm />, { route, path: '/register/invite' })
}

let assignMock: ReturnType<typeof vi.fn>
const originalLocation = window.location

// A fully-registered account, so resolvePostAuthRoute lands on /login-success —
// the branch that performs the post-invite external redirect.
const completeAccount = {
  email: 'ada@example.com',
  email_verified: true,
  profiles: [{ profile_id: 'p1', display_name: 'Ada', default: true }],
}

async function completeAndSubmit() {
  const user = userEvent.setup()
  await user.type(await screen.findByLabelText(/^Password/), 'Sup3rSecret!pass')
  await user.type(screen.getByLabelText(/^Confirm password/), 'Sup3rSecret!pass')
  await user.click(screen.getByLabelText(/I agree to the/))
  await user.click(screen.getByRole('button', { name: /Create account/ }))
}

describe('RegisterInviteForm invite validation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    registerInviteMock.mockResolvedValue({ data: {} })
    refreshAccountMock.mockResolvedValue(completeAccount)
    assignMock = vi.fn()
    // jsdom's location.assign is non-configurable, so swap the whole object.
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { origin: originalLocation.origin, href: `${originalLocation.origin}/register/invite`, assign: assignMock },
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
    sessionStorage.clear()
  })

  // The backend answers 410 Gone for a revoked or expired invite.
  it('refuses a dead invite before the user fills anything in', async () => {
    fetchInviteContextMock.mockRejectedValue(
      new ApiError({ message: 'Invite has expired', status: 410 }),
    )
    renderForm()

    expect(await screen.findByText("This invitation can't be used")).toBeInTheDocument()
    expect(screen.getByText(/Invite has expired/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to sign in' })).toBeInTheDocument()
    expect(screen.queryByLabelText(/^Password/)).not.toBeInTheDocument()
  })

  it('checks the invite before rendering the form, even when the URL carries a callback', async () => {
    fetchInviteContextMock.mockResolvedValue({
      invite_token: 'tok-1',
      email: 'ada@example.com',
      callback_url: 'https://app.example.com/welcome',
      status: 'pending',
    })
    renderForm(`${INVITE_ROUTE}&callback_url=https%3A%2F%2Fapp.example.com%2Fwelcome`)

    expect(await screen.findByLabelText(/^Password/)).toBeInTheDocument()
    expect(fetchInviteContextMock).toHaveBeenCalledWith('tok-1')
  })

  it('falls back to the invite\'s own email when the link omits it', async () => {
    fetchInviteContextMock.mockResolvedValue({
      invite_token: 'tok-1',
      email: 'ada@example.com',
      callback_url: null,
      status: 'pending',
    })
    renderForm('/register/invite?invite_token=tok-1&expires=1&sig=abc')

    expect(await screen.findByText('ada@example.com')).toBeInTheDocument()
    expect(screen.queryByText('Invalid invite link')).not.toBeInTheDocument()
  })

  it('shows a progress state while the invite is being checked', async () => {
    fetchInviteContextMock.mockReturnValue(new Promise(() => {}))
    renderForm()

    await waitFor(() => expect(screen.getByText('Checking your invitation…')).toBeInTheDocument())
    expect(screen.queryByLabelText(/^Password/)).not.toBeInTheDocument()
  })
})

describe('RegisterInviteForm post-registration callback', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    registerInviteMock.mockResolvedValue({ data: {} })
    refreshAccountMock.mockResolvedValue(completeAccount)
    assignMock = vi.fn()
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { origin: originalLocation.origin, href: `${originalLocation.origin}/register/invite`, assign: assignMock },
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
    sessionStorage.clear()
  })

  it('redirects to the callback the server validated for this invite', async () => {
    fetchInviteContextMock.mockResolvedValue({
      invite_token: 'tok-1',
      email: 'ada@example.com',
      callback_url: 'https://app.example.com/welcome',
      status: 'pending',
    })
    renderForm(`${INVITE_ROUTE}&callback_url=https%3A%2F%2Fapp.example.com%2Fwelcome`)
    await completeAndSubmit()

    await waitFor(() => expect(assignMock).toHaveBeenCalledWith('https://app.example.com/welcome'))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  // This used to assert the opposite by construction: the form preferred the raw
  // `callback_url` query parameter over the invite-context probe, so rewriting
  // the parameter on an invite link redirected the newly registered user to the
  // attacker's origin. Only the probe's value has been checked against the
  // inviting client's registered redirect URIs, so only it may be honoured.
  it('ignores a rewritten callback_url in the link and uses the server-validated one', async () => {
    fetchInviteContextMock.mockResolvedValue({
      invite_token: 'tok-1',
      email: 'ada@example.com',
      callback_url: 'https://app.example.com/welcome',
      status: 'pending',
    })
    renderForm(`${INVITE_ROUTE}&callback_url=https%3A%2F%2Fevil.test%2Fsteal`)
    await completeAndSubmit()

    await waitFor(() => expect(assignMock).toHaveBeenCalledWith('https://app.example.com/welcome'))
    expect(assignMock).not.toHaveBeenCalledWith(expect.stringContaining('evil.test'))
    expect(sessionStorage.getItem('maintainerd_auth_invite_callback')).toBe('https://app.example.com/welcome')
  })

  // Fail closed: no server-validated callback means no external redirect, even
  // though the link asks for one.
  it('performs no external redirect when the invite carries no server-validated callback', async () => {
    fetchInviteContextMock.mockResolvedValue({
      invite_token: 'tok-1',
      email: 'ada@example.com',
      callback_url: null,
      status: 'pending',
    })
    renderForm(`${INVITE_ROUTE}&callback_url=https%3A%2F%2Fevil.test%2Fsteal`)
    await completeAndSubmit()

    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith('/login-success', { replace: true }))
    expect(assignMock).not.toHaveBeenCalled()
    expect(sessionStorage.getItem('maintainerd_auth_invite_callback')).toBeNull()
  })

  it('drops a server callback that cannot pass the redirect guard', async () => {
    fetchInviteContextMock.mockResolvedValue({
      invite_token: 'tok-1',
      email: 'ada@example.com',
      callback_url: 'http://app.example.com/welcome',
      status: 'pending',
    })
    renderForm()
    await completeAndSubmit()

    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith('/login-success', { replace: true }))
    expect(assignMock).not.toHaveBeenCalled()
  })
})
