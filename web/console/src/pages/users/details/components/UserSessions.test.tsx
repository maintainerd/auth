import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { UserSessions } from "./UserSessions"
import type { UserSession } from "@/services/api/users/types"

const { useUserSessionsMock, revokeMutateAsync, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  useUserSessionsMock: vi.fn(),
  revokeMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/useUsers", () => ({
  useUserSessions: () => useUserSessionsMock(),
  useRevokeUserSession: () => ({ mutateAsync: revokeMutateAsync, isPending: false }),
  useRevokeAllUserSessions: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeSession(overrides: Partial<UserSession> = {}): UserSession {
  return {
    session_id: "s1",
    ip_address: "10.0.0.1",
    user_agent: "Firefox",
    last_used_at: "2024-01-02T00:00:00Z",
    created_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("UserSessions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUserSessionsMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders the loading skeleton", () => {
    useUserSessionsMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<UserSessions userId="u1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders the error message", () => {
    useUserSessionsMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<UserSessions userId="u1" />)
    expect(screen.getByText("Failed to load sessions")).toBeInTheDocument()
  })

  it("renders the empty state", () => {
    useUserSessionsMock.mockReturnValue({ data: [], isLoading: false, isError: false })
    renderWithProviders(<UserSessions userId="u1" />)
    expect(screen.getByText("No active sessions")).toBeInTheDocument()
  })

  it("renders sessions with user agent, ip, and formatted dates", () => {
    useUserSessionsMock.mockReturnValue({
      data: [makeSession()],
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserSessions userId="u1" />)
    expect(screen.getByText("Firefox")).toBeInTheDocument()
    expect(screen.getByText("10.0.0.1")).toBeInTheDocument()
  })

  it("falls back to 'Unknown device' and '—' for missing fields", () => {
    useUserSessionsMock.mockReturnValue({
      data: [
        makeSession({
          session_id: "s2",
          user_agent: null,
          ip_address: null,
          last_used_at: null,
          created_at: "not-a-date",
        }),
      ],
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserSessions userId="u1" />)
    expect(screen.getByText("Unknown device")).toBeInTheDocument()
    // last_used_at null and created_at invalid both render the em-dash.
    expect(screen.getAllByText(/—/).length).toBeGreaterThan(0)
  })

  it("revokes a session and shows success", async () => {
    const u = user()
    revokeMutateAsync.mockResolvedValueOnce(undefined)
    useUserSessionsMock.mockReturnValue({
      data: [makeSession()],
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserSessions userId="u1" />)
    await u.click(screen.getByRole("button", { name: /open menu/i }))
    await u.click(await screen.findByText("Revoke session"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Revoke" }))

    await waitFor(() =>
      expect(revokeMutateAsync).toHaveBeenCalledWith({ userId: "u1", sessionId: "s1" }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Session revoked")
  })

  it("shows error when revoke rejects", async () => {
    const u = user()
    const err = new Error("nope")
    revokeMutateAsync.mockRejectedValueOnce(err)
    useUserSessionsMock.mockReturnValue({
      data: [makeSession()],
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserSessions userId="u1" />)
    await u.click(screen.getByRole("button", { name: /open menu/i }))
    await u.click(await screen.findByText("Revoke session"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Revoke" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })
})
