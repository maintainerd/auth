import type {
  BrandingColors,
  BrandingFont,
  BrandingMetadata,
} from '@/services/api/tenants/types'

const COLOR_PROPERTIES: Record<keyof BrandingColors, readonly string[]> = {
  primary: ['--branding-primary', '--primary', '--ring', '--sidebar-primary', '--chart-1'],
  secondary: ['--branding-secondary', '--secondary'],
  accent: ['--branding-accent', '--accent'],
  appBackground: ['--branding-app-background', '--background'],
  topPanelBackground: ['--branding-top-panel-background', '--popover'],
  sidePanelBackground: ['--branding-side-panel-background', '--sidebar'],
  cardBackground: ['--branding-card-background', '--card'],
  textPrimary: [
    '--branding-text-primary',
    '--foreground',
    '--card-foreground',
    '--popover-foreground',
    '--sidebar-foreground',
  ],
  textMuted: ['--branding-text-muted', '--muted-foreground'],
  border: ['--branding-border', '--border', '--input', '--sidebar-border'],
  authPageBackground: ['--auth-page-background'],
  authFormPanelBackground: ['--auth-form-panel-background'],
  authFormPanelBorder: ['--auth-form-panel-border'],
  authFormPanelText: ['--auth-form-panel-foreground'],
  authVisualPanelBackground: ['--auth-visual-panel-background'],
  authVisualPanelText: ['--auth-visual-panel-foreground'],
  authVisualPanelOverlay: ['--auth-visual-panel-overlay'],
  authDecorativeLight: ['--auth-decorative-light'],
  authDecorativeDark: ['--auth-decorative-dark'],
  authProgressPanelBackground: ['--auth-progress-panel-background'],
  authSecurityPanelBackground: ['--auth-security-panel-background'],
}

const FOREGROUND_PROPERTIES: Partial<Record<keyof BrandingColors, readonly string[]>> = {
  primary: ['--primary-foreground'],
  secondary: ['--secondary-foreground'],
  accent: ['--accent-foreground'],
  topPanelBackground: ['--branding-top-panel-foreground'],
  sidePanelBackground: ['--branding-side-panel-foreground'],
}

const HEX_COLOR = /^#(?:[\da-f]{3}|[\da-f]{6})$/i

type PreviousProperty = {
  value: string
  priority: string
}

function contrastColor(value: string): string {
  const short = value.length === 4
  const hex = short
    ? value
        .slice(1)
        .split('')
        .map((digit) => `${digit}${digit}`)
        .join('')
    : value.slice(1)
  const [red, green, blue] = [0, 2, 4].map((offset) => Number.parseInt(hex.slice(offset, offset + 2), 16))
  const channels = [red, green, blue].map((channel) => {
    const normalized = channel / 255
    return normalized <= 0.03928
      ? normalized / 12.92
      : ((normalized + 0.055) / 1.055) ** 2.4
  })
  const luminance = 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2]
  const whiteContrast = 1.05 / (luminance + 0.05)
  const darkContrast = (luminance + 0.05) / 0.007
  return whiteContrast >= darkContrast ? '#ffffff' : '#0f172a'
}

function safeFontFamily(value: string | undefined): string | undefined {
  const trimmed = value?.trim()
  if (!trimmed || /[;{}]|url\s*\(/i.test(trimmed)) return undefined
  return trimmed
}

function safeBackground(value: string | undefined): string | undefined {
  const trimmed = value?.trim()
  if (!trimmed || /[;{}]|url\s*\(|image-set\s*\(/i.test(trimmed)) return undefined
  if (HEX_COLOR.test(trimmed) || /^(?:linear|radial)-gradient\(/i.test(trimmed)) return trimmed
  return undefined
}

/**
 * Returns the page background requested by branding metadata. Explicit gradient
 * and background values are supported for forward compatibility; the current
 * console-controlled appBackground color is the fallback.
 */
export function getBrandingBackground(metadata: BrandingMetadata | null | undefined): string | undefined {
  return safeBackground(metadata?.gradient)
    ?? safeBackground(metadata?.background)
    ?? safeBackground(metadata?.colors?.appBackground)
}

/**
 * Applies one branding palette to the document root and returns a cleanup that
 * restores every previous inline value. CSS owns the defaults; this helper only
 * supplies tenant overrides.
 */
export function applyBranding(
  colors: BrandingColors | undefined,
  font: BrandingFont | undefined,
  background: string | undefined,
): () => void {
  const root = document.documentElement
  const previous = new Map<string, PreviousProperty>()

  const setProperty = (property: string, value: string) => {
    if (!previous.has(property)) {
      previous.set(property, {
        value: root.style.getPropertyValue(property),
        priority: root.style.getPropertyPriority(property),
      })
    }
    root.style.setProperty(property, value)
  }

  for (const [name, properties] of Object.entries(COLOR_PROPERTIES) as [
    keyof BrandingColors,
    readonly string[],
  ][]) {
    const value = colors?.[name]?.trim()
    if (!value || !HEX_COLOR.test(value)) continue
    properties.forEach((property) => setProperty(property, value))

    const foregroundProperties = FOREGROUND_PROPERTIES[name]
    foregroundProperties?.forEach((property) => setProperty(property, contrastColor(value)))
  }

  const fontFamily = safeFontFamily(font?.family)
  if (fontFamily) setProperty('--font-family', fontFamily)

  const pageBackground = safeBackground(background)
    ?? safeBackground(colors?.authPageBackground)
    ?? safeBackground(colors?.appBackground)
  if (pageBackground) setProperty('--auth-page-background', pageBackground)

  return () => {
    for (const [property, original] of previous) {
      if (original.value) {
        root.style.setProperty(property, original.value, original.priority)
      } else {
        root.style.removeProperty(property)
      }
    }
  }
}
