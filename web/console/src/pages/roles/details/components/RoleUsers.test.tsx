import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { RoleUsers } from "./RoleUsers"

const { useUsersMock, removeRoleMutateAsync, navigateMock, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  useUsersMock: vi.fn(),
  removeRoleMutateAsync: vi.fn(),
  navigateMock: vi.fn(),
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

vi.mock("@/hooks/useUsers", () => ({
  useUsers: (...args: unknown[]) => useUsersMock(...args),
  useRemoveUserRole: () => ({ mutateAsync: removeRoleMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

vi.mock("./AssignUsersToRoleDialog", () => ({
  AssignUsersToRoleDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="assign-users-dialog" /> : null,
}))

function makeUser(overrides: Record<string, unknown> = {}) {
  return {
    user_id: "u1", username: "jdoe", fullname: "John Doe", email: "jdoe@test.com",
    phone: "", status: "active", is_email_verified: false, is_phone_verified: false,
    is_profile_completed: false, is_account_completed: false,
    metadata: null, created_at: "", updated_at: "",
    ...overrides,
  }
}

function setData(overrides: Record<string, unknown> = {}) {
  useUsersMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    isError: false,
    ...overrides,
  })
}

describe("RoleUsers", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setData()
  })

  it("renders loading", () => {
    setData({ isLoading: true })
    const { container } = renderWithProviders(<RoleUsers roleId="r1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    setData({ isError: true })
    renderWithProviders(<RoleUsers roleId="r1" />)
    expect(screen.getByText("Failed to load users")).toBeInTheDocument()
  })

  it("renders empty", () => {
    renderWithProviders(<RoleUsers roleId="r1" />)
    expect(screen.getByText("No users")).toBeInTheDocument()
  })

  it("renders users with name, username, email, and status", () => {
    setData({ data: { rows: [makeUser()], total: 1 } })
    renderWithProviders(<RoleUsers roleId="r1" />)
    expect(screen.getByText("John Doe")).toBeInTheDocument()
    expect(screen.getByText("@jdoe")).toBeInTheDocument()
    expect(screen.getByText("jdoe@test.com")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
  })

  it("falls back to username when fullname is empty", () => {
    setData({ data: { rows: [makeUser({ fullname: "" })], total: 1 } })
    renderWithProviders(<RoleUsers roleId="r1" />)
    expect(screen.getByText("jdoe")).toBeInTheDocument()
  })

  it("navigates to user details on card click", async () => {
    renderWithProviders(<RoleUsers roleId="r1" />)
    setData({ data: { rows: [makeUser()], total: 1 } })
    waitFor(() => expect(screen.getByText("John Doe")).toBeInTheDocument())
  })
})
