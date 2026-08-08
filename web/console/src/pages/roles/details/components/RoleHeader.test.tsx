import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { RoleHeader } from "./RoleHeader"
import type { Role } from "@/services/api/roles/types"

const { navigateMock, deleteMutateAsync, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  deleteMutateAsync: vi.fn(),
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

vi.mock("@/hooks/useRoles", () => ({
  useDeleteRole: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateRoleStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeRole(overrides: Partial<Role> = {}): Role {
  return {
    role_id: "r1", name: "admin", description: "Admin role",
    is_default: false, is_system: false, status: "active",
    created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("RoleHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the role name and status badge", () => {
    renderWithProviders(<RoleHeader role={makeRole()} roleId="r1" />)
    expect(screen.getByText("admin")).toBeInTheDocument()
    expect(screen.getByText("Admin role")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
  })

  it("shows system and default badges", () => {
    renderWithProviders(<RoleHeader role={makeRole({ is_system: true, is_default: true })} roleId="r1" />)
    expect(screen.getByText("System Role")).toBeInTheDocument()
    expect(screen.getByText("Default role assigned to new users")).toBeInTheDocument()
  })

  it("navigates to edit with router state", async () => {
    renderWithProviders(<RoleHeader role={makeRole()} roleId="r1" />)
    await u().click(screen.getByRole("button", { name: /edit/i }))
    expect(navigateMock).toHaveBeenCalledWith("/roles/r1/edit", expect.objectContaining({
      state: expect.objectContaining({ from: "/roles/r1" }),
    }))
  })

  it("deletes the role, shows success, and navigates back", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<RoleHeader role={makeRole({ name: "admin" })} roleId="r1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete Role"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "admin")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("r1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Role deleted successfully")
    expect(navigateMock).toHaveBeenCalledWith("/roles")
  })

  it("shows an error and does not navigate when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<RoleHeader role={makeRole({ name: "admin" })} roleId="r1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete Role"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "admin")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("hides the actions menu for system roles", () => {
    renderWithProviders(<RoleHeader role={makeRole({ is_system: true })} roleId="r1" />)
    expect(screen.queryByRole("button", { name: /open actions/i })).not.toBeInTheDocument()
  })

  it("hides the edit button for system roles", () => {
    // The role form is permanently disabled for system roles, so offering Edit
    // only ever navigates to a dead end — ApiHeader/PolicyHeader/ServiceHeader
    // all hide the whole actions block instead.
    renderWithProviders(<RoleHeader role={makeRole({ is_system: true })} roleId="r1" />)
    expect(screen.queryByRole("button", { name: /edit/i })).not.toBeInTheDocument()
  })

  it("keeps the edit button for a non-system default role", () => {
    // is_default only blocks deactivate/delete — the role is still editable, so
    // the actions block stays even though the dropdown has nothing to offer.
    renderWithProviders(<RoleHeader role={makeRole({ is_default: true })} roleId="r1" />)
    expect(screen.getByRole("button", { name: /edit/i })).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /open actions/i })).not.toBeInTheDocument()
  })

  it("offers activate for an inactive non-system role", async () => {
    renderWithProviders(<RoleHeader role={makeRole({ status: "inactive" })} roleId="r1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    expect(await screen.findByText("Activate Role")).toBeInTheDocument()
  })
})
