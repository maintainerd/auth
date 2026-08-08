import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useParams, useLocation } from "react-router-dom"
import { renderWithProviders } from "@/test/utils"
import ApiAddOrUpdateForm from "./ApiAddOrUpdateForm"

const {
  useApiMock,
  useServicesMock,
  createMutateAsync,
  updateMutateAsync,
  navigateMock,
  showSuccessMock,
  showErrorMock,
  parseErrorMock,
} = vi.hoisted(() => ({
  useApiMock: vi.fn(),
  useServicesMock: vi.fn(),
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

vi.mock("@/hooks/useApis", () => ({
  useApi: (...args: unknown[]) => useApiMock(...args),
  useCreateApi: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateApi: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useServices", () => ({
  useServices: (...args: unknown[]) => useServicesMock(...args),
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
  vi.mocked(useParams).mockReturnValue({ apiId: "a1" })
}

function makeService(overrides: Record<string, unknown> = {}) {
  return {
    service_id: "s1",
    name: "billing-service",
    display_name: "Billing Service",
    description: "", version: "1.0.0", status: "active", is_system: false,
    api_count: 0, policy_count: 0, created_at: "", updated_at: "",
    ...overrides,
  }
}

function makeApi(overrides: Record<string, unknown> = {}) {
  return {
    api_id: "a1",
    name: "billing-api",
    display_name: "Billing API",
    description: "Invoices and payments",
    identifier: "https://api.example.com/billing",
    service: makeService(),
    status: "active",
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("ApiAddOrUpdateForm", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useApiMock.mockReturnValue({ data: undefined, isLoading: false })
    useServicesMock.mockReturnValue({
      data: { rows: [makeService()], total: 1 },
      isLoading: false,
    })
    parseErrorMock.mockImplementation((error: unknown) => ({
      message: error instanceof Error ? error.message : String(error),
    }))
    vi.mocked(useLocation).mockReturnValue({ state: null } as ReturnType<typeof useLocation>)
    setCreateMode()
  })

  // ── Create mode ──────────────────────────────────────────────────────────
  it("renders the create form with its fields", () => {
    renderWithProviders(<ApiAddOrUpdateForm />)
    const headings = screen.getAllByText("Create API")
    expect(headings.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText("Basic Information")).toBeInTheDocument()
    expect(screen.getByLabelText(/^api name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^display name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^description/i)).toBeInTheDocument()
  })

  it("submits create with a selected service and navigates back", async () => {
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ApiAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^api name/i), "billing")
    await u().type(screen.getByLabelText(/^display name/i), "Billing API")
    await u().type(screen.getByLabelText(/^description/i), "Invoices and payments")

    // Pick the owning service through the Radix select.
    const [serviceSelect] = screen.getAllByRole("combobox")
    await u().click(serviceSelect)
    await u().click(await screen.findByRole("option", { name: "Billing Service" }))

    await u().click(screen.getByRole("button", { name: /create api/i }))

    await waitFor(() =>
      expect(createMutateAsync).toHaveBeenCalledWith({
        name: "billing",
        display_name: "Billing API",
        description: "Invoices and payments",
        status: "active",
        service_id: "s1",
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("API created successfully")
    expect(navigateMock).toHaveBeenCalledWith("/apis")
  })

  it("sanitizes the API name as a slug while typing", async () => {
    renderWithProviders(<ApiAddOrUpdateForm />)
    const nameInput = screen.getByLabelText(/^api name/i)
    await u().type(nameInput, "billing_api!")
    expect(nameInput).toHaveValue("billingapi")
  })

  it("shows validation errors for blank required fields on create", async () => {
    renderWithProviders(<ApiAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: /create api/i }))
    expect(await screen.findByText("API name is required")).toBeInTheDocument()
    expect(screen.getByText("Display name is required")).toBeInTheDocument()
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("maps structured server field errors onto the offending inputs", async () => {
    const err = new Error("validation failed")
    createMutateAsync.mockRejectedValueOnce(err)
    parseErrorMock.mockReturnValueOnce({
      message: "validation failed",
      fieldErrors: { name: "API name already exists" },
      isValidationError: true,
    })
    renderWithProviders(<ApiAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^api name/i), "billing")
    await u().type(screen.getByLabelText(/^display name/i), "Billing API")
    await u().type(screen.getByLabelText(/^description/i), "Invoices and payments")
    const [serviceSelect] = screen.getAllByRole("combobox")
    await u().click(serviceSelect)
    await u().click(await screen.findByRole("option", { name: "Billing Service" }))
    await u().click(screen.getByRole("button", { name: /create api/i }))

    expect(await screen.findByText("API name already exists")).toBeInTheDocument()
    expect(showErrorMock).toHaveBeenCalledWith(err)
    expect(navigateMock).not.toHaveBeenCalled()
  })

  // ── Unsaved-changes guard ────────────────────────────────────────────────
  it("cancel with a pristine form navigates immediately", async () => {
    renderWithProviders(<ApiAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(navigateMock).toHaveBeenCalledWith("/apis")
    expect(screen.queryByText("Discard changes?")).not.toBeInTheDocument()
  })

  it("cancel with unsaved changes prompts before discarding", async () => {
    renderWithProviders(<ApiAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^api name/i), "dirty")
    await u().click(screen.getByRole("button", { name: "Cancel" }))

    expect(await screen.findByText("Discard changes?")).toBeInTheDocument()
    expect(navigateMock).not.toHaveBeenCalled()

    await u().click(screen.getByRole("button", { name: "Discard changes" }))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/apis"))
  })

  // ── Edit mode ────────────────────────────────────────────────────────────
  it("shows the loading skeleton while fetching the API to edit", () => {
    setEditMode()
    useApiMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<ApiAddOrUpdateForm />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("shows the not-found state when the API doesn't exist", () => {
    setEditMode()
    useApiMock.mockReturnValue({ data: undefined, isLoading: false })
    renderWithProviders(<ApiAddOrUpdateForm />)
    expect(screen.getByText("API not found")).toBeInTheDocument()
  })

  it("pre-fills the form and submits an update", async () => {
    setEditMode()
    useApiMock.mockReturnValue({ data: makeApi(), isLoading: false })
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ApiAddOrUpdateForm />)

    expect(screen.getByLabelText(/^api name/i)).toHaveValue("billing-api")
    expect(screen.getByLabelText(/^display name/i)).toHaveValue("Billing API")

    await u().click(screen.getByRole("button", { name: /update api/i }))

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith({
        apiId: "a1",
        data: expect.objectContaining({
          name: "billing-api",
          display_name: "Billing API",
          service_id: "s1",
        }),
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("API updated successfully")
    expect(navigateMock).toHaveBeenCalledWith("/apis")
  })

  it("disables submit and shows the system badge and warning for system APIs", () => {
    setEditMode()
    useApiMock.mockReturnValue({ data: makeApi({ is_system: true }), isLoading: false })
    renderWithProviders(<ApiAddOrUpdateForm />)
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText(/this is a system api/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^api name/i)).toBeDisabled()
    expect(screen.getByRole("button", { name: /update api/i })).toBeDisabled()
  })

  it("honours location.state for the back destination", async () => {
    setEditMode()
    useApiMock.mockReturnValue({ data: makeApi(), isLoading: false })
    vi.mocked(useLocation).mockReturnValue({
      state: { from: "/apis/a1", backLabel: "Back to API Details" },
    } as unknown as ReturnType<typeof useLocation>)
    renderWithProviders(<ApiAddOrUpdateForm />)

    await u().click(screen.getByRole("button", { name: /back to api details/i }))
    expect(navigateMock).toHaveBeenCalledWith("/apis/a1")
  })
})
