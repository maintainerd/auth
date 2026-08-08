/**
 * Branding theme tokens - a fixed, system-defined set of theme variables.
 * Keys are not user-editable; only their values are. The shape mirrors backend
 * branding metadata and stays additive so older themes fall back to defaults.
 */

export type ThemeTokenKind = "color" | "text" | "select" | "heading"

export type ThemeToken = {
  section: string
  path: string[]
  label: string
  kind: ThemeTokenKind
  options?: string[]
  group?: string
}

export type ThemeSection = {
  id: string
  title: string
  description: string
  tokens: ThemeToken[]
}

const sizeOptions = ["sm", "md", "lg"]

const componentFields = [
  ["background", "Background", "color"],
  ["hoverColor", "Hover color", "color"],
  ["borderColor", "Border color", "color"],
  ["borderThickness", "Border thickness", "text"],
  ["borderRadius", "Border radius", "text"],
  ["textColor", "Text color", "color"],
  ["size", "Size", "select"],
] as const

const statusFields = [
  ["background", "Background", "color"],
  ["hoverColor", "Hover color", "color"],
  ["borderColor", "Border color", "color"],
  ["borderThickness", "Border thickness", "text"],
  ["borderRadius", "Border radius", "text"],
  ["textColor", "Text color", "color"],
  ["dotColor", "Dot color", "color"],
  ["size", "Size", "select"],
] as const

const alertFields = [
  ["background", "Background", "color"],
  ["borderColor", "Border color", "color"],
  ["borderThickness", "Border thickness", "text"],
  ["borderRadius", "Border radius", "text"],
  ["textColor", "Text color", "color"],
] as const

const switchFields = [
  ["background", "Background", "color"],
  ["hoverColor", "Hover color", "color"],
  ["borderColor", "Border color", "color"],
  ["borderThickness", "Border thickness", "text"],
  ["borderRadius", "Border radius", "text"],
  ["textColor", "Text color", "color"],
  ["size", "Size", "select"],
  ["uncheckedBackground", "Unchecked background", "color"],
  ["thumbColor", "Thumb background", "color"],
] as const

const componentTokens = (
  section: string,
  component: string,
  label: string,
): ThemeToken[] =>
  componentFields.map(([key, suffix, kind]) => ({
    section,
    path: ["components", component, key],
    label: `${label} ${suffix.toLowerCase()}`,
    kind,
    options: kind === "select" ? sizeOptions : undefined,
  }))

const badgeTokens = (group: string): ThemeToken[] =>
  statusFields.map(([key, suffix, kind]) => ({
    section: "badges",
    path: ["components", "badges", group, key],
    label: `${humanize(group)} ${suffix.toLowerCase()}`,
    kind,
    options: kind === "select" ? sizeOptions : undefined,
    group,
  }))

/** Component tokens for one button type, tagged with its sub-group so the
 *  merged Buttons section can render each button's preview and rows together. */
const buttonTokens = (component: string, label: string): ThemeToken[] =>
  componentTokens("buttons", component, label).map((t) => ({ ...t, group: component }))

const topPanelComponentTokens = (component: string, label: string): ThemeToken[] =>
  componentTokens("top-panel", component, label).map((t) => ({ ...t, group: component }))

const sidePanelComponentTokens = (component: string, label: string, group: string): ThemeToken[] =>
  componentTokens("side-panel", component, label).map((t) => ({ ...t, group }))

/** Square or round icon wells used beside resources in listings, detail
 *  headers, empty states, and setup blocks. */
const iconContainerTokens = (): ThemeToken[] =>
  componentTokens("icon-containers", "iconContainer", "Icon container")

/** Component tokens for the base card, tagged with its sub-group so the merged
 *  Card section can render the card and listing-card previews and rows together. */
const cardTokens = (component: string, label: string): ThemeToken[] =>
  componentTokens("card", component, label).map((t) => ({ ...t, group: "card" }))

/** Component tokens for a listing surface, tagged with the listing-card
 *  sub-group inside the merged Card section. */
const listingCardTokens = (component: string, label: string): ThemeToken[] =>
  componentTokens("card", component, label).map((t) => ({ ...t, group: "listing-card" }))

/** Component tokens for a sub-container inside a listing card (metadata rows,
 *  nested surfaces, key/value pairs), tagged with the sub-container sub-group.
 *  Every sub-container type shares this one config. */
const subContainerTokens = (component: string, label: string): ThemeToken[] =>
  componentTokens("card", component, label).map((t) => ({ ...t, group: "sub-container" }))

/** Component tokens for a clickable option row (quick actions, shortcuts,
 *  navigation rows), tagged with the option-card sub-group. */
const optionCardTokens = (component: string, label: string): ThemeToken[] =>
  componentTokens("card", component, label).map((t) => ({ ...t, group: "option-card" }))

