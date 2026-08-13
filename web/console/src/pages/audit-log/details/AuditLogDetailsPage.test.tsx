import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import AuditLogDetailsPage from "./AuditLogDetailsPage"
import type { AuditLogEntry } from "@/services/api/audit-log/types"

const { navigateMock, useAuditLogEntryMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  useAuditLogEntryMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useAuditLog", () => ({
  useAuditLogEntry: (...args: unknown[]) => useAuditLogEntryMock(...args),
}))

function makeAuditEntry(overrides: Partial<AuditLogEntry> = {}): AuditLogEntry {
  return {
    uuid: "audit-1",
    action: "tenant.updated",
    resource_type: "tenant",
    resource_id: "tenant-123",
    changes: { name: "Maintainerd" },
    ip_address: "10.0.0.1",
    actor_user_name: "Jane Admin",
    actor_client_name: null,
    outcome: "success",
    created_at: "2026-07-24T10:00:00Z",
    ...overrides,
  }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("AuditLogDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("shows the loading skeleton without dereferencing a missing entry", () => {
    useAuditLogEntryMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })

    renderWithProviders(<AuditLogDetailsPage />, {
      route: "/audit-log/audit-1",
      path: "/audit-log/:uuid",
    })

    expect(screen.queryByText("tenant.updated")).not.toBeInTheDocument()
    expect(useAuditLogEntryMock).toHaveBeenCalledWith("audit-1")
  })

  it("shows the not-found state when the entry is missing", () => {
    useAuditLogEntryMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })

    renderWithProviders(<AuditLogDetailsPage />, {
      route: "/audit-log/audit-1",
      path: "/audit-log/:uuid",
    })

    expect(screen.getByRole("heading", { name: "Audit entry not found" })).toBeInTheDocument()
  })

  it("renders audit entry details and navigates back to the monitoring audit tab", async () => {
    useAuditLogEntryMock.mockReturnValue({
      data: makeAuditEntry({ uuid: "audit-9" }),
      isLoading: false,
      isError: false,
    })
    const u = user()

    renderWithProviders(<AuditLogDetailsPage />, {
      route: "/audit-log/audit-9",
      path: "/audit-log/:uuid",
    })

    expect(screen.getByRole("heading", { name: "tenant.updated" })).toBeInTheDocument()
    expect(screen.getByText("audit-9")).toBeInTheDocument()
    expect(screen.getByText("tenant-123")).toBeInTheDocument()
    expect(screen.getByText("Jane Admin")).toBeInTheDocument()
    expect(screen.getByText("User #1")).toBeInTheDocument()
    expect(screen.getByText(/Maintainerd/)).toBeInTheDocument()

    await u.click(screen.getByRole("button", { name: /back to audit log/i }))
    expect(navigateMock).toHaveBeenCalledWith("/monitoring?tab=audit")
  })

  it("parses stringified JSON changes into multiline formatted JSON", () => {
    useAuditLogEntryMock.mockReturnValue({
      data: makeAuditEntry({
        changes: JSON.stringify({
          display_name: { old: "Old Tenant", new: "New Tenant" },
          security: {
            mfa_required: { before: false, after: true },
          },
          allowed_scopes: ["openid", "profile"],
        }),
      }),
      isLoading: false,
      isError: false,
    })

    const { container } = renderWithProviders(<AuditLogDetailsPage />, {
      route: "/audit-log/audit-1",
      path: "/audit-log/:uuid",
    })

    const changesBlock = container.querySelector("pre")

    expect(changesBlock?.textContent).toContain('{\n  "display_name": {\n    "old": "Old Tenant"')
    expect(changesBlock?.textContent).toContain('"after": true')
    expect(changesBlock?.textContent).toContain('"new": "New Tenant"')
    expect(changesBlock?.textContent).toContain('"openid"')
  })
})
