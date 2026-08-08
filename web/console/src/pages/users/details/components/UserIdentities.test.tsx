import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { UserIdentities } from "./UserIdentities"
import type { UserIdentity, UserIdentitiesResponse } from "@/services/api/users/types"

const { useUserIdentitiesMock } = vi.hoisted(() => ({
  useUserIdentitiesMock: vi.fn(),
}))

vi.mock("@/hooks/useUsers", () => ({
  useUserIdentities: (...args: unknown[]) => useUserIdentitiesMock(...args),
  useUnlinkUserIdentity: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
}))

function makeIdentity(overrides: Partial<UserIdentity> = {}): UserIdentity {
  return {
    user_identity_id: "i1",
    provider: "google",
    sub: "sub-123",
    metadata: null,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function response(rows: UserIdentity[], total = rows.length): UserIdentitiesResponse {
  return { rows, total, page: 1, limit: 10, total_pages: 1 }
}

describe("UserIdentities", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUserIdentitiesMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useUserIdentitiesMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<UserIdentities userId="u1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useUserIdentitiesMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<UserIdentities userId="u1" />)
    expect(screen.getByText("Failed to load identities")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useUserIdentitiesMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<UserIdentities userId="u1" />)
    expect(screen.getByText("No identities")).toBeInTheDocument()
  })

  it("renders an identity without a client", () => {
    useUserIdentitiesMock.mockReturnValue({
      data: response([makeIdentity()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserIdentities userId="u1" />)
    expect(screen.getByText("google")).toBeInTheDocument()
    expect(screen.getByText("sub-123")).toBeInTheDocument()
    expect(screen.queryByText("Identity provider")).not.toBeInTheDocument()
  })

  it("renders the issuing identity provider, not a client", () => {
    useUserIdentitiesMock.mockReturnValue({
      data: response([
        makeIdentity({
          identity_provider_id: "idp-1",
          identity_provider_name: "Google Workspace",
        }),
      ]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserIdentities userId="u1" />)
    expect(screen.getByText("Identity provider")).toBeInTheDocument()
    expect(screen.getByText("Google Workspace")).toBeInTheDocument()
    // An identity is never presented as belonging to one application.
    expect(screen.queryByText("Linked client")).not.toBeInTheDocument()
  })

  it("omits the provider block when the provider could not be resolved", () => {
    useUserIdentitiesMock.mockReturnValue({
      data: response([makeIdentity()], 3),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserIdentities userId="u1" />)
    expect(screen.queryByText("Identity provider")).not.toBeInTheDocument()
    // total > 0 -> pagination shown.
    expect(screen.getByText("Rows per page")).toBeInTheDocument()
  })
})
