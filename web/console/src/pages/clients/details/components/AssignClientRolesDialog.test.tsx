import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { AssignClientRolesDialog } from "./AssignClientRolesDialog"
import type { Role } from "@/services/api/roles/types"

const { useRolesMock, addRoleMutateAsync, addRoleState, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    useRolesMock: vi.fn(),
    addRoleMutateAsync: vi.fn(),
    addRoleState: { isPending: false },
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("@/hooks/useRoles", () => ({
  useRoles: (...args: unknown[]) => useRolesMock(...args),
}))

vi.mock("@/hooks/useClients", () => ({
  useAddClientRole: () => ({ mutateAsync: addRoleMutateAsync, isPending: addRoleState.isPending }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeRole(overrides: Partial<Role> = {}): Role {
  return {
    role_id: "r1",
    name: "admin",
    description: "Administrator role",
    is_default: false,
    is_system: false,
    status: "active",
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

function setRoles(rows: Role[], overrides: Record<string, unknown> = {}) {
  useRolesMock.mockReturnValue({ data: { rows, total: rows.length }, isLoading: false, ...overrides })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

const baseProps = {
  open: true,
  onOpenChange: vi.fn(),
  clientId: "c1",
  existingRoleIds: [] as string[],
}

describe("AssignClientRolesDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    addRoleState.isPending = false
    setRoles([])
  })

  it("disables the roles query while closed", () => {
    setRoles([])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} open={false} />)
    expect(useRolesMock).toHaveBeenCalledWith(expect.anything(), { enabled: false })
  })

  it("shows the loading state while roles load", () => {
    useRolesMock.mockReturnValue({ data: undefined, isLoading: true })
    renderWithProviders(<AssignClientRolesDialog {...baseProps} />)
    expect(screen.getByText("Loading roles...")).toBeInTheDocument()
  })

  it("shows the all-assigned empty state and excludes assigned roles", () => {
    setRoles([makeRole()])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} existingRoleIds={["r1"]} />)
    expect(screen.getByText("All available roles are already assigned")).toBeInTheDocument()
    expect(screen.queryByText("admin")).not.toBeInTheDocument()
  })

  it("filters by search and shows the no-match state", async () => {
    setRoles([makeRole()])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} />)
    await u().type(screen.getByPlaceholderText("Search roles..."), "zzz")
    expect(screen.getByText("No roles found matching your search")).toBeInTheDocument()
  })

  it("renders system and default badges on roles", () => {
    setRoles([makeRole({ is_system: true, is_default: true })])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} />)
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText("Default")).toBeInTheDocument()
  })

  it("toggles select-all and deselect-all", async () => {
    setRoles([makeRole(), makeRole({ role_id: "r2", name: "viewer" })])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} />)
    await u().click(screen.getByRole("button", { name: "Select All" }))
    expect(screen.getByText("2 roles selected")).toBeInTheDocument()
    await u().click(screen.getByRole("button", { name: "Deselect All" }))
    expect(screen.queryByText("2 roles selected")).not.toBeInTheDocument()
  })

  it("keeps the Assign button disabled with no selection", () => {
    setRoles([makeRole()])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} />)
    expect(screen.getByRole("button", { name: /assign roles/i })).toBeDisabled()
  })

  it("assigns the selected roles and closes on success", async () => {
    addRoleMutateAsync.mockResolvedValue(undefined)
    const onOpenChange = vi.fn()
    setRoles([makeRole()])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} onOpenChange={onOpenChange} />)

    await u().click(screen.getByRole("checkbox"))
    await u().click(screen.getByRole("button", { name: /assign roles/i }))

    await waitFor(() => expect(addRoleMutateAsync).toHaveBeenCalledWith("r1"))
    expect(showSuccessMock).toHaveBeenCalledWith("1 role assigned successfully")
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it("assigns multiple roles and shows the plural success message", async () => {
    addRoleMutateAsync.mockResolvedValue(undefined)
    setRoles([makeRole(), makeRole({ role_id: "r2", name: "viewer" })])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} />)

    await u().click(screen.getByRole("button", { name: "Select All" }))
    await u().click(screen.getByRole("button", { name: /assign roles/i }))

    await waitFor(() => expect(addRoleMutateAsync).toHaveBeenCalledTimes(2))
    expect(showSuccessMock).toHaveBeenCalledWith("2 roles assigned successfully")
  })

  it("shows an error when assigning rejects", async () => {
    const err = new Error("fail")
    addRoleMutateAsync.mockRejectedValueOnce(err)
    setRoles([makeRole()])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} />)

    await u().click(screen.getByRole("checkbox"))
    await u().click(screen.getByRole("button", { name: /assign roles/i }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("shows the Assigning... label while the mutation is pending", async () => {
    addRoleState.isPending = true
    setRoles([makeRole()])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} />)
    expect(screen.getByText("Assigning...")).toBeInTheDocument()
  })

  it("cancel calls onOpenChange(false)", async () => {
    const onOpenChange = vi.fn()
    setRoles([makeRole()])
    renderWithProviders(<AssignClientRolesDialog {...baseProps} onOpenChange={onOpenChange} />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
