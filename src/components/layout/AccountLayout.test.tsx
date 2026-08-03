import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import AccountLayout from './AccountLayout'

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({
    currentTenant: {
      branding: {
        company_name: 'Acme',
        logo_label: 'Acme ID',
        show_logo_label: true,
        logo_url: 'https://console-assets.example/logo.svg',
        metadata: {
          login_form_logo_detail: 'Identity access',
        },
      },
    },
  }),
}))

// Sign-out goes through the auth store, not the bare API call, so that Redux
// actually drops the session instead of the shell continuing to render as a
// signed-in user.
const logoutMock = vi.fn<() => Promise<void>>().mockResolvedValue(undefined)

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ logout: logoutMock }),
}))

function renderAccountLayout(path = '/account/security') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/login" element={<span>Login page</span>} />
          <Route
            path="*"
            element={
              <AccountLayout title="Security">
                <span>Account content</span>
              </AccountLayout>
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AccountLayout', () => {
  it('exposes reusable theme slots for account chrome', () => {
    const { container } = renderAccountLayout()

    expect(container.querySelector('[data-auth-identity-account-shell]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-top-panel]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-top-logout]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-sidebar]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-sidebar-section]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-sidebar-section-label]')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Security' })).toHaveAttribute('data-active', 'true')
    expect(screen.getByText('Account content')).toBeInTheDocument()
  })

  it('uses login-template label details and ignores console asset logo URLs in the top panel', () => {
    renderAccountLayout()

    expect(screen.getByText('Acme ID')).toBeInTheDocument()
    expect(screen.getByText('Identity access')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Acme ID' })).toHaveAttribute('src', '/logo.png')
  })

  it('lands on the login page after signing out', async () => {
    logoutMock.mockResolvedValueOnce(undefined)
    renderAccountLayout()

    await userEvent.click(screen.getByRole('button', { name: /sign out/i }))

    await waitFor(() => expect(screen.getByText('Login page')).toBeInTheDocument())
  })

  // The redirect used to hang off onSuccess, so any error response skipped it and
  // left the user on an account page with no session behind it. A 401 here is the
  // NORMAL case when the other surface in this browser already ended the shared
  // session — this browser is signed out either way and must land on /login.
  it('still lands on the login page when the logout request fails', async () => {
    logoutMock.mockRejectedValueOnce(new Error('session already gone'))
    renderAccountLayout()

    await userEvent.click(screen.getByRole('button', { name: /sign out/i }))

    await waitFor(() => expect(screen.getByText('Login page')).toBeInTheDocument())
  })
})
