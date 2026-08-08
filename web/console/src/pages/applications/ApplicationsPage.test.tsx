import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import ApplicationsPage from "./ApplicationsPage"

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

vi.mock("@/pages/clients/components/ClientListing", () => ({
  ClientListing: () => <div data-testid="tab-clients" />,
}))
vi.mock("@/pages/workload-identity/components/WorkloadIdentityListing", () => ({
  WorkloadIdentityListing: () => <div data-testid="tab-workload-identity" />,
}))

describe("ApplicationsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
  })

  it("renders the page title and tab triggers", () => {
    renderWithProviders(<ApplicationsPage />)
    expect(screen.getByRole("heading", { name: "Applications" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Clients" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Workload Identity" })).toBeInTheDocument()
  })

  it("defaults to clients", () => {
    renderWithProviders(<ApplicationsPage />)
    expect(screen.getByTestId("tab-clients")).toBeInTheDocument()
  })

  it("uses the route default tab", () => {
    renderWithProviders(<ApplicationsPage defaultTab="workload-identity" />)
    expect(screen.getByTestId("tab-workload-identity")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Workload Identity" })).toHaveAttribute("aria-selected", "true")
  })

  it("activates the tab from the query param", () => {
    searchParamsRef.current = new URLSearchParams("tab=workload-identity")
    renderWithProviders(<ApplicationsPage />)
    expect(screen.getByTestId("tab-workload-identity")).toBeInTheDocument()
  })
})
