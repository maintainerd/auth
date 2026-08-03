import type { BrandingLayout, BrandingMetadata } from '@/services/api/tenants/types'

export const AUTH_UI_TEMPLATE_IDS = [
  'centered-card',
  'split-showcase',
  'stepper-flow',
  'editorial-cover',
] as const

export type AuthUiTemplateId = typeof AUTH_UI_TEMPLATE_IDS[number]

export type LogoPlacement = 'inside-form' | 'above-form'

export const SPLIT_VISUAL_STYLES = [
  'default',
  'identity-mesh',
  'access-grid',
  'security-radar',
  'trust-circuit',
  'audit-trail',
  'session-orbit',
  'image',
] as const

export type SplitVisualStyle = typeof SPLIT_VISUAL_STYLES[number]

export type AuthTemplatePresentation = {
  logoPlacement: LogoPlacement
  logoDetail: string
  splitShowcaseVisualStyle: SplitVisualStyle
  splitShowcaseTitle: string
  splitShowcaseSubtitle: string
  splitShowcaseImageUrl: string
}

export const DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION: AuthTemplatePresentation = {
  logoPlacement: 'inside-form',
  logoDetail: '',
  splitShowcaseVisualStyle: 'default',
  splitShowcaseTitle: 'Secure access for your workspace',
  splitShowcaseSubtitle: 'Sign in with the protections, policies, and identity controls your team expects.',
  splitShowcaseImageUrl: '',
}

export function authUiTemplateIdFromMetadata(
  metadata: BrandingMetadata | null | undefined,
  fallbackLayout: BrandingLayout,
): AuthUiTemplateId {
  const value = metadata?.auth_ui_template
  if (typeof value === 'string' && AUTH_UI_TEMPLATE_IDS.includes(value as AuthUiTemplateId)) {
    return value as AuthUiTemplateId
  }
  if (fallbackLayout === 'split') return 'split-showcase'
  if (fallbackLayout === 'full_page') return 'stepper-flow'
  return 'centered-card'
}

export function authUiTemplatePresentationFromMetadata(
  metadata: BrandingMetadata | null | undefined,
): AuthTemplatePresentation {
  const rawLogoPlacement = metadata?.login_form_logo_placement
  const rawLogoDetail = metadata?.login_form_logo_detail
  const rawVisualStyle = metadata?.split_showcase_visual_style
  return {
    logoPlacement: rawLogoPlacement === 'above-form' || rawLogoPlacement === 'inside-form'
      ? rawLogoPlacement
      : DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.logoPlacement,
    logoDetail: readString(rawLogoDetail) ?? DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.logoDetail,
    splitShowcaseVisualStyle:
      typeof rawVisualStyle === 'string' && SPLIT_VISUAL_STYLES.includes(rawVisualStyle as SplitVisualStyle)
        ? rawVisualStyle as SplitVisualStyle
        : DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.splitShowcaseVisualStyle,
    splitShowcaseTitle: readString(metadata?.split_showcase_panel_title)
      ?? DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.splitShowcaseTitle,
    splitShowcaseSubtitle: readString(metadata?.split_showcase_panel_subtitle)
      ?? DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.splitShowcaseSubtitle,
    splitShowcaseImageUrl: readString(metadata?.split_showcase_image_url)
      ?? DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.splitShowcaseImageUrl,
  }
}

export function safeAuthTemplateImageUrl(value: string): string | undefined {
  if (!value || /[<>"'{}]/.test(value)) return undefined
  if (value.startsWith('/') || value.startsWith('https://') || value.startsWith('http://')) return value
  return undefined
}

function readString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}
