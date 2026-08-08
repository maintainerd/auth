import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import UserManagementPage from "./UserManagementPage"

const { searchParamsRef, navigateMock } = vi.hoisted(() => ({
  searchParamsRef: { current: new URLSearchParams() },
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return {
    ...actual,
    useNavigate: () => navigateMock,
    useSearchParams: () => [searchParamsRef.current, vi.fn()],
  }
})

vi.mock("@/pages/users/components/UserListing", () => ({
  UserListing: () => <div data-testid="tab-users" />,
}))
vi.mock("@/pages/roles/components/RoleListing", () => ({
  RoleListing: () => <div data-testid="tab-roles" />,
}))
vi.mock("@/pages/invitations/components/InvitationListing", () => ({
  InvitationListing: () => <div data-testid="tab-invitations" />,
}))

describe("UserManagementPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
  })

  it("renders the page title, description, and all tab triggers", () => {
    renderWithProviders(<UserManagementPage />)
    expect(screen.getByRole("heading", { name: "User Management" })).toBeInTheDocument()
    expect(screen.getByText(/manage users, roles, and invitations/i)).toBeInTheDocument()
    for (const label of ["Users", "Roles", "Invitations"]) {
      expect(screen.getByRole("tab", { name: label })).toBeInTheDocument()
    }
  })

  it("defaults to the users tab", () => {
    renderWithProviders(<UserManagementPage />)
    expect(screen.getByTestId("tab-users")).toBeInTheDocument()
    expect(screen.queryByTestId("tab-roles")).not.toBeInTheDocument()
  })

  it("uses the route default tab when no valid query tab is present", () => {
    renderWithProviders(<UserManagementPage defaultTab="roles" />)
    expect(screen.getByTestId("tab-roles")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Roles" })).toHaveAttribute("aria-selected", "true")
  })

  it("activates the tab from the ?tab search param", () => {
    searchParamsRef.current = new URLSearchParams("tab=invitations")
    renderWithProviders(<UserManagementPage />)
    expect(screen.getByTestId("tab-invitations")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Invitations" })).toHaveAttribute("aria-selected", "true")
  })
})
