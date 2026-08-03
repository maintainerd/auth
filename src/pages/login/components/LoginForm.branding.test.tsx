/**
 * Guards the contract between the console's login-template preview and what the
 * identity app actually renders: the operator-configured copy must win, and the
 * element order must match the preview's arrangement.
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/utils'
import LoginForm from './LoginForm'

const { fetchOAuthConnectionsMock, brandingMetadata, bootstrapConnections, magicLinkEnabled } = vi.hoisted(() => ({
  fetchOAuthConnectionsMock: vi.fn(),
  brandingMetadata: { value: null as Record<string, unknown> | null },
  bootstrapConnections: { value: [] as unknown[] },
  magicLinkEnabled: { value: false },
}))

vi.mock('@/services/api/oauth', async () => {
  const actual = await vi.importActual<typeof import('@/services/api/oauth')>('@/services/api/oauth')
  return { ...actual, fetchOAuthConnections: fetchOAuthConnectionsMock }
})

vi.mock('@/hooks/useAuth', () => ({ useAuth: () => ({ login: vi.fn() }) }))
vi.mock('@/hooks/useToast', () => ({ useToast: () => ({ showSuccess: vi.fn() }) }))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({
    currentTenant: {
      name: 'acme',
      registration_config: { self_registration_enabled: true },
      branding: { metadata: brandingMetadata.value },
    },
    getCurrentTenant: () => ({
      name: 'acme',
      registration_config: { self_registration_enabled: true },
      branding: { metadata: brandingMetadata.value },
    }),
    bootstrap: {
      client: { client_id: 'surface-client' },
      connections: bootstrapConnections.value,
      magic_link_enabled: magicLinkEnabled.value,
    },
  }),
}))

const cognito = {
  identifier: 'idp-cognito',
  display_name: 'AWS Cognito',
  provider: 'cognito',
  provider_type: 'enterprise',
  is_default: false,
  display_order: 1,
}

describe('LoginForm branding conformance', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    brandingMetadata.value = null
    bootstrapConnections.value = []
    magicLinkEnabled.value = false
  })

  it('renders the default heading when the tenant has no configured copy', () => {
    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    expect(screen.getByRole('heading', { name: 'Welcome back' })).toBeInTheDocument()
    expect(screen.getByText('Sign in to your account to continue.')).toBeInTheDocument()
  })

  it('uses the tenant copy configured in the console branding editor', () => {
    brandingMetadata.value = {
      login_page_content: {
        login: { title: 'Sign in to Acme', subtitle: 'Use your Acme workspace account.' },
      },
    }

    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    expect(screen.getByRole('heading', { name: 'Sign in to Acme' })).toBeInTheDocument()
    expect(screen.getByText('Use your Acme workspace account.')).toBeInTheDocument()
  })

  it('falls back to the default when a configured value is blank', () => {
    brandingMetadata.value = {
      login_page_content: { login: { title: '   ', subtitle: 'Only the subtitle is set.' } },
    }

    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    expect(screen.getByRole('heading', { name: 'Welcome back' })).toBeInTheDocument()
    expect(screen.getByText('Only the subtitle is set.')).toBeInTheDocument()
  })

  it('renders bootstrap providers on a direct visit, with no client_id or return_to', () => {
    bootstrapConnections.value = [cognito]

    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    expect(screen.getByRole('button', { name: /Continue with AWS Cognito/ })).toBeInTheDocument()
    // The bootstrap already carried them — no second round trip.
    expect(fetchOAuthConnectionsMock).not.toHaveBeenCalled()
  })

  it('orders providers above the email form, matching the console preview', () => {
    bootstrapConnections.value = [cognito]

    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    const provider = screen.getByRole('button', { name: /Continue with AWS Cognito/ })
    const email = screen.getByRole('textbox', { name: /email/i })
    const signIn = screen.getByRole('button', { name: 'Sign in' })

    // Node.DOCUMENT_POSITION_FOLLOWING === 4
    expect(provider.compareDocumentPosition(email) & 4).toBeTruthy()
    expect(email.compareDocumentPosition(signIn) & 4).toBeTruthy()
    expect(screen.getByText('or continue with email')).toBeInTheDocument()
  })

  it('omits the email divider when no providers are present', () => {
    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    expect(screen.queryByText('or continue with email')).not.toBeInTheDocument()
  })

  it('puts "Forgot password?" on the password label row, not below the input', () => {
    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    const forgot = screen.getByRole('link', { name: 'Forgot password?' })
    const label = document.querySelector('label[for="password"]')
    const input = document.getElementById('password')

    // Shares a row with the label…
    expect(forgot.parentElement).toBe(label?.parentElement)
    expect(forgot.parentElement?.className).toContain('justify-between')
    // …and sits above the input, not after it.
    expect(forgot.compareDocumentPosition(input!) & 4).toBeTruthy()
  })

  it('reads the sign-up prompt as a sentence with only the action linked', () => {
    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    const signUp = screen.getByRole('link', { name: 'Sign up' })
    expect(signUp.parentElement).toHaveTextContent("Don't have an account? Sign up")
    expect(signUp.parentElement).toHaveClass('text-muted-foreground')
    expect(screen.queryByText('Create an account')).not.toBeInTheDocument()
  })

  // Passwordless email sign-in weakens authentication to inbox possession, so
  // it must stay off until an operator turns it on for the client.
  it('hides magic-link sign-in unless the client enables it', () => {
    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    expect(screen.queryByRole('button', { name: /Email me a sign-in link/ })).not.toBeInTheDocument()
  })

  it('offers magic-link sign-in once the client enables it', () => {
    magicLinkEnabled.value = true

    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    expect(screen.getByRole('button', { name: /Email me a sign-in link/ })).toBeInTheDocument()
  })

  it('spaces every field label identically, per the shared Field primitive', () => {
    renderWithProviders(<LoginForm />, { route: '/login', path: '/login' })

    const emailField = screen.getByRole('textbox', { name: /email/i }).closest('[data-slot="field"]')
    const passwordField = document.getElementById('password')?.closest('[data-slot="field"]')

    // A stacked space-y-* on either would add to Field's gap-3 and knock the
    // two labels out of alignment.
    expect(emailField?.className).not.toMatch(/space-y-/)
    expect(passwordField?.className).not.toMatch(/space-y-/)
  })
})
