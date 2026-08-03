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

function parentLockupForText(text: string, root: ParentNode = document.body): HTMLElement {
  const label = Array.from(root.querySelectorAll('p')).find((element) => element.textContent === text)
  if (!(label instanceof HTMLElement)) {
    throw new Error(`Unable to find logo label for ${text}`)
  }
  const lockup = label.closest('.flex.items-center.gap-3')
  if (!(lockup instanceof HTMLElement)) {
    throw new Error(`Unable to find logo lockup for ${text}`)
  }
  return lockup
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
    expect(screen.getByRole('main')).toHaveAttribute('data-auth-identity-shell')
    expect(screen.getByText('Authentication form')).toBeInTheDocument()
    expect(screen.getAllByText('Acme ID').length).toBeGreaterThan(0)
  })

  it('renders the template brand lockup as logo beside logo label', () => {
    render(
      <LoginLayout branding={{ ...branding, metadata: { login_form_logo_detail: 'Identity access' } }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    const logo = screen.getByAltText('Acme ID')
    const label = screen.getByText('Acme ID')
    expect(logo.parentElement).toHaveClass('flex', 'items-center', 'gap-3')
    expect(logo.parentElement).toContainElement(label)
    expect(screen.getByText('Identity access')).toBeInTheDocument()
  })

  it('centers form-owned logo lockups for centered and cover templates', () => {
    const { rerender } = render(
      <LoginLayout branding={{ ...branding, metadata: { auth_ui_template: 'centered-card' } }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    expect(parentLockupForText('Acme ID')).toHaveClass('justify-center')

    rerender(
      <LoginLayout branding={{ ...branding, layout: 'full_page', metadata: { auth_ui_template: 'stepper-flow' } }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    expect(parentLockupForText('Acme ID')).toHaveClass('justify-center')
  })

  it('left-aligns brand-panel and editorial logo lockups like the template previews', () => {
    const { rerender } = render(
      <LoginLayout branding={{ ...branding, layout: 'split', metadata: { auth_ui_template: 'split-showcase' } }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    expect(parentLockupForText('Acme ID', screen.getByTestId('split-brand-panel'))).not.toHaveClass('justify-center')

    rerender(
      <LoginLayout branding={{ ...branding, layout: 'split', metadata: { auth_ui_template: 'editorial-cover' } }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    const editorialPanel = screen.getByRole('main').querySelector('section[data-auth-identity-card]')
    expect(parentLockupForText('Acme ID', editorialPanel ?? document.body)).not.toHaveClass('justify-center')
  })

  it('uses the identity fallback label and enlarges it when logo detail is blank', () => {
    render(
      <LoginLayout branding={null}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    const label = screen.getByText('Maintainerd')
    expect(label).toHaveClass('text-lg')
    expect(screen.queryByText('Identity access')).not.toBeInTheDocument()
  })

  it.each([
    ['centered-card', 'centered'],
    ['split-showcase', 'split'],
    ['stepper-flow', 'full_page'],
    ['editorial-cover', 'split'],
  ] as const)('honors the %s auth UI template structure', (templateId, layout) => {
    render(
      <LoginLayout
        branding={{
          ...branding,
          layout,
          metadata: {
            auth_ui_template: templateId,
            login_form_logo_placement: 'inside-form',
            split_showcase_panel_title: 'Access the workspace',
            split_showcase_panel_subtitle: 'Use your verified identity to continue.',
            split_showcase_visual_style: 'identity-mesh',
          },
        }}
      >
        <span>Authentication form</span>
      </LoginLayout>,
    )

    expect(screen.getByRole('main')).toHaveAttribute('data-auth-ui-template', templateId)
    expect(screen.getByRole('main')).toHaveAttribute('data-layout', layout)
    if (templateId !== 'centered-card') {
      expect(screen.getByTestId('split-brand-panel')).toBeInTheDocument()
      expect(screen.getByText('Access the workspace')).toBeInTheDocument()
      expect(screen.getByText('Use your verified identity to continue.')).toBeInTheDocument()
    }
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
      <LoginLayout branding={{ ...branding, show_logo_label: false, metadata: { login_form_logo_detail: 'Identity access' } }}>
        <span>Authentication form</span>
      </LoginLayout>,
    )

    expect(screen.queryByText('Acme ID')).not.toBeInTheDocument()
    expect(screen.queryByText('Identity access')).not.toBeInTheDocument()
  })
})
