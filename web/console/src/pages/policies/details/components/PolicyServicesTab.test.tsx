import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { PolicyServicesTab } from "./PolicyServicesTab"
import type { Service } from "@/services/api/services/types"

const { navigateMock, useServicesByPolicyMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  useServicesByPolicyMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("../hooks/useServicesByPolicy", () => ({
  useServicesByPolicy: (...args: unknown[]) => useServicesByPolicyMock(...args),
}))

function makeService(overrides: Partial<Service> = {}): Service {
  return {
    service_id: "s1",
    name: "auth-service",
    display_name: "Auth Service",
    description: "Handles authentication",
    version: "1.0.0",
    status: "active",
    is_system: false,
    api_count: 0,
    policy_count: 0,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function response(rows: Service[], total = rows.length) {
  return { rows, total, page: 1, limit: 10, total_pages: 1 }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("PolicyServicesTab", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useServicesByPolicyMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useServicesByPolicyMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<PolicyServicesTab policyId="p1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useServicesByPolicyMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<PolicyServicesTab policyId="p1" />)
    expect(screen.getByText("Failed to load services")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useServicesByPolicyMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<PolicyServicesTab policyId="p1" />)
    expect(screen.getByText("No services")).toBeInTheDocument()
  })

  it("renders a service row with badges and pagination", () => {
    useServicesByPolicyMock.mockReturnValue({
      data: response([makeService({ is_system: true })], 5),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<PolicyServicesTab policyId="p1" />)
    expect(screen.getByText("Auth Service")).toBeInTheDocument()
    expect(screen.getByText("auth-service")).toBeInTheDocument()
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText("Rows per page")).toBeInTheDocument()
  })

  it("navigates to the service with back-state when the row is clicked", async () => {
    const u = user()
    useServicesByPolicyMock.mockReturnValue({ data: response([makeService()]), isLoading: false, isError: false })
    renderWithProviders(<PolicyServicesTab policyId="p1" />)
    await u.click(screen.getByText("Auth Service"))
    expect(navigateMock).toHaveBeenCalledWith("/services/s1", expect.objectContaining({
      state: expect.objectContaining({ from: "/policies/p1" }),
    }))
  })
})
