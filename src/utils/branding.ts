import type {
  BrandingColors,
  BrandingFont,
  BrandingMetadata,
  BrandingPublic,
} from '@/services/api/tenants/types'

type PreviousProperty = {
  value: string
  priority: string
}

type ComponentConfig = {
  background?: string
  hoverColor?: string
  borderColor?: string
  borderThickness?: string
  borderRadius?: string
  textColor?: string
  size?: string
}

type NormalizedBranding = {
  metadata?: BrandingMetadata | null
  title?: string
  faviconUrl?: string
  legacyFont?: BrandingFont
  legacyBackground?: string
}

const MANAGED_PREFIXES = ['--branding-', '--auth-', '--md-']

const ROOT_THEME_PROPERTIES = [
  '--primary',
  '--primary-foreground',
  '--secondary',
  '--secondary-foreground',
  '--accent',
  '--accent-foreground',
  '--muted',
  '--muted-foreground',
  '--background',
  '--foreground',
  '--card',
  '--card-foreground',
  '--popover',
  '--popover-foreground',
  '--border',
  '--input',
  '--ring',
  '--chart-1',
  '--sidebar',
  '--sidebar-foreground',
  '--sidebar-primary',
  '--sidebar-primary-foreground',
  '--sidebar-accent',
  '--sidebar-accent-foreground',
  '--sidebar-border',
  '--font-family',
] as const

const COLOR_TOKEN_VARS: Record<keyof BrandingColors, readonly string[]> = {
  primary: ['--branding-primary', '--primary', '--ring', '--sidebar-primary', '--chart-1'],
  secondary: ['--branding-secondary', '--secondary'],
  accent: ['--branding-accent', '--accent', '--muted', '--sidebar-accent'],
  appBackground: ['--branding-app-background', '--background'],
  topPanelBackground: ['--branding-top-panel-background', '--md-top-panel-bg'],
  sidePanelBackground: ['--branding-side-panel-background', '--sidebar', '--md-side-panel-bg'],
  cardBackground: ['--branding-card-background', '--card', '--popover'],
  textPrimary: [
    '--branding-text-primary',
    '--foreground',
    '--card-foreground',
    '--popover-foreground',
    '--secondary-foreground',
    '--accent-foreground',
    '--sidebar-foreground',
    '--sidebar-accent-foreground',
  ],
  textMuted: ['--branding-text-muted', '--muted-foreground'],
  border: ['--branding-border', '--border', '--input', '--sidebar-border'],
  authPageBackground: ['--auth-page-background', '--md-auth-page-bg'],
  authFormPanelBackground: ['--auth-form-panel-background', '--md-auth-form-bg'],
  authFormPanelBorder: ['--auth-form-panel-border', '--md-auth-form-border'],
  authFormPanelText: ['--auth-form-panel-foreground', '--md-auth-form-text'],
  authVisualPanelBackground: ['--auth-visual-panel-background'],
  authVisualPanelText: ['--auth-visual-panel-foreground'],
  authVisualPanelOverlay: ['--auth-visual-panel-overlay'],
  authDecorativeLight: ['--auth-decorative-light'],
  authDecorativeDark: ['--auth-decorative-dark'],
  authProgressPanelBackground: ['--auth-progress-panel-background'],
  authSecurityPanelBackground: ['--auth-security-panel-background'],
}

const FOREGROUND_TOKEN_VARS: Partial<Record<keyof BrandingColors, readonly string[]>> = {
  primary: ['--primary-foreground', '--sidebar-primary-foreground'],
  secondary: ['--secondary-foreground'],
  accent: ['--accent-foreground', '--sidebar-accent-foreground'],
  topPanelBackground: ['--branding-top-panel-foreground', '--md-top-panel-text'],
  sidePanelBackground: ['--branding-side-panel-foreground', '--sidebar-foreground'],
}

