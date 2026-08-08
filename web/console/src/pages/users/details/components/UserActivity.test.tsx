import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { UserActivity } from "./UserActivity"
import type { AuthEvent, UserActivityResponse } from "@/services/api/users/types"

const { useUserActivityMock } = vi.hoisted(() => ({
  useUserActivityMock: vi.fn(),
}))

vi.mock("@/hooks/useUsers", () => ({
  useUserActivity: (...args: unknown[]) => useUserActivityMock(...args),
}))

function makeEvent(overrides: Partial<AuthEvent> = {}): AuthEvent {
  return {
    auth_event_id: "e1",
    category: "auth",
    event_type: "user.login_success",
    severity: "INFO",
    result: "SUCCESS",
    ip_address: "10.0.0.1",
    description: "Login worked",
    error_reason: null,
    created_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function response(rows: AuthEvent[], total = rows.length): UserActivityResponse {
  return { rows, total, page: 1, limit: 10, total_pages: 1 }
}

describe("UserActivity", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUserActivityMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useUserActivityMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<UserActivity userId="u1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useUserActivityMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<UserActivity userId="u1" />)
    expect(screen.getByText("Failed to load activity")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useUserActivityMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<UserActivity userId="u1" />)
    expect(screen.getByText("No activity")).toBeInTheDocument()
  })

  it("renders a SUCCESS/INFO event with description and ip, humanizing the event type", () => {
    useUserActivityMock.mockReturnValue({
      data: response([makeEvent()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserActivity userId="u1" />)
    expect(screen.getByText("User login success")).toBeInTheDocument()
    expect(screen.getByText("success")).toBeInTheDocument()
    expect(screen.getByText("Login worked")).toBeInTheDocument()
    expect(screen.getByText("10.0.0.1")).toBeInTheDocument()
    // INFO severity does not render a badge.
    expect(screen.queryByText("info")).not.toBeInTheDocument()
  })

  it("renders a FAILURE event with severity badge, error_reason fallback, and no ip", () => {
    useUserActivityMock.mockReturnValue({
      data: response([
        makeEvent({
          auth_event_id: "e2",
          event_type: "user.login_failed",
          result: "FAILURE",
          severity: "WARNING",
          description: null,
          error_reason: "Bad password",
          ip_address: "",
        }),
      ]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserActivity userId="u1" />)
    expect(screen.getByText("failure")).toBeInTheDocument()
    expect(screen.getByText("warning")).toBeInTheDocument()
    expect(screen.getByText("Bad password")).toBeInTheDocument()
  })

  it("renders an unknown result and an event with neither description nor error_reason", () => {
    useUserActivityMock.mockReturnValue({
      data: response([
        makeEvent({
          auth_event_id: "e3",
          result: "PENDING",
          description: null,
          error_reason: null,
        }),
      ]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserActivity userId="u1" />)
    expect(screen.getByText("pending")).toBeInTheDocument()
    expect(screen.queryByText("Login worked")).not.toBeInTheDocument()
  })

  it("renders pagination when total > 0", () => {
    useUserActivityMock.mockReturnValue({
      data: response([makeEvent()], 5),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserActivity userId="u1" />)
    expect(screen.getByText("Rows per page")).toBeInTheDocument()
  })
})
