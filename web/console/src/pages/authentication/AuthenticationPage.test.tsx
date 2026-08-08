import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import AuthenticationPage from "./AuthenticationPage"

const { searchParamsRef, navigateMock } = vi.hoisted(() => ({
  searchParamsRef: { current: new URLSearchParams() },
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return {
    ...actual,
    useNavigate: () => navigateMock,
    useSearchParams: () => [searchParamsRef.current, vi.fn()],
  }
})

vi.mock("@/pages/identity-providers/components/IdentityProviderListing", () => ({
  IdentityProviderListing: () => <div data-testid="tab-identity-providers" />,
}))
vi.mock("@/pages/registration-flows/components/RegistrationFlowListing", () => ({
  RegistrationFlowListing: () => <div data-testid="tab-registration-flows" />,
}))

describe("AuthenticationPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
  })

  it("renders the page title and tab triggers", () => {
    renderWithProviders(<AuthenticationPage />)
    expect(screen.getByRole("heading", { name: "Authentication" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Identity Providers" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Registration Flows" })).toBeInTheDocument()
  })

  it("defaults to identity providers", () => {
    renderWithProviders(<AuthenticationPage />)
    expect(screen.getByTestId("tab-identity-providers")).toBeInTheDocument()
  })

  it("uses the route default tab", () => {
    renderWithProviders(<AuthenticationPage defaultTab="registration-flows" />)
    expect(screen.getByTestId("tab-registration-flows")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Registration Flows" })).toHaveAttribute("aria-selected", "true")
  })

  it("activates the tab from the query param", () => {
    searchParamsRef.current = new URLSearchParams("tab=registration-flows")
    renderWithProviders(<AuthenticationPage />)
    expect(screen.getByTestId("tab-registration-flows")).toBeInTheDocument()
  })
})
