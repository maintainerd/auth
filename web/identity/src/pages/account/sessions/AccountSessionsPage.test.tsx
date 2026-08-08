import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import AccountSessionsPage from './AccountSessionsPage'

const { fetchSessionsMock, revokeAllSessionsMock, logoutMock, navigateMock } = vi.hoisted(() => ({
  fetchSessionsMock: vi.fn(),
  revokeAllSessionsMock: vi.fn(),
  logoutMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock('@/services/api/account', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/account')>('@/services/api/account')
  return { ...actual, fetchSessions: fetchSessionsMock, revokeAllSessions: revokeAllSessionsMock }
})

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ logout: logoutMock }),
}))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({ currentTenant: null }),
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => navigateMock }
})

function renderPage() {
  return renderWithProviders(<AccountSessionsPage />, {
    route: '/account/sessions',
    path: '/account/sessions',
  })
}

describe('AccountSessionsPage revoke-all', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    logoutMock.mockResolvedValue(undefined)
    revokeAllSessionsMock.mockResolvedValue(undefined)
    fetchSessionsMock.mockResolvedValue([
      {
        session_id: 'session-1',
        ip_address: '203.0.113.4',
        user_agent: 'Mozilla/5.0 (Macintosh) Chrome/120',
        created_at: '2024-05-01T10:00:00Z',
      },
    ])
  })

  // DELETE /account/sessions revokes every session, the caller's included.
  it('names the action for what the endpoint actually does', async () => {
    renderPage()

    expect(await screen.findByRole('button', { name: 'Sign out everywhere' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /all others/i })).not.toBeInTheDocument()
  })

  it('warns that this device is signed out too before revoking', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByRole('button', { name: 'Sign out everywhere' }))

    expect(await screen.findByText('Sign out everywhere?')).toBeInTheDocument()
    expect(screen.getByText(/including this one, on this device/i)).toBeInTheDocument()
    expect(revokeAllSessionsMock).not.toHaveBeenCalled()
  })

  it('clears local auth state and lands on sign-in once the sessions are gone', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByRole('button', { name: 'Sign out everywhere' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Sign out everywhere' }))

    await waitFor(() => expect(revokeAllSessionsMock).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(logoutMock).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith('/login', { replace: true }))
  })

  it('still signs this browser out when the logout call is rejected by the dead session', async () => {
    const user = userEvent.setup()
    logoutMock.mockRejectedValueOnce(new Error('unauthorized'))
    renderPage()

    await user.click(await screen.findByRole('button', { name: 'Sign out everywhere' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Sign out everywhere' }))

    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith('/login', { replace: true }))
  })
})
