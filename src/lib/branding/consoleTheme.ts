import type { BrandingPublic } from "@/services/api/tenants/types"
import {
  BADGE_GROUP_MEMBERS,
  STATUS_BADGE_TYPES,
  tokensFromMetadata,
} from "@/pages/branding/templates/themeTokens"

const THEME_ATTRIBUTE = "data-console-theme"

const GLOBAL_TOKEN_VARS: Record<string, string[]> = {
  "colors.primary": ["--primary", "--ring", "--sidebar-primary", "--chart-1"],
  "colors.secondary": ["--secondary"],
  "colors.accent": ["--accent", "--muted", "--sidebar-accent"],
  "colors.appBackground": ["--background"],
  "colors.sidePanelBackground": ["--sidebar"],
  "colors.cardBackground": ["--card", "--popover"],
  "colors.textPrimary": [
    "--foreground",
    "--card-foreground",
    "--popover-foreground",
    "--secondary-foreground",
    "--accent-foreground",
    "--sidebar-foreground",
    "--sidebar-accent-foreground",
  ],
  "colors.textMuted": ["--muted-foreground"],
  "colors.border": ["--border", "--sidebar-border"],
  "font.family": ["--md-font-family"],
}

const DIRECT_TOKEN_VARS: Record<string, string> = {
  "colors.topPanelBackground": "--md-top-panel-bg",
  "colors.topPanelBorder": "--md-top-panel-border",
  "colors.topPanelText": "--md-top-panel-text",
  "colors.authPageBackground": "--md-auth-page-bg",
  "colors.authFormPanelBackground": "--md-auth-form-bg",
  "colors.authFormPanelBorder": "--md-auth-form-border",
  "colors.authFormPanelText": "--md-auth-form-text",
  "colors.sidePanelBackground": "--md-side-panel-bg",
  "colors.sidePanelBorder": "--md-side-panel-border",
  "colors.sidePanelSectionText": "--md-side-panel-section-text",
  "colors.sidePanelItemIcon": "--md-side-panel-item-icon",
  "colors.sidePanelItemIconHover": "--md-side-panel-item-icon-hover",
  "colors.sidePanelItemActiveIcon": "--md-side-panel-item-active-icon",
  "colors.sidePanelItemHoverText": "--md-side-panel-item-hover-text",
  "colors.sidePanelChevron": "--md-side-panel-chevron",
  "effects.authFormPanelShadow": "--md-auth-form-shadow",
}

const COMPONENT_PREFIXES: Record<string, string> = {
  topPanelControl: "--md-top-control",
  topPanelDropdownTrigger: "--md-top-dropdown",
  topPanelProfileTrigger: "--md-top-profile",
  topPanelCreateButton: "--md-top-create",
  sidePanelSectionLabel: "--md-sidebar-section",
  sidePanelItem: "--md-sidebar-item",
  sidePanelItemActive: "--md-sidebar-item-active",
  sidePanelSubItem: "--md-sidebar-sub-item",
  sidePanelSubItemActive: "--md-sidebar-sub-item-active",
  primaryButton: "--md-button-primary",
  secondaryButton: "--md-button-secondary",
  outlineButton: "--md-outline-button",
  destructiveButton: "--md-button-destructive",
  ghostButton: "--md-ghost-button",
  tableContainer: "--md-table-container",
  tableHeader: "--md-table-header",
  tableRow: "--md-table-row",
  tableCell: "--md-table-cell",
  iconContainer: "--md-icon-container",
  listingItem: "--md-listing-item",
  listingItemIcon: "--md-listing-icon",
  listingItemMeta: "--md-listing-meta",
  listingSubContainer: "--md-listing-sub",
  optionCard: "--md-option-card",
  card: "--md-card",
  alert: "--md-alert",
  input: "--md-input",
  textarea: "--md-textarea",
  datePicker: "--md-date-picker",
  switch: "--md-switch",
  switchSubContainer: "--md-switch-sub",
  checkboxSubContainer: "--md-checkbox-sub",
}

