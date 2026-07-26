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
    expect(pages.find((page) => page.id === "step-up-methods")).toMatchObject({
      title: "Confirm it's you",
      group: "Security",
    })
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
})
