import type { ReactElement } from "react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Provider } from "react-redux"
import { configureStore } from "@reduxjs/toolkit"
import { renderWithProviders as baseRender } from "@/test/utils"
import { authReducer } from "@/store/auth/reducers"
import { tenantReducer } from "@/store/tenant/reducers"
import UserAddOrUpdateForm from "./UserAddOrUpdateForm"
import { useParams, useLocation } from "react-router-dom"

// The form reads the tenant password policy from the Redux store, so it must
// render inside a <Provider>. A fresh store per render (with a minimal tenant
// password_config) keeps the tenant slice populated and avoids cross-test leak.
function makeStore() {
  return configureStore({
    reducer: { auth: authReducer, tenant: tenantReducer },
    preloadedState: {
      tenant: {
        currentTenant: {
          tenant_id: "t1",
          name: "acme",
          display_name: "Acme",
          description: "",
          status: "active" as const,
          is_default: false,
          is_system: false,
          created_at: "2024-01-01T00:00:00Z",
          updated_at: "2024-01-01T00:00:00Z",
          password_config: {
            min_length: 8,
            max_length: 128,
            require_uppercase: true,
            require_lowercase: true,
            require_number: true,
            require_symbol: true,
            min_strength_score: 0,
            reject_common_passwords: false,
            check_hibp: false,
          },
        },
        surface: null,
        identityUrl: null,
        consoleUrl: null,
        consoleClient: null,
        isLoading: false,
        error: null,
      },
    },
  })
}

function renderWithProviders(
  ui: ReactElement,
  options?: Parameters<typeof baseRender>[1],
) {
  return baseRender(<Provider store={makeStore()}>{ui}</Provider>, options)
}

const {
  useUserMock,
  createMutateAsync,
  updateMutateAsync,
  navigateMock,
  showSuccessMock,
  showErrorMock,
} = vi.hoisted(() => ({
  useUserMock: vi.fn(),
  createMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
  navigateMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return {
    ...actual,
    useParams: vi.fn(() => ({})),
    useNavigate: () => navigateMock,
    useLocation: vi.fn(() => ({ state: null })),
  }
})

vi.mock("@/hooks/useUsers", () => ({
  useUser: (...args: unknown[]) => useUserMock(...args),
  useCreateUser: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateUser: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock,
    // The form's catch block calls parseError(error) before showError, so the
    // mock must provide it. Return the real ParsedError shape ({ message, ... }).
    parseError: (error: unknown) => ({
      message: error instanceof Error ? error.message : String(error),
    }),
  }),
}))

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

function setEditMode() {
  vi.mocked(useParams).mockReturnValue({ userId: "u1" })
}

function setCreateMode() {
  vi.mocked(useParams).mockReturnValue({})
}