/** Alert banner tokens tagged with the alert sub-group inside the merged Card
 *  section. */
const alertTokens = (): ThemeToken[] =>
  alertFields.map(([key, suffix, kind]) => ({
    section: "card",
    path: ["components", "alert", key],
    label: `Alert ${suffix.toLowerCase()}`,
    kind,
    group: "alert",
  }))

/** Input/select component tokens tagged with the inputs sub-group inside the
 *  merged Inputs and selects section. */
const inputTokens = (component: string, label: string): ThemeToken[] =>
  componentTokens("inputs", component, label).map((t) => ({ ...t, group: "inputs" }))

/** Textarea tokens tagged with the textarea sub-group inside the merged Inputs
 *  and selects section, so textareas keep their own radius instead of sharing
 *  the single-line input radius. */
const textareaTokens = (): ThemeToken[] =>
  componentFields.map(([key, suffix, kind]) => ({
    section: "inputs",
    path: ["components", "textarea", key],
    label: `Textarea ${suffix.toLowerCase()}`,
    kind,
    options: kind === "select" ? sizeOptions : undefined,
    group: "textarea",
  }))

/** Switch tokens tagged with the switch sub-group inside the merged Inputs and
 *  selects section. */
const switchTokens = (): ThemeToken[] =>
  switchFields.map(([key, suffix, kind]) => ({
    section: "inputs",
    path: ["components", "switch", key],
    label: `Switch ${suffix.toLowerCase()}`,
    kind,
    options: kind === "select" ? sizeOptions : undefined,
    group: "switch",
  }))

/** Tokens for the bordered box that wraps a switch field (allow registration,
 *  token federation, JIT provisioning), tagged with the switch-sub-container
 *  sub-group inside the merged Inputs and selects section. */
const switchSubContainerTokens = (): ThemeToken[] =>
  componentFields.map(([key, suffix, kind]) => ({
    section: "inputs",
    path: ["components", "switchSubContainer", key],
    label: `Switch box ${suffix.toLowerCase()}`,
    kind,
    options: kind === "select" ? sizeOptions : undefined,
    group: "switch-sub-container",
  }))

/** Tokens for the bordered box that wraps a checkbox option list (roles,
 *  permissions pickers), tagged with the checkbox-sub-container sub-group
 *  inside the merged Inputs and selects section. */
const checkboxSubContainerTokens = (): ThemeToken[] =>
  componentFields.map(([key, suffix, kind]) => ({
    section: "inputs",
    path: ["components", "checkboxSubContainer", key],
    label: `Checkbox list ${suffix.toLowerCase()}`,
    kind,
    options: kind === "select" ? sizeOptions : undefined,
    group: "checkbox-sub-container",
  }))

function humanize(value: string) {
  return value
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase())
}

export const STATUS_BADGE_TYPES = [
  "active",
  "enabled",
  "verified",
  "accepted",
  "allow",
  "pending",
  "draft",
  "configuring",
  "maintenance",
  "inactive",
  "disabled",
  "archived",
  "expired",
  "deny",
  "suspended",
  "blocked",
  "revoked",
  "quarantined",
  "deprecated",
] as const

/** Semantic sub-groups for the Badges section, mirroring how buttons are
 *  grouped. Each group renders its own preview and token rows. */
export const BADGE_GROUP_MEMBERS: Record<string, readonly (typeof STATUS_BADGE_TYPES)[number][]> = {
  positive: ["active", "enabled", "verified", "accepted", "allow"],
  "in-progress": ["pending", "draft", "configuring", "maintenance"],
  neutral: ["inactive", "disabled", "archived", "expired"],
  negative: ["deny", "suspended", "blocked", "revoked", "quarantined", "deprecated"],
}

