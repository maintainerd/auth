import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { BrandingPublic } from '@/services/api/tenants/types'
import LoginLayout from './LoginLayout'

type MockTenantState = { currentTenant: { branding: BrandingPublic } | null }

const useTenantMock = vi.hoisted(() => vi.fn<() => MockTenantState>(() => ({ currentTenant: null })))

vi.mock('@/hooks/useTenant', () => ({
  useTenant: useTenantMock,
}))

const branding: BrandingPublic = {
  layout: 'centered',
  company_name: 'Acme',
  logo_label: 'Acme ID',
  show_logo_label: true,
  logo_url: '',
  favicon_url: '',
  support_url: 'https://example.com/support',
  privacy_policy_url: '',
  terms_of_service_url: '',
  metadata: null,
}

describe('LoginLayout', () => {
  beforeEach(() => {
    useTenantMock.mockReturnValue({ currentTenant: null })
  })

  it.each(['centered', 'full_page', 'split'] as const)('renders the %s layout', (layout) => {
    render(
      <LoginLayout branding={{ ...branding, layout }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    expect(screen.getByRole('main')).toHaveAttribute('data-layout', layout)
    expect(screen.getByText('Authentication form')).toBeInTheDocument()
    expect(screen.getAllByText('Acme ID').length).toBeGreaterThan(0)
  })

  it('renders the split brand panel only for split layout', () => {
    const { rerender } = render(
      <LoginLayout branding={{ ...branding, layout: 'split' }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )
    expect(screen.getByTestId('split-brand-panel')).toBeInTheDocument()

    rerender(
      <LoginLayout branding={{ ...branding, layout: 'centered' }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )
    expect(screen.queryByTestId('split-brand-panel')).not.toBeInTheDocument()
  })

  it('defaults to centered when branding is absent', () => {
    render(
      <LoginLayout>
        <span>Setup form</span>
      </LoginLayout>,
    )

    expect(screen.getByRole('main')).toHaveAttribute('data-layout', 'centered')
  })

  it('inherits current tenant branding when no branding prop is provided', () => {
    useTenantMock.mockReturnValue({
      currentTenant: { branding: { ...branding, layout: 'split', company_name: 'Storefront', logo_label: 'Storefront ID' } },
    })

    render(
      <LoginLayout>
        <span>Inherited form</span>
      </LoginLayout>,
    )

    expect(screen.getByRole('main')).toHaveAttribute('data-layout', 'split')
    expect(screen.getAllByText('Storefront ID').length).toBeGreaterThan(0)
  })

  it('hides the logo label when branding disables it', () => {
    render(
      <LoginLayout branding={{ ...branding, show_logo_label: false }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    expect(screen.queryByText('Acme ID')).not.toBeInTheDocument()
  })
})
