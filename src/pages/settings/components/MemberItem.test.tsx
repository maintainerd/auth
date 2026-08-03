import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

import { MemberItem } from "./MemberItem"
import type { TenantMember } from "@/services/api/tenants/members"

const member: TenantMember = {
  tenant_member_id: "tm_1",
  role: "admin",
  created_at: "2025-01-10T00:00:00.000Z",
  updated_at: "2025-01-10T00:00:00.000Z",
  user: {
    user_id: "user_1",
    username: "ada",
    fullname: "Ada Lovelace",
    email: "ada@example.com",
    phone: "+15555550123",
    is_email_verified: true,
    is_phone_verified: true,
    is_profile_completed: true,
    is_account_completed: true,
    status: "active",
    metadata: {},
    created_at: "2025-01-01T00:00:00.000Z",
    updated_at: "2025-01-01T00:00:00.000Z",
  },
}

describe("MemberItem", () => {
  it("renders with the shared listing row and icon hooks", () => {
    const { container } = render(<MemberItem member={member} />)

    expect(screen.getByText("Ada Lovelace")).toBeInTheDocument()
    expect(screen.getByText("ada@example.com")).toBeInTheDocument()
    expect(screen.getByText("@ada")).toBeInTheDocument()
    expect(container.querySelector("[data-md-listing-item]")).toBeInTheDocument()
    expect(container.querySelector("[data-md-listing-icon]")).toBeInTheDocument()
  })

  it("keeps member actions available", async () => {
    const onDelete = vi.fn()
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    render(<MemberItem member={member} onDelete={onDelete} />)

    await user.click(screen.getByRole("button", { name: "Open menu" }))
    await user.click(screen.getByRole("menuitem", { name: "Remove Member" }))

    expect(onDelete).toHaveBeenCalledWith("tm_1", "Ada Lovelace")
  })
})
