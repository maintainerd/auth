import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import AccountLayout from './AccountLayout'

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({
    currentTenant: {
      branding: {
        company_name: 'Acme',
        logo_label: 'Acme ID',
        show_logo_label: true,
        logo_url: '',
      },
    },
  }),
}))

vi.mock('@/services/api/auth', () => ({
  logout: vi.fn().mockResolvedValue(undefined),
}))

function renderAccountLayout(path = '/account/security') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <AccountLayout title="Security">
          <span>Account content</span>
        </AccountLayout>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AccountLayout', () => {
  it('exposes reusable theme slots for account chrome', () => {
    const { container } = renderAccountLayout()

    expect(container.querySelector('[data-auth-identity-account-shell]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-top-panel]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-top-profile-trigger]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-sidebar]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-sidebar-section]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-sidebar-section-label]')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Security' })).toHaveAttribute('data-active', 'true')
    expect(screen.getByText('Account content')).toBeInTheDocument()
  })
})