function makeUser(overrides: Record<string, unknown> = {}) {
  return {
    user_id: "u1",
    username: "jdoe",
    email: "jdoe@test.com",
    phone: null,
    status: "active",
    fullname: "John Doe",
    is_email_verified: true,
    is_phone_verified: false,
    is_profile_completed: false,
    is_account_completed: false,
    metadata: null,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("UserAddOrUpdateForm", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUserMock.mockReturnValue({ data: undefined, isLoading: false })
    setCreateMode()
  })

  // ── Create mode ──────────────────────────────────────────────────────────
  it("renders the create form with account fields and the password input", () => {
    renderWithProviders(<UserAddOrUpdateForm />)
    const headings = screen.getAllByText("Create User")
    expect(headings.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/phone/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
  })

  it("submits create with a phone number and navigates back on success", async () => {
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<UserAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/username/i), "newuser")
    await u().type(screen.getByLabelText(/email/i), "new@test.com")
    await u().type(screen.getByLabelText(/phone/i), "5551234567")
    await u().type(screen.getByLabelText(/password/i), "P@ssword1")
    await u().click(screen.getByRole("button", { name: /create user/i }))

    await waitFor(() =>
      expect(createMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          username: "newuser",
          email: "new@test.com",
          // The country-code phone field emits the dial code + formatted local number.
          phone: "+1 (555) 123-4567",
          password: "P@ssword1",
        }),
      ),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("User created successfully")
    expect(navigateMock).toHaveBeenCalledWith("/users")
  })

  it("shows an error when create rejects", async () => {
    const err = new Error("create failed")
    createMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<UserAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/username/i), "newuser")
    await u().type(screen.getByLabelText(/email/i), "new@test.com")
    await u().type(screen.getByLabelText(/password/i), "P@ssword1")
    await u().click(screen.getByRole("button", { name: /create user/i }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("shows validation errors for blank required fields on create", async () => {
    renderWithProviders(<UserAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: /create user/i }))
    await waitFor(() => {
      expect(screen.getByText("Username is required")).toBeInTheDocument()
      expect(screen.getByText("Email is required")).toBeInTheDocument()
    })
    // Password field renders in create mode — assert it exists
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
  })

  // ── Edit mode: loading skeleton ──────────────────────────────────────────
  it("renders a loading skeleton when fetching the user for edit", () => {
    setEditMode()
    useUserMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<UserAddOrUpdateForm />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  // ── Edit mode: not-found ─────────────────────────────────────────────────
  it("shows the not-found state when editing a non-existent user and navigates back", async () => {
    setEditMode()
    useUserMock.mockReturnValue({ data: undefined, isLoading: false })
    renderWithProviders(<UserAddOrUpdateForm />)
    expect(screen.getByText("User not found")).toBeInTheDocument()
    const backButtons = screen.getAllByRole("button", { name: /back to users/i })
    await u().click(backButtons[backButtons.length - 1])
    expect(navigateMock).toHaveBeenCalledWith("/users")
  })

  // ── Edit mode: pre-fill and submit ───────────────────────────────────────
  it("pre-fills the form with existing user data and hides the password field", async () => {
    setEditMode()
    useUserMock.mockReturnValue({
      data: makeUser({ phone: "+1 5551234567" }),
      isLoading: false,
    })
    renderWithProviders(<UserAddOrUpdateForm />)
    await waitFor(() => {
      expect(screen.getByDisplayValue("jdoe")).toBeInTheDocument()
      expect(screen.getByDisplayValue("jdoe@test.com")).toBeInTheDocument()
      // The country-code phone field parses the stored value and shows the
      // formatted local number in the input.
      expect(screen.getByDisplayValue("(555) 123-4567")).toBeInTheDocument()
    })
    expect(screen.queryByLabelText(/password/i)).not.toBeInTheDocument()
  })

  it("submits update (keeping the existing phone) and navigates back on success", async () => {
    setEditMode()
    useUserMock.mockReturnValue({ data: makeUser({ phone: "5551234567" }), isLoading: false })
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<UserAddOrUpdateForm />)
    await waitFor(() => expect(screen.getByDisplayValue("jdoe")).toBeInTheDocument())
    await u().clear(screen.getByDisplayValue("jdoe"))
    await u().type(screen.getByLabelText(/username/i), "jdoe2")
    await u().click(screen.getByRole("button", { name: /update user/i }))

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith({
        userId: "u1",
        data: expect.objectContaining({ username: "jdoe2", phone: "5551234567" }),
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("User updated successfully")
    expect(navigateMock).toHaveBeenCalledWith("/users")
  })

  it("titles the edit page with the username when the user has no full name", async () => {
    setEditMode()
    useUserMock.mockReturnValue({
      data: makeUser({ fullname: "" }),
      isLoading: false,
    })
    renderWithProviders(<UserAddOrUpdateForm />)
    await waitFor(() =>
      expect(screen.getAllByText("Edit jdoe").length).toBeGreaterThanOrEqual(1),
    )
  })

  it("shows field validation errors for a weak password and a too-short phone on create", async () => {
    renderWithProviders(<UserAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/username/i), "newuser")
    await u().type(screen.getByLabelText(/email/i), "new@test.com")
    await u().type(screen.getByLabelText(/phone/i), "123")
    await u().type(screen.getByLabelText(/password/i), "Aa1!")
    await u().click(screen.getByRole("button", { name: /create user/i }))

    await waitFor(() => {
      expect(screen.getByText("Invalid phone number format")).toBeInTheDocument()
      expect(screen.getByText("Password must be at least 8 characters")).toBeInTheDocument()
    })
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("shows an error when update rejects", async () => {
    setEditMode()
    const err = new Error("update failed")
    updateMutateAsync.mockRejectedValueOnce(err)
    useUserMock.mockReturnValue({ data: makeUser(), isLoading: false })
    renderWithProviders(<UserAddOrUpdateForm />)
    await waitFor(() => expect(screen.getByDisplayValue("jdoe")).toBeInTheDocument())
    await u().click(screen.getByRole("button", { name: /update user/i }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  // ── Metadata ─────────────────────────────────────────────────────────────
  it("adds and removes custom metadata fields", async () => {
    renderWithProviders(<UserAddOrUpdateForm />)
    const addButtons = screen.getAllByText("Add Field")
    await u().click(addButtons[addButtons.length - 1])
    expect(screen.getAllByLabelText("Metadata key").length).toBe(1)
    const removeButtons = screen.getAllByText("Remove field")
    await u().click(removeButtons[0])
    expect(screen.queryByLabelText("Metadata key")).not.toBeInTheDocument()
  })

  it("stops submit and shows an error on duplicate metadata keys", async () => {
    renderWithProviders(<UserAddOrUpdateForm />)
    const addButtons = screen.getAllByText("Add Field")
    await u().click(addButtons[addButtons.length - 1])
    await u().click(screen.getByText("Add Field"))
    const keys = screen.getAllByLabelText("Metadata key")
    await u().type(keys[0], "dup")
    await u().type(keys[1], "dup")
    await waitFor(() => {
      expect(screen.getByText(/Duplicate keys: dup/i)).toBeInTheDocument()
    })
    await u().type(screen.getByLabelText(/username/i), "newuser")
    await u().type(screen.getByLabelText(/email/i), "new@test.com")
    await u().type(screen.getByLabelText(/password/i), "P@ssword1")
    await u().click(screen.getByRole("button", { name: /create user/i }))
    await waitFor(() => {
      expect(showErrorMock).toHaveBeenCalled()
    })
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("submits with custom metadata", async () => {
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<UserAddOrUpdateForm />)
    const addButtons = screen.getAllByText("Add Field")
    await u().click(addButtons[addButtons.length - 1])
    const keys = screen.getAllByLabelText("Metadata key")
    const values = screen.getAllByLabelText("Metadata value")
    await u().type(keys[0], "dept")
    await u().type(values[0], "engineering")
    await u().type(screen.getByLabelText(/username/i), "newuser")
    await u().type(screen.getByLabelText(/email/i), "new@test.com")
    await u().type(screen.getByLabelText(/password/i), "P@ssword1")
    await u().click(screen.getByRole("button", { name: /create user/i }))
    await waitFor(() =>
      expect(createMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({ metadata: { dept: "engineering" } }),
      ),
    )
  })

  // ── Navigation ───────────────────────────────────────────────────────────
  it("cancel button navigates back", async () => {
    renderWithProviders(<UserAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(navigateMock).toHaveBeenCalledWith("/users")
  })

  it("back navigation honours location.state", async () => {
    vi.mocked(useLocation).mockReturnValue({
      state: { from: "/users/u1", backLabel: "Back to Details" },
    } as unknown as ReturnType<typeof useLocation>)
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<UserAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/username/i), "newuser")
    await u().type(screen.getByLabelText(/email/i), "new@test.com")
    await u().type(screen.getByLabelText(/password/i), "P@ssword1")
    await u().click(screen.getByRole("button", { name: /create user/i }))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/users/u1"))
  })

  it("falls back to a generic 'Back' label when navigated from a non-default page without a label", () => {
    vi.mocked(useLocation).mockReturnValue({
      state: { from: "/users/u1" },
    } as unknown as ReturnType<typeof useLocation>)
    renderWithProviders(<UserAddOrUpdateForm />)
    expect(screen.getByRole("button", { name: "Back" })).toBeInTheDocument()
  })
})
