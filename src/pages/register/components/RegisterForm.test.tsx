import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '@/test/utils'
import RegisterForm from './RegisterForm'
import { ApiError } from '@/services/api/client'

const {
  fetchRegistrationContextMock,
  registerMock,
  refreshAccountMock,
  navigateMock,
  finishAuthStepMock,
} = vi.hoisted(() => ({
  fetchRegistrationContextMock: vi.fn(),
  registerMock: vi.fn(),
  refreshAccountMock: vi.fn(),
  navigateMock: vi.fn(),
  finishAuthStepMock: vi.fn(),
}))

vi.mock('@/services/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/auth')>('@/services/api/auth')
  return { ...actual, fetchRegistrationContext: fetchRegistrationContextMock }
})

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ register: registerMock, refreshAccount: refreshAccountMock }),
}))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({
    getCurrentTenant: () => ({ identifier: 'acme', registration_config: {} }),
  }),
}))

vi.mock('@/hooks/useToast', () => ({
  useToast: () => ({ showSuccess: vi.fn() }),
}))

vi.mock('@/utils/oauthContinuation', () => ({
  finishAuthStep: (...args: unknown[]) => finishAuthStepMock(...args),
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock('@/utils/clientContext', async () => {
  const actual = await vi.importActual<typeof import('@/utils/clientContext')>('@/utils/clientContext')
  // resolvePublicAuthContext is what the registration-context hook reads, so it
  // resolves the same client the sibling register() call sends.
  return {
    ...actual,
    currentPublicAuthContext: () => ({ clientId: 'storefront-abc123' }),
    resolvePublicAuthContext: () => ({ clientId: 'storefront-abc123' }),
  }
})

const FLOW_ROUTE = '/register?client_id=storefront-abc123&registration_flow=partner-signup'
const PLAIN_ROUTE = '/register?client_id=storefront-abc123'

function render(route: string) {
  return renderWithProviders(<RegisterForm />, { route, path: '/register' })
}

// The form renders a skeleton until the registration context settles, so every
// interaction must wait for the real fields first.
async function awaitForm() {
  return screen.findByLabelText(/^email/i, undefined, { timeout: 4000 })
}

async function fillCredentials(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/^email/i), 'buyer@example.com')
  await user.type(screen.getByLabelText(/^password/i), 'Str0ng!Passw0rd')
  await user.type(screen.getByLabelText(/confirm password/i), 'Str0ng!Passw0rd')
  await user.click(screen.getByRole('checkbox'))
}

describe('RegisterForm registration-flow wiring', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    refreshAccountMock.mockResolvedValue({})
    registerMock.mockResolvedValue({ data: {} })
  })

  // No flow in the link is ordinary self-service signup: no context request, and
  // none of the conditional fields.
  it('asks for nothing extra when the link names no flow', async () => {
    render(PLAIN_ROUTE)

    await awaitForm()
    expect(fetchRegistrationContextMock).not.toHaveBeenCalled()
    expect(screen.queryByLabelText(/full name/i)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/phone/i)).not.toBeInTheDocument()
  })

  it('renders only the fields the flow requires', async () => {
    fetchRegistrationContextMock.mockResolvedValue({
      registration_flow: 'partner-signup',
      required_fields: ['fullname'],
      verification_required: false,
    })
    render(FLOW_ROUTE)

    expect(await screen.findByLabelText(/full name/i)).toBeInTheDocument()
    expect(screen.queryByLabelText(/phone/i)).not.toBeInTheDocument()
  })

  // The regression this whole endpoint exists to prevent: a flow requiring a
  // field the form never collected produced a 400 no user could resolve.
  it('blocks submit until a required field is provided, then sends it', async () => {
    fetchRegistrationContextMock.mockResolvedValue({
      registration_flow: 'partner-signup',
      required_fields: ['fullname'],
      verification_required: false,
    })
    const user = userEvent.setup()
    render(FLOW_ROUTE)

    await screen.findByLabelText(/full name/i)
    await fillCredentials(user)
    await user.click(screen.getByRole('button', { name: /create account/i }))

    expect(await screen.findByText(/full name is required/i)).toBeInTheDocument()
    expect(registerMock).not.toHaveBeenCalled()

    await user.type(screen.getByLabelText(/full name/i), 'Ada Lovelace')
    await user.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => expect(registerMock).toHaveBeenCalled())
    expect(registerMock.mock.calls[0][0]).toMatchObject({
      email: 'buyer@example.com',
      fullname: 'Ada Lovelace',
    })
  })

  it('omits a field the flow does not require rather than sending an empty string', async () => {
    fetchRegistrationContextMock.mockResolvedValue({
      registration_flow: 'partner-signup',
      required_fields: [],
      verification_required: false,
    })
    const user = userEvent.setup()
    render(FLOW_ROUTE)

    await awaitForm()
    await fillCredentials(user)
    await user.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => expect(registerMock).toHaveBeenCalled())
    const payload = registerMock.mock.calls[0][0]
    expect(payload.fullname).toBeFalsy()
    expect(payload.phone).toBeFalsy()
  })

  it('validates phone against the backend rule, not a looser one', async () => {
    fetchRegistrationContextMock.mockResolvedValue({
      registration_flow: 'partner-signup',
      required_fields: ['phone'],
      verification_required: false,
    })
    const user = userEvent.setup()
    render(FLOW_ROUTE)

    await screen.findByLabelText(/phone/i)
    await fillCredentials(user)

    // Leading zero is rejected by internal/platform/valid/valid.go.
    await user.type(screen.getByLabelText(/phone/i), '0123456789')
    await user.click(screen.getByRole('button', { name: /create account/i }))
    expect(await screen.findByText(/country code/i)).toBeInTheDocument()
    expect(registerMock).not.toHaveBeenCalled()

    await user.clear(screen.getByLabelText(/phone/i))
    await user.type(screen.getByLabelText(/phone/i), '+1 212 555 1234')
    await user.click(screen.getByRole('button', { name: /create account/i }))
    await waitFor(() => expect(registerMock).toHaveBeenCalled())
  })

  // An authoritative refusal must not render a form that cannot succeed, and must
  // not silently fall back to plain signup — that would create an account missing
  // the flow's roles while looking like success.
  it('refuses signup outright when the link is no longer valid', async () => {
    fetchRegistrationContextMock.mockRejectedValue(new ApiError({ message: 'not found', status: 404 }))
    render(FLOW_ROUTE)

    expect(await screen.findByText(/no longer valid/i, undefined, { timeout: 4000 })).toBeInTheDocument()
    expect(screen.queryByLabelText(/^email/i)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /create account/i })).not.toBeInTheDocument()
  })

  // A transport failure is different: the server still enforces the requirement,
  // so degrading to the plain form is safe and refusing would break a healthy flow.
  it('still allows signup when the requirements cannot be reached', async () => {
    fetchRegistrationContextMock.mockRejectedValue(new ApiError({ message: 'boom', status: 500 }))
    render(FLOW_ROUTE)

    // A 5xx is retried once before the hook reports it, so allow for the backoff.
    expect(await screen.findByText(/could not confirm/i, undefined, { timeout: 8000 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument()
  })

  // registerAsync rejects with a plain object, so `instanceof Error` alone
  // discarded every actionable server message.
  it('surfaces the server message verbatim', async () => {
    fetchRegistrationContextMock.mockResolvedValue({
      registration_flow: 'partner-signup',
      required_fields: [],
      verification_required: false,
    })
    registerMock.mockRejectedValue({
      message: 'fullname is required by the registration flow',
      status: 400,
    })
    const user = userEvent.setup()
    render(FLOW_ROUTE)

    await awaitForm()
    await fillCredentials(user)
    await user.click(screen.getByRole('button', { name: /create account/i }))

    expect(
      await screen.findByText(/fullname is required by the registration flow/i),
    ).toBeInTheDocument()
  })

  // screen_hint=signup is what routed the user here, so forwarding it to /login
  // bounced them straight back — an inescapable loop.
  it('does not forward screen_hint to the sign-in link', async () => {
    fetchRegistrationContextMock.mockResolvedValue({
      registration_flow: 'partner-signup',
      required_fields: [],
      verification_required: false,
    })
    render(`${FLOW_ROUTE}&screen_hint=signup`)

    await awaitForm()
    const signIn = await screen.findByRole('link', { name: /sign in/i })
    expect(signIn.getAttribute('href')).not.toContain('screen_hint')
    expect(signIn.getAttribute('href')).toContain('client_id=storefront-abc123')
  })
})
