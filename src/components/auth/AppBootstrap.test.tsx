import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { TenantEntity } from '@/services/api/tenants/types'
import { AppBootstrap } from './AppBootstrap'

const initializeTenantMock = vi.hoisted(() => vi.fn())

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
