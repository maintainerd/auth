import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, fireEvent } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { AssignUserRolesDialog } from "./AssignUserRolesDialog"
import type { Role, RoleListResponse } from "@/services/api/roles/types"

const { useRolesMock, assignMutateAsync, assignState, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    useRolesMock: vi.fn(),
    assignMutateAsync: vi.fn(),
    assignState: { isPending: false },
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("@/hooks/useRoles", () => ({
  useRoles: (...args: unknown[]) => useRolesMock(...args),
}))

vi.mock("@/hooks/useUsers", () => ({
  useAssignUserRoles: () => ({ mutateAsync: assignMutateAsync, isPending: assignState.isPending }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeRole(overrides: Partial<Role> = {}): Role {
  return {
    role_id: "r1",
    name: "Admin",
    description: "Administrator role",
    is_default: false,
    is_system: false,
    status: "active",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function response(rows: Role[]): RoleListResponse {
  return { rows, total: rows.length, page: 1, limit: 100, total_pages: 1 }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

const noop = () => {}

describe("AssignUserRolesDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    assignState.isPending = false
    useRolesMock.mockReturnValue({ data: undefined, isLoading: false })
  })

  it("shows the 'Assigning...' label while the mutation is pending", () => {
    assignState.isPending = true
    useRolesMock.mockReturnValue({ data: response([makeRole()]), isLoading: false })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )
    expect(screen.getByText("Assigning...")).toBeInTheDocument()
  })

  it("disables the roles query when closed", () => {
    renderWithProviders(
      <AssignUserRolesDialog open={false} onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )
    expect(useRolesMock).toHaveBeenCalledWith(expect.anything(), { enabled: false })
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  })

  it("shows the loading state while roles load", () => {
    useRolesMock.mockReturnValue({ data: undefined, isLoading: true })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )
    expect(useRolesMock).toHaveBeenCalledWith(expect.anything(), { enabled: true })
    expect(screen.getByText("Loading roles...")).toBeInTheDocument()
  })

  it("shows the 'all assigned' empty state when every role is already assigned", () => {
    useRolesMock.mockReturnValue({ data: response([makeRole()]), isLoading: false })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={["r1"]} />,
    )
    expect(screen.getByText("All available roles are already assigned")).toBeInTheDocument()
  })

  it("filters by name and description and shows a no-match state, with null-description guard", async () => {
    const u = user()
    useRolesMock.mockReturnValue({
      data: response([
        makeRole({ role_id: "r1", name: "Admin", description: "Full access" }),
        makeRole({ role_id: "r2", name: "Editor", description: "Can write content" }),
        makeRole({ role_id: "r3", name: "Viewer", description: null as unknown as string }),
      ]),
      isLoading: false,
    })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )

    const search = screen.getByPlaceholderText("Search roles...")
    // Matches a description.
    await u.type(search, "write")
    expect(screen.getByText("Editor")).toBeInTheDocument()
    expect(screen.queryByText("Admin")).not.toBeInTheDocument()

    // No match.
    await u.clear(search)
    await u.type(search, "zzz")
    expect(screen.getByText("No roles found matching your search")).toBeInTheDocument()

    // Matching the role with a null description by its name still works.
    await u.clear(search)
    await u.type(search, "Viewer")
    expect(screen.getByText("Viewer")).toBeInTheDocument()
  })

  it("excludes already-assigned roles from the list", () => {
    useRolesMock.mockReturnValue({
      data: response([
        makeRole({ role_id: "r1", name: "Admin" }),
        makeRole({ role_id: "r2", name: "Editor" }),
      ]),
      isLoading: false,
    })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={["r1"]} />,
    )
    expect(screen.queryByText("Admin")).not.toBeInTheDocument()
    expect(screen.getByText("Editor")).toBeInTheDocument()
  })

  it("renders system and default badges on roles", () => {
    useRolesMock.mockReturnValue({
      data: response([makeRole({ is_system: true, is_default: true })]),
      isLoading: false,
    })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText("Default")).toBeInTheDocument()
  })

  it("selects and deselects a role", async () => {
    const u = user()
    useRolesMock.mockReturnValue({ data: response([makeRole()]), isLoading: false })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )
    const checkbox = screen.getByRole("checkbox")
    await u.click(checkbox)
    expect(screen.getByText("1 role selected")).toBeInTheDocument()
    await u.click(checkbox)
    expect(screen.queryByText("1 role selected")).not.toBeInTheDocument()
  })

  it("toggles select-all and deselect-all", async () => {
    const u = user()
    useRolesMock.mockReturnValue({
      data: response([
        makeRole({ role_id: "r1", name: "Admin" }),
        makeRole({ role_id: "r2", name: "Editor" }),
      ]),
      isLoading: false,
    })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )
    await u.click(screen.getByRole("button", { name: "Select All" }))
    expect(screen.getByText("2 roles selected")).toBeInTheDocument()
    await u.click(screen.getByRole("button", { name: "Deselect All" }))
    expect(screen.queryByText(/roles selected/)).not.toBeInTheDocument()
  })

  it("keeps the Assign button disabled and the mutation idle with no selection", () => {
    useRolesMock.mockReturnValue({ data: response([makeRole()]), isLoading: false })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )
    expect(screen.getByRole("button", { name: /assign roles/i })).toBeDisabled()
    expect(assignMutateAsync).not.toHaveBeenCalled()
  })

  it("shows an error when the form is submitted with no roles selected (guard clause)", () => {
    useRolesMock.mockReturnValue({ data: response([makeRole()]), isLoading: false })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={noop} userId="u1" existingRoleIds={[]} />,
    )
    const form = document.querySelector("form")!
    fireEvent.submit(form)
    expect(showErrorMock).toHaveBeenCalledWith("Please select at least one role")
    expect(assignMutateAsync).not.toHaveBeenCalled()
  })

  it("assigns the selected roles and shows success", async () => {
    const u = user()
    assignMutateAsync.mockResolvedValueOnce(undefined)
    const onOpenChange = vi.fn()
    useRolesMock.mockReturnValue({ data: response([makeRole()]), isLoading: false })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={onOpenChange} userId="u1" existingRoleIds={[]} />,
    )
    await u.click(screen.getByRole("checkbox"))
    await u.click(screen.getByRole("button", { name: /assign roles/i }))

    await waitFor(() =>
      expect(assignMutateAsync).toHaveBeenCalledWith({
        userId: "u1",
        data: { role_ids: ["r1"] },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("1 role assigned successfully")
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it("assigns multiple roles and shows the plural success message", async () => {
    const u = user()
    assignMutateAsync.mockResolvedValueOnce(undefined)
    const onOpenChange = vi.fn()
    useRolesMock.mockReturnValue({
      data: response([
        makeRole({ role_id: "r1", name: "Admin" }),
        makeRole({ role_id: "r2", name: "Editor" }),
      ]),
      isLoading: false,
    })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={onOpenChange} userId="u1" existingRoleIds={[]} />,
    )
    await u.click(screen.getByRole("button", { name: "Select All" }))
    await u.click(screen.getByRole("button", { name: /assign roles/i }))

    await waitFor(() =>
      expect(assignMutateAsync).toHaveBeenCalledWith({
        userId: "u1",
        data: { role_ids: ["r1", "r2"] },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("2 roles assigned successfully")
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it("shows an error when assigning rejects", async () => {
    const u = user()
    const err = new Error("nope")
    assignMutateAsync.mockRejectedValueOnce(err)
    useRolesMock.mockReturnValue({
      data: response([
        makeRole({ role_id: "r1", name: "Admin" }),
        makeRole({ role_id: "r2", name: "Editor" }),
      ]),
      isLoading: false,
    })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={vi.fn()} userId="u1" existingRoleIds={[]} />,
    )
    // Select all -> two roles -> exercises the plural success message branch too.
    await u.click(screen.getByRole("button", { name: "Select All" }))
    await u.click(screen.getByRole("button", { name: /assign roles/i }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("cancel calls onOpenChange(false)", async () => {
    const u = user()
    const onOpenChange = vi.fn()
    useRolesMock.mockReturnValue({ data: response([makeRole()]), isLoading: false })
    renderWithProviders(
      <AssignUserRolesDialog open onOpenChange={onOpenChange} userId="u1" existingRoleIds={[]} />,
    )
    await u.click(screen.getByRole("button", { name: "Cancel" }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