const DIRECT_TOKEN_VARS = [
  { path: ['colors', 'topPanelBorder'], properties: ['--md-top-panel-border'] },
  { path: ['colors', 'topPanelText'], properties: ['--md-top-panel-text'] },
  { path: ['colors', 'sidePanelBorder'], properties: ['--md-side-panel-border'] },
  { path: ['colors', 'sidePanelSectionText'], properties: ['--md-side-panel-section-text'] },
  { path: ['colors', 'sidePanelItemIcon'], properties: ['--md-side-panel-item-icon'] },
  { path: ['colors', 'sidePanelItemIconHover'], properties: ['--md-side-panel-item-icon-hover'] },
  { path: ['colors', 'sidePanelItemActiveIcon'], properties: ['--md-side-panel-item-active-icon'] },
  { path: ['colors', 'sidePanelItemHoverText'], properties: ['--md-side-panel-item-hover-text'] },
  { path: ['colors', 'sidePanelChevron'], properties: ['--md-side-panel-chevron'] },
  { path: ['effects', 'authFormPanelShadow'], properties: ['--md-auth-form-shadow'] },
] as const

const COMPONENT_PREFIXES: Record<string, string> = {
  topPanelControl: '--md-top-control',
  topPanelDropdownTrigger: '--md-top-dropdown',
  topPanelProfileTrigger: '--md-top-profile',
  topPanelCreateButton: '--md-top-create',
  sidePanelSectionLabel: '--md-sidebar-section',
  sidePanelItem: '--md-sidebar-item',
  sidePanelItemActive: '--md-sidebar-item-active',
  sidePanelSubItem: '--md-sidebar-sub-item',
  sidePanelSubItemActive: '--md-sidebar-sub-item-active',
  primaryButton: '--md-button-primary',
  secondaryButton: '--md-button-secondary',
  outlineButton: '--md-outline-button',
  destructiveButton: '--md-button-destructive',
  ghostButton: '--md-ghost-button',
  iconContainer: '--md-icon-container',
  listingItem: '--md-listing-item',
  listingItemIcon: '--md-listing-icon',
  listingItemMeta: '--md-listing-meta',
  listingSubContainer: '--md-listing-sub',
  optionCard: '--md-option-card',
  card: '--md-card',
  alert: '--md-alert',
  input: '--md-input',
  textarea: '--md-textarea',
  datePicker: '--md-date-picker',
  switch: '--md-switch',
  switchSubContainer: '--md-switch-sub',
  checkboxSubContainer: '--md-checkbox-sub',
}

const SIZE_STYLES: Record<string, {
  height: string
  paddingX: string
  fontSize: string
  dotSize: string
  thumbSize: string
  switchWidth: string
}> = {
  sm: {
    height: '2rem',
    paddingX: '0.75rem',
    fontSize: '0.8125rem',
    dotSize: '0.3125rem',
    thumbSize: '0.875rem',
    switchWidth: '2rem',
  },
  md: {
    height: '2.25rem',
    paddingX: '1rem',
    fontSize: '0.875rem',
    dotSize: '0.375rem',
    thumbSize: '1rem',
    switchWidth: '2.5rem',
  },
  lg: {
    height: '2.5rem',
    paddingX: '1.25rem',
    fontSize: '0.9375rem',
    dotSize: '0.4375rem',
    thumbSize: '1.125rem',
    switchWidth: '2.75rem',
  },
}

