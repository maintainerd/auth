import { describe, expect, it } from "vitest"
import {
  DEFAULT_LOGIN_PAGE_PREVIEWS,
  LOGIN_PAGE_PREVIEW_GROUPS,
  LOGIN_PAGE_PREVIEW_IDS,
  loginPageContentCollectionMetadata,
  loginPageContentMetadata,
  loginPagePreviewsFromMetadata,
  loginTemplateImageUrlFromMetadata,
} from "./loginPageContent"

describe("loginPageContent", () => {
  it("has a default preview for every fixed identity page state", () => {
    expect(Object.keys(DEFAULT_LOGIN_PAGE_PREVIEWS)).toEqual([...LOGIN_PAGE_PREVIEW_IDS])
    expect(loginPagePreviewsFromMetadata(null).map((page) => page.id)).toEqual([...LOGIN_PAGE_PREVIEW_IDS])
  })

  it("uses only configured preview groups", () => {
    const groups = new Set(LOGIN_PAGE_PREVIEW_GROUPS)

    expect(loginPagePreviewsFromMetadata(null).every((page) => groups.has(page.group))).toBe(true)
  })

  it("returns fixed identity page previews when metadata has no configured copy", () => {
    const pages = loginPagePreviewsFromMetadata(null)

    expect(pages.find((page) => page.id === "login")).toMatchObject({
      title: "Welcome back",
      group: "Sign-in",
    })
    expect(pages.find((page) => page.id === "oauth-consent-request")).toMatchObject({
      label: "OAuth Consent: Request",
      group: "OAuth",
    })
  })

  it("keeps alternate page states as separate previews", () => {
    const pages = loginPagePreviewsFromMetadata(null)

    expect(pages.find((page) => page.id === "invite-accept")).toMatchObject({
      title: "Accept your invitation",
    })
    expect(pages.find((page) => page.id === "invite-invalid")).toMatchObject({
      title: "Invalid invite link",
    })
    expect(pages.find((page) => page.id === "reset-password-form")).toMatchObject({
      title: "Reset your password",
    })
    expect(pages.find((page) => page.id === "reset-password-invalid")).toMatchObject({
      title: "Invalid reset link",
    })
    expect(pages.find((page) => page.id === "oauth-device-approved")).toMatchObject({
      title: "Device approved",
    })
    expect(pages.find((page) => page.id === "oauth-device-denied")).toMatchObject({
      title: "Device denied",
    })
    expect(pages.find((page) => page.id === "login-methods-unavailable")).toMatchObject({
      title: "Welcome back",
    })
    expect(pages.find((page) => page.id === "registration-invalid-link")).toMatchObject({
      title: "This sign-up link is no longer valid",
    })
    expect(pages.find((page) => page.id === "phone-verification-code")).toMatchObject({
      title: "Verify Phone",
      group: "Account",
    })
    expect(pages.find((page) => page.id === "account-erasure-confirm")).toMatchObject({
      title: "Request account deletion",
      group: "Account",
    })
  })

  it("does not expose dialog-only identity components as hosted page previews", () => {
    expect([...LOGIN_PAGE_PREVIEW_IDS].some((id) => id.startsWith("step-up"))).toBe(false)
  })

  it("overrides only configured copy fields", () => {
    const pages = loginPagePreviewsFromMetadata({
      login_page_content: {
        login: {
          title: "Welcome to Acme",
          primaryAction: "Ignored legacy action",
        },
      },
    })

    expect(pages.find((page) => page.id === "login")).toMatchObject({
      title: "Welcome to Acme",
      subtitle: "Sign in to your account to continue.",
    })
  })

  it("preserves other metadata while saving selected page copy", () => {
    const pages = loginPagePreviewsFromMetadata(null)
    const login = pages.find((page) => page.id === "login")!

    const metadata = loginPageContentMetadata(
      { colors: { primary: "#000000" }, auth_ui_template: "centered-card" },
      {
        ...login,
        title: "Welcome back",
        subtitle: "Use your company account.",
      },
    )

    expect(metadata).toMatchObject({
      colors: { primary: "#000000" },
      auth_ui_template: "centered-card",
      login_page_content: {
        login: {
          title: "Welcome back",
          subtitle: "Use your company account.",
        },
      },
    })
    const configuredLogin = (metadata.login_page_content as Record<string, Record<string, unknown>>).login
    expect(configuredLogin).not.toHaveProperty("primaryAction")
  })

  it("saves all configured page copy values together", () => {
    const pages = loginPagePreviewsFromMetadata(null)
    const login = pages.find((page) => page.id === "login")!
    const inviteInvalid = pages.find((page) => page.id === "invite-invalid")!

    const metadata = loginPageContentCollectionMetadata(
      { auth_ui_template: "centered-card" },
      [
        { ...login, title: "Tenant sign in", subtitle: "Use your tenant account." },
        { ...inviteInvalid, title: "Invite expired", subtitle: "Ask your administrator for a new invite." },
      ],
    )

    expect(metadata).toMatchObject({
      auth_ui_template: "centered-card",
      login_page_content: {
        login: {
          title: "Tenant sign in",
          subtitle: "Use your tenant account.",
        },
        "invite-invalid": {
          title: "Invite expired",
          subtitle: "Ask your administrator for a new invite.",
        },
      },
    })
  })

  it("stores and reads the login template image URL", () => {
    const pages = loginPagePreviewsFromMetadata(null)
    const metadata = loginPageContentCollectionMetadata(
      {},
      [pages[0]],
      " https://example.com/login-side.jpg ",
    )

    expect(metadata).toMatchObject({
      login_template_image_url: "https://example.com/login-side.jpg",
    })
    expect(loginTemplateImageUrlFromMetadata(metadata)).toBe("https://example.com/login-side.jpg")
    expect(loginTemplateImageUrlFromMetadata(null)).toBe("")
  })

  it("matches the identity default registration fields", () => {
    const registration = DEFAULT_LOGIN_PAGE_PREVIEWS.registration
    const fieldLabels = registration.elements
      .filter((element) => element.type === "field")
      .map((element) => element.label)

    expect(fieldLabels).toEqual(["Email", "Password", "Confirm password"])
    expect(registration.elements.some((element) => element.type === "checkbox")).toBe(true)
    expect(registration.elements.some((element) => element.type === "field" && element.label === "Full name")).toBe(false)
    expect(registration.elements.some((element) => element.type === "field" && element.label === "Phone")).toBe(false)
  })

  it("keeps login MFA method choices identity-complete and one column", () => {
    const mfaMethods = DEFAULT_LOGIN_PAGE_PREVIEWS["login-mfa-code"].elements.find(
      (element) => element.type === "tile-list",
    )

    expect(mfaMethods).toMatchObject({ columns: 1 })
    expect(mfaMethods?.items.map((item) => item.title)).toEqual([
      "Authenticator app",
      "Passkey",
      "Text message",
      "Email OTP",
      "Backup code",
    ])
    expect(DEFAULT_LOGIN_PAGE_PREVIEWS["login-mfa-code"].elements).toContainEqual({
      type: "checkbox",
      label: "Trust this device — skip verification here next time",
    })
  })

  it("removes email OTP from magic-link MFA because email is already proven", () => {
    const mfaMethods = DEFAULT_LOGIN_PAGE_PREVIEWS["magic-link-mfa"].elements.find(
      (element) => element.type === "tile-list",
    )
    const labels = DEFAULT_LOGIN_PAGE_PREVIEWS["magic-link-mfa"].elements.map((element) =>
      element.type === "button" ? element.label : "",
    )

    expect(mfaMethods?.items.map((item) => item.title)).toEqual([
      "Authenticator app",
      "Passkey",
      "Text message",
      "Backup code",
    ])
    expect(labels).not.toContain("Send code to my email")
  })

  it("keeps the console sign-in loading preview with the full login form shape", () => {
    const loading = DEFAULT_LOGIN_PAGE_PREVIEWS["login-methods-loading"].elements

    expect(loading).toEqual([
      {
        type: "section",
        title: "Loading sign-in methods...",
        description: "Available password and identity provider options are loading.",
      },
      { type: "field", label: "Email", value: "you@company.com", kind: "email" },
      { type: "field", label: "Password", value: "********", kind: "password" },
      { type: "button", label: "Sign in" },
    ])
  })

  it("models connected app grants as per-grant rows with revoke actions", () => {
    const grants = DEFAULT_LOGIN_PAGE_PREVIEWS["oauth-grants-list"].elements.find(
      (element) => element.type === "tile-list",
    )

    expect(grants).toMatchObject({ columns: 1 })
    expect(grants?.items[0].scopes).toEqual(["openid", "profile", "email"])
    expect(grants?.items.every((item) => Boolean(item.actionLabel))).toBe(true)
  })
})
