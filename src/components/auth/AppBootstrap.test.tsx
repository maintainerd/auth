import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { TenantEntity } from '@/services/api/tenants/types'
import { AppBootstrap } from './AppBootstrap'

const initializeTenantMock = vi.hoisted(() => vi.fn())
const authStatus = vi.hoisted(() => ({ value: 'authenticated' as string }))

const tenant = {
  branding: {
    layout: 'centered',
    company_name: 'Acme',
    logo_label: 'Acme ID',
    show_logo_label: true,
    logo_url: '',
    favicon_url: '',
    support_url: '',
    privacy_policy_url: '',
    terms_of_service_url: '',
    metadata: {
      colors: {
        primary: '#7c3aed',
        appBackground: '#f5f3ff',
        cardBackground: '#ffffff',
        textPrimary: '#2e1065',
      },
      font: { family: 'Georgia, serif' },
      gradient: 'linear-gradient(180deg, #f5f3ff 0%, #ddd6fe 100%)',
    },
  },
} as TenantEntity

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({
    initializeAuth: vi.fn().mockResolvedValue(undefined),
    isInitialized: true,
    status: authStatus.value,
  }),
}))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: () => ({
    initializeTenant: initializeTenantMock,
    currentTenant: tenant,
    error: null,
  }),
}))

vi.mock('./RouteGuard', () => ({
  RouteGuard: ({ children }: { children: React.ReactNode }) => children,
}))

afterEach(() => {
  initializeTenantMock.mockReset()
  initializeTenantMock.mockResolvedValue(tenant)
  document.documentElement.removeAttribute('style')
})

describe('AppBootstrap branding', () => {
  beforeEach(() => {
    initializeTenantMock.mockResolvedValue(tenant)
  })

  it('applies tenant branding to every hosted auth route and cleans it up on unmount', async () => {
    const { unmount } = render(
      <MemoryRouter initialEntries={['/login']}>
        <AppBootstrap>
          <span>Hosted login</span>
        </AppBootstrap>
      </MemoryRouter>,
    )

    expect(await screen.findByText('Hosted login')).toBeInTheDocument()

    await waitFor(() => {
      const style = document.documentElement.style
      expect(style.getPropertyValue('--primary')).toBe('#7c3aed')
      expect(style.getPropertyValue('--background')).toBe('#f5f3ff')
      expect(style.getPropertyValue('--card')).toBe('#ffffff')
      expect(style.getPropertyValue('--foreground')).toBe('#2e1065')
      expect(style.getPropertyValue('--font-family')).toBe('Georgia, serif')
      expect(style.getPropertyValue('--auth-page-background')).toBe(
        'linear-gradient(180deg, #f5f3ff 0%, #ddd6fe 100%)',
      )
    })

    unmount()
    expect(document.documentElement.style.getPropertyValue('--primary')).toBe('')
    expect(document.documentElement.style.getPropertyValue('--auth-page-background')).toBe('')
  })

  it('passes URL client_id to tenant bootstrap', async () => {
    render(
      <MemoryRouter initialEntries={['/oauth/authorize?client_id=client-abc']}>
        <AppBootstrap>
          <span>Authorize</span>
        </AppBootstrap>
      </MemoryRouter>,
    )

    expect(await screen.findByText('Authorize')).toBeInTheDocument()
    expect(initializeTenantMock).toHaveBeenCalledWith('client-abc')
  })
})

/**
 * The gate owns the decision. Nothing downstream may render — and therefore
 * nothing may redirect — until the session verdict is in. This is what stops a
 * signed-in user glimpsing the login / no-access page on first load.
 */
describe('AppBootstrap gate', () => {
  afterEach(() => {
    authStatus.value = 'authenticated'
  })

  it('holds the loading screen while the session is still unknown', async () => {
    authStatus.value = 'unknown'

    render(
      <MemoryRouter>
        <AppBootstrap>
          <div data-testid="route-tree">routes</div>
        </AppBootstrap>
      </MemoryRouter>,
    )

    await waitFor(() => expect(initializeTenantMock).toHaveBeenCalled())
    // Route tree must NOT have rendered: a guard inside it would redirect on a
    // session state we have not established yet.
    expect(screen.queryByTestId('route-tree')).not.toBeInTheDocument()
  })

  it('renders the route tree once the verdict is in', async () => {
    authStatus.value = 'anonymous'

    render(
      <MemoryRouter>
        <AppBootstrap>
          <div data-testid="route-tree">routes</div>
        </AppBootstrap>
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByTestId('route-tree')).toBeInTheDocument())
  })
})
