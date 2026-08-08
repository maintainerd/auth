import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { IpRestrictionListing } from "./IpRestrictionListing"
import type { IpRestrictionRule } from "@/services/api/ip-restriction-rules/types"

const { useIpRestrictionRulesMock, createMutateAsync, updateMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    useIpRestrictionRulesMock: vi.fn(),
    createMutateAsync: vi.fn(),
    updateMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("@/hooks/useIpRestrictionRules", () => ({
  useIpRestrictionRules: (...args: unknown[]) => useIpRestrictionRulesMock(...args),
  useCreateIpRestrictionRule: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateIpRestrictionRule: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
  useDeleteIpRestrictionRule: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateIpRestrictionRuleStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeRule(overrides: Partial<IpRestrictionRule> = {}): IpRestrictionRule {
  return {
    ipRestrictionRuleId: "r1",
    description: "Office VPN",
    type: "allow",
    ipAddress: "203.0.113.0/24",
    status: "active",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function setRules(overrides: Record<string, unknown> = {}) {
  useIpRestrictionRulesMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...overrides,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("IpRestrictionListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setRules()
  })

  it("renders the loading state", () => {
    setRules({ isLoading: true })
    const { container } = renderWithProviders(<IpRestrictionListing />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders rows", () => {
    setRules({ data: { rows: [makeRule({ ipAddress: "10.0.0.1" })], total: 1 } })
    renderWithProviders(<IpRestrictionListing />)
    expect(screen.getByText("10.0.0.1")).toBeInTheDocument()
    expect(screen.getByText("Office VPN")).toBeInTheDocument()
    expect(screen.getByText("Allow")).toBeInTheDocument()
  })

  it("clicking Add IP Rule opens the dialog", async () => {
    renderWithProviders(<IpRestrictionListing />)
    const buttons = screen.getAllByRole("button", { name: /add ip rule/i })
    await u().click(buttons[0])
    expect(await screen.findByRole("dialog")).toBeInTheDocument()
  })
})
