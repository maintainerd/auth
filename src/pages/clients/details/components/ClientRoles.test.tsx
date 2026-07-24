import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ClientRoles } from "./ClientRoles"
import type { Role } from "@/services/api/roles/types"

const { navigateMock, useClientRolesMock, removeRoleMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    useClientRolesMock: vi.fn(),
    removeRoleMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useClients", () => ({
  useClientRoles: (...args: unknown[]) => useClientRolesMock(...args),
  useRemoveClientRole: () => ({ mutateAsync: removeRoleMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

// AssignClientRolesDialog is tested separately; stub it so ClientRoles is isolated.
vi.mock("./AssignClientRolesDialog", () => ({
  AssignClientRolesDialog: () => null,
}))

function makeRole(overrides: Partial<Role> = {}): Role {
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

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

// The row is a div[role="button"] whose accessible name also contains the
// menu trigger's "Open menu" text, so disambiguate to the real <button>.
function getMenuButton() {
  return screen
    .getAllByRole("button", { name: /open menu/i })
    .find((el) => el.tagName === "BUTTON") as HTMLElement
}

describe("ClientRoles", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useClientRolesMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useClientRolesMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<ClientRoles clientId="c1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useClientRolesMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ClientRoles clientId="c1" />)
    expect(screen.getByText("Failed to load roles")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useClientRolesMock.mockReturnValue({ data: [], isLoading: false, isError: false })
    renderWithProviders(<ClientRoles clientId="c1" />)
    expect(screen.getByText("No roles assigned")).toBeInTheDocument()
  })

  it("renders a role with status, system and default badges and pagination", () => {
    useClientRolesMock.mockReturnValue({
      data: [makeRole({ is_system: true, is_default: true })],
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<ClientRoles clientId="c1" />)
    expect(screen.getByText("Admin")).toBeInTheDocument()
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText("Default")).toBeInTheDocument()
    expect(screen.getByText("Rows per page")).toBeInTheDocument()
  })

  it("renders a role without system/default badges", () => {
    useClientRolesMock.mockReturnValue({
      data: [makeRole({ is_system: false, is_default: false })],
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<ClientRoles clientId="c1" />)
    expect(screen.getByText("Admin")).toBeInTheDocument()
    expect(screen.queryByText("System")).not.toBeInTheDocument()
    expect(screen.queryByText("Default")).not.toBeInTheDocument()
  })

  it("navigates to the role when the row body is clicked", async () => {
    const u = user()
    useClientRolesMock.mockReturnValue({ data: [makeRole()], isLoading: false, isError: false })
    renderWithProviders(<ClientRoles clientId="c1" />)
    await u.click(screen.getByText("Admin"))
    expect(navigateMock).toHaveBeenCalledWith("/roles/r1")
  })

  it("navigates on Enter and Space, but ignores other keys", () => {
    useClientRolesMock.mockReturnValue({ data: [makeRole()], isLoading: false, isError: false })
    renderWithProviders(<ClientRoles clientId="c1" />)
    const row = screen.getByRole("button", { name: /Admin/i })

    row.focus()
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
    useClientRolesMock.mockReturnValue({ data: [makeRole()], isLoading: false, isError: false })
    renderWithProviders(<ClientRoles clientId="c1" />)
    await u.click(getMenuButton())
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("views a role via the menu", async () => {
    const u = user()
    useClientRolesMock.mockReturnValue({ data: [makeRole()], isLoading: false, isError: false })
    renderWithProviders(<ClientRoles clientId="c1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/roles/r1")
  })

  it("removes a role from the client and shows success", async () => {
    const u = user()
    removeRoleMutateAsync.mockResolvedValueOnce(undefined)
    useClientRolesMock.mockReturnValue({ data: [makeRole()], isLoading: false, isError: false })
    renderWithProviders(<ClientRoles clientId="c1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Remove from Client"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Remove" }))

    await waitFor(() => expect(removeRoleMutateAsync).toHaveBeenCalledWith("r1"))
    expect(showSuccessMock).toHaveBeenCalledWith('Role "Admin" removed successfully')
  })

  it("shows an error when removing a role rejects", async () => {
    const u = user()
    const err = new Error("nope")
    removeRoleMutateAsync.mockRejectedValueOnce(err)
    useClientRolesMock.mockReturnValue({ data: [makeRole()], isLoading: false, isError: false })
    renderWithProviders(<ClientRoles clientId="c1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Remove from Client"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Remove" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })
})
