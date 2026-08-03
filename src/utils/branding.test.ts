import { afterEach, describe, expect, it } from 'vitest'
import { applyBranding, getBrandingBackground, resolveBrandingLogoUrl } from './branding'

afterEach(() => {
  document.documentElement.removeAttribute('style')
  document.documentElement.removeAttribute('data-identity-theme')
  document.title = ''
})

describe('applyBranding', () => {
  it('maps the complete tenant palette, font, and reusable component tokens', () => {
    const cleanup = applyBranding({
      layout: 'centered',
      company_name: 'Acme',
      logo_url: '',
      favicon_url: '',
      support_url: '',
      privacy_policy_url: '',
      terms_of_service_url: '',
      metadata: {
        colors: {
          primary: '#2563eb',
          secondary: '#64748b',
          accent: '#0ea5e9',
          appBackground: '#f8fafc',
          topPanelBackground: '#ffffff',
          sidePanelBackground: '#0f172a',
          cardBackground: '#fefefe',
          textPrimary: '#111827',
          textMuted: '#6b7280',
          border: '#e5e7eb',
          authPageBackground: '#eef2f8',
          authFormPanelBackground: '#ffffff',
          authFormPanelBorder: '#cbd5e1',
          authFormPanelText: '#172033',
          authVisualPanelBackground: '#1d4ed8',
          authVisualPanelText: '#f8fafc',
          authVisualPanelOverlay: '#0f172a',
          authDecorativeLight: '#ffffff',
          authDecorativeDark: '#000000',
          authProgressPanelBackground: '#f8fafc',
          authSecurityPanelBackground: '#f9fafb',
        },
        font: { family: 'Inter, system-ui, sans-serif' },
        background: '#f8fafc',
        effects: {
          authFormPanelShadow: '0 20px 60px rgba(15, 23, 42, 0.24)',
        },
        components: {
          primaryButton: {
            background: '#0f172a',
            hoverColor: '#1e293b',
            borderColor: '#334155',
            borderThickness: '2px',
            borderRadius: '6px',
            textColor: '#ffffff',
            size: 'lg',
          },
          card: {
            background: '#fefefe',
            borderColor: '#dbe4ef',
            borderThickness: '1px',
            borderRadius: '8px',
            textColor: '#111827',
            size: 'md',
          },
          input: {
            background: '#ffffff',
            hoverColor: '#f8fafc',
            borderColor: '#cbd5e1',
            borderRadius: '5px',
            textColor: '#111827',
            size: 'sm',
          },
          iconContainer: {
            background: '#e0f2fe',
            borderColor: '#bae6fd',
            textColor: '#075985',
            size: 'md',
          },
        },
      },
    })

    const style = document.documentElement.style
    expect(document.documentElement).toHaveAttribute('data-identity-theme', 'active')
    expect(style.getPropertyValue('--primary')).toBe('#2563eb')
    expect(style.getPropertyValue('--ring')).toBe('#2563eb')
    expect(style.getPropertyValue('--secondary')).toBe('#64748b')
    expect(style.getPropertyValue('--accent')).toBe('#0ea5e9')
    expect(style.getPropertyValue('--background')).toBe('#f8fafc')
    expect(style.getPropertyValue('--md-top-panel-bg')).toBe('#ffffff')
    expect(style.getPropertyValue('--popover')).toBe('#fefefe')
    expect(style.getPropertyValue('--sidebar')).toBe('#0f172a')
    expect(style.getPropertyValue('--branding-side-panel-foreground')).toBe('#ffffff')
    expect(style.getPropertyValue('--card')).toBe('#fefefe')
    expect(style.getPropertyValue('--foreground')).toBe('#111827')
    expect(style.getPropertyValue('--muted-foreground')).toBe('#6b7280')
    expect(style.getPropertyValue('--border')).toBe('#e5e7eb')
    expect(style.getPropertyValue('--font-family')).toBe('Inter, system-ui, sans-serif')
    expect(style.getPropertyValue('--auth-page-background')).toBe('#f8fafc')
    expect(style.getPropertyValue('--auth-form-panel-background')).toBe('#ffffff')
    expect(style.getPropertyValue('--auth-form-panel-border')).toBe('#cbd5e1')
    expect(style.getPropertyValue('--auth-form-panel-foreground')).toBe('#172033')
    expect(style.getPropertyValue('--auth-visual-panel-background')).toBe('#1d4ed8')
    expect(style.getPropertyValue('--auth-visual-panel-foreground')).toBe('#f8fafc')
    expect(style.getPropertyValue('--auth-visual-panel-overlay')).toBe('#0f172a')
    expect(style.getPropertyValue('--auth-decorative-light')).toBe('#ffffff')
    expect(style.getPropertyValue('--auth-decorative-dark')).toBe('#000000')
    expect(style.getPropertyValue('--auth-progress-panel-background')).toBe('#f8fafc')
    expect(style.getPropertyValue('--auth-security-panel-background')).toBe('#f9fafb')
    expect(style.getPropertyValue('--md-auth-form-shadow')).toBe('0 20px 60px rgba(15, 23, 42, 0.24)')
    expect(style.getPropertyValue('--md-button-primary-bg')).toBe('#0f172a')
    expect(style.getPropertyValue('--md-button-primary-hover')).toBe('#1e293b')
    expect(style.getPropertyValue('--md-button-primary-border')).toBe('#334155')
    expect(style.getPropertyValue('--md-button-primary-border-width')).toBe('2px')
    expect(style.getPropertyValue('--md-button-primary-radius')).toBe('6px')
    expect(style.getPropertyValue('--md-button-primary-text')).toBe('#ffffff')
    expect(style.getPropertyValue('--md-button-primary-height')).toBe('2.75rem')
    expect(style.getPropertyValue('--md-card-bg')).toBe('#fefefe')
    expect(style.getPropertyValue('--md-card-radius')).toBe('8px')
    expect(style.getPropertyValue('--md-input-bg')).toBe('#ffffff')
    expect(style.getPropertyValue('--md-input-height')).toBe('2.25rem')
    expect(style.getPropertyValue('--md-icon-container-bg')).toBe('#e0f2fe')

    cleanup()
    expect(style.getPropertyValue('--primary')).toBe('')
    expect(style.getPropertyValue('--font-family')).toBe('')
    expect(style.getPropertyValue('--md-button-primary-bg')).toBe('')
    expect(document.documentElement).not.toHaveAttribute('data-identity-theme')
  })

  it('restores pre-existing inline theme values during cleanup', () => {
    document.documentElement.style.setProperty('--primary', '#123456')

    const cleanup = applyBranding({ primary: '#abcdef' }, undefined, undefined)
    expect(document.documentElement.style.getPropertyValue('--primary')).toBe('#abcdef')

    cleanup()
    expect(document.documentElement.style.getPropertyValue('--primary')).toBe('#123456')
  })

  it('ignores unsafe or malformed theme values', () => {
    applyBranding(
      { primary: 'url(https://example.test/pixel)', appBackground: 'red; color: transparent' },
      { family: 'Inter; display: none' },
      'url(https://example.test/background)',
    )

    const style = document.documentElement.style
    expect(style.getPropertyValue('--primary')).toBe('')
    expect(style.getPropertyValue('--font-family')).toBe('')
    expect(style.getPropertyValue('--auth-page-background')).toBe('')
  })
})

