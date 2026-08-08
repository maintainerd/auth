import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ServicePoliciesTab } from "./ServicePoliciesTab"
import type { Policy } from "@/services/api/policies/types"

const { navigateMock, useServicePoliciesMock, removeMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    useServicePoliciesMock: vi.fn(),
    removeMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("../hooks/useServicePolicies", () => ({
  useServicePolicies: (...args: unknown[]) => useServicePoliciesMock(...args),
}))

vi.mock("../hooks/useServicePolicyMutations", () => ({
  useServicePolicyMutations: () => ({
    assignPolicy: { mutateAsync: vi.fn(), isPending: false },
    removePolicy: { mutateAsync: removeMutateAsync, isPending: false },
  }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

// PolicyAssignDialog is tested separately; stub it so the tab is isolated.
vi.mock("./PolicyAssignDialog", () => ({
  PolicyAssignDialog: () => null,
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

function response(rows: Policy[], total = rows.length) {
  return { rows, total, page: 1, limit: 10, total_pages: 1 }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

function getMenuButton() {
  return screen
    .getAllByRole("button", { name: /open menu/i })
    .find((el) => el.tagName === "BUTTON") as HTMLElement
}

describe("ServicePoliciesTab", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useServicePoliciesMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useServicePoliciesMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<ServicePoliciesTab serviceId="s1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useServicePoliciesMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ServicePoliciesTab serviceId="s1" />)
    expect(screen.getByText("Failed to load policies")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useServicePoliciesMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<ServicePoliciesTab serviceId="s1" />)
    expect(screen.getByText("No policies")).toBeInTheDocument()
  })

  it("renders a policy row and navigates on row click", async () => {
    const u = user()
    useServicePoliciesMock.mockReturnValue({ data: response([makePolicy()], 5), isLoading: false, isError: false })
    renderWithProviders(<ServicePoliciesTab serviceId="s1" />)
    expect(screen.getByText("Read only access")).toBeInTheDocument()
    expect(screen.getByText("Rows per page")).toBeInTheDocument()

    await u.click(screen.getByText("read-only"))
    expect(navigateMock).toHaveBeenCalledWith("/policies/p1", expect.objectContaining({
      state: expect.objectContaining({ from: "/services/s1" }),
    }))
  })

  it("shows the remove relationship action for system policies", async () => {
    const u = user()
    useServicePoliciesMock.mockReturnValue({
      data: response([makePolicy({ is_system: true })]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<ServicePoliciesTab serviceId="s1" />)
    await u.click(getMenuButton())
    expect(await screen.findByText("View Details")).toBeInTheDocument()
    expect(screen.getByText("Remove from Service")).toBeInTheDocument()
  })

  it("removes a policy from the service and shows success", async () => {
    const u = user()
    removeMutateAsync.mockResolvedValueOnce(undefined)
    useServicePoliciesMock.mockReturnValue({ data: response([makePolicy()]), isLoading: false, isError: false })
    renderWithProviders(<ServicePoliciesTab serviceId="s1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Remove from Service"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Remove" }))

    await waitFor(() => expect(removeMutateAsync).toHaveBeenCalledWith("p1"))
    expect(showSuccessMock).toHaveBeenCalledWith('Policy "read-only" removed successfully')
  })

  it("shows an error when removing a policy rejects", async () => {
    const u = user()
    const err = new Error("nope")
    removeMutateAsync.mockRejectedValueOnce(err)
    useServicePoliciesMock.mockReturnValue({ data: response([makePolicy()]), isLoading: false, isError: false })
    renderWithProviders(<ServicePoliciesTab serviceId="s1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Remove from Service"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Remove" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })
})