export const THEME_SECTIONS: ThemeSection[] = [
  {
    id: "app-foundation",
    title: "App foundation",
    description: "Global palette, text, borders, and typography used as the fallback when component tokens are not set.",
    tokens: [
      { section: "app-foundation", path: ["colors", "primary"], label: "Primary", kind:  "color", group: "colors" },
      { section: "app-foundation", path: ["colors", "secondary"], label: "Secondary", kind:  "color", group: "colors" },
      { section: "app-foundation", path: ["colors", "accent"], label: "Accent", kind:  "color", group: "colors" },
      { section: "app-foundation", path: ["colors", "appBackground"], label: "App background", kind:  "color", group: "colors" },
      { section: "app-foundation", path: ["colors", "textPrimary"], label: "Primary text", kind:  "color", group: "colors" },
      { section: "app-foundation", path: ["colors", "textMuted"], label: "Muted text", kind:  "color", group: "colors" },
      { section: "app-foundation", path: ["colors", "border"], label: "Border", kind:  "color", group: "colors" },
      { section: "app-foundation", path: ["font", "family"], label: "Font family", kind: "text" },
    ],
  },
  {
    id: "login-template",
    title: "Login template",
    description: "Hosted-auth layout surfaces that change between centered, split, full-page, modal, guided, and security templates.",
    tokens: [
      { section: "login-template", path: ["colors", "authPageBackground"], label: "Page background", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authFormPanelBackground"], label: "Form panel background", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authFormPanelBorder"], label: "Form panel border", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authFormPanelText"], label: "Form panel text", kind:  "color", group: "colors" },
      { section: "login-template", path: ["effects", "authFormPanelShadow"], label: "Form panel shadow", kind: "text" },
      { section: "login-template", path: ["colors", "authVisualPanelBackground"], label: "Visual panel background", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authVisualPanelText"], label: "Visual panel text", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authVisualPanelOverlay"], label: "Visual panel overlay", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authDecorativeLight"], label: "Decorative light shape", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authDecorativeDark"], label: "Decorative dark shape", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authProgressPanelBackground"], label: "Progress panel background", kind:  "color", group: "colors" },
      { section: "login-template", path: ["colors", "authSecurityPanelBackground"], label: "Security panel background", kind:  "color", group: "colors" },
    ],
  },
  {
    id: "top-panel",
    title: "Top panel",
    description: "Top navigation surface plus hamburger, tenant dropdown, and profile controls.",
    tokens: [
      { section: "top-panel", path: ["colors", "topPanelBackground"], label: "Top panel background", kind: "color", group: "colors" },
      { section: "top-panel", path: ["colors", "topPanelBorder"], label: "Top panel border", kind: "color", group: "colors" },
      { section: "top-panel", path: ["colors", "topPanelText"], label: "Top panel text", kind: "color", group: "colors" },
      ...topPanelComponentTokens("topPanelControl", "Icon control"),
      ...topPanelComponentTokens("topPanelDropdownTrigger", "Tenant dropdown trigger"),
      ...topPanelComponentTokens("topPanelSearchTrigger", "Search trigger"),
      ...topPanelComponentTokens("topPanelProfileTrigger", "Profile trigger"),
      ...topPanelComponentTokens("topPanelCreateButton", "Create button"),
    ],
  },
  {
    id: "side-panel",
    title: "Side panel",
    description: "Side navigation surface, labels, parent rows, active rows, and sub-items.",
    tokens: [
      { section: "side-panel", path: ["colors", "sidePanelBackground"], label: "Side panel background", kind:  "color", group: "colors" },
      { section: "side-panel", path: ["colors", "sidePanelBorder"], label: "Side panel border", kind:  "color", group: "colors" },
      { section: "side-panel", path: ["colors", "sidePanelSectionText"], label: "Section label text", kind:  "color", group: "colors" },
      { section: "side-panel", path: ["colors", "sidePanelItemIcon"], label: "Item icon", kind:  "color", group: "colors" },
      { section: "side-panel", path: ["colors", "sidePanelItemIconHover"], label: "Item icon hover", kind:  "color", group: "colors" },
      { section: "side-panel", path: ["colors", "sidePanelItemActiveIcon"], label: "Active item icon", kind:  "color", group: "colors" },
      { section: "side-panel", path: ["colors", "sidePanelItemHoverText"], label: "Hover item text", kind:  "color", group: "colors" },
      { section: "side-panel", path: ["colors", "sidePanelChevron"], label: "Chevron", kind:  "color", group: "colors" },
      ...sidePanelComponentTokens("sidePanelSectionLabel", "Section label", "sidePanelSectionLabel"),
      ...sidePanelComponentTokens("sidePanelItem", "Parent item", "parent-item"),
      ...sidePanelComponentTokens("sidePanelItemActive", "Active parent item", "parent-item"),
      ...sidePanelComponentTokens("sidePanelSubItem", "Sub-item", "sub-item"),
      ...sidePanelComponentTokens("sidePanelSubItemActive", "Active sub-item", "sub-item"),
    ],
  },
  {
    id: "buttons",
    title: "Buttons",
    description: "Action buttons across the console — primary, secondary, outline, destructive, and ghost.",
    tokens: [
      ...buttonTokens("primaryButton", "Primary button"),
      ...buttonTokens("secondaryButton", "Secondary button"),
      ...buttonTokens("outlineButton", "Outline button"),
      ...buttonTokens("destructiveButton", "Destructive button"),
      ...buttonTokens("ghostButton", "Ghost button"),
    ],
  },
  {
    id: "card",
    title: "Card",
    description: "Card surface defaults used by form panels, details, and repeated content, plus the listing cards, sub-containers (metadata rows), option cards, and alert banners used across the console.",
    tokens: [
      { section: "card", path: ["colors", "cardBackground"], label: "Card background", kind: "color", group: "card" },
      ...cardTokens("card", "Card"),
      ...listingCardTokens("listingItem", "Listing item"),
      ...subContainerTokens("listingSubContainer", "Sub-container"),
      ...optionCardTokens("optionCard", "Option card"),
      ...alertTokens(),
    ],
  },
  {
    id: "icon-containers",
    title: "Icon containers",
    description: "Small icon wells used beside resources in listing rows, role tabs, detail headers, empty states, and setup panels.",
    tokens: iconContainerTokens(),
  },
  {
    id: "table",
    title: "Table",
    description: "Table containers, headers, rows, hover states, and cells.",
    tokens: [
      ...componentTokens("table", "tableContainer", "Table container"),
      ...componentTokens("table", "tableHeader", "Table header"),
      ...componentTokens("table", "tableRow", "Table row"),
      ...componentTokens("table", "tableCell", "Table cell"),
    ],
  },
  {
    id: "inputs",
    title: "Inputs and selects",
    description: "Text inputs, textareas, ordinary dropdown/select triggers, date pickers, toggle switches, and the switch boxes that wrap them outside the top panel.",
    tokens: [
      ...inputTokens("input", "Input and select"),
      ...textareaTokens(),
      ...inputTokens("datePicker", "Date picker"),
      ...switchTokens(),
      ...switchSubContainerTokens(),
      ...checkboxSubContainerTokens(),
    ],
  },
  {
    id: "badges",
    title: "Badges",
    description: "Badge styling shared by every badge in each tone group — positive, in progress, neutral, and negative. Status, system, and default badges all follow these styles.",
    tokens: Object.keys(BADGE_GROUP_MEMBERS).flatMap((group) => badgeTokens(group)),
  },
]

