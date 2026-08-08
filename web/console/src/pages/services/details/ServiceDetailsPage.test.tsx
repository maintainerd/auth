import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import ServiceDetailsPage from "./ServiceDetailsPage"
import type { ServiceResponse } from "@/services/api/services/types"

const { useServiceMock, navigateMock, searchParamsRef } = vi.hoisted(() => ({
  useServiceMock: vi.fn(),
  navigateMock: vi.fn(),
  searchParamsRef: { current: new URLSearchParams() },
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({ serviceId: "s1" }),
    useNavigate: () => navigateMock,
    useSearchParams: () => [searchParamsRef.current, vi.fn()],
  }
})

vi.mock("@/hooks/useServices", () => ({
  useService: (...args: unknown[]) => useServiceMock(...args),
}))

vi.mock("./components", () => ({
  ServiceHeader: ({ service }: { service: ServiceResponse }) => (
    <div data-testid="service-header">{service.display_name}</div>
  ),
  ServiceApisTab: () => <div data-testid="apis-tab" />,
  ServicePoliciesTab: () => <div data-testid="policies-tab" />,
}))

function makeService(overrides: Partial<ServiceResponse> = {}): ServiceResponse {
  return {
    service_id: "s1", name: "auth-service", display_name: "Auth Service",
    description: "Handles authentication", version: "1.0.0", status: "active",
    is_system: false, api_count: 0, policy_count: 0,
    created_at: "", updated_at: "", ...overrides,
  }
}

describe("ServiceDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
    useServiceMock.mockReturnValue({ data: undefined, isLoading: false })
  })

  it("renders the loading state", () => {
    useServiceMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<ServiceDetailsPage />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders the not-found state", () => {
    useServiceMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ServiceDetailsPage />)
    expect(screen.getByText("Service not found")).toBeInTheDocument()
  })

  it("renders the service header with display name", () => {
    useServiceMock.mockReturnValue({ data: makeService(), isLoading: false })
    renderWithProviders(<ServiceDetailsPage />)
    expect(screen.getByTestId("service-header")).toHaveTextContent("Auth Service")
  })

  it("renders the APIs and Policies tabs", () => {
    useServiceMock.mockReturnValue({ data: makeService(), isLoading: false })
    renderWithProviders(<ServiceDetailsPage />)
    expect(screen.getByRole("tab", { name: "APIs" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Policies" })).toBeInTheDocument()
  })

  it("falls back to the APIs tab for an unknown ?tab value", () => {
    searchParamsRef.current = new URLSearchParams("tab=bogus")
    useServiceMock.mockReturnValue({ data: makeService(), isLoading: false })
    renderWithProviders(<ServiceDetailsPage />)
    expect(screen.getByTestId("apis-tab")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "APIs" })).toHaveAttribute("aria-selected", "true")
  })

  it("activates the tab from the ?tab search param", () => {
    searchParamsRef.current = new URLSearchParams("tab=policies")
    useServiceMock.mockReturnValue({ data: makeService(), isLoading: false })
    renderWithProviders(<ServiceDetailsPage />)
    expect(screen.getByTestId("policies-tab")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Policies" })).toHaveAttribute("aria-selected", "true")
  })
})
