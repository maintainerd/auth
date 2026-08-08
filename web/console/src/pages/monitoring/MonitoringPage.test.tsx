import { describe, expect, it, vi } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import MonitoringPage from "./MonitoringPage"

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

vi.mock("../log-monitoring/LogMonitoringPage", () => ({
  default: ({ standalone = true }: { standalone?: boolean }) => (
    <div data-testid="tab-logs">Sign-in Logs content: {String(standalone)}</div>
  ),
}))

vi.mock("../audit-log/AuditLogPage", () => ({
  default: ({ standalone = true }: { standalone?: boolean }) => (
    <div data-testid="tab-audit">Audit Log content: {String(standalone)}</div>
  ),
}))

describe("MonitoringPage", () => {
  it("renders both tab triggers", () => {
    renderWithProviders(<MonitoringPage />, { route: "/monitoring", path: "/monitoring" })

    expect(screen.getByRole("heading", { name: "Monitoring" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Sign-in Logs" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Audit Log" })).toBeInTheDocument()
  })

  it("defaults to the sign-in logs tab", () => {
    renderWithProviders(<MonitoringPage />, { route: "/monitoring", path: "/monitoring" })

    expect(screen.getByTestId("tab-logs")).toBeInTheDocument()
    expect(screen.queryByTestId("tab-audit")).not.toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Sign-in Logs" })).toHaveAttribute("aria-selected", "true")
  })

  it("activates the audit log tab from the search param", () => {
    renderWithProviders(<MonitoringPage />, {
      route: "/monitoring?tab=audit",
      path: "/monitoring",
    })

    expect(screen.getByTestId("tab-audit")).toBeInTheDocument()
    expect(screen.queryByTestId("tab-logs")).not.toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Audit Log" })).toHaveAttribute("aria-selected", "true")
  })

  it("falls back to sign-in logs for an unknown tab", () => {
    renderWithProviders(<MonitoringPage />, {
      route: "/monitoring?tab=unknown",
      path: "/monitoring",
    })

    expect(screen.getByTestId("tab-logs")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Sign-in Logs" })).toHaveAttribute("aria-selected", "true")
  })

  it("switches between sign-in logs and audit log tabs", async () => {
    const user = u()
    renderWithProviders(<MonitoringPage />, { route: "/monitoring", path: "/monitoring" })

    await user.click(screen.getByRole("tab", { name: "Audit Log" }))
    expect(screen.getByTestId("tab-audit")).toBeInTheDocument()
    expect(screen.queryByTestId("tab-logs")).not.toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Audit Log" })).toHaveAttribute("aria-selected", "true")

    await user.click(screen.getByRole("tab", { name: "Sign-in Logs" }))
    expect(screen.getByTestId("tab-logs")).toBeInTheDocument()
    expect(screen.queryByTestId("tab-audit")).not.toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Sign-in Logs" })).toHaveAttribute("aria-selected", "true")
  })
})
