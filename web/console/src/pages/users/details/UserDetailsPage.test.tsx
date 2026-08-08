import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import UserDetailsPage from "./UserDetailsPage"
import type { User } from "@/services/api/users/types"

const { navigateMock, useUserMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  useUserMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useUsers", () => ({
  useUser: (...args: unknown[]) => useUserMock(...args),
}))

// Isolate the page from its tab content by stubbing each child component.
vi.mock("./components", () => ({
  UserHeader: ({ user }: { user: User }) => <div data-testid="user-header">{user.username}</div>,
  UserProfiles: () => <div data-testid="tab-profiles">profiles</div>,
  UserRoles: () => <div data-testid="tab-roles">roles</div>,
  UserIdentities: () => <div data-testid="tab-identities">identities</div>,
  UserSessions: () => <div data-testid="tab-sessions">sessions</div>,
  UserActivity: () => <div data-testid="tab-activity">activity</div>,
  UserMFA: () => <div data-testid="tab-mfa">mfa</div>,
  UserConsents: () => <div data-testid="tab-consents">consents</div>,
  UserTrustedDevices: () => <div data-testid="tab-devices">devices</div>,
  UserMetadata: () => <div data-testid="tab-metadata">metadata</div>,
}))

function makeUser(): User {
  return {
    user_id: "u1",
    username: "jdoe",
    fullname: "John Doe",
    email: "jdoe@example.com",
    phone: "123",
    is_email_verified: true,
    is_phone_verified: false,
    is_profile_completed: true,
    is_account_completed: false,
    status: "active",
    metadata: {},
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("UserDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("shows the loading skeleton while the user query is loading", () => {
    useUserMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    renderWithProviders(<UserDetailsPage />, { route: "/users/u1", path: "/users/:userId" })
    expect(screen.queryByTestId("user-header")).not.toBeInTheDocument()
  })

  it("shows the not-found state on error", () => {
    useUserMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<UserDetailsPage />, { route: "/users/u1", path: "/users/:userId" })
    expect(screen.getByRole("heading", { name: "User not found" })).toBeInTheDocument()
  })

  it("shows the not-found state when there is no user (no error)", () => {
    useUserMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
    renderWithProviders(<UserDetailsPage />, { route: "/users/u1", path: "/users/:userId" })
    expect(screen.getByRole("heading", { name: "User not found" })).toBeInTheDocument()
  })

  it("renders the header and the default Profiles tab on success", () => {
    useUserMock.mockReturnValue({ data: makeUser(), isLoading: false, isError: false })
    renderWithProviders(<UserDetailsPage />, { route: "/users/u1", path: "/users/:userId" })
    expect(screen.getByTestId("user-header")).toHaveTextContent("jdoe")
    expect(screen.getByTestId("tab-profiles")).toBeInTheDocument()
    // useUser is called with the userId param.
    expect(useUserMock).toHaveBeenCalledWith("u1")
  })

  it("honors the ?tab query param to select a non-default tab", () => {
    useUserMock.mockReturnValue({ data: makeUser(), isLoading: false, isError: false })
    renderWithProviders(<UserDetailsPage />, {
      route: "/users/u1?tab=roles",
      path: "/users/:userId",
    })
    expect(screen.getByTestId("tab-roles")).toBeInTheDocument()
  })

  it("switches tabs and updates the search params", async () => {
    const u = user()
    useUserMock.mockReturnValue({ data: makeUser(), isLoading: false, isError: false })
    renderWithProviders(<UserDetailsPage />, { route: "/users/u1", path: "/users/:userId" })
    await u.click(screen.getByRole("tab", { name: /metadata/i }))
    expect(screen.getByTestId("tab-metadata")).toBeInTheDocument()
  })

  it("navigates back to the users list via the back action", async () => {
    const u = user()
    useUserMock.mockReturnValue({ data: makeUser(), isLoading: false, isError: false })
    renderWithProviders(<UserDetailsPage />, { route: "/users/u1", path: "/users/:userId" })
    await u.click(screen.getByRole("button", { name: /back to users/i }))
    expect(navigateMock).toHaveBeenCalledWith("/users")
  })

  it("falls back to an empty userId when the route param is missing", () => {
    useUserMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    renderWithProviders(<UserDetailsPage />, { route: "/", path: "/*" })
    expect(useUserMock).toHaveBeenCalledWith("")
  })
})
