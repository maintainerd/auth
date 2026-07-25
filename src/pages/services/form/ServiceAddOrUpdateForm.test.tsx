import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useParams, useLocation } from "react-router-dom"
import { renderWithProviders } from "@/test/utils"
import ServiceAddOrUpdateForm from "./ServiceAddOrUpdateForm"

const {
  useServiceMock,
  createMutateAsync,
  updateMutateAsync,
  navigateMock,
  showSuccessMock,
  showErrorMock,
  parseErrorMock,
} = vi.hoisted(() => ({
  useServiceMock: vi.fn(),
  createMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
  navigateMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  parseErrorMock: vi.fn(),
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

vi.mock("@/hooks/useServices", () => ({
  useService: (...args: unknown[]) => useServiceMock(...args),
  useCreateService: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateService: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock,
    parseError: parseErrorMock,
  }),
}))

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

function setCreateMode() {
  vi.mocked(useParams).mockReturnValue({})
}

function setEditMode() {
  vi.mocked(useParams).mockReturnValue({ serviceId: "s1" })
}

function makeService(overrides: Record<string, unknown> = {}) {
  return {
    service_id: "s1",
    name: "auth-service",
    display_name: "Auth Service",
    description: "Handles authentication flows",
    version: "v1.0.0",
    status: "active",
    is_system: false,
    api_count: 0,
    policy_count: 0,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("ServiceAddOrUpdateForm", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useServiceMock.mockReturnValue({ data: undefined, isLoading: false })
    parseErrorMock.mockImplementation((error: unknown) => ({
      message: error instanceof Error ? error.message : String(error),
    }))
    vi.mocked(useLocation).mockReturnValue({ state: null } as ReturnType<typeof useLocation>)
    setCreateMode()
  })

  // ── Create mode ──────────────────────────────────────────────────────────
  it("renders the create form with its fields", () => {
    renderWithProviders(<ServiceAddOrUpdateForm />)
    const headings = screen.getAllByText("Create Service")
    expect(headings.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText("Basic Information")).toBeInTheDocument()
    expect(screen.getByLabelText(/^service name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^display name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^description/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^version/i)).toBeInTheDocument()
  })

  it("submits create and navigates back on success", async () => {
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ServiceAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^service name/i), "billing")
    await u().type(screen.getByLabelText(/^display name/i), "Billing Service")
    await u().type(screen.getByLabelText(/^description/i), "Handles invoices and payments")
    await u().click(screen.getByRole("button", { name: /create service/i }))

    await waitFor(() =>
      expect(createMutateAsync).toHaveBeenCalledWith({
        name: "billing",
        display_name: "Billing Service",
        description: "Handles invoices and payments",
        version: "v0.1.0",
        status: "active",
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Service created successfully")
    expect(navigateMock).toHaveBeenCalledWith("/services")
  })

  it("sanitizes the service name as a slug while typing", async () => {
    renderWithProviders(<ServiceAddOrUpdateForm />)
    const nameInput = screen.getByLabelText(/^service name/i)
    await u().type(nameInput, "Billing_Service!")
    expect(nameInput).toHaveValue("illingervice")
  })

  it("shows validation errors for blank required fields on create", async () => {
    renderWithProviders(<ServiceAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: /create service/i }))
    expect(await screen.findByText("Service name is required")).toBeInTheDocument()
    expect(screen.getByText("Display name is required")).toBeInTheDocument()
    // Description is optional, matching the backend (Length(0, 255)) and the
    // NOT NULL DEFAULT '' column. The form used to demand 10+ characters, blocking
    // a create the server would have accepted.
    expect(screen.queryByText(/description is required/i)).not.toBeInTheDocument()
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  // Every bound below used to disagree with the server, so the form accepted values
  // that came back as a 422.
  it("enforces the backend's field limits", async () => {
    const user = u()
    renderWithProviders(<ServiceAddOrUpdateForm />)

    await user.type(screen.getByLabelText(/service name/i), "ab")
    await user.type(screen.getByLabelText(/display name/i), "ab")
    await user.type(screen.getByLabelText(/version/i), "v".repeat(21))
    await user.click(screen.getByRole("button", { name: /create service/i }))

    // "Service name…" and "Display name…" both match /name must be at least/, so
    // anchor on the distinct prefixes.
    expect(
      await screen.findByText(/service name must be at least 3 characters/i),
    ).toBeInTheDocument()
    expect(screen.getByText(/display name must be at least 3 characters/i)).toBeInTheDocument()
    expect(screen.getByText(/version must not exceed 20 characters/i)).toBeInTheDocument()
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("maps structured server field errors onto the offending inputs", async () => {
    const err = new Error("validation failed")
    createMutateAsync.mockRejectedValueOnce(err)
    parseErrorMock.mockReturnValueOnce({
      message: "validation failed",
      fieldErrors: { name: "Service name already exists" },
      isValidationError: true,
    })
    renderWithProviders(<ServiceAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^service name/i), "billing")
    await u().type(screen.getByLabelText(/^display name/i), "Billing Service")
    await u().type(screen.getByLabelText(/^description/i), "Handles invoices and payments")
    await u().click(screen.getByRole("button", { name: /create service/i }))

    expect(await screen.findByText("Service name already exists")).toBeInTheDocument()
    expect(showErrorMock).toHaveBeenCalledWith(err)
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("keyword-maps unstructured server errors onto a field", async () => {
    const err = new Error("display name is already taken")
    createMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ServiceAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^service name/i), "billing")
    await u().type(screen.getByLabelText(/^display name/i), "Billing Service")
    await u().type(screen.getByLabelText(/^description/i), "Handles invoices and payments")
    await u().click(screen.getByRole("button", { name: /create service/i }))

    expect(await screen.findByText("display name is already taken")).toBeInTheDocument()
    expect(showErrorMock).toHaveBeenCalledWith(err)
  })

  // ── Unsaved-changes guard ────────────────────────────────────────────────
  it("cancel with a pristine form navigates immediately", async () => {
    renderWithProviders(<ServiceAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(navigateMock).toHaveBeenCalledWith("/services")
    expect(screen.queryByText("Discard changes?")).not.toBeInTheDocument()
  })

  it("cancel with unsaved changes prompts before discarding", async () => {
    renderWithProviders(<ServiceAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^service name/i), "dirty")
    await u().click(screen.getByRole("button", { name: "Cancel" }))

    expect(await screen.findByText("Discard changes?")).toBeInTheDocument()
    expect(navigateMock).not.toHaveBeenCalled()

    await u().click(screen.getByRole("button", { name: "Discard changes" }))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/services"))
  })

  // ── Edit mode ────────────────────────────────────────────────────────────
  it("shows the loading skeleton while fetching the service to edit", () => {
    setEditMode()
    useServiceMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<ServiceAddOrUpdateForm />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("shows the not-found state when the service doesn't exist", () => {
    setEditMode()
    useServiceMock.mockReturnValue({ data: undefined, isLoading: false })
    renderWithProviders(<ServiceAddOrUpdateForm />)
    expect(screen.getByText("Service not found")).toBeInTheDocument()
  })

  it("pre-fills the form and submits an update", async () => {
    setEditMode()
    useServiceMock.mockReturnValue({ data: makeService(), isLoading: false })
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ServiceAddOrUpdateForm />)

    expect(screen.getByLabelText(/^service name/i)).toHaveValue("auth-service")
    expect(screen.getByLabelText(/^display name/i)).toHaveValue("Auth Service")

    await u().click(screen.getByRole("button", { name: /update service/i }))

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith({
        serviceId: "s1",
        data: expect.objectContaining({
          name: "auth-service",
          display_name: "Auth Service",
        }),
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Service updated successfully")
    expect(navigateMock).toHaveBeenCalledWith("/services")
  })

  it("disables submit and shows the system badge and warning for system services", () => {
    setEditMode()
    useServiceMock.mockReturnValue({ data: makeService({ is_system: true }), isLoading: false })
    renderWithProviders(<ServiceAddOrUpdateForm />)
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText(/this is a system service/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^service name/i)).toBeDisabled()
    expect(screen.getByRole("button", { name: /update service/i })).toBeDisabled()
  })

  it("honours location.state for the back destination", async () => {
    setEditMode()
    useServiceMock.mockReturnValue({ data: makeService(), isLoading: false })
    vi.mocked(useLocation).mockReturnValue({
      state: { from: "/services/s1", backLabel: "Back to Service Details" },
    } as unknown as ReturnType<typeof useLocation>)
    renderWithProviders(<ServiceAddOrUpdateForm />)

    await u().click(screen.getByRole("button", { name: /back to service details/i }))
    expect(navigateMock).toHaveBeenCalledWith("/services/s1")
  })
})
