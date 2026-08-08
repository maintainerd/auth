import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import { ApiError } from '@/services/api/client'
import ChangeEmailPage from './ChangeEmailPage'

const { initiateEmailChangeMock, showErrorMock, navigateMock } = vi.hoisted(() => ({
  initiateEmailChangeMock: vi.fn(),
  showErrorMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock('@/services/api/account', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/account')>('@/services/api/account')
  return { ...actual, initiateEmailChange: initiateEmailChangeMock }
})

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ logout: vi.fn().mockResolvedValue(undefined) }),
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

function serverError(status: number, message: string) {
  const error = new ApiError({ message, status })
  error.responseData = { error: message }
  return error
}

async function submitEmail(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/New email address/), 'taken@example.com')
  await user.type(screen.getByLabelText(/Current password/), 'Password123!')
  await user.click(screen.getByRole('button', { name: 'Send verification code' }))
}

describe('ChangeEmailPage conflict handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // apperror.NewValidation("email address is already in use") → HTTP 400.
  it('puts the in-use message on the field for the 400 the server sends', async () => {
    const user = userEvent.setup()
    initiateEmailChangeMock.mockRejectedValueOnce(serverError(400, 'email address is already in use'))
    renderWithProviders(<ChangeEmailPage />, {
      route: '/account/security/email',
      path: '/account/security/email',
    })

    await submitEmail(user)

    expect(await screen.findByText('That email address is already in use.')).toBeInTheDocument()
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('leaves an unrelated 400 to the toast', async () => {
    const user = userEvent.setup()
    initiateEmailChangeMock.mockRejectedValueOnce(serverError(400, 'account has no password set'))
    renderWithProviders(<ChangeEmailPage />, {
      route: '/account/security/email',
      path: '/account/security/email',
    })

    await submitEmail(user)

    await waitFor(() => expect(showErrorMock).toHaveBeenCalled())
    expect(screen.queryByText('That email address is already in use.')).not.toBeInTheDocument()
  })
})
