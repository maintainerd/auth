import type { BrandingLayout } from "@/services/api/branding/types"

export const AUTH_UI_TEMPLATE_IDS = [
  "centered-card",
  "split-showcase",
  "stepper-flow",
  "editorial-cover",
] as const

export type AuthUiTemplateId = typeof AUTH_UI_TEMPLATE_IDS[number]

export const LOGIN_FORM_LOGO_PLACEMENTS = [
  { value: "inside-form", label: "Inside form" },
  { value: "above-form", label: "Above form" },
] as const

export type LoginFormLogoPlacement = typeof LOGIN_FORM_LOGO_PLACEMENTS[number]["value"]

export const SPLIT_SHOWCASE_VISUAL_STYLES = [
  { value: "default", label: "Default" },
  { value: "identity-mesh", label: "Identity mesh" },
  { value: "access-grid", label: "Access grid" },
  { value: "security-radar", label: "Security radar" },
  { value: "trust-circuit", label: "Trust circuit" },
  { value: "audit-trail", label: "Audit trail" },
  { value: "session-orbit", label: "Session orbit" },
  { value: "image", label: "Image URL" },
] as const

export type SplitShowcaseVisualStyle = typeof SPLIT_SHOWCASE_VISUAL_STYLES[number]["value"]

export type AuthUiTemplatePresentation = {
  logoPlacement: LoginFormLogoPlacement
  logoDetail: string
  splitShowcaseVisualStyle: SplitShowcaseVisualStyle
  splitShowcaseTitle: string
  splitShowcaseSubtitle: string
  splitShowcaseImageUrl: string
}

export const DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION: AuthUiTemplatePresentation = {
  // Brand above the card, so the logo reads as the page's identity rather than
  // as a row inside the form. Kept in step with the backend seeder default.
  logoPlacement: "above-form",
  logoDetail: "Open-source Cloud Platform",
  splitShowcaseVisualStyle: "default",
  splitShowcaseTitle: "Secure access for your workspace",
  splitShowcaseSubtitle: "Sign in with the protections, policies, and identity controls your team expects.",
  splitShowcaseImageUrl: "",
}

export interface AuthUiTemplate {
  id: AuthUiTemplateId
  label: string
  summary: string
  layout: BrandingLayout
  previewVariant: AuthUiTemplateId
  bestFor: string
  flowTreatment: string
  features: string[]
}

export const DEFAULT_AUTH_UI_TEMPLATE_ID: AuthUiTemplateId = "centered-card"

export const AUTH_UI_TEMPLATES: AuthUiTemplate[] = [
  {
    id: "centered-card",
    label: "Centered Card",
    summary: "A balanced card layout with brand presence, calm spacing, and predictable auth actions.",
    layout: "centered",
    previewVariant: "centered-card",
    bestFor: "General SaaS login and registration",
    flowTreatment: "All auth steps reuse a single centered container.",
    features: ["Login", "Registration", "MFA", "Recovery", "Link account"],
  },
  {
    id: "split-showcase",
    label: "Split Showcase",
    summary: "A polished two-column experience with a visual brand panel beside the auth form.",
    layout: "split",
    previewVariant: "split-showcase",
    bestFor: "Product-led and customer-facing tenants",
    flowTreatment: "Brand panel remains stable while each auth step swaps in the form panel.",
    features: ["Login", "Registration", "MFA", "Invites", "Legal links"],
  },
  {
    id: "stepper-flow",
    label: "Cover Card",
    summary: "A centered auth card with the form and visual cover enclosed in one polished surface.",
    layout: "full_page",
    previewVariant: "stepper-flow",
    bestFor: "Enterprise IAM and tenant-branded access",
    flowTreatment: "A single centered card keeps the active auth form beside a configurable visual cover.",
    features: ["Login", "Registration", "MFA", "Recovery", "Invites"],
  },
  {
    id: "editorial-cover",
    label: "Editorial Cover",
    summary: "A split cover layout with the visual panel on the opposite side and the logo reserved for the form.",
    layout: "split",
    previewVariant: "editorial-cover",
    bestFor: "Consumer-facing auth and branded communities",
    flowTreatment: "The visual panel uses the same configurable split artwork while the auth form owns the brand lockup.",
    features: ["Login", "Registration", "Social login", "Invites", "Marketing consent"],
  },
]

