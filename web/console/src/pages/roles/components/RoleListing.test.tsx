import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { RoleListing } from "./RoleListing"
import type { Role } from "@/services/api/roles/types"

const { useRolesMock, navigateMock } = vi.hoisted(() => ({
  useRolesMock: vi.fn(),
  navigateMock: vi.fn(),
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
  useRoles: (...args: unknown[]) => useRolesMock(...args),
  useDeleteRole: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateRoleStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeRole(overrides: Partial<Role> = {}): Role {
  return {
    role_id: "r1",
    name: "admin",
    description: "Administrator role",
    is_default: false,
    is_system: false,
    status: "active",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function setRoles(overrides: Record<string, unknown> = {}) {
  useRolesMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...overrides,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("RoleListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setRoles()
  })

  it("renders the loading state", () => {
    setRoles({ isLoading: true })
    const { container } = renderWithProviders(<RoleListing />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders rows and navigates on row click", async () => {
    setRoles({ data: { rows: [makeRole({ role_id: "r9", name: "viewer" })], total: 1 } })
    renderWithProviders(<RoleListing />)
    const cell = screen.getByText("viewer")
    await u().click(cell)
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/roles/r9"))
  })

  it("clicking New Role navigates to the create route", async () => {
    renderWithProviders(<RoleListing />)
    const buttons = screen.getAllByRole("button", { name: /new role/i })
    await u().click(buttons[0])
    expect(navigateMock).toHaveBeenCalledWith("/roles/create")
  })
})