const HEX_COLOR = /^#(?:[\da-f]{3}|[\da-f]{4}|[\da-f]{6}|[\da-f]{8})$/i
const HEX_RGB_COLOR = /^#(?:[\da-f]{3}|[\da-f]{6})$/i
const CSS_COLOR_FUNCTION = /^(?:rgb|rgba|hsl|hsla|oklch|color-mix)\(/i
const CSS_GRADIENT = /^(?:linear|radial)-gradient\(/i

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function safeCssValue(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  if (!trimmed || trimmed.length > 240 || /[;{}]|url\s*\(|image-set\s*\(|@import/i.test(trimmed)) return undefined
  return trimmed
}

function safeColor(value: unknown): string | undefined {
  const trimmed = safeCssValue(value)
  if (!trimmed) return undefined
  if (trimmed === 'transparent' || HEX_COLOR.test(trimmed) || CSS_COLOR_FUNCTION.test(trimmed)) return trimmed
  return undefined
}

function safeBackground(value: unknown): string | undefined {
  const trimmed = safeCssValue(value)
  if (!trimmed) return undefined
  if (HEX_COLOR.test(trimmed) || CSS_COLOR_FUNCTION.test(trimmed) || CSS_GRADIENT.test(trimmed)) return trimmed
  return undefined
}

function safeFontFamily(value: unknown): string | undefined {
  return safeCssValue(value)
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

function readPath(source: unknown, path: readonly string[]): unknown {
  return path.reduce<unknown>((current, key) => {
    if (!isPlainObject(current)) return undefined
    return current[key]
  }, source)
}

function rememberProperty(
  root: HTMLElement,
  previous: Map<string, PreviousProperty>,
  property: string,
) {
  if (previous.has(property)) return
  previous.set(property, {
    value: root.style.getPropertyValue(property),
    priority: root.style.getPropertyPriority(property),
  })
}

function setProperty(
  root: HTMLElement,
  previous: Map<string, PreviousProperty>,
  property: string,
  value: string,
) {
  rememberProperty(root, previous, property)
  root.style.setProperty(property, value)
}

function removeProperty(root: HTMLElement, previous: Map<string, PreviousProperty>, property: string) {
  rememberProperty(root, previous, property)
  root.style.removeProperty(property)
}

function isManagedProperty(property: string): boolean {
  return MANAGED_PREFIXES.some((prefix) => property.startsWith(prefix)) ||
    ROOT_THEME_PROPERTIES.includes(property as (typeof ROOT_THEME_PROPERTIES)[number])
}

function clearManagedProperties(root: HTMLElement, previous: Map<string, PreviousProperty>) {
  const properties: string[] = []
  for (let index = 0; index < root.style.length; index += 1) {
    const property = root.style.item(index)
    if (isManagedProperty(property)) properties.push(property)
  }
  properties.forEach((property) => removeProperty(root, previous, property))
}

function normalizeBranding(
  brandingOrColors?: BrandingPublic | BrandingColors | null,
  font?: BrandingFont,
  background?: string,
): NormalizedBranding {
  if (isPlainObject(brandingOrColors) && 'metadata' in brandingOrColors) {
    const branding = brandingOrColors as unknown as BrandingPublic
    return {
      metadata: branding.metadata,
      title: branding.company_name || branding.logo_label,
      faviconUrl: branding.favicon_url,
    }
  }

  return {
    metadata: {
      colors: brandingOrColors as BrandingColors | undefined,
      font,
      background,
    },
    legacyFont: font,
    legacyBackground: background,
  }
}

function setComponentVars(
  root: HTMLElement,
  previous: Map<string, PreviousProperty>,
  metadata: BrandingMetadata,
) {
  for (const [componentName, prefix] of Object.entries(COMPONENT_PREFIXES)) {
    const component = readPath(metadata, ['components', componentName])
    if (!isPlainObject(component)) continue
    const config = component as ComponentConfig

    const background = safeColor(config.background)
    const hoverColor = safeColor(config.hoverColor)
    const borderColor = safeColor(config.borderColor)
    const borderThickness = safeCssValue(config.borderThickness)
    const borderRadius = safeCssValue(config.borderRadius)
    const textColor = safeColor(config.textColor)

    if (background) setProperty(root, previous, `${prefix}-bg`, background)
    if (hoverColor) setProperty(root, previous, `${prefix}-hover`, hoverColor)
    if (borderColor) setProperty(root, previous, `${prefix}-border`, borderColor)
    if (borderThickness) setProperty(root, previous, `${prefix}-border-width`, borderThickness)
    if (borderRadius) setProperty(root, previous, `${prefix}-radius`, borderRadius)
    if (textColor) setProperty(root, previous, `${prefix}-text`, textColor)

    const size = config.size?.trim()
    const sizeStyle = size ? SIZE_STYLES[size] : undefined
    if (!sizeStyle) continue
    setProperty(root, previous, `${prefix}-height`, sizeStyle.height)
    setProperty(root, previous, `${prefix}-padding-x`, sizeStyle.paddingX)
    setProperty(root, previous, `${prefix}-font-size`, sizeStyle.fontSize)
    setProperty(root, previous, `${prefix}-dot-size`, sizeStyle.dotSize)
    setProperty(root, previous, `${prefix}-thumb-size`, sizeStyle.thumbSize)
    if (componentName === 'switch') setProperty(root, previous, `${prefix}-width`, sizeStyle.switchWidth)
  }

  const switchConfig = readPath(metadata, ['components', 'switch'])
  if (isPlainObject(switchConfig)) {
    const uncheckedBackground = safeColor(switchConfig.uncheckedBackground)
    const thumbColor = safeColor(switchConfig.thumbColor)
    if (uncheckedBackground) setProperty(root, previous, '--md-switch-unchecked-bg', uncheckedBackground)
    if (thumbColor) setProperty(root, previous, '--md-switch-thumb-bg', thumbColor)
  }
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
 * Applies the resolved tenant/client branding to identity's semantic CSS API.
 * CSS owns the defaults; this helper only supplies loaded template overrides.
 */
export function applyBranding(
  brandingOrColors?: BrandingPublic | BrandingColors | null,
  font?: BrandingFont,
  background?: string,
): () => void {
  const root = document.documentElement
  const previous = new Map<string, PreviousProperty>()
  const previousAttribute = root.getAttribute('data-identity-theme')
  const previousTitle = document.title
  const iconLink = document.querySelector<HTMLLinkElement>("link[rel*='icon']")
  const previousFavicon = iconLink?.href
  const { metadata, title, faviconUrl, legacyFont, legacyBackground } = normalizeBranding(
    brandingOrColors,
    font,
    background,
  )

  clearManagedProperties(root, previous)

  if (!metadata) {
    root.removeAttribute('data-identity-theme')
    return () => {
      for (const [property, original] of previous) {
        if (original.value) root.style.setProperty(property, original.value, original.priority)
        else root.style.removeProperty(property)
      }
      if (previousAttribute === null) root.removeAttribute('data-identity-theme')
      else root.setAttribute('data-identity-theme', previousAttribute)
    }
  }

  const colors = metadata.colors
  if (colors) {
    for (const [name, properties] of Object.entries(COLOR_TOKEN_VARS) as [
      keyof BrandingColors,
      readonly string[],
    ][]) {
      const value = safeColor(colors[name])
      if (!value) continue
      properties.forEach((property) => setProperty(root, previous, property, value))

      if (HEX_RGB_COLOR.test(value)) {
        FOREGROUND_TOKEN_VARS[name]?.forEach((property) => {
          setProperty(root, previous, property, contrastColor(value))
        })
      }
    }
  }

  DIRECT_TOKEN_VARS.forEach(({ path, properties }) => {
    const value = path[0] === 'effects' ? safeCssValue(readPath(metadata, path)) : safeColor(readPath(metadata, path))
    if (!value) return
    properties.forEach((property) => setProperty(root, previous, property, value))
  })

  const pageBackground = getBrandingBackground(metadata)
    ?? safeBackground(legacyBackground)
    ?? safeBackground(colors?.authPageBackground)
    ?? safeBackground(colors?.appBackground)
  if (pageBackground) {
    setProperty(root, previous, '--auth-page-background', pageBackground)
    setProperty(root, previous, '--md-auth-page-bg', pageBackground)
  }

  const fontFamily = safeFontFamily(metadata.font?.family) ?? safeFontFamily(legacyFont?.family)
  if (fontFamily) {
    setProperty(root, previous, '--font-family', fontFamily)
    setProperty(root, previous, '--md-font-family', fontFamily)
  }

  setComponentVars(root, previous, metadata)
  root.setAttribute('data-identity-theme', 'active')

  if (title) document.title = title
  if (faviconUrl && iconLink) iconLink.href = faviconUrl

  return () => {
    clearManagedProperties(root, previous)
    for (const [property, original] of previous) {
      if (original.value) root.style.setProperty(property, original.value, original.priority)
      else root.style.removeProperty(property)
    }
    if (previousAttribute === null) root.removeAttribute('data-identity-theme')
    else root.setAttribute('data-identity-theme', previousAttribute)
    document.title = previousTitle
    if (iconLink && previousFavicon) iconLink.href = previousFavicon
  }
}
