import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import RoleAddOrUpdateForm from "./RoleAddOrUpdateForm"
import { useParams, useLocation } from "react-router-dom"

const { useRoleMock, createMutateAsync, updateMutateAsync, navigateMock, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    useRoleMock: vi.fn(),
    createMutateAsync: vi.fn(),
    updateMutateAsync: vi.fn(),
    navigateMock: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: vi.fn(() => ({})),
    useNavigate: () => navigateMock,
    useLocation: vi.fn(() => ({ state: null })),
  }
})

vi.mock("@/hooks/useRoles", () => ({
  useRole: (...args: unknown[]) => useRoleMock(...args),
  useCreateRole: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateRole: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock,
    // The form's catch block calls parseError(error) before showError.
    parseError: (error: unknown) => ({
      message: error instanceof Error ? error.message : String(error),
    }),
  }),
}))

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

function setEditMode() {
  vi.mocked(useParams).mockReturnValue({ roleId: "r1" })
}

function setCreateMode() {
  vi.mocked(useParams).mockReturnValue({})
}

function makeRole(overrides: Record<string, unknown> = {}) {
  return {
    role_id: "r1", name: "admin", description: "Admin role",
    is_default: false, is_system: false, status: "active",
    created_at: "", updated_at: "",
    ...overrides,
  }
}

describe("RoleAddOrUpdateForm", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useRoleMock.mockReturnValue({ data: undefined, isLoading: false })
    setCreateMode()
  })

  it("renders the create form", () => {
    renderWithProviders(<RoleAddOrUpdateForm />)
    const headings = screen.getAllByText("Create Role")
    expect(headings.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByLabelText(/name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/description/i)).toBeInTheDocument()
  })

  it("submits create and navigates back on success", async () => {
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<RoleAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/name/i), "viewer")
    await u().type(screen.getByLabelText(/description/i), "Viewer role")
    await u().click(screen.getByRole("button", { name: /create role/i }))

    await waitFor(() =>
      expect(createMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({ name: "viewer", description: "Viewer role" }),
      ),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Role created successfully")
    expect(navigateMock).toHaveBeenCalledWith("/roles")
  })

  it("shows an error when create rejects", async () => {
    const err = new Error("fail")
    createMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<RoleAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/name/i), "viewer")
    await u().type(screen.getByLabelText(/description/i), "Viewer role")
    await u().click(screen.getByRole("button", { name: /create role/i }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("shows validation errors for blank required fields", async () => {
    renderWithProviders(<RoleAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: /create role/i }))
    // Name is required; description is optional (matches the backend validation).
    await waitFor(() => {
      expect(screen.getByText(/name is required/i)).toBeInTheDocument()
    })
    expect(screen.queryByText(/description is required/i)).not.toBeInTheDocument()
  })

  it("renders loading skeleton when editing", () => {
    setEditMode()
    useRoleMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<RoleAddOrUpdateForm />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("shows not-found state when editing non-existent role", () => {
    setEditMode()
    useRoleMock.mockReturnValue({ data: undefined, isLoading: false })
    renderWithProviders(<RoleAddOrUpdateForm />)
    expect(screen.getByText("Role not found")).toBeInTheDocument()
  })

  it("pre-fills form with existing role data", async () => {
    setEditMode()
    useRoleMock.mockReturnValue({ data: makeRole(), isLoading: false })
    renderWithProviders(<RoleAddOrUpdateForm />)
    await waitFor(() => {
      expect(screen.getByDisplayValue("admin")).toBeInTheDocument()
    })
  })

  it("submits update and navigates back", async () => {
    setEditMode()
    useRoleMock.mockReturnValue({ data: makeRole(), isLoading: false })
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<RoleAddOrUpdateForm />)
    await waitFor(() => expect(screen.getByDisplayValue("admin")).toBeInTheDocument())
    await u().clear(screen.getByDisplayValue("admin"))
    await u().type(screen.getByLabelText(/name/i), "superadmin")
    await u().click(screen.getByRole("button", { name: /update role/i }))

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith({
        roleId: "r1",
        data: expect.objectContaining({ name: "superadmin" }),
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Role updated successfully")
  })

  it("back navigation honours location.state", async () => {
    vi.mocked(useLocation).mockReturnValue({
      pathname: "/roles/create",
      search: "",
      hash: "",
      state: { from: "/roles/r1", backLabel: "Back to Details" },
      key: "test",
    })
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<RoleAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/name/i), "viewer")
    await u().type(screen.getByLabelText(/description/i), "Viewer role")
    await u().click(screen.getByRole("button", { name: /create role/i }))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/roles/r1"))
  })
})
