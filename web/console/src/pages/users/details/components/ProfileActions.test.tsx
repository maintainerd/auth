import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ProfileActions } from "./ProfileActions"
import type { UserProfile } from "@/services/api/users/types"

function makeProfile(overrides: Partial<UserProfile> = {}): UserProfile {
  return {
    profile_id: "p1",
    first_name: "John",
    last_name: "Doe",
    is_default: false,
    metadata: null,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ProfileActions", () => {
  const onEdit = vi.fn()
  const onDelete = vi.fn()
  const onSetAsDefault = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  function setup(profile: UserProfile) {
    return renderWithProviders(
      <ProfileActions
        profile={profile}
        onEdit={onEdit}
        onDelete={onDelete}
        onSetAsDefault={onSetAsDefault}
      />,
    )
  }

  it("shows only Edit for the default profile", async () => {
    const u = user()
    const profile = makeProfile({ is_default: true })
    setup(profile)
    await u.click(screen.getByRole("button", { name: /open menu/i }))

    expect(await screen.findByText("Edit Profile")).toBeInTheDocument()
    expect(screen.queryByText("Set As Default")).not.toBeInTheDocument()
    expect(screen.queryByText("Delete Profile")).not.toBeInTheDocument()

    await u.click(screen.getByText("Edit Profile"))
    expect(onEdit).toHaveBeenCalledWith(profile)
  })

  it("shows Edit, Set Default and Delete for a non-default profile and each fires its callback", async () => {
    const profile = makeProfile({ is_default: false })

    const u1 = user()
    setup(profile)
    await u1.click(screen.getByRole("button", { name: /open menu/i }))
    await u1.click(await screen.findByText("Set As Default"))
    expect(onSetAsDefault).toHaveBeenCalledWith(profile)

    await u1.click(screen.getByRole("button", { name: /open menu/i }))
    await u1.click(await screen.findByText("Delete Profile"))
    expect(onDelete).toHaveBeenCalledWith(profile)

    await u1.click(screen.getByRole("button", { name: /open menu/i }))
    await u1.click(await screen.findByText("Edit Profile"))
    expect(onEdit).toHaveBeenCalledWith(profile)
  })
})
