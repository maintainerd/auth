import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { RoleActions } from "./RoleActions"
import type { Role } from "@/services/api/roles/types"

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

vi.mock("@/hooks/useRoles", () => ({
  useDeleteRole: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateRoleStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeRole(overrides: Partial<Role> = {}): Role {
  return {
    role_id: "r1",
    name: "admin",
    description: "Admin role",
    is_default: false,
    is_system: false,
    status: "active",
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("RoleActions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("navigates to view details", async () => {
    renderWithProviders(<RoleActions role={makeRole()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/roles/r1")
  })

  it("navigates to edit", async () => {
    renderWithProviders(<RoleActions role={makeRole()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Edit Role"))
    expect(navigateMock).toHaveBeenCalledWith("/roles/r1/edit")
  })

  it("deactivates an active role with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<RoleActions role={makeRole({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Role"))
    await u().click(screen.getByRole("button", { name: "Deactivate" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        roleId: "r1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Role status updated to inactive")
  })

  it("activates an inactive role with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<RoleActions role={makeRole({ status: "inactive" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Activate Role"))
    await u().click(screen.getByRole("button", { name: "Activate" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        roleId: "r1",
        data: { status: "active" },
      }),
    )
  })

  it("shows error when status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<RoleActions role={makeRole({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Role"))
    await u().click(screen.getByRole("button", { name: "Deactivate" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes a non-system role with confirmation", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<RoleActions role={makeRole({ name: "admin" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Role"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "admin")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("r1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Role deleted successfully")
  })

  it("shows error when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<RoleActions role={makeRole({ name: "admin" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Role"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "admin")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("hides delete action for system roles", async () => {
    renderWithProviders(<RoleActions role={makeRole({ is_system: true })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    expect(screen.queryByText("Delete Role")).not.toBeInTheDocument()
  })
})
