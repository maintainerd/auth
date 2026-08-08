import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import ApisResourcesPage from "./ApisResourcesPage"

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

vi.mock("@/pages/services/components/ServiceListing", () => ({
  ServiceListing: () => <div data-testid="tab-services" />,
}))
vi.mock("@/pages/apis/components/ApiListing", () => ({
  ApiListing: () => <div data-testid="tab-apis" />,
}))
vi.mock("@/pages/policies/components/PolicyListing", () => ({
  PolicyListing: () => <div data-testid="tab-policies" />,
}))

describe("ApisResourcesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
  })

  it("renders the page title and tab triggers", () => {
    renderWithProviders(<ApisResourcesPage />)
    expect(screen.getByRole("heading", { name: "APIs & Resources" })).toBeInTheDocument()
    for (const label of ["Services", "APIs", "Policies"]) {
      expect(screen.getByRole("tab", { name: label })).toBeInTheDocument()
    }
  })

  it("defaults to services", () => {
    renderWithProviders(<ApisResourcesPage />)
    expect(screen.getByTestId("tab-services")).toBeInTheDocument()
  })

  it("uses the route default tab", () => {
    renderWithProviders(<ApisResourcesPage defaultTab="policies" />)
    expect(screen.getByTestId("tab-policies")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Policies" })).toHaveAttribute("aria-selected", "true")
  })

  it("activates the tab from the query param", () => {
    searchParamsRef.current = new URLSearchParams("tab=apis")
    renderWithProviders(<ApisResourcesPage />)
    expect(screen.getByTestId("tab-apis")).toBeInTheDocument()
  })
})
