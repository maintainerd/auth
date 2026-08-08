import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { UserListing } from "./UserListing"
import type { User } from "@/services/api/users/types"
import type { ServerListResult } from "@/components/data-table"

const { navigateMock, useUsersMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  useUsersMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useUsers", () => ({
  useUsers: (...args: unknown[]) => useUsersMock(...args),
  useUpdateUserStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteUser: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeUser(overrides: Partial<User> = {}): User {
  return {
    user_id: "u1",
    username: "jdoe",
    fullname: "John Doe",
    email: "jdoe@example.com",
    phone: "12345",
    is_email_verified: true,
    is_phone_verified: false,
    is_profile_completed: true,
    is_account_completed: false,
    status: "active",
    metadata: {},
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

interface UseUsersReturn {
  data?: ServerListResult<User>
  isLoading: boolean
  error: Error | null
}

function setUsers(value: Partial<UseUsersReturn>) {
  useUsersMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...value,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("UserListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setUsers({})
  })

  it("renders rows and navigates on row click", async () => {
    setUsers({ data: { rows: [makeUser({ user_id: "u9", username: "janedoe" })], total: 1 } })
    const user = u()
    renderWithProviders(<UserListing />)

    const cell = screen.getByText("janedoe")
    await user.click(cell)
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/users/u9"))
  })

  it("renders the loading state", () => {
    setUsers({ isLoading: true })
    renderWithProviders(<UserListing />)
    expect(screen.getByPlaceholderText(/search users/i)).toBeInTheDocument()
  })

  it("renders the error state", () => {
    setUsers({ error: new Error("failed") })
    renderWithProviders(<UserListing />)
    expect(screen.getByText(/failed to load data/i)).toBeInTheDocument()
  })

  it("clicking New User navigates to the create route", async () => {
    setUsers({ data: { rows: [makeUser()], total: 1 } })
    const user = u()
    renderWithProviders(<UserListing />)

    await user.click(screen.getByRole("button", { name: /new user/i }))
    expect(navigateMock).toHaveBeenCalledWith("/users/create")
  })

  it("typing in the search box debounces into the API params", async () => {
    setUsers({ data: { rows: [makeUser()], total: 1 } })
    renderWithProviders(<UserListing />)

    const input = screen.getByPlaceholderText(/search users/i)
    const user = u()
    await user.type(input, "alice")

    await waitFor(() =>
      expect(useUsersMock).toHaveBeenCalledWith(
        expect.objectContaining({ search: "alice" }),
      ),
    )
  })

  it("toggling a Status filter checkbox updates the API params and shows the chip", async () => {
    setUsers({ data: { rows: [makeUser()], total: 1 } })
    const user = u()
    renderWithProviders(<UserListing />)

    await user.click(screen.getByRole("button", { name: /filters/i }))
    const checkbox = await screen.findByRole("checkbox", { name: "active" })
    await user.click(checkbox)

    await waitFor(() =>
      expect(useUsersMock).toHaveBeenCalledWith(expect.objectContaining({ status: "active" })),
    )
    expect(await screen.findByText("Status: active")).toBeInTheDocument()
  })

  it("Clear all removes the active filters", async () => {
    setUsers({ data: { rows: [makeUser()], total: 1 } })
    const user = u()
    renderWithProviders(<UserListing />)

    await user.click(screen.getByRole("button", { name: /filters/i }))
    const checkbox = await screen.findByRole("checkbox", { name: "active" })
    await user.click(checkbox)

    expect(await screen.findByText("Status: active")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Clear all" }))
    await waitFor(() => expect(screen.queryByText("Status: active")).not.toBeInTheDocument())
  })

  // The column id is the display label ("Status"); sort_by must be the API column,
  // because the backend sanitizes against an allowlist and silently ignores anything
  // else — which made a header click look like a no-op.
  it("clicking a sortable column header toggles sort in the API params", async () => {
    setUsers({ data: { rows: [makeUser()], total: 1 } })
    const user = u()
    renderWithProviders(<UserListing />)

    await user.click(screen.getByRole("button", { name: /^status$/i }))
    await waitFor(() =>
      expect(useUsersMock).toHaveBeenCalledWith(
        expect.objectContaining({ sort_by: "status", sort_order: "asc" }),
      ),
    )
    // Toggle again -> desc
    await user.click(screen.getByRole("button", { name: /^status$/i }))
    await waitFor(() =>
      expect(useUsersMock).toHaveBeenCalledWith(
        expect.objectContaining({ sort_by: "status", sort_order: "desc" }),
      ),
    )
  })
})
