import type { BrandingLayout } from "@/services/api/branding/types"

export const AUTH_UI_TEMPLATE_IDS = [
  "centered-card",
  "split-showcase",
  "full-page-minimal",
  "side-panel",
  "stepper-flow",
  "compact-modal",
  "security-console",
  "editorial-cover",
] as const

export type AuthUiTemplateId = typeof AUTH_UI_TEMPLATE_IDS[number]

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
    id: "full-page-minimal",
    label: "Full Page Minimal",
    summary: "A restrained full-page layout with a slim header and generous reading space.",
    layout: "full_page",
    previewVariant: "full-page-minimal",
    bestFor: "Enterprise portals and internal tools",
    flowTreatment: "Each step gets a full-page section with restrained navigation.",
    features: ["Login", "Passwordless", "MFA", "Account linking", "Error states"],
  },
  {
    id: "side-panel",
    label: "Side Panel",
    summary: "A flexible side-context layout with brand, support, and policy cues beside the auth form.",
    layout: "split",
    previewVariant: "side-panel",
    bestFor: "SaaS products, customer portals, workforce tools, and partner apps",
    flowTreatment: "A consistent context panel stays visible while each fixed auth page renders in the main area.",
    features: ["Login", "Registration", "Recovery", "Support links", "Brand context"],
  },
  {
    id: "stepper-flow",
    label: "Guided Steps",
    summary: "A professional step-by-step onboarding layout for longer registration and recovery journeys.",
    layout: "full_page",
    previewVariant: "stepper-flow",
    bestFor: "Onboarding-heavy tenants",
    flowTreatment: "Progress markers orient users across registration and verification steps.",
    features: ["Registration", "MFA setup", "Email verify", "Phone verify", "Link account"],
  },
  {
    id: "compact-modal",
    label: "Compact Dialog",
    summary: "A polished modal-style surface that keeps auth focused above a quiet app backdrop.",
    layout: "centered",
    previewVariant: "compact-modal",
    bestFor: "Embedded sign-in, product overlays, portals, and developer tools",
    flowTreatment: "Auth pages stay inside a compact dialog with a visible brand header and light surrounding context.",
    features: ["Login", "MFA", "Passkeys", "Recovery", "Invites"],
  },
  {
    id: "security-console",
    label: "Assurance Console",
    summary: "A trust-forward layout with session, policy, and device context presented cleanly.",
    layout: "full_page",
    previewVariant: "security-console",
    bestFor: "High-assurance and workforce identity",
    flowTreatment: "Challenge screens reserve space for risk, device, and policy messaging.",
    features: ["Login", "Step-up", "MFA", "Device trust", "Threat prompts"],
  },
  {
    id: "editorial-cover",
    label: "Editorial Cover",
    summary: "A premium cover layout with an expressive visual panel and composed form placement.",
    layout: "split",
    previewVariant: "editorial-cover",
    bestFor: "Consumer-facing auth and branded communities",
    flowTreatment: "Cover panel remains expressive while auth steps stay predictable.",
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
  if (fallbackLayout === "full_page") return "full-page-minimal"
  return DEFAULT_AUTH_UI_TEMPLATE_ID
}

export function authUiTemplateOptions() {
  return AUTH_UI_TEMPLATES.map((template) => ({
    value: template.id,
    label: template.label,
  }))
}

export function authUiTemplateSupportsImage(template: AuthUiTemplate): boolean {
  return template.layout === "split"
}
