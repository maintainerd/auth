import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { RegistrationFlowListing } from "./RegistrationFlowListing"
import type { RegistrationFlow } from "@/services/api/registration-flows/types"

const { useListMock, navigateMock } = vi.hoisted(() => ({
  useListMock: vi.fn(),
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

vi.mock("@/hooks/useRegistrationFlows", () => ({
  useRegistrationFlowsList: (...args: unknown[]) => useListMock(...args),
  useDeleteRegistrationFlow: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateRegistrationFlowStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeFlow(overrides: Partial<RegistrationFlow> = {}): RegistrationFlow {
  return {
    registration_flow_id: "f1",
    name: "Seller Signup",
    description: "Sellers onboard here",
    status: "active",
    client_id: "c1",
    verification_required: false,
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function setFlows(overrides: Record<string, unknown> = {}) {
  useListMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...overrides,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("RegistrationFlowListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setFlows()
  })

  it("sorts by the backend column name, not a display label", () => {
    renderWithProviders(<RegistrationFlowListing />)
    // "Created" is not in the backend's sort allow-list and was silently ignored.
    expect(useListMock).toHaveBeenCalledWith(
      expect.objectContaining({ sort_by: "created_at", sort_order: "desc" }),
    )
  })

  it("maps free-text search onto the backend's `search` filter (name + identifier)", async () => {
    renderWithProviders(<RegistrationFlowListing />)

    await u().type(screen.getByPlaceholderText(/search registration flows/i), "seller")

    await waitFor(() =>
      expect(useListMock).toHaveBeenCalledWith(expect.objectContaining({ search: "seller" })),
    )
  })

  it("advertises identifier search in the placeholder", () => {
    renderWithProviders(<RegistrationFlowListing />)
    expect(screen.getByPlaceholderText(/name or description/i)).toBeInTheDocument()
  })

  it("renders the loading state", () => {
    setFlows({ isLoading: true })
    const { container } = renderWithProviders(<RegistrationFlowListing />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders rows and navigates on row click", async () => {
    setFlows({ data: { rows: [makeFlow({ registration_flow_id: "f9", name: "Buyer Signup" })], total: 1 } })
    renderWithProviders(<RegistrationFlowListing />)

    await u().click(screen.getByText("Buyer Signup"))

    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/registration-flows/f9"))
  })

  it("clicking New Registration Flow navigates to the create route", async () => {
    renderWithProviders(<RegistrationFlowListing />)
    const buttons = screen.getAllByRole("button", { name: /new registration flow/i })
    await u().click(buttons[0])
    expect(navigateMock).toHaveBeenCalledWith("/registration-flows/create")
  })

  it("offers status and type filters (no draft status)", async () => {
    renderWithProviders(<RegistrationFlowListing />)

    await u().click(screen.getByRole("button", { name: /filters/i }))

    expect(screen.getByLabelText("active")).toBeInTheDocument()
    expect(screen.getByLabelText("inactive")).toBeInTheDocument()
    expect(screen.queryByLabelText("draft")).not.toBeInTheDocument()
    expect(screen.getByLabelText("system")).toBeInTheDocument()
    expect(screen.getByLabelText("regular")).toBeInTheDocument()
  })
})
