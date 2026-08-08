import { useState } from "react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ProfileFormDialog } from "./ProfileFormDialog"
import type { UserProfile } from "@/services/api/users/types"

const { createMutateAsync, updateMutateAsync, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  createMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/useUsers", () => ({
  useCreateUserProfile: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateUserProfile: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeProfile(overrides: Partial<UserProfile> = {}): UserProfile {
  return {
    profile_id: "p1",
    first_name: "John",
    middle_name: "M",
    last_name: "Doe",
    display_name: "Johnny",
    birthdate: "1990-05-01T00:00:00Z",
    gender: "male",
    email: "john@example.com",
    timezone: "UTC",
    language: "en",
    profile_url: "https://example.com/a.png",
    is_default: false,
    metadata: { "employee-id": "E1" },
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const user = () => userEvent.setup({ pointerEventsCheck: 0 })

const noop = () => {}

/**
 * Render the dialog with a real `open` state so calling `onOpenChange(false)`
 * actually closes (and unmounts) it. This keeps each test from leaking a
 * permanently-open Radix dialog into the next one. `onOpenChange` forwards to
 * the optional spy so tests can still assert it was called.
 */
function renderForm(props: {
  userId?: string
  profile?: UserProfile
  onOpenChange?: (open: boolean) => void
}) {
  function Host() {
    const [open, setOpen] = useState(true)
    return (
      <ProfileFormDialog
        open={open}
        onOpenChange={(next) => {
          props.onOpenChange?.(next)
          setOpen(next)
        }}
        userId={props.userId ?? "u1"}
        profile={props.profile}
      />
    )
  }
  return renderWithProviders(<Host />)
}

describe("ProfileFormDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders nothing meaningful when closed", () => {
    renderWithProviders(<ProfileFormDialog open={false} onOpenChange={noop} userId="u1" />)
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  })

  it("renders the create dialog with empty fields", () => {
    renderForm({})
    expect(screen.getByRole("heading", { name: "Create Profile" })).toBeInTheDocument()
    expect(screen.getByText("Fill in the profile information below")).toBeInTheDocument()
    expect(screen.getByText("No custom metadata yet.")).toBeInTheDocument()
  })

  it("loads profile values (incl. middle_name/profile_url) when editing", () => {
    renderForm({ profile: makeProfile() })
    expect(screen.getByRole("heading", { name: "Edit Profile" })).toBeInTheDocument()
    expect(screen.getByText("Update the profile information below")).toBeInTheDocument()
    expect(screen.getByDisplayValue("John")).toBeInTheDocument()
    expect(screen.getByDisplayValue("M")).toBeInTheDocument()
    expect(screen.getByDisplayValue("https://example.com/a.png")).toBeInTheDocument()
    // Existing metadata loads into custom fields.
    expect(screen.getByDisplayValue("employee-id")).toBeInTheDocument()
    expect(screen.getByDisplayValue("E1")).toBeInTheDocument()
  })

  it("loads a sparse profile (all optional fields empty) when editing", () => {
    // Drives the falsy side of every `profile.field || undefined` in the reset,
    // the `birthdate ? ... : undefined` else-branch, and null metadata.
    const sparse: UserProfile = {
      profile_id: "p9",
      first_name: "Solo",
      middle_name: "",
      last_name: "",
      display_name: "",
      birthdate: undefined,
      gender: "",
      email: "",
      timezone: "",
      language: "",
      profile_url: "",
      is_default: false,
      metadata: null,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    }
    renderForm({ profile: sparse })
    expect(screen.getByDisplayValue("Solo")).toBeInTheDocument()
    // No metadata rows for a null-metadata profile.
    expect(screen.getByText("No custom metadata yet.")).toBeInTheDocument()
  })

  it("creates a profile with metadata on submit", async () => {
    const u = user()
    const onOpenChange = vi.fn()
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderForm({ onOpenChange })

    await u.type(screen.getByLabelText(/first name/i), "Jane")
    // Add a custom metadata field.
    await u.click(screen.getByRole("button", { name: /add field/i }))
    await u.type(screen.getByLabelText("Metadata key"), "team")
    await u.type(screen.getByLabelText("Metadata value"), "core")

    await u.click(screen.getByRole("button", { name: "Create Profile" }))

    await waitFor(() => expect(createMutateAsync).toHaveBeenCalled())
    const arg = createMutateAsync.mock.calls[0][0]
    expect(arg.userId).toBe("u1")
    expect(arg.data.first_name).toBe("Jane")
    expect(arg.data.metadata).toEqual({ team: "core" })
    expect(showSuccessMock).toHaveBeenCalledWith("Profile created successfully")
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it("updates a profile on submit when editing", async () => {
    const u = user()
    const onOpenChange = vi.fn()
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderForm({ onOpenChange, profile: makeProfile() })
    await u.click(screen.getByRole("button", { name: "Update Profile" }))

    await waitFor(() => expect(updateMutateAsync).toHaveBeenCalled())
    const arg = updateMutateAsync.mock.calls[0][0]
    expect(arg).toMatchObject({ userId: "u1", profileId: "p1" })
    expect(arg.data.first_name).toBe("John")
    expect(showSuccessMock).toHaveBeenCalledWith("Profile updated successfully")
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it("surfaces a mutation error via the toast", async () => {
    const u = user()
    const err = new Error("nope")
    createMutateAsync.mockRejectedValueOnce(err)
    renderForm({})
    await u.type(screen.getByLabelText(/first name/i), "Jane")
    await u.click(screen.getByRole("button", { name: "Create Profile" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("shows a validation error when first name is blank", async () => {
    const u = user()
    renderForm({})
    await u.click(screen.getByRole("button", { name: "Create Profile" }))
    expect(await screen.findByText("First name is required")).toBeInTheDocument()
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("blocks submission when profile url is invalid", async () => {
    const u = user()
    renderForm({})
    await u.type(screen.getByLabelText(/first name/i), "Jane")
    await u.type(screen.getByLabelText(/profile picture url/i), "ftp://bad")
    await u.click(screen.getByRole("button", { name: "Create Profile" }))

    // Validation fails, so the create mutation is never invoked.
    await waitFor(() => expect(createMutateAsync).not.toHaveBeenCalled())
    expect(showSuccessMock).not.toHaveBeenCalled()
  })

  it("adds and removes custom metadata fields", async () => {
    const u = user()
    renderForm({})
    await u.click(screen.getByRole("button", { name: /add field/i }))
    expect(screen.getByLabelText("Metadata key")).toBeInTheDocument()
    await u.click(screen.getByRole("button", { name: /remove field/i }))
    expect(screen.queryByLabelText("Metadata key")).not.toBeInTheDocument()
    expect(screen.getByText("No custom metadata yet.")).toBeInTheDocument()
  })

  it("blocks submit and shows an error on duplicate metadata keys", async () => {
    const u = user()
    renderForm({})
    await u.type(screen.getByLabelText(/first name/i), "Jane")

    await u.click(screen.getByRole("button", { name: /add field/i }))
    await u.click(screen.getByRole("button", { name: /add field/i }))
    const keys = screen.getAllByLabelText("Metadata key")
    await u.type(keys[0], "dup")
    await u.type(keys[1], "dup")

    expect(await screen.findByText(/Duplicate keys: dup/)).toBeInTheDocument()
    // The submit button is disabled while there's a metadata error.
    expect(screen.getByRole("button", { name: "Create Profile" })).toBeDisabled()
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("aborts the submit handler when a metadata error is present", async () => {
    const u = user()
    renderForm({})
    await u.type(screen.getByLabelText(/first name/i), "Jane")
    // Create duplicate metadata keys to set the metadata error.
    await u.click(screen.getByRole("button", { name: /add field/i }))
    await u.click(screen.getByRole("button", { name: /add field/i }))
    const keys = screen.getAllByLabelText("Metadata key")
    await u.type(keys[0], "dup")
    await u.type(keys[1], "dup")
    expect(await screen.findByText(/Duplicate keys: dup/)).toBeInTheDocument()

    // The submit button is disabled, but submitting the form directly still runs
    // the handler, which must bail out via the metadata-error guard.
    fireEvent.submit(document.querySelector("form")!)
    await waitFor(() =>
      expect(showErrorMock).toHaveBeenCalledWith("Duplicate keys: dup"),
    )
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("selects a gender from the gender dropdown", async () => {
    const u = user()
    renderForm({})
    await u.click(screen.getByRole("combobox", { name: /gender/i }))
    await u.click(await screen.findByRole("option", { name: "Female" }))
    expect(screen.getByRole("combobox", { name: /gender/i })).toHaveTextContent("Female")
  })

  it("picks a birthdate from the date field", async () => {
    const u = user()
    renderForm({})
    // The trigger's accessible name comes from its associated "Birthdate" label.
    const trigger = screen.getByRole("button", { name: /birthdate/i })
    expect(trigger).toHaveTextContent("Pick a date")
    await u.click(trigger)
    // Choose any selectable day in the calendar grid.
    const days = await screen.findAllByRole("gridcell")
    const enabledDay = days
      .map((cell) => within(cell).queryByRole("button"))
      .find((btn): btn is HTMLElement => !!btn && !btn.hasAttribute("disabled"))
    expect(enabledDay).toBeTruthy()
    await u.click(enabledDay!)
    // After selecting, the trigger shows a formatted date instead of the placeholder.
    expect(screen.getByRole("button", { name: /birthdate/i })).not.toHaveTextContent("Pick a date")
  })

  it("closes via Cancel", async () => {
    const u = user()
    const onOpenChange = vi.fn()
    renderForm({ onOpenChange })
    await u.click(screen.getByRole("button", { name: "Cancel" }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
