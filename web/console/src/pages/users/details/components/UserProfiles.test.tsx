import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { UserProfiles } from "./UserProfiles"
import type { UserProfile, UserProfilesResponse } from "@/services/api/users/types"

const {
  useUserProfilesMock,
  deleteMutateAsync,
  setDefaultMutateAsync,
  showSuccessMock,
  showErrorMock,
} = vi.hoisted(() => ({
  useUserProfilesMock: vi.fn(),
  deleteMutateAsync: vi.fn(),
  setDefaultMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/useUsers", () => ({
  useUserProfiles: (...args: unknown[]) => useUserProfilesMock(...args),
  useDeleteUserProfile: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useSetUserProfileAsDefault: () => ({ mutateAsync: setDefaultMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

// ProfileFormDialog is covered separately; render a lightweight stub that
// surfaces whether it's open and which profile (create vs edit) it received.
vi.mock("./ProfileFormDialog", () => ({
  ProfileFormDialog: ({ open, profile }: { open: boolean; profile?: UserProfile }) => (
    <div data-testid="profile-form" style={{ display: open ? "block" : "none" }}>
      {open ? (profile ? `edit:${profile.profile_id}` : "create") : null}
    </div>
  ),
}))

// ConfirmationDialog / DeleteConfirmationDialog use Radix <Dialog> with focus-
// trapping portals.  Clicking a button that lies behind the overlay (e.g.
// "Create Profile") causes Radix to close the dialog via onOpenChange, so the
// "bails-out" tests never reach the guard clauses.  Replacing them with
// simple stubs that always keep a mounted DOM wrapper avoids this.
vi.mock("@/components/dialog", () => ({
  ConfirmationDialog: ({
    open,
    onOpenChange,
    onConfirm,
    confirmText = "Confirm",
    cancelText = "Cancel",
    isLoading = false,
  }: {
    open: boolean
    onOpenChange: (o: boolean) => void
    onConfirm: () => void
    title: string
    description: string
    confirmText?: string
    cancelText?: string
    isLoading?: boolean
  }) => (
    <div data-testid="confirm-dialog" style={{ display: open ? "block" : "none" }}>
      {open && (
        <div role="dialog">
          <button onClick={() => onOpenChange(false)} disabled={isLoading}>
            {cancelText}
          </button>
          <button
            onClick={async () => {
              await onConfirm()
              onOpenChange(false)
            }}
            disabled={isLoading}
          >
            {confirmText}
          </button>
        </div>
      )}
    </div>
  ),
  DeleteConfirmationDialog: ({
    open,
    onOpenChange,
    onConfirm,
    isDeleting = false,
    confirmLabel = "Delete",
  }: {
    open: boolean
    onOpenChange: (o: boolean) => void
    onConfirm: () => void
    title: string
    description: string
    itemName?: string
    isDeleting?: boolean
    confirmLabel?: string
  }) => (
    <div data-testid="delete-dialog" style={{ display: open ? "block" : "none" }}>
      {open && (
        <div role="dialog">
          <input role="textbox" disabled={isDeleting} />
          <button onClick={() => onOpenChange(false)} disabled={isDeleting}>
            Cancel
          </button>
          <button
            onClick={async () => {
              await onConfirm()
              onOpenChange(false)
            }}
            disabled={isDeleting}
          >
            {confirmLabel}
          </button>
        </div>
      )}
    </div>
  ),
}))

function makeProfile(overrides: Partial<UserProfile> = {}): UserProfile {
  return {
    profile_id: "p1",
    first_name: "John",
    last_name: "Doe",
    is_default: false,
    metadata: null,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
    ...overrides,
  }
}

function response(rows: UserProfile[], total = rows.length): UserProfilesResponse {
  return { rows, total, page: 1, limit: 10, total_pages: 1 }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

function getMenuButton() {
  return screen
    .getAllByRole("button", { name: /open menu/i })
    .find((el) => el.tagName === "BUTTON") as HTMLElement
}

describe("UserProfiles", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUserProfilesMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useUserProfilesMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<UserProfiles userId="u1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useUserProfilesMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<UserProfiles userId="u1" />)
    expect(screen.getByText("Failed to load profiles")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useUserProfilesMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<UserProfiles userId="u1" />)
    expect(screen.getByText("No profiles")).toBeInTheDocument()
  })

  it("renders a default profile with image, attributes, metadata, and the display-name subpart", () => {
    useUserProfilesMock.mockReturnValue({
      data: response(
        [
          makeProfile({
            is_default: true,
            display_name: "Johnny",
            first_name: "John",
            last_name: "Doe",
            gender: "male",
            email: "john@example.com",
            birthdate: "1990-05-01",
            timezone: "UTC",
            language: "en",
            profile_url: "https://example.com/a.png",
            metadata: { employee_id: "E1" },
          }),
        ],
        5,
      ),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)

    // Display name is the heading; legal name appears as a subpart, plus gender.
    expect(screen.getByText("Johnny")).toBeInTheDocument()
    expect(screen.getByText("John Doe · Male")).toBeInTheDocument()
    expect(screen.getByText("Default")).toBeInTheDocument()
    expect(screen.getByText("john@example.com")).toBeInTheDocument()
    // Metadata row.
    expect(screen.getByText("employee_id")).toBeInTheDocument()
    expect(screen.getByText("E1")).toBeInTheDocument()
    // Image avatar.
    expect(screen.getByRole("img", { name: "Johnny" })).toBeInTheDocument()
  })

  it("renders a non-default profile using initials, no display name, no attributes/metadata", () => {
    useUserProfilesMock.mockReturnValue({
      data: response([
        makeProfile({
          profile_id: "p2",
          first_name: "Jane",
          last_name: "Smith",
          display_name: undefined,
          metadata: {},
        }),
      ]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    // Composed legal name as heading, initials avatar (no img).
    expect(screen.getByText("Jane Smith")).toBeInTheDocument()
    expect(screen.queryByRole("img")).not.toBeInTheDocument()
    expect(screen.getByText("JS")).toBeInTheDocument()
    expect(screen.queryByText("Default")).not.toBeInTheDocument()
  })

  it("falls back to 'Unnamed profile' when no names exist", () => {
    useUserProfilesMock.mockReturnValue({
      data: response([
        makeProfile({ profile_id: "p3", first_name: "", last_name: undefined, display_name: undefined }),
      ]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    // profileName falls back to "Unnamed profile", whose initials are "UP".
    expect(screen.getByText("Unnamed profile")).toBeInTheDocument()
    expect(screen.getByText("UP")).toBeInTheDocument()
  })

  it("renders an em dash when a date cannot be parsed", () => {
    useUserProfilesMock.mockReturnValue({
      data: response([makeProfile({ created_at: "nope", updated_at: "nope" })]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    expect(screen.getAllByText(/—/).length).toBeGreaterThan(0)
  })

  it("opens the create form via the Create Profile button", async () => {
    const u = user()
    useUserProfilesMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<UserProfiles userId="u1" />)
    await u.click(screen.getByRole("button", { name: /create profile/i, hidden: true }))
    expect(screen.getByTestId("profile-form")).toHaveTextContent("create")
  })

  it("opens the edit form from a profile's actions menu", async () => {
    const u = user()
    useUserProfilesMock.mockReturnValue({
      data: response([makeProfile()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Edit Profile"))
    expect(screen.getByTestId("profile-form")).toHaveTextContent("edit:p1")
  })

  it("sets a profile as default (success then closes)", async () => {
    const u = user()
    setDefaultMutateAsync.mockResolvedValueOnce(undefined)
    useUserProfilesMock.mockReturnValue({
      data: response([makeProfile()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Set As Default"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Set as Default" }))

    await waitFor(() =>
      expect(setDefaultMutateAsync).toHaveBeenCalledWith({ userId: "u1", profileId: "p1" }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Profile set as default successfully")
  })

  it("shows an error when set-as-default rejects", async () => {
    const u = user()
    const err = new Error("nope")
    setDefaultMutateAsync.mockRejectedValueOnce(err)
    useUserProfilesMock.mockReturnValue({
      data: response([makeProfile()]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Set As Default"))

    const dialog = await screen.findByRole("dialog")
    await u.click(within(dialog).getByRole("button", { name: "Set as Default" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes a profile (success then closes)", async () => {
    const u = user()
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    useUserProfilesMock.mockReturnValue({
      data: response([makeProfile({ display_name: "Johnny" })]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Delete Profile"))

    const dialog = await screen.findByRole("dialog")
    await u.type(within(dialog).getByRole("textbox"), "Johnny")
    await u.click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() =>
      expect(deleteMutateAsync).toHaveBeenCalledWith({ userId: "u1", profileId: "p1" }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Profile deleted successfully")
  })

  it("shows an error when delete rejects", async () => {
    const u = user()
    const err = new Error("nope")
    deleteMutateAsync.mockRejectedValueOnce(err)
    useUserProfilesMock.mockReturnValue({
      data: response([makeProfile({ display_name: "Johnny" })]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Delete Profile"))

    const dialog = await screen.findByRole("dialog")
    await u.type(within(dialog).getByRole("textbox"), "Johnny")
    await u.click(within(dialog).getByRole("button", { name: "Delete" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  // The confirm handlers guard against a missing selectedProfile. Reach that
  // guard by opening the confirm dialog (which sets selectedProfile) and then
  // clicking "Create Profile" — which resets selectedProfile to undefined while
  // the confirm dialog stays mounted — before confirming.
  it("set-as-default confirm bails out when no profile is selected", async () => {
    const u = user()
    useUserProfilesMock.mockReturnValue({
      data: response([makeProfile({ display_name: "Johnny" })]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Set As Default"))
    const setDefaultDialog = await screen.findByRole("dialog")

    // Reset selectedProfile to undefined (create flow) without closing the dialog.
    await u.click(screen.getByRole("button", { name: /create profile/i, hidden: true }))
    await u.click(within(setDefaultDialog).getByRole("button", { name: "Set as Default", hidden: true }))

    expect(setDefaultMutateAsync).not.toHaveBeenCalled()
  })

  it("delete confirm bails out when no profile is selected", async () => {
    const u = user()
    useUserProfilesMock.mockReturnValue({
      data: response([makeProfile({ display_name: "Johnny" })]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<UserProfiles userId="u1" />)
    await u.click(getMenuButton())
    await u.click(await screen.findByText("Delete Profile"))
    const deleteDialog = await screen.findByRole("dialog")

    // Reset selectedProfile to undefined (create flow) without closing the dialog.
    // The required confirmation text becomes "" (no selected profile), so an empty
    // input enables the Delete button; confirming then hits the guard clause.
    await u.click(screen.getByRole("button", { name: /create profile/i, hidden: true }))
    await u.click(within(deleteDialog).getByRole("button", { name: "Delete" }))

    expect(deleteMutateAsync).not.toHaveBeenCalled()
  })
})
