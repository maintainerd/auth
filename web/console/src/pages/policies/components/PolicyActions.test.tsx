import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { PolicyActions } from "./PolicyActions"
import type { Policy } from "@/services/api/policies/types"

const { navigateMock, updateStatusMutateAsync, deleteMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    updateStatusMutateAsync: vi.fn(),
    deleteMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
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
  useDeletePolicy: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdatePolicyStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makePolicy(overrides: Partial<Policy> = {}): Policy {
  return {
    policy_id: "p1",
    name: "read-only",
    description: "Read only access",
    version: "1.0.0",
    status: "active",
    is_system: false,
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("PolicyActions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("navigates to view details", async () => {
    renderWithProviders(<PolicyActions policy={makePolicy()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/policies/p1")
  })

  it("navigates to edit", async () => {
    renderWithProviders(<PolicyActions policy={makePolicy()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Edit Policy"))
    expect(navigateMock).toHaveBeenCalledWith("/policies/p1/edit")
  })

  it("deactivates an active policy with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<PolicyActions policy={makePolicy({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Policy"))
    await u().click(screen.getByRole("button", { name: "Deactivate" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        policyId: "p1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Policy status updated to inactive")
  })

  it("activates an inactive policy with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<PolicyActions policy={makePolicy({ status: "inactive" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Activate Policy"))
    await u().click(screen.getByRole("button", { name: "Activate" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        policyId: "p1",
        data: { status: "active" },
      }),
    )
  })

  it("shows error when status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<PolicyActions policy={makePolicy({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Policy"))
    await u().click(screen.getByRole("button", { name: "Deactivate" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes a non-system policy with confirmation", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<PolicyActions policy={makePolicy({ name: "read-only" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Policy"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "read-only")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("p1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Policy deleted successfully")
  })

  it("shows error when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<PolicyActions policy={makePolicy({ name: "read-only" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Policy"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "read-only")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("hides edit, status, and delete actions for system policies", async () => {
    renderWithProviders(<PolicyActions policy={makePolicy({ is_system: true })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    expect(await screen.findByText("View Details")).toBeInTheDocument()
    expect(screen.queryByText("Edit Policy")).not.toBeInTheDocument()
    expect(screen.queryByText("Deactivate Policy")).not.toBeInTheDocument()
    expect(screen.queryByText("Delete Policy")).not.toBeInTheDocument()
  })
})
