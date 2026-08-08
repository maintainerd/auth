import { beforeEach, describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

const mocks = vi.hoisted(() => ({
  useRateLimitConfig: vi.fn(),
  useUpdateRateLimitConfig: vi.fn(),
  useMaintenanceConfig: vi.fn(),
  useUpdateMaintenanceConfig: vi.fn(),
  useAuditConfig: vi.fn(),
  useUpdateAuditConfig: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock("@/hooks/useRateLimitConfig", () => ({
  useRateLimitConfig: mocks.useRateLimitConfig,
  useUpdateRateLimitConfig: mocks.useUpdateRateLimitConfig,
}))
vi.mock("@/hooks/useMaintenanceConfig", () => ({
  useMaintenanceConfig: mocks.useMaintenanceConfig,
  useUpdateMaintenanceConfig: mocks.useUpdateMaintenanceConfig,
}))
vi.mock("@/hooks/useAuditConfig", () => ({
  useAuditConfig: mocks.useAuditConfig,
  useUpdateAuditConfig: mocks.useUpdateAuditConfig,
}))
vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: mocks.showSuccess, showError: mocks.showError }),
}))

import { MaintenanceSettingsPanel, RateLimitSettingsPanel } from "./SettingsPanels"
import { toDatetimeLocalInput } from "@/lib/datetime"

// The exact key set ValidateRateLimitConfig accepts in
// internal/tenant/validation_setting.go — it 422s on the first key outside it.
const ACCEPTED_RATE_LIMIT_KEYS = [
  "enabled",
  "per_ip",
  "requests_per_window",
  "window_duration_seconds",
  "exempt_ips",
  "endpoint_overrides",
]

// The shape Go's time.RFC3339 accepts for scheduled_start / scheduled_end.
const RFC3339 = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/

function mutation() {
  const mutateAsync = vi.fn().mockResolvedValue(undefined)
  return { mutateAsync, isPending: false }
}

function loaded(data: unknown) {
  return { data, isLoading: false, isError: false }
}

const save = () => screen.getByRole("button", { name: /save changes/i })

beforeEach(() => {
  vi.clearAllMocks()
  mocks.useAuditConfig.mockReturnValue(loaded(undefined))
  mocks.useUpdateAuditConfig.mockReturnValue(mutation())
})

describe("RateLimitSettingsPanel", () => {
  const savedConfig = {
    enabled: true,
    requests_per_window: 250,
    window_duration_seconds: 30,
    per_ip: true,
    exempt_ips: [],
    endpoint_overrides: {},
  }

  it("submits only keys the API accepts, never per_api_key", async () => {
    const update = mutation()
    mocks.useRateLimitConfig.mockReturnValue(loaded(savedConfig))
    mocks.useUpdateRateLimitConfig.mockReturnValue(update)
    const user = userEvent.setup({ pointerEventsCheck: 0 })

    render(<RateLimitSettingsPanel />)
    await user.click(save())

    await waitFor(() => expect(update.mutateAsync).toHaveBeenCalledTimes(1))
    const payload = update.mutateAsync.mock.calls[0][0]
    expect(payload).not.toHaveProperty("per_api_key")
    expect(Object.keys(payload).sort()).toEqual([
      "enabled",
      "per_ip",
      "requests_per_window",
      "window_duration_seconds",
    ])
    for (const key of Object.keys(payload)) {
      expect(ACCEPTED_RATE_LIMIT_KEYS).toContain(key)
    }
  })

  it("keeps the per-IP scope switch and drops the per-API-key one the backend never had", () => {
    mocks.useRateLimitConfig.mockReturnValue(loaded(savedConfig))
    mocks.useUpdateRateLimitConfig.mockReturnValue(mutation())

    render(<RateLimitSettingsPanel />)

    expect(screen.getByRole("switch", { name: "Per IP" })).toBeInTheDocument()
    expect(screen.queryByRole("switch", { name: "Per API Key" })).not.toBeInTheDocument()
  })

  it("carries edited threshold values through unchanged", async () => {
    const update = mutation()
    mocks.useRateLimitConfig.mockReturnValue(loaded(savedConfig))
    mocks.useUpdateRateLimitConfig.mockReturnValue(update)
    const user = userEvent.setup({ pointerEventsCheck: 0 })

    render(<RateLimitSettingsPanel />)
    fireEvent.change(screen.getByLabelText("Requests per Window"), { target: { value: "500" } })
    await user.click(save())

    await waitFor(() => expect(update.mutateAsync).toHaveBeenCalledTimes(1))
    expect(update.mutateAsync.mock.calls[0][0]).toEqual({
      enabled: true,
      requests_per_window: 500,
      window_duration_seconds: 30,
      per_ip: true,
    })
  })
})

