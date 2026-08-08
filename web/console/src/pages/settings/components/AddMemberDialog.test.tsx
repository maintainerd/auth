import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

import { renderWithProviders } from "@/test/utils"
import { AddMemberDialog } from "./AddMemberDialog"
import type { TenantEntity } from "@/services/api/tenants/types"

const { addMemberMutateAsync, useCandidatesMock, useTenantMembersMock, currentTenantMock, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    addMemberMutateAsync: vi.fn(),
    useCandidatesMock: vi.fn(),
    useTenantMembersMock: vi.fn(),
    currentTenantMock: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("@/hooks/useTenantMembers", () => ({
  useAddTenantMember: () => ({ mutateAsync: addMemberMutateAsync, isPending: false }),
  useTenantMembers: (...args: unknown[]) => useTenantMembersMock(...args),
}))

vi.mock("@/hooks/useUsers", () => ({
  useMembershipCandidates: (...args: unknown[]) => useCandidatesMock(...args),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

vi.mock("@/store/hooks", () => ({
  useAppSelector: (selector: (state: { tenant: { currentTenant: TenantEntity | null } }) => unknown) =>
    selector({ tenant: { currentTenant: currentTenantMock() } }),
}))

function makeTenant(is_system: boolean): TenantEntity {
  return {
    tenant_id: "t-1",
    name: is_system ? "system" : "acme-corp",
    display_name: is_system ? "System" : "Acme Corporation",
    description: "A tenant",
    status: "active",
    is_system,
    created_at: "",
    updated_at: "",
  }
}

const USER = {
  user_id: "u-1",
  username: "ada",
  fullname: "Ada Lovelace",
  email: "ada@example.com",
}

beforeEach(() => {
  vi.clearAllMocks()
  useCandidatesMock.mockReturnValue({ data: { rows: [USER] }, isLoading: false })
  useTenantMembersMock.mockReturnValue({ data: { data: { rows: [] } } })
})

describe("AddMemberDialog in the system tenant", () => {
  beforeEach(() => currentTenantMock.mockReturnValue(makeTenant(true)))

  it("offers the user picker, because these users really are eligible", () => {
    renderWithProviders(<AddMemberDialog open onOpenChange={vi.fn()} />)

    expect(screen.getByText("Ada Lovelace")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("Search users by name or email...")).toBeInTheDocument()
  })

  it("submits the selected user", async () => {
    addMemberMutateAsync.mockResolvedValue({})
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<AddMemberDialog open onOpenChange={vi.fn()} />)

    await user.click(screen.getByText("Ada Lovelace"))
    await user.click(screen.getByRole("button", { name: /Add Member/i }))

    expect(addMemberMutateAsync).toHaveBeenCalledWith({ user_id: "u-1", role: "member" })
  })
})

// The picker is sourced from /users/membership-candidates — the SYSTEM-tenant
// users that tenant.CreateByUserUUID actually accepts. It used to be sourced
// from the caller's own tenant, so outside the system tenant every option was
// guaranteed to 403 and the dialog had to block itself entirely. Both the
// blocking and the doomed options are gone.
describe("AddMemberDialog outside the system tenant", () => {
  beforeEach(() => currentTenantMock.mockReturnValue(makeTenant(false)))

  it("still offers a picker, because candidates come from the system tenant", () => {
    renderWithProviders(<AddMemberDialog open onOpenChange={vi.fn()} />)

    expect(screen.getByText("Ada Lovelace")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("Search users by name or email...")).toBeInTheDocument()
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
  })

  it("submits a candidate that the backend will accept", async () => {
    addMemberMutateAsync.mockResolvedValue({})
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<AddMemberDialog open onOpenChange={vi.fn()} />)

    await user.click(screen.getByText("Ada Lovelace"))
    await user.click(screen.getByRole("button", { name: /Add Member/i }))

    expect(addMemberMutateAsync).toHaveBeenCalledWith({ user_id: "u-1", role: "member" })
  })

  it("keeps the submit disabled until a candidate is chosen", () => {
    renderWithProviders(<AddMemberDialog open onOpenChange={vi.fn()} />)
    expect(screen.getByRole("button", { name: /Add Member/i })).toBeDisabled()
  })
})

// The candidates query is gated on `open` so closing the dialog does not keep a
// system-tenant user list in flight.
describe("AddMemberDialog query gating", () => {
  beforeEach(() => currentTenantMock.mockReturnValue(makeTenant(false)))

  it("does not fetch candidates while closed", () => {
    renderWithProviders(<AddMemberDialog open={false} onOpenChange={vi.fn()} />)
    expect(useCandidatesMock).toHaveBeenCalledWith(expect.anything(), { enabled: false })
  })
})