const COMPONENT_FIELD_SUFFIXES: Record<string, string> = {
  background: "bg",
  hoverColor: "hover",
  borderColor: "border",
  borderThickness: "border-width",
  borderRadius: "radius",
  textColor: "text",
  uncheckedBackground: "unchecked-bg",
  thumbColor: "thumb-bg",
}

const COMPONENT_SIZES: Record<string, { height: string; paddingX: string; fontSize: string }> = {
  sm: { height: "2.25rem", paddingX: "0.75rem", fontSize: "0.75rem" },
  md: { height: "2.5rem", paddingX: "0.875rem", fontSize: "0.875rem" },
  lg: { height: "2.75rem", paddingX: "1rem", fontSize: "0.9375rem" },
}

const SWITCH_SIZES: Record<string, { height: string; width: string; thumb: string }> = {
  sm: { height: "1rem", width: "1.75rem", thumb: "0.8125rem" },
  md: { height: "1.15rem", width: "2rem", thumb: "1rem" },
  lg: { height: "1.35rem", width: "2.375rem", thumb: "1.125rem" },
}

const STATUS_SIZES: Record<string, { paddingX: string; fontSize: string; dot: string }> = {
  sm: { paddingX: "0.5rem", fontSize: "0.75rem", dot: "0.375rem" },
  md: { paddingX: "0.625rem", fontSize: "0.8125rem", dot: "0.4375rem" },
  lg: { paddingX: "0.75rem", fontSize: "0.875rem", dot: "0.5rem" },
}

const MANAGED_PREFIXES = [
  "--md-",
  "--primary",
  "--ring",
  "--sidebar-primary",
  "--sidebar",
  "--chart-1",
  "--secondary",
  "--accent",
  "--muted",
  "--sidebar-accent",
  "--background",
  "--card",
  "--popover",
  "--foreground",
  "--card-foreground",
  "--popover-foreground",
  "--secondary-foreground",
  "--accent-foreground",
  "--sidebar-foreground",
  "--sidebar-accent-foreground",
  "--muted-foreground",
  "--border",
  "--sidebar-border",
  "--input",
  "--destructive",
]

function normalizeMetadata(metadata: unknown): Record<string, unknown> | null {
  if (!metadata) return null
  if (typeof metadata === "string") {
    try {
      const parsed = JSON.parse(metadata)
      return parsed && typeof parsed === "object" && !Array.isArray(parsed)
        ? parsed as Record<string, unknown>
        : null
    } catch {
      return null
    }
  }
  return typeof metadata === "object" && !Array.isArray(metadata)
    ? metadata as Record<string, unknown>
    : null
}

function set(vars: Record<string, string>, name: string, value: string | undefined) {
  if (typeof value === "string" && value.trim()) {
    vars[name] = value.trim()
  }
}

function setComponentVars(vars: Record<string, string>, tokens: Record<string, string>, component: string) {
  const prefix = COMPONENT_PREFIXES[component]
  if (!prefix) return

  for (const [field, suffix] of Object.entries(COMPONENT_FIELD_SUFFIXES)) {
    set(vars, `${prefix}-${suffix}`, tokens[`components.${component}.${field}`])
  }

  const size = tokens[`components.${component}.size`]
  const componentSize = COMPONENT_SIZES[size] ?? COMPONENT_SIZES.md
  set(vars, `${prefix}-height`, componentSize.height)
  set(vars, `${prefix}-padding-x`, componentSize.paddingX)
  set(vars, `${prefix}-font-size`, componentSize.fontSize)

  if (component === "switch") {
    const switchSize = SWITCH_SIZES[size] ?? SWITCH_SIZES.md
    set(vars, "--md-switch-height", switchSize.height)
    set(vars, "--md-switch-width", switchSize.width)
    set(vars, "--md-switch-thumb-size", switchSize.thumb)
  }
}