describe("MaintenanceSettingsPanel", () => {
  const savedConfig = {
    enabled: true,
    message: "Back shortly.",
    scheduled_start: "2026-08-05T14:30:00Z",
    scheduled_end: "2026-08-05T16:30:00Z",
  }

  it("hydrates the datetime-local inputs from the RFC3339 values the API returns", () => {
    mocks.useMaintenanceConfig.mockReturnValue(loaded(savedConfig))
    mocks.useUpdateMaintenanceConfig.mockReturnValue(mutation())

    render(<MaintenanceSettingsPanel />)

    expect(screen.getByLabelText("Scheduled Start")).toHaveValue(
      toDatetimeLocalInput(savedConfig.scheduled_start),
    )
    expect(screen.getByLabelText("Scheduled End")).toHaveValue(
      toDatetimeLocalInput(savedConfig.scheduled_end),
    )
  })

  it("submits RFC3339 timestamps, not the raw datetime-local values", async () => {
    const update = mutation()
    mocks.useMaintenanceConfig.mockReturnValue(loaded(savedConfig))
    mocks.useUpdateMaintenanceConfig.mockReturnValue(update)
    const user = userEvent.setup({ pointerEventsCheck: 0 })

    render(<MaintenanceSettingsPanel />)
    await user.click(save())

    await waitFor(() => expect(update.mutateAsync).toHaveBeenCalledTimes(1))
    const payload = update.mutateAsync.mock.calls[0][0]
    expect(payload.scheduled_start).toMatch(RFC3339)
    expect(payload.scheduled_end).toMatch(RFC3339)
    // The load → display → save round trip must not move the scheduled window.
    expect(new Date(payload.scheduled_start).getTime()).toBe(
      new Date(savedConfig.scheduled_start).getTime(),
    )
    expect(new Date(payload.scheduled_end).getTime()).toBe(
      new Date(savedConfig.scheduled_end).getTime(),
    )
    expect(Object.keys(payload).sort()).toEqual([
      "enabled",
      "message",
      "scheduled_end",
      "scheduled_start",
    ])
  })

  // The regression that made every scheduled window 422: a value picked in the
  // control is "2026-09-01T08:15" — no seconds, no offset — and was submitted raw.
  it("converts a window picked in the datetime-local control into RFC3339", async () => {
    const update = mutation()
    mocks.useMaintenanceConfig.mockReturnValue(loaded(savedConfig))
    mocks.useUpdateMaintenanceConfig.mockReturnValue(update)
    const user = userEvent.setup({ pointerEventsCheck: 0 })

    render(<MaintenanceSettingsPanel />)
    fireEvent.change(screen.getByLabelText("Scheduled Start"), { target: { value: "2026-09-01T08:15" } })
    fireEvent.change(screen.getByLabelText("Scheduled End"), { target: { value: "2026-09-01T10:15" } })
    await user.click(save())

    await waitFor(() => expect(update.mutateAsync).toHaveBeenCalledTimes(1))
    const payload = update.mutateAsync.mock.calls[0][0]
    expect(payload.scheduled_start).toMatch(RFC3339)
    expect(payload.scheduled_end).toMatch(RFC3339)
    // The picked wall clock is local time, so the instant is offset-dependent —
    // assert against the same local construction rather than a fixed UTC string.
    expect(new Date(payload.scheduled_start).getTime()).toBe(new Date(2026, 8, 1, 8, 15).getTime())
    expect(new Date(payload.scheduled_end).getTime()).toBe(new Date(2026, 8, 1, 10, 15).getTime())
  })

  it("sends null for a cleared schedule field, because the API rejects an empty string", async () => {
    const update = mutation()
    mocks.useMaintenanceConfig.mockReturnValue(loaded(savedConfig))
    mocks.useUpdateMaintenanceConfig.mockReturnValue(update)
    const user = userEvent.setup({ pointerEventsCheck: 0 })

    render(<MaintenanceSettingsPanel />)
    fireEvent.change(screen.getByLabelText("Scheduled End"), { target: { value: "" } })
    fireEvent.change(screen.getByLabelText("Scheduled Start"), { target: { value: "" } })
    await user.click(save())

    await waitFor(() => expect(update.mutateAsync).toHaveBeenCalledTimes(1))
    const payload = update.mutateAsync.mock.calls[0][0]
    expect(payload.scheduled_start).toBeNull()
    expect(payload.scheduled_end).toBeNull()
  })

  it("blocks an inverted window on the field instead of letting the API reject it", async () => {
    const update = mutation()
    mocks.useMaintenanceConfig.mockReturnValue(loaded(savedConfig))
    mocks.useUpdateMaintenanceConfig.mockReturnValue(update)
    const user = userEvent.setup({ pointerEventsCheck: 0 })

    render(<MaintenanceSettingsPanel />)
    fireEvent.change(screen.getByLabelText("Scheduled End"), { target: { value: "2026-08-05T09:00" } })
    await user.click(save())

    expect(
      await screen.findByText("Scheduled end must be after scheduled start"),
    ).toBeInTheDocument()
    expect(update.mutateAsync).not.toHaveBeenCalled()
  })
})
