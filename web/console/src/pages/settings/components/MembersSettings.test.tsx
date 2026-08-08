import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

import { renderWithProviders } from "@/test/utils"
import { MembersSettings } from "./MembersSettings"
import type { TenantMember } from "@/services/api/tenants/members"

const { useTenantMembersMock, deleteMutateAsync, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  useTenantMembersMock: vi.fn(),
  deleteMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/useTenantMembers", () => ({
  useTenantMembers: (...args: unknown[]) => useTenantMembersMock(...args),
  useDeleteTenantMember: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useAddTenantMember: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useTransferTenantOwnership: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useUsers", () => ({
  useUsers: () => ({ data: { rows: [] }, isLoading: false }),
  // AddMemberDialog now sources its picker from the membership-candidates
  // endpoint, since those are the only users the backend will accept.
  useMembershipCandidates: () => ({ data: { rows: [] }, isLoading: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

vi.mock("@/store/hooks", () => ({
  useAppSelector: () => null,
}))

function makeMember(n: number): TenantMember {
  return {
    tenant_member_id: `tm_${n}`,
    role: "member",
    created_at: "2025-01-01T00:00:00.000Z",
    updated_at: "2025-01-01T00:00:00.000Z",
    user: {
      user_id: `u_${n}`,
      username: `user${n}`,
      fullname: `Member ${n}`,
      email: `member${n}@example.com`,
      phone: "",
      is_email_verified: true,
      is_phone_verified: false,
      status: "active",
      metadata: {},
      created_at: "2025-01-01T00:00:00.000Z",
      updated_at: "2025-01-01T00:00:00.000Z",
    },
  }
}

/** One page of 10 rows out of a 25-member tenant — the shape that broke. */
function pageOfTwentyFive() {
  return {
    data: {
      data: {
        rows: Array.from({ length: 10 }, (_, i) => makeMember(i + 1)),
        total: 25,
        page: 1,
        limit: 10,
        total_pages: 3,
      },
    },
    isLoading: false,
    error: null,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  useTenantMembersMock.mockReturnValue(pageOfTwentyFive())
})

describe("MembersSettings pagination", () => {
  // pageCount was derived from the current page's row count, so a full page of
  // 10 always computed to exactly 1 page and members 11+ were unreachable.
  it("reports the server's total, not the current page's row count", () => {
    renderWithProviders(<MembersSettings tenantId="t-1" />)

    expect(screen.getByText(/1-10 of 25 results/)).toBeInTheDocument()
  })

  it("enables next-page navigation when the server reports more pages", () => {
    renderWithProviders(<MembersSettings tenantId="t-1" />)

    expect(screen.getByRole("button", { name: "Go to next page" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "Go to last page" })).toBeEnabled()
  })

  it("requests the next page from the server when paging forward", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    renderWithProviders(<MembersSettings tenantId="t-1" />)

    await user.click(screen.getByRole("button", { name: "Go to next page" }))

    // Not toHaveBeenLastCalledWith: the nested AddMemberDialog calls the same
    // hook with its own params, so the listing's call is not necessarily last.
    expect(useTenantMembersMock).toHaveBeenCalledWith("t-1", expect.objectContaining({ page: 2 }))
  })

  it("does not offer a next page when the server reports a single page", () => {
    useTenantMembersMock.mockReturnValue({
      data: { data: { rows: [makeMember(1)], total: 1, page: 1, limit: 10, total_pages: 1 } },
      isLoading: false,
      error: null,
    })

    renderWithProviders(<MembersSettings tenantId="t-1" />)

    expect(screen.getByText(/1-1 of 1 results/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Go to next page" })).toBeDisabled()
  })
})