const TEMPLATE_MAP = new Map(AUTH_UI_TEMPLATES.map((template) => [template.id, template]))

export function getAuthUiTemplate(id: unknown): AuthUiTemplate {
  return typeof id === "string" && TEMPLATE_MAP.has(id as AuthUiTemplateId)
    ? TEMPLATE_MAP.get(id as AuthUiTemplateId)!
    : TEMPLATE_MAP.get(DEFAULT_AUTH_UI_TEMPLATE_ID)!
}

export function authUiTemplateIdFromMetadata(
  metadata: Record<string, unknown> | null | undefined,
  fallbackLayout?: BrandingLayout,
): AuthUiTemplateId {
  const value = metadata?.auth_ui_template
  if (typeof value === "string" && TEMPLATE_MAP.has(value as AuthUiTemplateId)) {
    return value as AuthUiTemplateId
  }

  if (fallbackLayout === "split") return "split-showcase"
  if (fallbackLayout === "full_page") return "stepper-flow"
  return DEFAULT_AUTH_UI_TEMPLATE_ID
}

export function authUiTemplateOptions() {
  return AUTH_UI_TEMPLATES.map((template) => ({
    value: template.id,
    label: template.label,
  }))
}

export function authUiTemplateSupportsImage(template: AuthUiTemplate): boolean {
  return template.layout === "split" || template.previewVariant === "stepper-flow"
}

export function authUiTemplatePresentationFromMetadata(
  metadata: Record<string, unknown> | null | undefined,
): AuthUiTemplatePresentation {
  const rawLogoPlacement = metadata?.login_form_logo_placement
  const rawLogoDetail = metadata?.login_form_logo_detail
  const rawVisualStyle = metadata?.split_showcase_visual_style
  const rawSplitTitle = metadata?.split_showcase_panel_title
  const rawSplitSubtitle = metadata?.split_showcase_panel_subtitle
  const rawSplitImageUrl = metadata?.split_showcase_image_url

  return {
    logoPlacement: isLoginFormLogoPlacement(rawLogoPlacement)
      ? rawLogoPlacement
      : DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.logoPlacement,
    logoDetail: typeof rawLogoDetail === 'string'
      ? rawLogoDetail.trim()
      : DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.logoDetail,
    splitShowcaseVisualStyle: isSplitShowcaseVisualStyle(rawVisualStyle)
      ? rawVisualStyle
      : DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.splitShowcaseVisualStyle,
    splitShowcaseTitle: readString(rawSplitTitle) ?? DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.splitShowcaseTitle,
    splitShowcaseSubtitle: readString(rawSplitSubtitle) ?? DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.splitShowcaseSubtitle,
    splitShowcaseImageUrl: readString(rawSplitImageUrl) ?? DEFAULT_AUTH_UI_TEMPLATE_PRESENTATION.splitShowcaseImageUrl,
  }
}

export function authUiTemplatePresentationMetadata(
  currentMetadata: Record<string, unknown> | null | undefined,
  presentation: AuthUiTemplatePresentation,
): Record<string, unknown> {
  const metadata = { ...(currentMetadata ?? {}) }
  delete metadata.login_form_card_shadow

  return {
    ...metadata,
    login_form_logo_placement: presentation.logoPlacement,
    login_form_logo_detail: presentation.logoDetail.trim(),
    split_showcase_visual_style: presentation.splitShowcaseVisualStyle,
    split_showcase_panel_title: presentation.splitShowcaseTitle.trim(),
    split_showcase_panel_subtitle: presentation.splitShowcaseSubtitle.trim(),
    split_showcase_image_url: presentation.splitShowcaseImageUrl.trim(),
  }
}

function isLoginFormLogoPlacement(value: unknown): value is LoginFormLogoPlacement {
  return LOGIN_FORM_LOGO_PLACEMENTS.some((option) => option.value === value)
}

function isSplitShowcaseVisualStyle(value: unknown): value is SplitShowcaseVisualStyle {
  return SPLIT_SHOWCASE_VISUAL_STYLES.some((option) => option.value === value)
}

function readString(value: unknown) {
  return typeof value === "string" && value.trim() ? value.trim() : undefined
}
