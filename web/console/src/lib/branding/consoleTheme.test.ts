import { describe, expect, it, afterEach } from "vitest"
import {
  applyConsoleTheme,
  clearConsoleTheme,
  consoleThemeVariablesFromBranding,
} from "./consoleTheme"
import type { BrandingPublic } from "@/services/api/tenants/types"

const branding = (metadata: BrandingPublic["metadata"], companyName = "Acme IAM"): BrandingPublic => ({
  layout: "centered",
  company_name: companyName,
  logo_label: "Acme",
  logo_detail: "",
  show_logo_label: true,
  identity_logo_label: "",
  identity_show_logo_label: true,
  logo_url: "",
  favicon_url: "",
  support_url: "",
  privacy_policy_url: "",
  terms_of_service_url: "",
  metadata,
})

afterEach(() => {
  clearConsoleTheme()
})

describe("console theme runtime", () => {
  it("maps bootstrap branding metadata to global and component CSS variables", () => {
    const vars = consoleThemeVariablesFromBranding(branding({
      colors: {
        primary: "#123456cc",
        topPanelBackground: "#101820",
        authPageBackground: "#090f1a",
        authFormPanelBackground: "#121a28",
        authFormPanelBorder: "#263348",
        authFormPanelText: "#edf4ff",
        sidePanelBackground: "#fdfefe",
        textPrimary: "#172033",
      },
      effects: {
        authFormPanelShadow: "0 16px 40px -24px rgba(0,0,0,0.8)",
      },
      font: {
        family: "Example Sans, system-ui, sans-serif",
      },
      components: {
        input: {
          background: "#ffffff80",
          hoverColor: "#f4f7fb",
          borderColor: "#abc123",
          borderThickness: "2px",
          borderRadius: "14px",
          textColor: "#111827",
          size: "lg",
        },
        textarea: {
          background: "#fefefe",
          hoverColor: "#f1f5f9",
          borderColor: "#94a3b8",
          borderThickness: "1px",
          borderRadius: "8px",
          textColor: "#0f172a",
          size: "md",
        },
        tableRow: {
          background: "#ffffff",
          hoverColor: "#f1f5f9",
          borderColor: "#cbd5e1",
          borderThickness: "2px",
          borderRadius: "0px",
          textColor: "#111827",
          size: "md",
        },
        iconContainer: {
          background: "#101827",
          hoverColor: "#17233a",
          borderColor: "#2b3b54",
          borderThickness: "1px",
          borderRadius: "7px",
          textColor: "#c8d7ef",
          size: "md",
        },
        listingItem: {
          background: "#ffffff",
          hoverColor: "#f8fafc",
          borderColor: "#dbe4ef",
          borderThickness: "1px",
          borderRadius: "10px",
          textColor: "#111827",
          size: "md",
        },
        listingSubContainer: {
          background: "#f8fafc",
          hoverColor: "#f1f5f9",
          borderColor: "#cbd5e1",
          borderThickness: "1px",
          borderRadius: "8px",
          textColor: "#0f172a",
          size: "sm",
        },
        optionCard: {
          background: "#ffffff",
          hoverColor: "#eef2ff",
          borderColor: "#94a3b8",
          borderThickness: "1px",
          borderRadius: "6px",
          textColor: "#0f172a",
          size: "md",
        },
        datePicker: {
          background: "#ffffff",
          hoverColor: "#f8fafc",
          borderColor: "#64748b",
          borderThickness: "1px",
          borderRadius: "6px",
          textColor: "#111827",
          size: "lg",
        },
        detailsEditButton: {
          background: "#ffffff",
          hoverColor: "#e9edf3",
          borderColor: "#d1d8e2",
          borderThickness: "1px",
          borderRadius: "8px",
          textColor: "#1f252e",
          size: "sm",
        },
        detailsMenuButton: {
          background: "#fefefe",
          hoverColor: "#eef2ff",
          borderColor: "#94a3b8",
          borderThickness: "2px",
          borderRadius: "6px",
          textColor: "#0f172a",
          size: "sm",
        },
        detailsRestoreButton: {
          background: "#f8fafc",
          hoverColor: "#eef2ff",
          borderColor: "#94a3b8",
          borderThickness: "1px",
          borderRadius: "8px",
          textColor: "#0f172a",
          size: "sm",
        },
        switch: {
          background: "#2563eb",
          hoverColor: "#1d4ed8",
          borderColor: "transparent",
          borderThickness: "0px",
          borderRadius: "999px",
          textColor: "#ffffff",
          size: "md",
          uncheckedBackground: "#d1d8e2",
          thumbColor: "#f6f7f9",
        },
        switchSubContainer: {
          background: "#ffffff",
          hoverColor: "#f8fafc",
          borderColor: "#dce1e8",
          borderThickness: "1px",
          borderRadius: "8px",
          textColor: "#1f252e",
          size: "md",
        },
        checkboxSubContainer: {
          background: "#ffffff",
          hoverColor: "#f1f5f9",
          borderColor: "#cbd5e1",
          borderThickness: "1px",
          borderRadius: "6px",
          textColor: "#0f172a",
          size: "md",
        },
        alert: {
          background: "#fefce8",
          borderColor: "#fde047",
          borderThickness: "1px",
          borderRadius: "12px",
          textColor: "#713f12",
        },
        ghostButton: {
          background: "transparent",
          hoverColor: "#eef2ff",
          borderColor: "transparent",
          borderThickness: "0px",
          borderRadius: "6px",
          textColor: "#0f172a",
          size: "sm",
        },
        outlineButton: {
          background: "#ffffff",
          hoverColor: "#eef2ff",
          borderColor: "#94a3b8",
          borderThickness: "1px",
          borderRadius: "6px",
          textColor: "#0f172a",
          size: "md",
        },
        badges: {
          positive: {
            background: "#e7fff6",
            hoverColor: "#d8f7ec",
            borderColor: "#98e6c3",
            borderThickness: "1px",
            borderRadius: "999px",
            textColor: "#065f46",
            dotColor: "#10b981",
            size: "md",
          },
        },
      },
    }))

    expect(vars?.["--primary"]).toBe("#123456cc")
    expect(vars?.["--md-top-panel-bg"]).toBe("#101820")
    expect(vars?.["--md-auth-page-bg"]).toBe("#090f1a")
    expect(vars?.["--md-auth-form-bg"]).toBe("#121a28")
    expect(vars?.["--md-auth-form-border"]).toBe("#263348")
    expect(vars?.["--md-auth-form-text"]).toBe("#edf4ff")
    expect(vars?.["--md-auth-form-shadow"]).toBe("0 16px 40px -24px rgba(0,0,0,0.8)")
    expect(vars?.["--md-side-panel-bg"]).toBe("#fdfefe")
    expect(vars?.["--sidebar"]).toBe("#fdfefe")
    expect(vars?.["--foreground"]).toBe("#172033")
    expect(vars?.["--md-font-family"]).toBe("Example Sans, system-ui, sans-serif")
    expect(vars?.["--md-input-bg"]).toBe("#ffffff80")
    expect(vars?.["--md-input-radius"]).toBe("14px")
    expect(vars?.["--md-input-height"]).toBe("2.75rem")
    expect(vars?.["--md-textarea-bg"]).toBe("#fefefe")
    expect(vars?.["--md-textarea-radius"]).toBe("8px")
    expect(vars?.["--md-textarea-height"]).toBe("2.5rem")
    expect(vars?.["--md-table-row-border-width"]).toBe("2px")
    expect(vars?.["--md-icon-container-bg"]).toBe("#101827")
    expect(vars?.["--md-icon-container-border"]).toBe("#2b3b54")
    expect(vars?.["--md-icon-container-radius"]).toBe("7px")
    expect(vars?.["--md-icon-container-text"]).toBe("#c8d7ef")
    expect(vars?.["--md-listing-item-radius"]).toBe("10px")
    expect(vars?.["--md-listing-sub-bg"]).toBe("#f8fafc")
    expect(vars?.["--md-listing-sub-radius"]).toBe("8px")
    expect(vars?.["--md-listing-sub-font-size"]).toBe("0.75rem")
    expect(vars?.["--md-option-card-bg"]).toBe("#ffffff")
    expect(vars?.["--md-option-card-hover"]).toBe("#eef2ff")
    expect(vars?.["--md-option-card-radius"]).toBe("6px")
    expect(vars?.["--md-switch-sub-bg"]).toBe("#ffffff")
    expect(vars?.["--md-switch-sub-border"]).toBe("#dce1e8")
    expect(vars?.["--md-switch-sub-radius"]).toBe("8px")
    expect(vars?.["--md-checkbox-sub-bg"]).toBe("#ffffff")
    expect(vars?.["--md-checkbox-sub-border"]).toBe("#cbd5e1")
    expect(vars?.["--md-checkbox-sub-radius"]).toBe("6px")
    expect(vars?.["--md-date-picker-border"]).toBe("#64748b")
    expect(vars?.["--md-ghost-button-radius"]).toBe("6px")
    expect(vars?.["--md-ghost-button-bg"]).toBe("transparent")
    expect(vars?.["--md-ghost-button-height"]).toBe("2.25rem")
    expect(vars?.["--md-outline-button-radius"]).toBe("6px")
    expect(vars?.["--md-outline-button-border"]).toBe("#94a3b8")
    expect(vars?.["--md-outline-button-height"]).toBe("2.5rem")
    expect(vars?.["--md-status-active-dot"]).toBe("#10b981")
    expect(vars?.["--md-status-active-padding-x"]).toBe("0.625rem")
    expect(vars?.["--md-status-enabled-bg"]).toBe("#e7fff6")
    expect(vars?.["--md-status-blocked-dot"]).toBe("#fb2c36")
    expect(vars?.["--md-alert-bg"]).toBe("#fefce8")
    expect(vars?.["--md-alert-border"]).toBe("#fde047")
    expect(vars?.["--md-alert-radius"]).toBe("12px")
    expect(vars?.["--md-alert-text"]).toBe("#713f12")
  })

  it("supports legacy flat metadata keys while keeping default tokens for missing fields", () => {
    const vars = consoleThemeVariablesFromBranding(branding({
      "color.primary": "#3344ff",
      "color.background": "#fafafa",
      "font.family": "Legacy Sans, system-ui",
    }))

    expect(vars?.["--primary"]).toBe("#3344ff")
    expect(vars?.["--background"]).toBe("#fafafa")
    expect(vars?.["--md-font-family"]).toBe("Legacy Sans, system-ui")
    expect(vars?.["--md-side-panel-bg"]).toBe("#f6f7f9")
  })

  it("activates only when bootstrap branding exists and clears back to hardcoded fallback", () => {
    applyConsoleTheme(branding({ colors: { topPanelBackground: "#010203" } }, "Tenant Name"))

    expect(document.documentElement.getAttribute("data-console-theme")).toBe("active")
    expect(document.documentElement.style.getPropertyValue("--md-top-panel-bg")).toBe("#010203")
    expect(document.title).toBe("Tenant Name")

    applyConsoleTheme(null)

    expect(document.documentElement.hasAttribute("data-console-theme")).toBe(false)
    expect(document.documentElement.style.getPropertyValue("--md-top-panel-bg")).toBe("")
    expect(document.title).toBe("Maintainerd")
  })
})
