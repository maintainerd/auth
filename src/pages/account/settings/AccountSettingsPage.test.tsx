import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import AccountSettingsPage from './AccountSettingsPage'

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

vi.mock('@/services/api/account', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/account')>()
  return {
    ...actual,
    fetchUserSettings: vi.fn().mockResolvedValue({ language: 'en-US', timezone: 'UTC' }),
    updateUserSettings: vi.fn().mockResolvedValue(undefined),
  }
})

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/account/settings']}>
        <AccountSettingsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AccountSettingsPage', () => {
  it('uses the console-style preferences shape without an identity theme switch', async () => {
    renderPage()

    expect(await screen.findByText('Localization')).toBeInTheDocument()
    expect(screen.getByText('Your timezone and language preferences.')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('Timezone')).toBeInTheDocument())
    expect(screen.getByText('Language')).toBeInTheDocument()
    expect(screen.queryByText('Theme')).not.toBeInTheDocument()
    expect(screen.queryByText('Notifications')).not.toBeInTheDocument()
  })
})
