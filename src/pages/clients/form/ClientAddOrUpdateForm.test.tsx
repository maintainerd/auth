import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useParams, useLocation } from "react-router-dom"
import { renderWithProviders } from "@/test/utils"
import ClientAddOrUpdateForm from "./ClientAddOrUpdateForm"

const {
  useClientMock,
  useClientConfigMock,
  createMutateAsync,
  updateMutateAsync,
  createUriMutateAsync,
  deleteUriMutateAsync,
  updateUriMutateAsync,
  navigateMock,
  showSuccessMock,
  showErrorMock,
  parseErrorMock,
} = vi.hoisted(() => ({
  useClientMock: vi.fn(),
  useClientConfigMock: vi.fn(),
  createMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
  createUriMutateAsync: vi.fn(),
  deleteUriMutateAsync: vi.fn(),
  updateUriMutateAsync: vi.fn(),
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

vi.mock("@/hooks/useClients", () => ({
  useClient: (...args: unknown[]) => useClientMock(...args),
  useCreateClient: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateClient: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
  useClientConfig: (...args: unknown[]) => useClientConfigMock(...args),
  useCreateClientUri: () => ({ mutateAsync: createUriMutateAsync, isPending: false }),
  useDeleteClientUri: () => ({ mutateAsync: deleteUriMutateAsync, isPending: false }),
  useUpdateClientUri: () => ({ mutateAsync: updateUriMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useBranding", () => ({
  useBrandings: () => ({ data: [] }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock,
    parseError: parseErrorMock,
  }),
}))

vi.mock("@/store/hooks", () => ({
  useAppSelector: (selector: (state: never) => unknown) =>
    selector({ tenant: { currentTenant: { tenant_id: "t1" } } } as never),
}))

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

function setCreateMode() {
  vi.mocked(useParams).mockReturnValue({})
}

function setEditMode() {
  vi.mocked(useParams).mockReturnValue({ clientId: "c1" })
}

function makeClient(overrides: Record<string, unknown> = {}) {
  return {
    client_id: "c1",
    name: "console",
    display_name: "Console App",
    client_type: "traditional",
    domain: "app.example.com",
    status: "active",
    is_default: false,
    is_system: false,
    branding_id: null,
    allow_registration: true,
    uris: [],
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("ClientAddOrUpdateForm", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useClientMock.mockReturnValue({ data: undefined, isLoading: false })
    useClientConfigMock.mockReturnValue({ data: undefined })
    parseErrorMock.mockImplementation((error: unknown) => ({
      message: error instanceof Error ? error.message : String(error),
    }))
    vi.mocked(useLocation).mockReturnValue({ state: null } as ReturnType<typeof useLocation>)
    setCreateMode()
  })

  // ── Create mode ──────────────────────────────────────────────────────────
  it("renders the create form with its card sections", () => {
    renderWithProviders(<ClientAddOrUpdateForm />)
    const headings = screen.getAllByText("Create Client")
    expect(headings.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText("Basic Information")).toBeInTheDocument()
    expect(screen.getByText("OAuth Flow Configuration")).toBeInTheDocument()
    expect(screen.getByText("Token Configuration")).toBeInTheDocument()
    expect(screen.getByText("Step-up & Session Security")).toBeInTheDocument()
    expect(screen.getByText("Metadata")).toBeInTheDocument()
    expect(screen.getByLabelText(/^name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^display name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^domain/i)).toBeInTheDocument()
  })

  it("submits create and navigates back on success", async () => {
    createMutateAsync.mockResolvedValueOnce({ client: { client_id: "c9" } })
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^name/i), "my-client")
    await u().type(screen.getByLabelText(/^display name/i), "My Client App")
    await u().type(screen.getByLabelText(/^domain/i), "app.example.com")
    await u().click(screen.getByRole("button", { name: /create client/i }))

    await waitFor(() =>
      expect(createMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "my-client",
          display_name: "My Client App",
          domain: "app.example.com",
          client_type: "spa",
          status: "active",
          allow_registration: true,
          config: expect.objectContaining({
            grant_types: ["authorization_code", "refresh_token"],
            pkce_required: true,
          }),
        }),
      ),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Client created successfully")
    expect(navigateMock).toHaveBeenCalledWith("/clients")
  })

  it("shows validation errors for blank required fields on create", async () => {
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: /create client/i }))
    expect(await screen.findByText("Client name is required")).toBeInTheDocument()
    expect(screen.getByText("Display name is required")).toBeInTheDocument()
    expect(screen.getByText("Domain is required")).toBeInTheDocument()
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("maps structured server field errors onto the offending inputs", async () => {
    const err = new Error("validation failed")
    createMutateAsync.mockRejectedValueOnce(err)
    parseErrorMock.mockReturnValueOnce({
      message: "validation failed",
      fieldErrors: { name: "Client name already exists" },
      isValidationError: true,
    })
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^name/i), "my-client")
    await u().type(screen.getByLabelText(/^display name/i), "My Client App")
    await u().type(screen.getByLabelText(/^domain/i), "app.example.com")
    await u().click(screen.getByRole("button", { name: /create client/i }))

    expect(await screen.findByText("Client name already exists")).toBeInTheDocument()
    expect(showErrorMock).toHaveBeenCalledWith(err)
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("keyword-maps unstructured server errors onto a field", async () => {
    const err = new Error("display name is already taken")
    createMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^name/i), "my-client")
    await u().type(screen.getByLabelText(/^display name/i), "My Client App")
    await u().type(screen.getByLabelText(/^domain/i), "app.example.com")
    await u().click(screen.getByRole("button", { name: /create client/i }))

    expect(await screen.findByText("display name is already taken")).toBeInTheDocument()
    expect(showErrorMock).toHaveBeenCalledWith(err)
  })

  // ── Unsaved-changes guard ────────────────────────────────────────────────
  it("cancel with a pristine form navigates immediately", async () => {
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(navigateMock).toHaveBeenCalledWith("/clients")
    expect(screen.queryByText("Discard changes?")).not.toBeInTheDocument()
  })

  it("cancel with unsaved changes prompts before discarding", async () => {
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^name/i), "dirty")
    await u().click(screen.getByRole("button", { name: "Cancel" }))

    expect(await screen.findByText("Discard changes?")).toBeInTheDocument()
    expect(navigateMock).not.toHaveBeenCalled()

    await u().click(screen.getByRole("button", { name: "Discard changes" }))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/clients"))
  })

  it("cancel after toggling a non-RHF control prompts before discarding", async () => {
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().click(screen.getByRole("switch", { name: /require consent/i }))
    await u().click(screen.getByRole("button", { name: "Cancel" }))

    expect(await screen.findByText("Discard changes?")).toBeInTheDocument()
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("cancel after editing metadata prompts before discarding", async () => {
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: /add field/i }))
    await u().click(screen.getByRole("button", { name: "Cancel" }))

    expect(await screen.findByText("Discard changes?")).toBeInTheDocument()
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("keeps the discard prompt closed while staying on the form", async () => {
    renderWithProviders(<ClientAddOrUpdateForm />)
    await u().click(screen.getByRole("switch", { name: /require consent/i }))
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    await u().click(await screen.findByRole("button", { name: "Keep editing" }))

    await waitFor(() =>
      expect(screen.queryByText("Discard changes?")).not.toBeInTheDocument(),
    )
    expect(navigateMock).not.toHaveBeenCalled()
  })

  // ── Edit mode ────────────────────────────────────────────────────────────
  it("shows the loading skeleton while fetching the client to edit", () => {
    setEditMode()
    useClientMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<ClientAddOrUpdateForm />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("shows the not-found state when the client doesn't exist", () => {
    setEditMode()
    useClientMock.mockReturnValue({ data: undefined, isLoading: false })
    renderWithProviders(<ClientAddOrUpdateForm />)
    expect(screen.getByText("Client not found")).toBeInTheDocument()
  })

  it("pre-fills the form and submits an update", async () => {
    setEditMode()
    useClientMock.mockReturnValue({ data: makeClient(), isLoading: false })
    useClientConfigMock.mockReturnValue({
      data: { grant_types: ["authorization_code"], access_token_lifetime: 3600 },
    })
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ClientAddOrUpdateForm />)

    expect(screen.getByLabelText(/^name/i)).toHaveValue("console")
    expect(screen.getByLabelText(/^display name/i)).toHaveValue("Console App")

    await u().click(screen.getByRole("button", { name: /update client/i }))

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith({
        clientId: "c1",
        data: expect.objectContaining({
          name: "console",
          display_name: "Console App",
          client_type: "traditional",
        }),
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Client updated successfully")
    expect(navigateMock).toHaveBeenCalledWith("/clients")
  })

  it("hydrates stored metadata into the shared editor", async () => {
    setEditMode()
    useClientMock.mockReturnValue({ data: makeClient(), isLoading: false })
    useClientConfigMock.mockReturnValue({
      data: { custom: { cognito_region: "us-east-1" } },
    })
    renderWithProviders(<ClientAddOrUpdateForm />)

    expect(await screen.findByDisplayValue("cognito_region")).toBeInTheDocument()
    expect(screen.getByDisplayValue("us-east-1")).toBeInTheDocument()
  })

  it("disables submit and shows the system badge and warning for system clients", () => {
    setEditMode()
    useClientMock.mockReturnValue({ data: makeClient({ is_system: true }), isLoading: false })
    renderWithProviders(<ClientAddOrUpdateForm />)
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText(/this is a system client/i)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /update client/i })).toBeDisabled()
  })

  it("honours location.state for the back destination", async () => {
    setEditMode()
    useClientMock.mockReturnValue({ data: makeClient(), isLoading: false })
    vi.mocked(useLocation).mockReturnValue({
      state: { from: "/clients/c1", backLabel: "Back to Client Details" },
    } as unknown as ReturnType<typeof useLocation>)
    renderWithProviders(<ClientAddOrUpdateForm />)

    await u().click(screen.getByRole("button", { name: /back to client details/i }))
    expect(navigateMock).toHaveBeenCalledWith("/clients/c1")
  })
})
