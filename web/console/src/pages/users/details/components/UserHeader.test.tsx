import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { UserHeader } from "./UserHeader"
import type { User } from "@/services/api/users/types"

const {
  navigateMock,
  deleteMutateAsync,
  verifyEmailMutateAsync,
  verifyPhoneMutateAsync,
  resetMfaMutateAsync,
  updateStatusMutateAsync,
  showSuccessMock,
  showErrorMock,
} = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  deleteMutateAsync: vi.fn(),
  verifyEmailMutateAsync: vi.fn(),
  verifyPhoneMutateAsync: vi.fn(),
  resetMfaMutateAsync: vi.fn(),
  updateStatusMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useUsers", () => {
  return {
    useDeleteUser: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
    useVerifyUserEmail: () => ({ mutateAsync: verifyEmailMutateAsync, isPending: false }),
    useVerifyUserPhone: () => ({ mutateAsync: verifyPhoneMutateAsync, isPending: false }),
    useResetUserMfa: () => ({ mutateAsync: resetMfaMutateAsync, isPending: false }),
    useUpdateUserStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
    useForcePasswordChange: () => ({ mutateAsync: vi.fn(), mutate: vi.fn(), isPending: false }),
    useCreateErasureRequest: () => ({ mutateAsync: vi.fn(), mutate: vi.fn(), isPending: false }),
    useUnlockUser: () => ({ mutateAsync: vi.fn(), mutate: vi.fn(), isPending: false }),
  }
})

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
    updated_at: "2024-02-01T00:00:00Z",
    ...overrides,
  }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

function getActionsButton() {
  return screen
    .getAllByRole("button", { name: /open actions/i })
    .find((el) => el.tagName === "BUTTON") as HTMLElement
}

describe("UserHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  function setup(overrides: Partial<User> = {}) {
    return renderWithProviders(
      <UserHeader user={makeUser(overrides)} userId="u1" />,
    )
  }

  it("renders verified email, phone, completion marks, and tenant when present", () => {
    setup({
      is_email_verified: true,
      phone: "12345",
      is_phone_verified: true,
      tenant: {
        tenant_id: "t1",
        name: "acme",
        display_name: "Acme Inc",
        description: "",
        status: "active",
        is_public: true,
        is_system: false,
        metadata: null,
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
      },
    })
    expect(screen.getByText("jdoe@example.com")).toBeInTheDocument()
    expect(screen.getByText("12345")).toBeInTheDocument()
    expect(screen.getByText("Acme Inc")).toBeInTheDocument()
    expect(screen.getByText("acme")).toBeInTheDocument()
    // Two "Verified" marks (email + phone) since both are verified.
    expect(screen.getAllByText("Verified").length).toBe(2)
  })

  it("renders unverified marks, no-phone placeholder, and no tenant block", () => {
    setup({ is_email_verified: false, phone: "", tenant: undefined })
    expect(screen.getAllByText("Not verified").length).toBeGreaterThan(0)
    // Phone falls back to the em-dash placeholder.
    expect(screen.getByText("—")).toBeInTheDocument()
    expect(screen.queryByText("Acme")).not.toBeInTheDocument()
  })

  it("uses initials from the fullname", () => {
    setup({ fullname: "John Doe" })
    expect(screen.getByText("JO")).toBeInTheDocument()
  })

  it("uses initials from the username when fullname is empty", () => {
    setup({ fullname: "", username: "zoe" })
    expect(screen.getByText("ZO")).toBeInTheDocument()
    // Title also falls back to the username.
    expect(screen.getByRole("heading", { name: "zoe" })).toBeInTheDocument()
  })

  it("navigates to edit with router state", async () => {
    const u = user()
    setup()
    await u.click(screen.getByRole("button", { name: /^edit$/i }))
    expect(navigateMock).toHaveBeenCalledWith("/users/u1/edit", {
      state: { from: "/users/u1", backLabel: "Back to User Details" },
    })
  })

  // Each non-destructive action: open the dialog, confirm, assert mutation + toast.
  const actionCases = [
    {
      menu: "Mark Email as Verified",
      mutate: () => verifyEmailMutateAsync,
      success: "Email verified successfully",
    },
    {
      menu: "Mark Phone as Verified",
      mutate: () => verifyPhoneMutateAsync,
      success: "Phone verified successfully",
    },
    {
      menu: "Reset MFA",
      mutate: () => resetMfaMutateAsync,
      success: "MFA reset successfully",
    },
  ] as const

  for (const { menu, mutate, success } of actionCases) {
    it(`"${menu}" confirms and shows success`, async () => {
      const u = user()
      mutate().mockResolvedValueOnce(undefined)
      setup()
      await u.click(getActionsButton())
      await u.click(await screen.findByText(menu))

      const dialog = await screen.findByRole("dialog")
      await u.click(within(dialog).getByRole("button", { name: /confirm|reset mfa/i }))

      await waitFor(() => expect(mutate()).toHaveBeenCalledWith("u1"))
      expect(showSuccessMock).toHaveBeenCalledWith(success)
    })

    it(`"${menu}" shows error when it rejects`, async () => {
      const u = user()
      const err = new Error("boom")
      mutate().mockRejectedValueOnce(err)
      setup()
      await u.click(getActionsButton())
      await u.click(await screen.findByText(menu))

      const dialog = await screen.findByRole("dialog")
      await u.click(within(dialog).getByRole("button", { name: /confirm|reset mfa/i }))

      await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    })
  }

  it("deletes the user, shows success, and navigates back to the list", async () => {
    const u = user()
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    setup({ fullname: "John Doe" })
    await u.click(getActionsButton())
    await u.click(await screen.findByText("Delete User"))

    const dialog = await screen.findByRole("dialog")
    await u.type(within(dialog).getByRole("textbox"), "John Doe")
    await u.click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("u1"))
    expect(showSuccessMock).toHaveBeenCalledWith("User deleted successfully")
    expect(navigateMock).toHaveBeenCalledWith("/users")
  })

  it("shows an error and does not navigate when delete rejects", async () => {
    const u = user()
    const err = new Error("nope")
    deleteMutateAsync.mockRejectedValueOnce(err)
    setup({ fullname: "John Doe" })
    await u.click(getActionsButton())
    await u.click(await screen.findByText("Delete User"))

    const dialog = await screen.findByRole("dialog")
    await u.type(within(dialog).getByRole("textbox"), "John Doe")
    await u.click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    expect(navigateMock).not.toHaveBeenCalledWith("/users")
  })
})
