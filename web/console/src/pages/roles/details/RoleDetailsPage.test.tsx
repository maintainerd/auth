import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import RoleDetailsPage from "./RoleDetailsPage"
import type { Role } from "@/services/api/roles/types"

const { useRoleMock, navigateMock } = vi.hoisted(() => ({
  useRoleMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({ roleId: "r1" }),
    useNavigate: () => navigateMock,
    useSearchParams: () => [new URLSearchParams(), vi.fn()],
  }
})

vi.mock("@/hooks/useRoles", () => ({
  useRole: (...args: unknown[]) => useRoleMock(...args),
  useDeleteRole: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

vi.mock("./components", () => ({
  RoleHeader: ({ role }: { role: Role }) => <div data-testid="role-header">{role.name}</div>,
  RolePermissionsTab: () => <div data-testid="permissions-tab" />,
  RoleUsers: () => <div data-testid="users-tab" />,
}))

function makeRole(overrides: Partial<Role> = {}): Role {
  return {
    role_id: "r1", name: "admin", description: "Admin role",
    is_default: false, is_system: false, status: "active",
    created_at: "", updated_at: "", ...overrides,
  }
}

describe("RoleDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useRoleMock.mockReturnValue({ data: undefined, isLoading: false })
  })

  it("renders the loading state", () => {
    useRoleMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<RoleDetailsPage />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders the not-found state", () => {
    useRoleMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<RoleDetailsPage />)
    expect(screen.getByText("Role not found")).toBeInTheDocument()
  })

  it("renders the role header with role name", () => {
    useRoleMock.mockReturnValue({ data: makeRole(), isLoading: false })
    renderWithProviders(<RoleDetailsPage />)
    expect(screen.getByTestId("role-header")).toHaveTextContent("admin")
  })

  it("renders the permissions and users tabs", () => {
    useRoleMock.mockReturnValue({ data: makeRole(), isLoading: false })
    renderWithProviders(<RoleDetailsPage />)
    expect(screen.getByText("Permissions")).toBeInTheDocument()
    expect(screen.getByText("Users")).toBeInTheDocument()
  })
})
