import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { IpRestrictionActions } from "./IpRestrictionActions"
import type { IpRestrictionRule } from "@/services/api/ip-restriction-rules/types"

const { updateStatusMutateAsync, deleteMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    updateStatusMutateAsync: vi.fn(),
    deleteMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("@/hooks/useIpRestrictionRules", () => ({
  useDeleteIpRestrictionRule: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateIpRestrictionRuleStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeRule(overrides: Partial<IpRestrictionRule> = {}): IpRestrictionRule {
  return {
    ipRestrictionRuleId: "r1",
    description: "Office VPN",
    type: "allow",
    ipAddress: "203.0.113.0/24",
    status: "active",
    createdAt: "",
    updatedAt: "",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("IpRestrictionActions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("opens the edit dialog via the onEdit callback", async () => {
    const onEdit = vi.fn()
    const rule = makeRule()
    renderWithProviders(<IpRestrictionActions rule={rule} onEdit={onEdit} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Edit Rule"))
    expect(onEdit).toHaveBeenCalledWith(rule)
  })

  it("deactivates an active rule with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<IpRestrictionActions rule={makeRule({ status: "active" })} onEdit={vi.fn()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Rule"))
    await u().click(screen.getByRole("button", { name: "Deactivate Rule" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        id: "r1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("IP rule deactivated successfully")
  })

  it("activates an inactive rule with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<IpRestrictionActions rule={makeRule({ status: "inactive" })} onEdit={vi.fn()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Activate Rule"))
    await u().click(screen.getByRole("button", { name: "Activate Rule" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        id: "r1",
        data: { status: "active" },
      }),
    )
  })

  it("shows error when status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<IpRestrictionActions rule={makeRule({ status: "active" })} onEdit={vi.fn()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Rule"))
    await u().click(screen.getByRole("button", { name: "Deactivate Rule" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes a rule with type-to-confirm", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<IpRestrictionActions rule={makeRule()} onEdit={vi.fn()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Rule"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "203.0.113.0/24")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("r1"))
    expect(showSuccessMock).toHaveBeenCalledWith("IP rule deleted successfully")
  })

  it("shows error when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<IpRestrictionActions rule={makeRule()} onEdit={vi.fn()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Rule"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "203.0.113.0/24")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })
})