/** Reverse lookup: each status keyword → its tone group. Badges in the same
 *  group share one set of tokens (components.badges.<group>.*), which the
 *  runtime maps back onto the per-status CSS variables used by the badges. */
const STATUS_GROUP_OF: Record<string, string> = Object.fromEntries(
  Object.entries(BADGE_GROUP_MEMBERS).flatMap(([group, statuses]) =>
    statuses.map((status) => [status, group]),
  ),
)

function setStatusVars(vars: Record<string, string>, tokens: Record<string, string>) {
  for (const status of STATUS_BADGE_TYPES) {
    const group = STATUS_GROUP_OF[status] ?? status
    const prefix = `--md-status-${status}`
    set(vars, `${prefix}-bg`, tokens[`components.badges.${group}.background`])
    set(vars, `${prefix}-hover`, tokens[`components.badges.${group}.hoverColor`])
    set(vars, `${prefix}-border`, tokens[`components.badges.${group}.borderColor`])
    set(vars, `${prefix}-border-width`, tokens[`components.badges.${group}.borderThickness`])
    set(vars, `${prefix}-radius`, tokens[`components.badges.${group}.borderRadius`])
    set(vars, `${prefix}-text`, tokens[`components.badges.${group}.textColor`])
    set(vars, `${prefix}-dot`, tokens[`components.badges.${group}.dotColor`])

    const size = tokens[`components.badges.${group}.size`]
    const statusSize = STATUS_SIZES[size] ?? STATUS_SIZES.sm
    set(vars, `${prefix}-padding-x`, statusSize.paddingX)
    set(vars, `${prefix}-font-size`, statusSize.fontSize)
    set(vars, `${prefix}-dot-size`, statusSize.dot)
  }
}

export function consoleThemeVariablesFromBranding(branding?: Pick<BrandingPublic, "metadata"> | null) {
  const metadata = normalizeMetadata(branding?.metadata)
  if (!branding || !metadata) return null

  const tokens = tokensFromMetadata(metadata)
  const vars: Record<string, string> = {}

  for (const [token, names] of Object.entries(GLOBAL_TOKEN_VARS)) {
    for (const name of names) set(vars, name, tokens[token])
  }
  for (const [token, name] of Object.entries(DIRECT_TOKEN_VARS)) {
    set(vars, name, tokens[token])
  }

  set(vars, "--input", tokens["components.input.borderColor"] || tokens["colors.border"])
  set(vars, "--destructive", tokens["components.destructiveButton.background"])

  for (const component of Object.keys(COMPONENT_PREFIXES)) {
    setComponentVars(vars, tokens, component)
  }
  setStatusVars(vars, tokens)

  return vars
}

export function clearConsoleTheme(root: HTMLElement = document.documentElement) {
  root.removeAttribute(THEME_ATTRIBUTE)
  for (let i = root.style.length - 1; i >= 0; i -= 1) {
    const name = root.style.item(i)
    if (MANAGED_PREFIXES.some((prefix) => name.startsWith(prefix))) {
      root.style.removeProperty(name)
    }
  }
  document.body.style.fontFamily = ""
  document.title = "Maintainerd IAM"
}

export function applyConsoleTheme(branding?: BrandingPublic | null, root: HTMLElement = document.documentElement) {
  const vars = consoleThemeVariablesFromBranding(branding)
  if (!vars) {
    clearConsoleTheme(root)
    return
  }

  clearConsoleTheme(root)
  for (const [name, value] of Object.entries(vars)) {
    root.style.setProperty(name, value)
  }
  root.setAttribute(THEME_ATTRIBUTE, "active")

  if (vars["--md-font-family"]) {
    document.body.style.fontFamily = vars["--md-font-family"]
  }
  if (branding?.company_name) {
    document.title = branding.company_name
  }
}
