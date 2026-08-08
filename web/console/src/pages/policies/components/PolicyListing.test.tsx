import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { PolicyListing } from "./PolicyListing"
import type { Policy } from "@/services/api/policies/types"

const { usePoliciesListMock, navigateMock } = vi.hoisted(() => ({
  usePoliciesListMock: vi.fn(),
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

vi.mock("@/hooks/usePolicies", () => ({
  usePoliciesList: (...args: unknown[]) => usePoliciesListMock(...args),
  useDeletePolicy: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdatePolicyStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makePolicy(overrides: Partial<Policy> = {}): Policy {
  return {
    policy_id: "p1",
    name: "read-only",
    description: "Read only access",
    version: "1.0.0",
    status: "active",
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function setPolicies(overrides: Record<string, unknown> = {}) {
  usePoliciesListMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...overrides,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("PolicyListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setPolicies()
  })

  it("renders the loading state", () => {
    setPolicies({ isLoading: true })
    const { container } = renderWithProviders(<PolicyListing />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders rows and navigates on row click", async () => {
    setPolicies({ data: { rows: [makePolicy({ policy_id: "p9", name: "admin-policy" })], total: 1 } })
    renderWithProviders(<PolicyListing />)
    const cell = screen.getByText("admin-policy")
    await u().click(cell)
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/policies/p9"))
  })

  it("clicking New Policy navigates to the create route", async () => {
    renderWithProviders(<PolicyListing />)
    const buttons = screen.getAllByRole("button", { name: /new policy/i })
    await u().click(buttons[0])
    expect(navigateMock).toHaveBeenCalledWith("/policies/create")
  })
})
