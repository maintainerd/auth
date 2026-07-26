/**
 * Branding theme tokens — a fixed, system-defined set of theme variables.
 * Keys are not user-editable; only their values are. Mirrors the shape produced
 * by the backend seeder (metadata.colors.* + metadata.font.family).
 */

export type ThemeToken = {
  group: "colors" | "font"
  key: string
  label: string
  kind: "color" | "text"
}

export const THEME_TOKENS: ThemeToken[] = [
  { group: "colors", key: "primary", label: "Primary", kind: "color" },
  { group: "colors", key: "secondary", label: "Secondary", kind: "color" },
  { group: "colors", key: "accent", label: "Accent", kind: "color" },
  { group: "colors", key: "appBackground", label: "App background", kind: "color" },
  { group: "colors", key: "topPanelBackground", label: "Top panel background", kind: "color" },
  { group: "colors", key: "sidePanelBackground", label: "Side panel background", kind: "color" },
  { group: "colors", key: "cardBackground", label: "Card background", kind: "color" },
  { group: "colors", key: "textPrimary", label: "Primary text", kind: "color" },
  { group: "colors", key: "textMuted", label: "Muted text", kind: "color" },
  { group: "colors", key: "border", label: "Border", kind: "color" },
  { group: "font", key: "family", label: "Font family", kind: "text" },
]

export const DEFAULT_TOKENS: Record<string, string> = {
  "colors.primary": "#2563eb",
  "colors.secondary": "#2d3748",
  "colors.accent": "#0ea5e9",
  "colors.appBackground": "#f6f7f9",
  "colors.topPanelBackground": "#0f172a",
  "colors.sidePanelBackground": "#f6f7f9",
  "colors.cardBackground": "#ffffff",
  "colors.textPrimary": "#1f252e",
  "colors.textMuted": "#647084",
  "colors.border": "#dce1e8",
  "font.family": "Inter, system-ui, sans-serif",
}

export const tokenId = (t: ThemeToken) => `${t.group}.${t.key}`

export const isHex = (value: string) => /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.test(value)

/** Flatten a branding's metadata into the fixed token map, with defaults filled in. */
export function tokensFromMetadata(
  metadata: Record<string, unknown> | null | undefined,
): Record<string, string> {
  const out: Record<string, string> = { ...DEFAULT_TOKENS }
  if (!metadata) return out
  for (const t of THEME_TOKENS) {
    const group = metadata[t.group]
    if (group && typeof group === "object") {
      const value = (group as Record<string, unknown>)[t.key]
      if (typeof value === "string" && value) out[tokenId(t)] = value
    }
  }
  return out
}

/** Build the metadata object the API expects from the fixed token map. */
export function metadataFromTokens(tokens: Record<string, string>): Record<string, Record<string, string>> {
  const metadata: Record<string, Record<string, string>> = { colors: {}, font: {} }
  for (const t of THEME_TOKENS) {
    metadata[t.group][t.key] = (tokens[tokenId(t)] ?? "").trim()
  }
  return metadata
}