export const THEME_TOKENS: ThemeToken[] = THEME_SECTIONS.flatMap((section) => section.tokens)

export const DEFAULT_TOKENS: Record<string, string> = {
  "colors.primary": "#2563eb",
  "colors.secondary": "#eef1f5",
  "colors.accent": "#e9edf3",
  "colors.appBackground": "#f6f7f9",
  "colors.topPanelBackground": "#0f172a",
  "colors.topPanelBorder": "#1e293b",
  "colors.topPanelText": "#ffffff",
  "colors.authPageBackground": "#f6f7f9",
  "colors.authFormPanelBackground": "#ffffff",
  "colors.authFormPanelBorder": "#dce1e8",
  "colors.authFormPanelText": "#1f252e",
  "effects.authFormPanelShadow": "0 1px 2px rgba(15,23,42,0.04), 0 16px 40px -20px rgba(15,23,42,0.25)",
  "colors.authVisualPanelBackground": "#2563eb",
  "colors.authVisualPanelText": "#ffffff",
  "colors.authVisualPanelOverlay": "#0f172a",
  "colors.authDecorativeLight": "#ffffff",
  "colors.authDecorativeDark": "#000000",
  "colors.authProgressPanelBackground": "#ffffff",
  "colors.authSecurityPanelBackground": "#ffffff",
  "colors.sidePanelBackground": "#f6f7f9",
  "colors.sidePanelBorder": "#cfd6e0",
  "colors.sidePanelSectionText": "#667085",
  "colors.sidePanelItemIcon": "#647084",
  "colors.sidePanelItemIconHover": "#2d3748",
  "colors.sidePanelItemActiveIcon": "#2563eb",
  "colors.sidePanelItemHoverText": "#111827",
  "colors.sidePanelChevron": "#7b8797",
  "colors.cardBackground": "#ffffff",
  "colors.textPrimary": "#1f252e",
  "colors.textMuted": "#647084",
  "colors.border": "#dce1e8",
  "font.family": "Inter, system-ui, sans-serif",

  "components.topPanelControl.background": "rgba(255,255,255,0.05)",
  "components.topPanelControl.hoverColor": "rgba(255,255,255,0.10)",
  "components.topPanelControl.borderColor": "transparent",
  "components.topPanelControl.borderThickness": "0px",
  "components.topPanelControl.borderRadius": "3px",
  "components.topPanelControl.textColor": "#cbd5e1",
  "components.topPanelControl.size": "md",

  "components.topPanelDropdownTrigger.background": "rgba(255,255,255,0.05)",
  "components.topPanelDropdownTrigger.hoverColor": "rgba(255,255,255,0.10)",
  "components.topPanelDropdownTrigger.borderColor": "#334155",
  "components.topPanelDropdownTrigger.borderThickness": "1px",
  "components.topPanelDropdownTrigger.borderRadius": "3px",
  "components.topPanelDropdownTrigger.textColor": "#cbd5e1",
  "components.topPanelDropdownTrigger.size": "md",

  "components.topPanelSearchTrigger.background": "rgba(255,255,255,0.05)",
  "components.topPanelSearchTrigger.hoverColor": "rgba(255,255,255,0.10)",
  "components.topPanelSearchTrigger.borderColor": "#334155",
  "components.topPanelSearchTrigger.borderThickness": "1px",
  "components.topPanelSearchTrigger.borderRadius": "3px",
  "components.topPanelSearchTrigger.textColor": "#cbd5e1",
  "components.topPanelSearchTrigger.size": "md",

  "components.topPanelProfileTrigger.background": "rgba(255,255,255,0.05)",
  "components.topPanelProfileTrigger.hoverColor": "rgba(255,255,255,0.10)",
  "components.topPanelProfileTrigger.borderColor": "transparent",
  "components.topPanelProfileTrigger.borderThickness": "0px",
  "components.topPanelProfileTrigger.borderRadius": "3px",
  "components.topPanelProfileTrigger.textColor": "#ffffff",
  "components.topPanelProfileTrigger.size": "md",

  "components.topPanelCreateButton.background": "#2563eb",
  "components.topPanelCreateButton.hoverColor": "#1d4ed8",
  "components.topPanelCreateButton.borderColor": "transparent",
  "components.topPanelCreateButton.borderThickness": "0px",
  "components.topPanelCreateButton.borderRadius": "3px",
  "components.topPanelCreateButton.textColor": "#ffffff",
  "components.topPanelCreateButton.size": "md",

  "components.sidePanelItem.background": "transparent",
  "components.sidePanelItem.hoverColor": "#edf1f6",
  "components.sidePanelItem.borderColor": "transparent",
  "components.sidePanelItem.borderThickness": "0px",
  "components.sidePanelItem.borderRadius": "3px",
  "components.sidePanelItem.textColor": "#475569",
  "components.sidePanelItem.size": "md",

  "components.sidePanelSectionLabel.background": "transparent",
  "components.sidePanelSectionLabel.hoverColor": "transparent",
  "components.sidePanelSectionLabel.borderColor": "transparent",
  "components.sidePanelSectionLabel.borderThickness": "0px",
  "components.sidePanelSectionLabel.borderRadius": "3px",
  "components.sidePanelSectionLabel.textColor": "#667085",
  "components.sidePanelSectionLabel.size": "sm",

  "components.sidePanelItemActive.background": "#e4eaf2",
  "components.sidePanelItemActive.hoverColor": "#e4eaf2",
  "components.sidePanelItemActive.borderColor": "transparent",
  "components.sidePanelItemActive.borderThickness": "0px",
  "components.sidePanelItemActive.borderRadius": "3px",
  "components.sidePanelItemActive.textColor": "#111827",
  "components.sidePanelItemActive.size": "md",

  "components.sidePanelSubItem.background": "transparent",
  "components.sidePanelSubItem.hoverColor": "#edf1f6",
  "components.sidePanelSubItem.borderColor": "transparent",
  "components.sidePanelSubItem.borderThickness": "0px",
  "components.sidePanelSubItem.borderRadius": "3px",
  "components.sidePanelSubItem.textColor": "#5b677a",
  "components.sidePanelSubItem.size": "md",

  "components.sidePanelSubItemActive.background": "#e4eaf2",
  "components.sidePanelSubItemActive.hoverColor": "#e4eaf2",
  "components.sidePanelSubItemActive.borderColor": "transparent",
  "components.sidePanelSubItemActive.borderThickness": "0px",
  "components.sidePanelSubItemActive.borderRadius": "3px",
  "components.sidePanelSubItemActive.textColor": "#111827",
  "components.sidePanelSubItemActive.size": "md",

  "components.primaryButton.background": "#2563eb",
  "components.primaryButton.hoverColor": "#1d4ed8",
  "components.primaryButton.borderColor": "transparent",
  "components.primaryButton.borderThickness": "0px",
  "components.primaryButton.borderRadius": "3px",
  "components.primaryButton.textColor": "#ffffff",
  "components.primaryButton.size": "md",

  "components.secondaryButton.background": "#eef1f5",
  "components.secondaryButton.hoverColor": "#e9edf3",
  "components.secondaryButton.borderColor": "transparent",
  "components.secondaryButton.borderThickness": "0px",
  "components.secondaryButton.borderRadius": "3px",
  "components.secondaryButton.textColor": "#2d3748",
  "components.secondaryButton.size": "md",

  "components.outlineButton.background": "#ffffff",
  "components.outlineButton.hoverColor": "#e9edf3",
  "components.outlineButton.borderColor": "#d1d8e2",
  "components.outlineButton.borderThickness": "1px",
  "components.outlineButton.borderRadius": "3px",
  "components.outlineButton.textColor": "#1f252e",
  "components.outlineButton.size": "md",

  "components.destructiveButton.background": "#dc2626",
  "components.destructiveButton.hoverColor": "#b91c1c",
  "components.destructiveButton.borderColor": "transparent",
  "components.destructiveButton.borderThickness": "0px",
  "components.destructiveButton.borderRadius": "3px",
  "components.destructiveButton.textColor": "#ffffff",
  "components.destructiveButton.size": "md",

  "components.ghostButton.background": "transparent",
  "components.ghostButton.hoverColor": "#e9edf3",
  "components.ghostButton.borderColor": "transparent",
  "components.ghostButton.borderThickness": "0px",
  "components.ghostButton.borderRadius": "3px",
  "components.ghostButton.textColor": "#2d3748",
  "components.ghostButton.size": "sm",

  "components.tableContainer.background": "#ffffff",
  "components.tableContainer.hoverColor": "#ffffff",
  "components.tableContainer.borderColor": "#dce1e8",
  "components.tableContainer.borderThickness": "1px",
  "components.tableContainer.borderRadius": "3px",
  "components.tableContainer.textColor": "#1f252e",
  "components.tableContainer.size": "md",

  "components.tableHeader.background": "#f6f7f9",
  "components.tableHeader.hoverColor": "#f6f7f9",
  "components.tableHeader.borderColor": "#dce1e8",
  "components.tableHeader.borderThickness": "1px",
  "components.tableHeader.borderRadius": "0px",
  "components.tableHeader.textColor": "#1f252e",
  "components.tableHeader.size": "md",

  "components.tableRow.background": "#ffffff",
  "components.tableRow.hoverColor": "#f6f7f9",
  "components.tableRow.borderColor": "#dce1e8",
  "components.tableRow.borderThickness": "1px",
  "components.tableRow.borderRadius": "0px",
  "components.tableRow.textColor": "#1f252e",
  "components.tableRow.size": "md",

  "components.tableCell.background": "transparent",
  "components.tableCell.hoverColor": "transparent",
  "components.tableCell.borderColor": "transparent",
  "components.tableCell.borderThickness": "0px",
  "components.tableCell.borderRadius": "0px",
  "components.tableCell.textColor": "#1f252e",
  "components.tableCell.size": "md",

  "components.iconContainer.background": "#f0f2f5",
  "components.iconContainer.hoverColor": "#e9edf3",
  "components.iconContainer.borderColor": "transparent",
  "components.iconContainer.borderThickness": "0px",
  "components.iconContainer.borderRadius": "3px",
  "components.iconContainer.textColor": "#647084",
  "components.iconContainer.size": "md",

  "components.listingItem.background": "#ffffff",
  "components.listingItem.hoverColor": "#f6f7f9",
  "components.listingItem.borderColor": "#dce1e8",
  "components.listingItem.borderThickness": "1px",
  "components.listingItem.borderRadius": "3px",
  "components.listingItem.textColor": "#1f252e",
  "components.listingItem.size": "md",

  "components.listingItemIcon.background": "#f0f2f5",
  "components.listingItemIcon.hoverColor": "#e9edf3",
  "components.listingItemIcon.borderColor": "transparent",
  "components.listingItemIcon.borderThickness": "0px",
  "components.listingItemIcon.borderRadius": "3px",
  "components.listingItemIcon.textColor": "#647084",
  "components.listingItemIcon.size": "md",

  "components.listingItemMeta.background": "transparent",
  "components.listingItemMeta.hoverColor": "transparent",
  "components.listingItemMeta.borderColor": "transparent",
  "components.listingItemMeta.borderThickness": "0px",
  "components.listingItemMeta.borderRadius": "0px",
  "components.listingItemMeta.textColor": "#647084",
  "components.listingItemMeta.size": "sm",

  "components.listingSubContainer.background": "#f6f7f9",
  "components.listingSubContainer.hoverColor": "#f6f7f9",
  "components.listingSubContainer.borderColor": "#dce1e8",
  "components.listingSubContainer.borderThickness": "1px",
  "components.listingSubContainer.borderRadius": "3px",
  "components.listingSubContainer.textColor": "#1f252e",
  "components.listingSubContainer.size": "md",

  "components.optionCard.background": "#ffffff",
  "components.optionCard.hoverColor": "#e9edf3",
  "components.optionCard.borderColor": "#dce1e8",
  "components.optionCard.borderThickness": "1px",
  "components.optionCard.borderRadius": "3px",
  "components.optionCard.textColor": "#1f252e",
  "components.optionCard.size": "md",

  "components.card.background": "#ffffff",
  "components.card.hoverColor": "#f9fafb",
  "components.card.borderColor": "#dce1e8",
  "components.card.borderThickness": "1px",
  "components.card.borderRadius": "3px",
  "components.card.textColor": "#1f252e",
  "components.card.size": "md",

  "components.alert.background": "#ffffff",
  "components.alert.borderColor": "#dce1e8",
  "components.alert.borderThickness": "1px",
  "components.alert.borderRadius": "0.5rem",
  "components.alert.textColor": "#1f252e",

  "components.input.background": "#ffffff",
  "components.input.hoverColor": "#f8fafc",
  "components.input.borderColor": "#d1d8e2",
  "components.input.borderThickness": "1px",
  "components.input.borderRadius": "3px",
  "components.input.textColor": "#1f252e",
  "components.input.size": "md",

  "components.textarea.background": "#ffffff",
  "components.textarea.hoverColor": "#f8fafc",
  "components.textarea.borderColor": "#d1d8e2",
  "components.textarea.borderThickness": "1px",
  "components.textarea.borderRadius": "3px",
  "components.textarea.textColor": "#1f252e",
  "components.textarea.size": "md",

  "components.datePicker.background": "#ffffff",
  "components.datePicker.hoverColor": "#f8fafc",
  "components.datePicker.borderColor": "#d1d8e2",
  "components.datePicker.borderThickness": "1px",
  "components.datePicker.borderRadius": "3px",
  "components.datePicker.textColor": "#1f252e",
  "components.datePicker.size": "md",

  "components.switchSubContainer.background": "#ffffff",
  "components.switchSubContainer.hoverColor": "#f8fafc",
  "components.switchSubContainer.borderColor": "#dce1e8",
  "components.switchSubContainer.borderThickness": "1px",
  "components.switchSubContainer.borderRadius": "3px",
  "components.switchSubContainer.textColor": "#1f252e",
  "components.switchSubContainer.size": "md",

  "components.checkboxSubContainer.background": "#ffffff",
  "components.checkboxSubContainer.hoverColor": "#f8fafc",
  "components.checkboxSubContainer.borderColor": "#dce1e8",
  "components.checkboxSubContainer.borderThickness": "1px",
  "components.checkboxSubContainer.borderRadius": "3px",
  "components.checkboxSubContainer.textColor": "#1f252e",
  "components.checkboxSubContainer.size": "md",

  "components.switch.background": "#2563eb",
  "components.switch.hoverColor": "#1d4ed8",
  "components.switch.borderColor": "transparent",
  "components.switch.borderThickness": "0px",
  "components.switch.borderRadius": "999px",
  "components.switch.textColor": "#ffffff",
  "components.switch.size": "md",
  "components.switch.uncheckedBackground": "#d1d8e2",
  "components.switch.thumbColor": "#f6f7f9",
}

