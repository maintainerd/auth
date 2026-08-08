import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import BackupCodeRecoveryPage from './BackupCodeRecoveryPage'

const { postMock, refreshAccountMock, navigateMock, showErrorMock } = vi.hoisted(() => ({
  postMock: vi.fn(),
  refreshAccountMock: vi.fn(),
  navigateMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock('@/services/api/client', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/client')>('@/services/api/client')
  return { ...actual, post: postMock }
})

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ refreshAccount: refreshAccountMock }),
}))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({ currentTenant: null }),
}))

vi.mock('@/hooks/useToast', () => ({
  useToast: () => ({ showError: showErrorMock, showSuccess: vi.fn() }),
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => navigateMock }
})

const RECOVERY_ROUTE = '/recovery?client_id=identity-client&provider_id=maintainerd'

function renderPage(route = RECOVERY_ROUTE) {
  return renderWithProviders(<BackupCodeRecoveryPage />, { route, path: '/recovery' })
}

describe('BackupCodeRecoveryPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    postMock.mockResolvedValue({ success: true })
    refreshAccountMock.mockResolvedValue({ user_id: 'user-1', email: 'ada@example.com' })
  })

  // user.VerifyBackupCodeDTO: email, code, client_id and provider_id, all Required.
  it('sends the four fields the endpoint validates', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText('Email address'), 'ada@example.com')
    await user.type(screen.getByLabelText('Backup code'), 'ABCD-1234')
    await user.click(screen.getByRole('button', { name: 'Recover Account' }))

    await waitFor(() => expect(postMock).toHaveBeenCalledTimes(1))
    expect(postMock).toHaveBeenCalledWith('/recovery/backup-code', {
      email: 'ada@example.com',
      code: 'ABCD-1234',
      client_id: 'identity-client',
      provider_id: 'maintainerd',
    })
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith('/', { replace: true }))
  })

  it('prefills the email the recovery link addressed', async () => {
    renderPage(`${RECOVERY_ROUTE}&email=ada%40example.com`)

    expect(screen.getByLabelText('Email address')).toHaveValue('ada@example.com')
  })

  it('asks for an email address instead of submitting an unusable request', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText('Backup code'), 'ABCD-1234')
    await user.click(screen.getByRole('button', { name: 'Recover Account' }))

    expect(await screen.findByText('Enter the email address on your account.')).toBeInTheDocument()
    expect(postMock).not.toHaveBeenCalled()
  })

  // Without a provider the request cannot pass validation, and a backup code is
  // single-use — burning one on a request guaranteed to 400 is not acceptable.
  it('blocks up front when the link carries no provider', () => {
    renderPage('/recovery?client_id=identity-client')

    expect(screen.getByText('This recovery link is incomplete')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Recover Account' })).not.toBeInTheDocument()
  })

  // The handler returns tokens via resp.Success, not SuccessWithCookies, so a
  // 200 leaves this browser without a session.
  it('does not claim a sign-in when no session was established', async () => {
    const user = userEvent.setup()
    refreshAccountMock.mockResolvedValue(null)
    renderPage()

    await user.type(screen.getByLabelText('Email address'), 'ada@example.com')
    await user.type(screen.getByLabelText('Backup code'), 'ABCD-1234')
    await user.click(screen.getByRole('button', { name: 'Recover Account' }))

    expect(await screen.findByText('Backup code accepted')).toBeInTheDocument()
    expect(screen.getByText(/could not sign you in on this device/i)).toBeInTheDocument()
    expect(navigateMock).not.toHaveBeenCalledWith('/', { replace: true })
  })

  it('surfaces a rejected code instead of routing away', async () => {
    const user = userEvent.setup()
    postMock.mockRejectedValueOnce(new Error('invalid email or backup code'))
    renderPage()

    await user.type(screen.getByLabelText('Email address'), 'ada@example.com')
    await user.type(screen.getByLabelText('Backup code'), 'WRONG-CODE')
    await user.click(screen.getByRole('button', { name: 'Recover Account' }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalled())
    expect(navigateMock).not.toHaveBeenCalledWith('/', { replace: true })
  })
})
