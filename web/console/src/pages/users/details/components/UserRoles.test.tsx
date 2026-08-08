import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { UserRoles } from "./UserRoles"
import type { UserRole, UserRolesResponse } from "@/services/api/users/types"

const { navigateMock, useUserRolesMock, removeRoleMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    useUserRolesMock: vi.fn(),
    removeRoleMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useUsers", () => ({
  useUserRoles: (...args: unknown[]) => useUserRolesMock(...args),
  useRemoveUserRole: () => ({ mutateAsync: removeRoleMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

// AssignUserRolesDialog is tested separately; stub it so UserRoles is isolated.
vi.mock("./AssignUserRolesDialog", () => ({
  AssignUserRolesDialog: () => null,
}))

function makeRole(overrides: Partial<UserRole> = {}): UserRole {
  return {
    role_id: "r1",
    name: "Admin",
    description: "Admin role",
    is_default: false,
    is_system: false,
    status: "active",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function response(rows: UserRole[], total = rows.length): UserRolesResponse {
  return { rows, total, page: 1, limit: 10, total_pages: 1 }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

// The row is a div[role="button"] whose accessible name also contains the
// menu trigger's "Open menu" text, so disambiguate to the real <button>.
function getMenuButton() {
  return screen
    .getAllByRole("button", { name: /open menu/i })
    .find((el) => el.tagName === "BUTTON") as HTMLElement
}

describe("UserRoles", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUserRolesMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useUserRolesMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<UserRoles userId="u1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useUserRolesMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<UserRoles userId="u1" />)
    expect(screen.getByText("Failed to load roles")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useUserRolesMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<UserRoles userId="u1" />)
    expect(screen.getByText("No roles assigned")).toBeInTheDocument()
  })

  it("renders a role with status, system and default badges and pagination", () => {
    useUserRolesMock.mockReturnValue({
      data: response([makeRole({ is_system: true, is_default: true })], 5),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    expect(screen.getByText("Admin")).toBeInTheDocument()
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText("Default")).toBeInTheDocument()
    expect(screen.getByText("Rows per page")).toBeInTheDocument()
  })

  it("renders a role without status/system/default badges", () => {
    useUserRolesMock.mockReturnValue({
      data: response([makeRole({ status: "", is_system: false, is_default: false })]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    expect(screen.getByText("Admin")).toBeInTheDocument()
    expect(screen.queryByText("System")).not.toBeInTheDocument()
    expect(screen.queryByText("Default")).not.toBeInTheDocument()
  })

  it("opens the assign-roles dialog via the Assign Role button", async () => {
    const u = user()
    useUserRolesMock.mockReturnValue({
      data: response([makeRole()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    // The dialog is stubbed, so just exercise the open handler without error.
    await u.click(screen.getByRole("button", { name: /assign role/i }))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("navigates to the role when the row body is clicked", async () => {
    const u = user()
    useUserRolesMock.mockReturnValue({
      data: response([makeRole()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    await u.click(screen.getByText("Admin"))
    expect(navigateMock).toHaveBeenCalledWith("/roles/r1")
  })

  it("navigates on Enter and Space, but ignores other keys", () => {
    useUserRolesMock.mockReturnValue({
      data: response([makeRole()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    const row = screen.getByRole("button", { name: /Admin/i })

    row.focus()
    // Other keys are ignored.
    const evt = (key: string) =>
      new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true })
    row.dispatchEvent(evt("a"))
    expect(navigateMock).not.toHaveBeenCalled()

    row.dispatchEvent(evt("Enter"))
    row.dispatchEvent(evt(" "))
    expect(navigateMock).toHaveBeenCalledTimes(2)
    expect(navigateMock).toHaveBeenCalledWith("/roles/r1")
  })

  it("does not navigate when interacting with the actions menu (guard)", async () => {
    const u = user()
    useUserRolesMock.mockReturnValue({
      data: response([makeRole()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    await u.click(getMenuButton())
    // The menu button click bubbled to the row handler but was guarded out.
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("ignores keydown that does not originate on the row itself", () => {
    useUserRolesMock.mockReturnValue({
      data: response([makeRole()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    // Pressing Enter while focus is on a child element (so e.target !== the row)
    // must hit the guard and not navigate.
    const child = screen.getByText("Admin")
    child.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("views a role via the menu", async () => {
    const u = user()
    useUserRolesMock.mockReturnValue({
      data: response([makeRole()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/roles/r1")
  })

  it("removes a role from the user and shows success", async () => {
    const u = user()
    removeRoleMutateAsync.mockResolvedValueOnce(undefined)
    useUserRolesMock.mockReturnValue({
      data: response([makeRole()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Remove from User"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Remove" }))

    await waitFor(() =>
      expect(removeRoleMutateAsync).toHaveBeenCalledWith({ userId: "u1", roleId: "r1" }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith('Role "Admin" removed successfully')
  })

  it("shows an error when removing a role rejects", async () => {
    const u = user()
    const err = new Error("nope")
    removeRoleMutateAsync.mockRejectedValueOnce(err)
    useUserRolesMock.mockReturnValue({
      data: response([makeRole()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserRoles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Remove from User"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Remove" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })
})
