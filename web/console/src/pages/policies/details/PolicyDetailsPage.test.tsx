import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import PolicyDetailsPage from "./PolicyDetailsPage"
import type { PolicyDetail } from "@/services/api/policies/types"

const { usePolicyMock, useServicesByPolicyMock, navigateMock, searchParamsRef, locationRef } =
  vi.hoisted(() => ({
    usePolicyMock: vi.fn(),
    useServicesByPolicyMock: vi.fn(),
    navigateMock: vi.fn(),
    searchParamsRef: { current: new URLSearchParams() },
    locationRef: { current: { state: null as unknown } },
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({ policyId: "p1" }),
    useNavigate: () => navigateMock,
    useLocation: () => locationRef.current,
    useSearchParams: () => [searchParamsRef.current, vi.fn()],
  }
})

vi.mock("@/hooks/usePolicies", () => ({
  usePolicy: (...args: unknown[]) => usePolicyMock(...args),
}))

vi.mock("./hooks/useServicesByPolicy", () => ({
  useServicesByPolicy: (...args: unknown[]) => useServicesByPolicyMock(...args),
}))

vi.mock("./components", () => ({
  PolicyHeader: ({ policy }: { policy: PolicyDetail }) => (
    <div data-testid="policy-header">{policy.name}</div>
  ),
  PolicyStatementsTab: () => <div data-testid="statements-tab" />,
  PolicyServicesTab: () => <div data-testid="services-tab" />,
  PolicyHistoryTab: () => <div data-testid="history-tab" />,
}))

function makePolicy(overrides: Partial<PolicyDetail> = {}): PolicyDetail {
  return {
    policy_id: "p1",
    name: "read-only",
    description: "Read only access",
    document: { version: "2012-10-17", statement: [] },
    version: "1.0.0",
    status: "active",
    is_system: false,
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

describe("PolicyDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
    locationRef.current = { state: null }
    usePolicyMock.mockReturnValue({ data: undefined, isLoading: false })
    useServicesByPolicyMock.mockReturnValue({ data: undefined })
  })

  it("renders the loading state", () => {
    usePolicyMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<PolicyDetailsPage />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders the not-found state", () => {
    usePolicyMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<PolicyDetailsPage />)
    expect(screen.getByText("Policy not found")).toBeInTheDocument()
  })

  it("renders the policy header and tabs with the services count", () => {
    usePolicyMock.mockReturnValue({ data: makePolicy(), isLoading: false })
    useServicesByPolicyMock.mockReturnValue({ data: { total: 3 } })
    renderWithProviders(<PolicyDetailsPage />)
    expect(screen.getByTestId("policy-header")).toHaveTextContent("read-only")
    expect(screen.getByRole("tab", { name: "Statements" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: /Services \(3\)/ })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "History" })).toBeInTheDocument()
  })

  it("falls back to the statements tab for an unknown ?tab value", () => {
    searchParamsRef.current = new URLSearchParams("tab=bogus")
    usePolicyMock.mockReturnValue({ data: makePolicy(), isLoading: false })
    renderWithProviders(<PolicyDetailsPage />)
    expect(screen.getByTestId("statements-tab")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Statements" })).toHaveAttribute("aria-selected", "true")
  })

  it("honours location.state for the back label", () => {
    locationRef.current = { state: { from: "/services/s1", backLabel: "Back to Service Details" } }
    usePolicyMock.mockReturnValue({ data: makePolicy(), isLoading: false })
    renderWithProviders(<PolicyDetailsPage />)
    expect(screen.getByRole("button", { name: /back to service details/i })).toBeInTheDocument()
  })
})
