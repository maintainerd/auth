import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { UserActions } from "./UserActions"
import type { User, UserStatus } from "@/services/api/users/types"

const {
  navigateMock,
  updateStatusMutateAsync,
  deleteMutateAsync,
  showSuccessMock,
  showErrorMock,
} = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  updateStatusMutateAsync: vi.fn(),
  deleteMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useUsers", () => ({
  useUpdateUserStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
  useDeleteUser: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeUser(overrides: Partial<User> = {}): User {
  return {
    user_id: "u1",
    username: "jdoe",
    fullname: "John Doe",
    email: "jdoe@example.com",
    phone: "12345",
    is_email_verified: true,
    is_phone_verified: false,
    is_profile_completed: true,
    is_account_completed: false,
    status: "active",
    metadata: {},
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

async function openMenu(u: ReturnType<typeof userEvent.setup>) {
  await u.click(screen.getByRole("button", { name: /open menu/i }))
}

describe("UserActions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("navigates to view details", async () => {
    const u = user()
    renderWithProviders(<UserActions user={makeUser()} />)
    await openMenu(u)
    await u.click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/users/u1")
  })

  it("navigates to edit", async () => {
    const u = user()
    renderWithProviders(<UserActions user={makeUser()} />)
    await openMenu(u)
    await u.click(await screen.findByText("Edit User"))
    expect(navigateMock).toHaveBeenCalledWith("/users/u1/edit")
  })

  // Covers each starting status -> the status menu items it offers.
  const statusCases: { status: UserStatus; label: string; target: UserStatus }[] = [
    { status: "inactive", label: "Activate User", target: "active" },
    { status: "active", label: "Deactivate User", target: "inactive" },
    { status: "active", label: "Suspend User", target: "suspended" },
    { status: "suspended", label: "Activate User", target: "active" },
    { status: "pending", label: "Activate User", target: "active" },
  ]

  for (const { status, label, target } of statusCases) {
    it(`status action "${label}" from ${status} confirms and updates status`, async () => {
      const u = user()
      updateStatusMutateAsync.mockResolvedValueOnce(undefined)
      renderWithProviders(<UserActions user={makeUser({ status })} />)
      await openMenu(u)
      await u.click(await screen.findByText(label))

      const dialog = await screen.findByRole("dialog")
      await u.click(within(dialog).getByRole("button", { name: label }))

      await waitFor(() =>
        expect(updateStatusMutateAsync).toHaveBeenCalledWith({
          userId: "u1",
          data: { status: target },
        }),
      )
      expect(showSuccessMock).toHaveBeenCalledWith(`User status updated to ${target}`)
    })
  }

  it("shows error when status update rejects", async () => {
    const u = user()
    const err = new Error("boom")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<UserActions user={makeUser({ status: "inactive" })} />)
    await openMenu(u)
    await u.click(await screen.findByText("Activate User"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Activate User" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    expect(showSuccessMock).not.toHaveBeenCalled()
  })

  it("deletes a user with a fullname and shows success", async () => {
    const u = user()
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    const target = makeUser({ fullname: "John Doe" })
    renderWithProviders(<UserActions user={target} />)
    await openMenu(u)
    await u.click(await screen.findByText("Delete User"))

    const dialog = await screen.findByRole("dialog")
    await u.type(within(dialog).getByRole("textbox"), "John Doe")
    await u.click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("u1"))
    expect(showSuccessMock).toHaveBeenCalledWith("User deleted successfully")
  })

  it("deletes a user without a fullname (falls back to username) and shows error on reject", async () => {
    const u = user()
    const err = new Error("nope")
    deleteMutateAsync.mockRejectedValueOnce(err)
    const target = makeUser({ fullname: "", username: "jdoe" })
    renderWithProviders(<UserActions user={target} />)
    await openMenu(u)
    await u.click(await screen.findByText("Delete User"))

    const dialog = await screen.findByRole("dialog")
    await u.type(within(dialog).getByRole("textbox"), "jdoe")
    await u.click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("u1"))
    expect(showErrorMock).toHaveBeenCalledWith(err)
  })
})