const statusDefaults: Record<string, { dot: string; text?: string }> = {
  positive: { dot: "#00bc7d" },
  "in-progress": { dot: "#f0b100" },
  neutral: { dot: "#647084" },
  negative: { dot: "#fb2c36" },
}

for (const group of Object.keys(BADGE_GROUP_MEMBERS)) {
  DEFAULT_TOKENS[`components.badges.${group}.background`] = "#f6f7f9"
  DEFAULT_TOKENS[`components.badges.${group}.hoverColor`] = "#f6f7f9"
  DEFAULT_TOKENS[`components.badges.${group}.borderColor`] = "#dce1e8"
  DEFAULT_TOKENS[`components.badges.${group}.borderThickness`] = "1px"
  DEFAULT_TOKENS[`components.badges.${group}.borderRadius`] = "3px"
  DEFAULT_TOKENS[`components.badges.${group}.textColor`] =
    statusDefaults[group].text ?? "#1f252e"
  DEFAULT_TOKENS[`components.badges.${group}.dotColor`] = statusDefaults[group].dot
  DEFAULT_TOKENS[`components.badges.${group}.size`] = "sm"
}

export const tokenId = (t: ThemeToken) => t.path.join(".")

export const isHex = (value: string) => /^#([0-9a-f]{3}|[0-9a-f]{4}|[0-9a-f]{6}|[0-9a-f]{8})$/i.test(value)

