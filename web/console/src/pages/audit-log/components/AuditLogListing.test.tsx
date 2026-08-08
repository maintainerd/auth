import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { AuditLogListing } from "./AuditLogListing"
import type { ServerListResult } from "@/components/data-table"
import type { AuditLogEntry } from "@/services/api/audit-log/types"

const { navigateMock, useAuditLogListMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  useAuditLogListMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useAuditLog", () => ({
  useAuditLogList: (...args: unknown[]) => useAuditLogListMock(...args),
}))

function makeAuditEntry(overrides: Partial<AuditLogEntry> = {}): AuditLogEntry {
  return {
    uuid: "audit-1",
    action: "user.updated",
    resource_type: "user",
    resource_id: "resource-123456789",
    changes: { status: "active" },
    ip_address: "127.0.0.1",
    actor_user_id: 1,
    actor_user_name: "John Doe",
    actor_client_id: null,
    actor_client_name: null,
    outcome: "success",
    created_at: "2026-07-24T10:00:00Z",
    ...overrides,
  }
}

interface UseAuditLogReturn {
  data?: ServerListResult<AuditLogEntry>
  isLoading: boolean
  error: Error | null
}

function setAuditLog(value: Partial<UseAuditLogReturn>) {
  useAuditLogListMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...value,
  })
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("AuditLogListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setAuditLog({})
  })

  it("renders audit entries with the shared listing pattern and opens details on row click", async () => {
    setAuditLog({ data: { rows: [makeAuditEntry({ uuid: "audit-9" })], total: 1 } })
    const u = user()

    renderWithProviders(<AuditLogListing tableInCard />)

    expect(screen.getByPlaceholderText(/search audit entries/i)).toBeInTheDocument()
    expect(screen.getByText("user.updated")).toBeInTheDocument()
    expect(screen.getByText("John Doe")).toBeInTheDocument()
    expect(screen.getByText("User #1")).toBeInTheDocument()

    await u.click(screen.getByText("user.updated"))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/audit-log/audit-9"))
  })

  it("passes filters and search params through the server listing hook", async () => {
    setAuditLog({ data: { rows: [makeAuditEntry()], total: 1 } })
    const u = user()

    renderWithProviders(<AuditLogListing />)

    await u.type(screen.getByPlaceholderText(/search audit entries/i), "role")
    await u.click(screen.getByRole("button", { name: /filters/i }))
    await u.click(await screen.findByRole("checkbox", { name: "failure" }))

    await waitFor(() =>
      expect(useAuditLogListMock).toHaveBeenCalledWith(
        expect.objectContaining({
          action: "role",
          resource_type: "role",
          outcome: "failure",
        }),
      ),
    )
  })
})
