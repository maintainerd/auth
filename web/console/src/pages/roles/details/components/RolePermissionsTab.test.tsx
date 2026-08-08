import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, fireEvent } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { RolePermissionsTab } from "./RolePermissionsTab"

const { useRolePermissionsMock, removePermissionMutateAsync, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  useRolePermissionsMock: vi.fn(),
  removePermissionMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/useRoles", () => ({
  useRolePermissions: (...args: unknown[]) => useRolePermissionsMock(...args),
  useRemoveRolePermission: () => ({ mutateAsync: removePermissionMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

vi.mock("./AddRolePermissionsDialog", () => ({
  AddRolePermissionsDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="add-permissions-dialog" /> : null,
}))

function makePermission(overrides: Record<string, unknown> = {}) {
  return {
    permission_id: "p1",
    name: "users.read",
    description: "Read users",
    is_system: false,
    status: "active",
    api: { display_name: "Users API" },
    created_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function setData(overrides: Record<string, unknown> = {}) {
  useRolePermissionsMock.mockReturnValue({
    data: { rows: [], total: 0, total_pages: 0 },
    isLoading: false,
    isError: false,
    ...overrides,
  })
}

describe("RolePermissionsTab", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setData()
  })

  it("renders loading", () => {
    setData({ isLoading: true })
    const { container } = renderWithProviders(<RolePermissionsTab roleId="r1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    setData({ isError: true })
    renderWithProviders(<RolePermissionsTab roleId="r1" />)
    expect(screen.getByText("Failed to load permissions")).toBeInTheDocument()
  })

  it("renders empty", () => {
    renderWithProviders(<RolePermissionsTab roleId="r1" />)
    expect(screen.getByText("No permissions")).toBeInTheDocument()
  })

  it("renders permissions with name, description, and API info", () => {
    setData({ data: { rows: [makePermission()], total: 1, total_pages: 1 } })
    renderWithProviders(<RolePermissionsTab roleId="r1" />)
    expect(screen.getByText("users.read")).toBeInTheDocument()
    expect(screen.getByText("Read users")).toBeInTheDocument()
    expect(screen.getByText("API: Users API")).toBeInTheDocument()
  })

  it("shows system badge for system permissions", () => {
    setData({ data: { rows: [makePermission({ is_system: true })], total: 1, total_pages: 1 } })
    renderWithProviders(<RolePermissionsTab roleId="r1" />)
    expect(screen.getByText("System")).toBeInTheDocument()
  })

  it("opens the add permissions dialog", async () => {
    renderWithProviders(<RolePermissionsTab roleId="r1" />)
    fireEvent.click(screen.getByText("Add Permission"))
    await waitFor(() => expect(screen.getByTestId("add-permissions-dialog")).toBeInTheDocument())
  })
})