export function hexToColorInputValue(value: string | undefined): string {
  if (!value || !isHex(value)) return "#000000"
  const normalized = value.toLowerCase()
  if (normalized.length === 4 || normalized.length === 5) {
    const [r, g, b] = normalized.slice(1)
    return `#${r}${r}${g}${g}${b}${b}`
  }
  return normalized.slice(0, 7)
}

export function hexToRgba(value: string | undefined, alphaMultiplier = 1, fallback = "37,99,235"): string {
  if (!value || !isHex(value)) return `rgba(${fallback},${alphaMultiplier})`
  const normalized = value.toLowerCase()
  const raw = normalized.slice(1)
  const channels = raw.length === 3 || raw.length === 4
    ? raw.split("").map((part) => `${part}${part}`)  // nosemgrep
    : raw.match(/.{2}/g) ?? []
  const [rRaw, gRaw, bRaw, aRaw] = channels
  const r = Number.parseInt(rRaw, 16)
  const g = Number.parseInt(gRaw, 16)
  const b = Number.parseInt(bRaw, 16)
  const baseAlpha = aRaw ? Number.parseInt(aRaw, 16) / 255 : 1
  const alpha = Math.max(0, Math.min(1, baseAlpha * alphaMultiplier))
  return `rgba(${r},${g},${b},${Number(alpha.toFixed(4))})`
}

