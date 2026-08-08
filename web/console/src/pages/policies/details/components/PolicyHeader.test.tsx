import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { PolicyHeader } from "./PolicyHeader"
import type { PolicyDetail } from "@/services/api/policies/types"

const { navigateMock, deleteMutateAsync, updateStatusMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    deleteMutateAsync: vi.fn(),
    updateStatusMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
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

function makePolicy(overrides: Partial<PolicyDetail> = {}): PolicyDetail {
  return {
    policy_id: "p1",
    name: "read-only",
    description: "Read only access",
    document: { version: "2012-10-17", statement: [] },
    version: "1.0.0",
    status: "active",
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("PolicyHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the name, description, badges, and attributes", () => {
    renderWithProviders(<PolicyHeader policy={makePolicy()} policyId="p1" />)
    expect(screen.getByText("read-only")).toBeInTheDocument()
    expect(screen.getByText("Read only access")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
    expect(screen.getByText("Custom Policy")).toBeInTheDocument()
    expect(screen.getByText("1.0.0")).toBeInTheDocument()
    expect(screen.getByText("2012-10-17")).toBeInTheDocument()
  })

  it("navigates to edit with router state", async () => {
    renderWithProviders(<PolicyHeader policy={makePolicy()} policyId="p1" />)
    await u().click(screen.getByRole("button", { name: /edit/i }))
    expect(navigateMock).toHaveBeenCalledWith("/policies/p1/edit", expect.objectContaining({
      state: expect.objectContaining({ from: "/policies/p1" }),
    }))
  })

  it("offers the other status in the actions menu and confirms a change", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<PolicyHeader policy={makePolicy({ status: "active" })} policyId="p1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    expect(await screen.findByText("Deactivate Policy")).toBeInTheDocument()
    expect(screen.queryByText("Activate Policy")).not.toBeInTheDocument()

    await u().click(screen.getByText("Deactivate Policy"))
    await u().click(screen.getByRole("button", { name: "Deactivate Policy" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        policyId: "p1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Policy status updated to inactive")
  })

  it("shows an error when the status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<PolicyHeader policy={makePolicy({ status: "inactive" })} policyId="p1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Activate Policy"))
    await u().click(screen.getByRole("button", { name: "Activate Policy" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes the policy and navigates to the back target", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(
      <PolicyHeader policy={makePolicy({ name: "read-only" })} policyId="p1" afterDeleteTo="/services/s1" />,
    )
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete Policy"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "read-only")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("p1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Policy deleted successfully")
    expect(navigateMock).toHaveBeenCalledWith("/services/s1")
  })

  it("shows an error and does not navigate when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<PolicyHeader policy={makePolicy({ name: "read-only" })} policyId="p1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete Policy"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "read-only")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("hides edit and the actions menu for system policies", () => {
    renderWithProviders(<PolicyHeader policy={makePolicy({ is_system: true })} policyId="p1" />)
    expect(screen.getByText("System Policy")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /edit/i })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /open actions/i })).not.toBeInTheDocument()
  })
})
