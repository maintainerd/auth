import { describe, expect, it } from "vitest"
import {
  BADGE_GROUP_MEMBERS,
  DEFAULT_TOKENS,
  STATUS_BADGE_TYPES,
  THEME_SECTIONS,
  THEME_TOKENS,
  hexToColorInputValue,
  hexToRgba,
  isHex,
  metadataFromTokens,
  tokensFromMetadata,
} from "./themeTokens"

describe("themeTokens", () => {
  it("includes a complete token set for every badge tone group", () => {
    for (const group of Object.keys(BADGE_GROUP_MEMBERS)) {
      for (const key of [
        "background",
        "hoverColor",
        "borderColor",
        "borderThickness",
        "borderRadius",
        "textColor",
        "dotColor",
        "size",
      ]) {
        expect(DEFAULT_TOKENS[`components.badges.${group}.${key}`]).toBeTruthy()
      }
    }
  })

  it("covers every status badge type in exactly one tone group", () => {
    const grouped = Object.values(BADGE_GROUP_MEMBERS).flat()
    expect(grouped).toHaveLength(STATUS_BADGE_TYPES.length)
    expect(new Set(grouped).size).toBe(STATUS_BADGE_TYPES.length)
    for (const status of STATUS_BADGE_TYPES) {
      expect(grouped).toContain(status)
    }
  })

  it("stores one shared token set per tone group, not per status", () => {
    const badges = THEME_SECTIONS.find((s) => s.id === "badges")
    expect(badges).toBeDefined()

    const expectedGroups = ["positive", "in-progress", "neutral", "negative"]
    expect(new Set(badges!.tokens.map((t) => t.group))).toEqual(new Set(expectedGroups))
    expect(badges!.tokens.length).toBe(expectedGroups.length * 8)

    const positivePaths = badges!.tokens
      .filter((t) => t.group === "positive")
      .map((t) => t.path.join("."))
    expect(positivePaths.every((p) => p.startsWith("components.badges.positive."))).toBe(true)
  })

  it("captures the actual switch states", () => {
    expect(DEFAULT_TOKENS["components.switch.background"]).toBe("#2563eb")
    expect(DEFAULT_TOKENS["components.switch.uncheckedBackground"]).toBe("#d1d8e2")
    expect(DEFAULT_TOKENS["components.switch.thumbColor"]).toBe("#f6f7f9")
  })

  it("captures hosted login template layout surfaces", () => {
    for (const key of [
      "authPageBackground",
      "authFormPanelBackground",
      "authFormPanelBorder",
      "authFormPanelText",
      "authVisualPanelBackground",
      "authVisualPanelText",
      "authVisualPanelOverlay",
      "authDecorativeLight",
      "authDecorativeDark",
      "authProgressPanelBackground",
      "authSecurityPanelBackground",
    ]) {
      expect(DEFAULT_TOKENS[`colors.${key}`]).toBeTruthy()
    }
  })

  it("includes console action, table, listing, and date picker tokens", () => {
    for (const component of [
      "topPanelCreateButton",
      "primaryButton",
      "secondaryButton",
      "outlineButton",
      "destructiveButton",
      "ghostButton",
      "tableContainer",
      "tableHeader",
      "tableRow",
      "tableCell",
      "iconContainer",
      "listingItem",
      "listingItemIcon",
      "listingSubContainer",
      "datePicker",
    ]) {
      for (const key of [
        "background",
        "hoverColor",
        "borderColor",
        "borderThickness",
        "borderRadius",
        "textColor",
        "size",
      ]) {
        expect(DEFAULT_TOKENS[`components.${component}.${key}`]).toBeTruthy()
      }
    }
  })

  it("includes alert banner tokens as an alert sub-group of the card section", () => {
    const card = THEME_SECTIONS.find((s) => s.id === "card")
    expect(card).toBeDefined()

    const alertTokens = card!.tokens.filter((t) => t.group === "alert")
    expect(alertTokens.length).toBeGreaterThan(0)

    for (const key of [
      "background",
      "borderColor",
      "borderThickness",
      "borderRadius",
      "textColor",
    ]) {
      expect(DEFAULT_TOKENS[`components.alert.${key}`]).toBeTruthy()
      expect(alertTokens.some((t) => t.path.join(".") === `components.alert.${key}`)).toBe(true)
    }

    expect(THEME_SECTIONS.some((s) => s.id === "alerts")).toBe(false)
  })

  it("round trips the nested branding metadata shape", () => {
    const metadata = metadataFromTokens({
      ...DEFAULT_TOKENS,
      "components.primaryButton.background": "#111827",
      "colors.authVisualPanelBackground": "#0f172a",
      "components.switch.uncheckedBackground": "#94a3b8",
      "components.badges.negative.dotColor": "#991b1b",
    })

    expect(metadata).toMatchObject({
      components: {
        primaryButton: {
          background: "#111827",
        },
        switch: {
          uncheckedBackground: "#94a3b8",
        },
        badges: {
          negative: {
            dotColor: "#991b1b",
          },
        },
      },
      colors: {
        authVisualPanelBackground: "#0f172a",
      },
    })

    expect(tokensFromMetadata(metadata)["components.primaryButton.background"]).toBe("#111827")
    expect(tokensFromMetadata(metadata)["colors.authVisualPanelBackground"]).toBe("#0f172a")
    expect(tokensFromMetadata(metadata)["components.switch.uncheckedBackground"]).toBe("#94a3b8")
    expect(tokensFromMetadata(metadata)["components.badges.negative.dotColor"]).toBe("#991b1b")
  })

  it("merges all button types into one section with tagged sub-groups", () => {
    const buttons = THEME_SECTIONS.find((s) => s.id === "buttons")
    expect(buttons).toBeDefined()
    const groups = new Set(buttons!.tokens.map((t) => t.group))
    expect([...groups]).toEqual([
      "primaryButton",
      "secondaryButton",
      "outlineButton",
      "destructiveButton",
      "ghostButton",
    ])

    for (const group of groups) {
      expect(
        buttons!.tokens.filter((t) => t.group === group).length,
        `${group} has one token per component field`,
      ).toBe(7)
    }

    for (const id of ["primary-button", "secondary-button", "outline-button", "destructive-button", "actions"]) {
      expect(THEME_SECTIONS.some((s) => s.id === id)).toBe(false)
    }
  })

  it("merges listing tokens into the card section as listing-card and sub-container sub-groups", () => {
    const card = THEME_SECTIONS.find((s) => s.id === "card")
    expect(card).toBeDefined()

    const groups = new Set(card!.tokens.map((t) => t.group))
    expect([...groups]).toEqual(["card", "listing-card", "sub-container", "option-card", "alert"])

    // The listing card is the main item surface only — icon and meta stay as
    // runtime defaults, not separately editable tokens.
    const listingTokens = card!.tokens.filter((t) => t.group === "listing-card")
    expect(listingTokens.length).toBe(7)
    expect(listingTokens.every((t) => t.path[1] === "listingItem")).toBe(true)
    expect(
      card!.tokens.some((t) => t.group === "listing-card" && t.path[1] === "listingItemIcon"),
    ).toBe(false)
    expect(
      card!.tokens.some((t) => t.group === "listing-card" && t.path[1] === "listingItemMeta"),
    ).toBe(false)

    const subContainerTokens = card!.tokens.filter((t) => t.group === "sub-container")
    expect(subContainerTokens.length).toBe(7)
    expect(subContainerTokens.every((t) => t.path[1] === "listingSubContainer")).toBe(true)

    const optionCardTokens = card!.tokens.filter((t) => t.group === "option-card")
    expect(optionCardTokens.length).toBe(7)
    expect(optionCardTokens.every((t) => t.path[1] === "optionCard")).toBe(true)

    expect(THEME_SECTIONS.some((s) => s.id === "listing")).toBe(false)
  })

  it("exposes icon wells in a dedicated icon containers section", () => {
    const section = THEME_SECTIONS.find((s) => s.id === "icon-containers")
    expect(section).toBeDefined()
    expect(section!.tokens).toHaveLength(7)
    expect(section!.tokens.every((t) => t.path[1] === "iconContainer")).toBe(true)

    for (const key of [
      "background",
      "hoverColor",
      "borderColor",
      "borderThickness",
      "borderRadius",
      "textColor",
      "size",
    ]) {
      expect(DEFAULT_TOKENS[`components.iconContainer.${key}`]).toBeTruthy()
    }
  })

  it("uses legacy listing item icon metadata as an icon container fallback", () => {
    const tokens = tokensFromMetadata({
      components: {
        listingItemIcon: {
          background: "#101827",
          hoverColor: "#17233a",
          borderColor: "#2b3b54",
          borderThickness: "1px",
          borderRadius: "7px",
          textColor: "#c8d7ef",
          size: "lg",
        },
      },
    })

    expect(tokens["components.iconContainer.background"]).toBe("#101827")
    expect(tokens["components.iconContainer.hoverColor"]).toBe("#17233a")
    expect(tokens["components.iconContainer.borderColor"]).toBe("#2b3b54")
    expect(tokens["components.iconContainer.borderThickness"]).toBe("1px")
    expect(tokens["components.iconContainer.borderRadius"]).toBe("7px")
    expect(tokens["components.iconContainer.textColor"]).toBe("#c8d7ef")
    expect(tokens["components.iconContainer.size"]).toBe("lg")
  })

  it("derives dark icon container colors from a dark palette when icon tokens are absent", () => {
    const tokens = tokensFromMetadata({
      colors: {
        appBackground: "#0d1117",
        cardBackground: "#111827",
        secondary: "#94a3b8",
        accent: "#38bdf8",
        textMuted: "#94a3b8",
        border: "#1f2937",
      },
    })

    expect(tokens["components.iconContainer.background"]).toBe("#111827")
    expect(tokens["components.iconContainer.hoverColor"]).toBe("#111827")
    expect(tokens["components.iconContainer.borderColor"]).toBe("#1f2937")
    expect(tokens["components.iconContainer.textColor"]).toBe("#94a3b8")
  })

  it("merges switch tokens into the inputs section as a switch sub-group", () => {
    const inputs = THEME_SECTIONS.find((s) => s.id === "inputs")
    expect(inputs).toBeDefined()

    const groups = new Set(inputs!.tokens.map((t) => t.group))
    expect([...groups]).toEqual([
      "inputs",
      "textarea",
      "switch",
      "switch-sub-container",
      "checkbox-sub-container",
    ])

    const switchGroup = inputs!.tokens.filter((t) => t.group === "switch")
    expect(switchGroup.length).toBe(9)
    for (const key of [
      "background",
      "hoverColor",
      "borderColor",
      "borderThickness",
      "borderRadius",
      "textColor",
      "size",
      "uncheckedBackground",
      "thumbColor",
    ]) {
      expect(switchGroup.some((t) => t.path.join(".") === `components.switch.${key}`)).toBe(true)
    }

    const switchSubGroup = inputs!.tokens.filter((t) => t.group === "switch-sub-container")
    expect(switchSubGroup.length).toBe(7)
    for (const key of [
      "background",
      "hoverColor",
      "borderColor",
      "borderThickness",
      "borderRadius",
      "textColor",
      "size",
    ]) {
      expect(switchSubGroup.some((t) => t.path.join(".") === `components.switchSubContainer.${key}`)).toBe(true)
    }
    for (const key of [
      "background",
      "hoverColor",
      "borderColor",
      "borderThickness",
      "borderRadius",
      "textColor",
      "size",
    ]) {
      expect(DEFAULT_TOKENS[`components.switchSubContainer.${key}`]).toBeTruthy()
    }

    const checkboxSubGroup = inputs!.tokens.filter((t) => t.group === "checkbox-sub-container")
    expect(checkboxSubGroup.length).toBe(7)
    for (const key of [
      "background",
      "hoverColor",
      "borderColor",
      "borderThickness",
      "borderRadius",
      "textColor",
      "size",
    ]) {
      expect(
        checkboxSubGroup.some((t) => t.path.join(".") === `components.checkboxSubContainer.${key}`),
      ).toBe(true)
      expect(DEFAULT_TOKENS[`components.checkboxSubContainer.${key}`]).toBeTruthy()
    }

    expect(THEME_SECTIONS.some((s) => s.id === "switch")).toBe(false)
  })

  it("gives textareas their own token set inside the inputs section", () => {
    const inputs = THEME_SECTIONS.find((s) => s.id === "inputs")
    expect(inputs).toBeDefined()

    const textareaGroup = inputs!.tokens.filter((t) => t.group === "textarea")
    expect(textareaGroup.length).toBe(7)
    for (const key of [
      "background",
      "hoverColor",
      "borderColor",
      "borderThickness",
      "borderRadius",
      "textColor",
      "size",
    ]) {
      expect(textareaGroup.some((t) => t.path.join(".") === `components.textarea.${key}`)).toBe(true)
      expect(DEFAULT_TOKENS[`components.textarea.${key}`]).toBeTruthy()
    }
  })

  it("keeps legacy flat metadata usable while newer themes use nested tokens", () => {
    const tokens = tokensFromMetadata({
      "color.primary": "#123456",
      "color.background": "#abcdef",
    })

    expect(tokens["colors.primary"]).toBe("#123456")
    expect(tokens["colors.appBackground"]).toBe("#abcdef")
    expect(THEME_TOKENS.length).toBeGreaterThan(100)
  })

  it("accepts alpha hex colors for transparent theme values", () => {
    expect(isHex("#fff")).toBe(true)
    expect(isHex("#ffff")).toBe(true)
    expect(isHex("#ffffff")).toBe(true)
    expect(isHex("#ffffff00")).toBe(true)
    expect(isHex("#fffffff")).toBe(false)
  })

  it("keeps alpha hex in metadata while using opaque rgb for native color inputs", () => {
    const metadata = metadataFromTokens({
      ...DEFAULT_TOKENS,
      "colors.authVisualPanelOverlay": "#ffffff00",
    })

    expect(tokensFromMetadata(metadata)["colors.authVisualPanelOverlay"]).toBe("#ffffff00")
    expect(hexToColorInputValue("#ffffff00")).toBe("#ffffff")
    expect(hexToColorInputValue("#abcd")).toBe("#aabbcc")
    expect(hexToRgba("#ffffff00", 0.42)).toBe("rgba(255,255,255,0)")
    expect(hexToRgba("#00000080", 1)).toBe("rgba(0,0,0,0.502)")
  })
})
