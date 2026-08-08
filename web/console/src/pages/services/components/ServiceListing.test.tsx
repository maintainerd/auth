import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ServiceListing } from "./ServiceListing"
import type { Service } from "@/services/api/services/types"

const { useServicesListMock, navigateMock } = vi.hoisted(() => ({
  useServicesListMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({}),
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useServices", () => ({
  useServicesList: (...args: unknown[]) => useServicesListMock(...args),
  useDeleteService: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateServiceStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
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
    api_count: 2,
    policy_count: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function setServices(overrides: Record<string, unknown> = {}) {
  useServicesListMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...overrides,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ServiceListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setServices()
  })

  it("renders the loading state", () => {
    setServices({ isLoading: true })
    const { container } = renderWithProviders(<ServiceListing />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders rows and navigates on row click", async () => {
    setServices({ data: { rows: [makeService({ service_id: "s9", display_name: "Billing" })], total: 1 } })
    renderWithProviders(<ServiceListing />)
    const cell = screen.getByText("Billing")
    await u().click(cell)
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/services/s9"))
  })

  it("clicking New Service navigates to the create route", async () => {
    renderWithProviders(<ServiceListing />)
    const buttons = screen.getAllByRole("button", { name: /new service/i })
    await u().click(buttons[0])
    expect(navigateMock).toHaveBeenCalledWith("/services/create")
  })
})
