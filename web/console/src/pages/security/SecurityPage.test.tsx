import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import SecurityPage from "./SecurityPage"

const { searchParamsRef } = vi.hoisted(() => ({
  searchParamsRef: { current: new URLSearchParams() },
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return {
    ...actual,
    useSearchParams: () => [searchParamsRef.current, vi.fn()],
  }
})

// Stub all tab content — each page manages its own loading/error/data states.
vi.mock("./password-policies/PasswordPolicyPage", () => ({
  default: () => <div data-testid="tab-password" />,
}))
vi.mock("./mfa/MfaViewPage", () => ({
  default: () => <div data-testid="tab-mfa" />,
}))
vi.mock("./lockout/LockoutViewPage", () => ({
  default: () => <div data-testid="tab-lockout" />,
}))
vi.mock("./session-management/SessionViewPage", () => ({
  default: () => <div data-testid="tab-sessions" />,
}))
vi.mock("./token/TokenViewPage", () => ({
  default: () => <div data-testid="tab-tokens" />,
}))
vi.mock("./registration/RegistrationViewPage", () => ({
  default: () => <div data-testid="tab-registration" />,
}))
vi.mock("./threat-detection/ThreatViewPage", () => ({
  default: () => <div data-testid="tab-threat" />,
}))
vi.mock("./ip-restrictions/IpRestrictionsPage", () => ({
  default: () => <div data-testid="tab-ip-restrictions" />,
}))

describe("SecurityPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
  })

  it("renders the page title, description, and all 8 tab triggers", () => {
    renderWithProviders(<SecurityPage />)
    expect(screen.getByRole("heading", { name: "Security" })).toBeInTheDocument()
    expect(
      screen.getByText(/manage authentication policies, session settings/i),
    ).toBeInTheDocument()
    for (const label of [
      "Password Policy", "Multi-Factor Auth", "Account Lockout", "Sessions",
      "Tokens", "Registration", "Threat Protection", "IP Restrictions",
    ]) {
      expect(screen.getByRole("tab", { name: label })).toBeInTheDocument()
    }
  })

  it("defaults to the password tab", () => {
    renderWithProviders(<SecurityPage />)
    expect(screen.getByTestId("tab-password")).toBeInTheDocument()
    expect(screen.queryByTestId("tab-mfa")).not.toBeInTheDocument()
  })

  it("falls back to password for an unknown ?tab value", () => {
    searchParamsRef.current = new URLSearchParams("tab=bogus")
    renderWithProviders(<SecurityPage />)
    expect(screen.getByTestId("tab-password")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Password Policy" })).toHaveAttribute("aria-selected", "true")
  })

  it("activates the tab from the ?tab search param", () => {
    searchParamsRef.current = new URLSearchParams("tab=lockout")
    renderWithProviders(<SecurityPage />)
    expect(screen.getByTestId("tab-lockout")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Account Lockout" })).toHaveAttribute("aria-selected", "true")
  })
})