export function valueAtPath(obj: unknown, path: string[]): unknown {
  let current = obj
  for (const part of path) {
    if (!current || typeof current !== "object") return undefined
    current = (current as Record<string, unknown>)[part]  // nosemgrep
  }
  return current
}

function setValueAtPath(obj: Record<string, unknown>, path: string[], value: string) {
  let current = obj
  path.forEach((part, index) => {
    if (index === path.length - 1) {
      current[part] = value
      return
    }
    if (!current[part] || typeof current[part] !== "object") {
      current[part] = {}
    }
    current = current[part] as Record<string, unknown>
  })
}

function hexLuminance(value: string | undefined): number | null {
  if (!value || !isHex(value)) return null
  const normalized = hexToColorInputValue(value)
  const r = Number.parseInt(normalized.slice(1, 3), 16) / 255
  const g = Number.parseInt(normalized.slice(3, 5), 16) / 255
  const b = Number.parseInt(normalized.slice(5, 7), 16) / 255
  const linear = [r, g, b].map((channel) =>
    channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4,
  )
  return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2]
}

function isDarkHex(value: string | undefined): boolean {
  const luminance = hexLuminance(value)
  return luminance !== null && luminance < 0.18
}

/** Flatten a branding's metadata into the fixed token map, with defaults filled in. */
export function tokensFromMetadata(
  metadata: Record<string, unknown> | null | undefined,
): Record<string, string> {
  const out: Record<string, string> = { ...DEFAULT_TOKENS }
  if (!metadata) return out

  for (const t of THEME_TOKENS) {
    const value = valueAtPath(metadata, t.path)
    if (typeof value === "string" && value) out[tokenId(t)] = value
  }

  const legacyColorMap: Record<string, string> = {
    "color.primary": "colors.primary",
    "color.background": "colors.appBackground",
    "color.card": "colors.cardBackground",
    "color.text": "colors.textPrimary",
    "color.muted": "colors.textMuted",
    "color.border": "colors.border",
  }
  for (const [legacyKey, modernKey] of Object.entries(legacyColorMap)) {
    if (out[modernKey] === DEFAULT_TOKENS[modernKey] && typeof metadata[legacyKey] === "string") {
      out[modernKey] = metadata[legacyKey] as string
    }
  }
  if (out["font.family"] === DEFAULT_TOKENS["font.family"] && typeof metadata["font.family"] === "string") {
    out["font.family"] = metadata["font.family"] as string
  }

  const iconContainerKeys = [
    "background",
    "hoverColor",
    "borderColor",
    "borderThickness",
    "borderRadius",
    "textColor",
    "size",
  ]
  const hasExplicitIconContainer = iconContainerKeys.some((key) =>
    typeof valueAtPath(metadata, ["components", "iconContainer", key]) === "string",
  )
  const hasLegacyListingIcon = iconContainerKeys.some((key) =>
    typeof valueAtPath(metadata, ["components", "listingItemIcon", key]) === "string",
  )

  for (const key of iconContainerKeys) {
    const modernKey = `components.iconContainer.${key}`
    const legacyValue = valueAtPath(metadata, ["components", "listingItemIcon", key])
    if (out[modernKey] === DEFAULT_TOKENS[modernKey] && typeof legacyValue === "string" && legacyValue) {
      out[modernKey] = legacyValue
    }
  }

  if (!hasExplicitIconContainer && !hasLegacyListingIcon && isDarkHex(out["colors.appBackground"])) {
    const iconBg = isDarkHex(out["colors.secondary"])
      ? out["colors.secondary"]
      : out["colors.cardBackground"]
    out["components.iconContainer.background"] = iconBg
    out["components.iconContainer.hoverColor"] = isDarkHex(out["colors.accent"])
      ? out["colors.accent"]
      : iconBg
    out["components.iconContainer.borderColor"] = out["colors.border"]
    out["components.iconContainer.textColor"] = out["colors.textMuted"]
  }

  return out
}

/** Build the metadata object the API expects from the fixed token map. */
export function metadataFromTokens(tokens: Record<string, string>): Record<string, unknown> {
  const metadata: Record<string, unknown> = {}
  for (const t of THEME_TOKENS) {
    setValueAtPath(metadata, t.path, (tokens[tokenId(t)] ?? "").trim())
  }
  return metadata
}