describe('getBrandingBackground', () => {
  it('prefers a gradient, then a background, then appBackground', () => {
    expect(getBrandingBackground({
      gradient: 'linear-gradient(180deg, #ffffff 0%, #2563eb 100%)',
      background: '#abcdef',
      colors: { appBackground: '#f8fafc' },
    })).toBe('linear-gradient(180deg, #ffffff 0%, #2563eb 100%)')

    expect(getBrandingBackground({
      background: '#abcdef',
      colors: { appBackground: '#f8fafc' },
    })).toBe('#abcdef')

    expect(getBrandingBackground({ colors: { appBackground: '#f8fafc' } })).toBe('#f8fafc')
  })
})

describe('resolveBrandingLogoUrl', () => {
  it('keeps direct logo URLs unchanged', () => {
    expect(resolveBrandingLogoUrl('https://example.com/logo.png')).toBe('https://example.com/logo.png')
  })

  it('keeps uploaded logo paths relative in the Vite proxy', () => {
    expect(resolveBrandingLogoUrl('/public/branding/acme/logo')).toBe('/public/branding/acme/logo')
  })

  it('returns null when no logo is configured', () => {
    expect(resolveBrandingLogoUrl('')).toBeNull()
    expect(resolveBrandingLogoUrl(null)).toBeNull()
    expect(resolveBrandingLogoUrl(undefined)).toBeNull()
  })
})
