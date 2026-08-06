import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import { ApiError } from '@/services/api/client'
import ChangeUsernamePage from './ChangeUsernamePage'

const { changeUsernameMock, showErrorMock, navigateMock } = vi.hoisted(() => ({
  changeUsernameMock: vi.fn(),
  showErrorMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock('@/services/api/account', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/account')>('@/services/api/account')
  return { ...actual, changeUsername: changeUsernameMock }
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

/** The shape client.ts builds from a backend error body. */
function serverError(status: number, message: string) {
  const error = new ApiError({ message, status })
  error.responseData = { error: message }
  return error
}

async function submitUsername(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/New username/), 'ada')
  await user.type(screen.getByLabelText(/Current password/), 'Password123!')
  await user.click(screen.getByRole('button', { name: 'Save' }))
}

describe('ChangeUsernamePage conflict handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // accountService returns apperror.NewValidation("username is already taken"),
  // which HandleServiceError maps to 400 — not the 409 this page used to expect.
  it('puts the taken-username message on the field for the 400 the server sends', async () => {
    const user = userEvent.setup()
    changeUsernameMock.mockRejectedValueOnce(serverError(400, 'username is already taken'))
    renderWithProviders(<ChangeUsernamePage />, {
      route: '/account/security/username',
      path: '/account/security/username',
    })

    await submitUsername(user)

    expect(await screen.findByText('That username is already taken.')).toBeInTheDocument()
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('still handles the conflict if the backend reclassifies it as 409', async () => {
    const user = userEvent.setup()
    changeUsernameMock.mockRejectedValueOnce(serverError(409, 'username is already taken'))
    renderWithProviders(<ChangeUsernamePage />, {
      route: '/account/security/username',
      path: '/account/security/username',
    })

    await submitUsername(user)

    expect(await screen.findByText('That username is already taken.')).toBeInTheDocument()
  })

  it('leaves an unrelated 400 to the toast rather than blaming the username field', async () => {
    const user = userEvent.setup()
    changeUsernameMock.mockRejectedValueOnce(serverError(400, 'account has no password set'))
    renderWithProviders(<ChangeUsernamePage />, {
      route: '/account/security/username',
      path: '/account/security/username',
    })

    await submitUsername(user)

    await waitFor(() => expect(showErrorMock).toHaveBeenCalled())
    expect(screen.queryByText('That username is already taken.')).not.toBeInTheDocument()
  })
})
