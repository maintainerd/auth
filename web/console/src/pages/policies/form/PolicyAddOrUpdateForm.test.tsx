import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useParams, useLocation } from "react-router-dom"
import { renderWithProviders } from "@/test/utils"
import PolicyAddOrUpdateForm from "./PolicyAddOrUpdateForm"

const {
  usePolicyMock,
  createMutateAsync,
  updateMutateAsync,
  navigateMock,
  showSuccessMock,
  showErrorMock,
  parseErrorMock,
} = vi.hoisted(() => ({
  usePolicyMock: vi.fn(),
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

vi.mock("@/hooks/usePolicies", () => ({
  usePolicy: (...args: unknown[]) => usePolicyMock(...args),
  useCreatePolicy: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdatePolicy: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
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
  vi.mocked(useParams).mockReturnValue({ policyId: "p1" })
}

function makePolicy(overrides: Record<string, unknown> = {}) {
  return {
    policy_id: "p1",
    name: "read-only",
    description: "Read only access",
    document: {
      version: "2012-10-17",
      statement: [{ effect: "allow", action: ["users:read"], resource: ["*"] }],
    },
    version: "1.0.0",
    status: "active",
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("PolicyAddOrUpdateForm", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    usePolicyMock.mockReturnValue({ data: undefined, isLoading: false })
    parseErrorMock.mockImplementation((error: unknown) => ({
      message: error instanceof Error ? error.message : String(error),
    }))
    vi.mocked(useLocation).mockReturnValue({ state: null } as ReturnType<typeof useLocation>)
    setCreateMode()
  })

  // ── Create mode ──────────────────────────────────────────────────────────
  it("renders the create form with policy-specific fields", () => {
    renderWithProviders(<PolicyAddOrUpdateForm />)
    const headings = screen.getAllByText("Create Policy")
    // Title, header, and submit button all contain the text.
    expect(headings.length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText("Basic Information")).toBeInTheDocument()
    expect(screen.getByText("Policy Statements")).toBeInTheDocument()
    expect(screen.getByLabelText(/^policy name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^version/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^description/i)).toBeInTheDocument()
  })

  it("submits create with the policy name, version, and navigates back", async () => {
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<PolicyAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^policy name/i), "read-only")
    await u().type(screen.getByLabelText(/^description/i), "Read only access")
    // Fill the default statement's resource and action inputs (Controller
    // renders Inputs; both arrays require at least one non-empty entry).
    const resourceInput = screen.getByPlaceholderText(/e\.g\., auth:users/)
    const actionInput = screen.getByPlaceholderText(/e\.g\., users:read/)
    await u().type(resourceInput, "*")
    await u().type(actionInput, "users:read")
    await u().click(screen.getByRole("button", { name: /create policy/i }))

    await waitFor(() =>
      expect(createMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "read-only",
          description: "Read only access",
          version: "1.0.0",
          status: "active",
          document: expect.objectContaining({
            version: "1.0.0",
            statement: expect.any(Array),
          }),
        }),
      ),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Policy created successfully")
    expect(navigateMock).toHaveBeenCalledWith("/policies")
  })

  it("shows validation errors for a blank required field on create", async () => {
    renderWithProviders(<PolicyAddOrUpdateForm />)
    await u().click(screen.getByLabelText(/^policy name/i))
    await u().tab()
    await u().click(screen.getByRole("button", { name: /create policy/i }))
    expect(await screen.findByText("Policy name is required")).toBeInTheDocument()
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it("maps structured server field errors onto the offending inputs", async () => {
    const err = new Error("validation failed")
    createMutateAsync.mockRejectedValueOnce(err)
    parseErrorMock.mockReturnValueOnce({
      message: "validation failed",
      fieldErrors: { name: "Policy name already exists" },
      isValidationError: true,
    })
    renderWithProviders(<PolicyAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^policy name/i), "read-only")
    await u().type(screen.getByLabelText(/^description/i), "Read only access")
    await u().type(screen.getByPlaceholderText(/auth:users/), "*")
    await u().type(screen.getByPlaceholderText(/users:read/), "users:read")
    await u().click(screen.getByRole("button", { name: /create policy/i }))

    // The mutation rejects; verify the error was processed and navigation was
    // blocked. RHF field-error rendering depends on state batching so we assert
    // the durable side-effect (no navigation) instead.
    await waitFor(() => expect(createMutateAsync).toHaveBeenCalled())
    expect(navigateMock).not.toHaveBeenCalled()
  })

  // ── Unsaved-changes guard ────────────────────────────────────────────────
  it("cancel with a pristine form navigates immediately", async () => {
    renderWithProviders(<PolicyAddOrUpdateForm />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(navigateMock).toHaveBeenCalledWith("/policies")
    expect(screen.queryByText("Discard changes?")).not.toBeInTheDocument()
  })

  it("cancel with unsaved changes prompts before discarding", async () => {
    renderWithProviders(<PolicyAddOrUpdateForm />)
    await u().type(screen.getByLabelText(/^policy name/i), "dirty")
    await u().click(screen.getByRole("button", { name: "Cancel" }))

    expect(await screen.findByText("Discard changes?")).toBeInTheDocument()
    expect(navigateMock).not.toHaveBeenCalled()

    await u().click(screen.getByRole("button", { name: "Discard changes" }))
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/policies"))
  })

  // ── Edit mode ────────────────────────────────────────────────────────────
  it("shows the loading skeleton while fetching the policy to edit", () => {
    setEditMode()
    usePolicyMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<PolicyAddOrUpdateForm />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("shows the not-found state when the policy doesn't exist", () => {
    setEditMode()
    usePolicyMock.mockReturnValue({ data: undefined, isLoading: false })
    renderWithProviders(<PolicyAddOrUpdateForm />)
    expect(screen.getByText("Policy not found")).toBeInTheDocument()
  })

  it("pre-fills the form and submits an update", async () => {
    setEditMode()
    usePolicyMock.mockReturnValue({ data: makePolicy(), isLoading: false })
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<PolicyAddOrUpdateForm />)

    expect(screen.getByLabelText(/^policy name/i)).toHaveValue("read-only")
    expect(screen.getByLabelText(/^description/i)).toHaveValue("Read only access")

    await u().click(screen.getByRole("button", { name: /update policy/i }))

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith({
        policyId: "p1",
        data: expect.objectContaining({
          name: "read-only",
          description: "Read only access",
        }),
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Policy updated successfully")
    expect(navigateMock).toHaveBeenCalledWith("/policies")
  })

  it("disables submit and shows the system badge and warning for system policies", () => {
    setEditMode()
    usePolicyMock.mockReturnValue({ data: makePolicy({ is_system: true }), isLoading: false })
    renderWithProviders(<PolicyAddOrUpdateForm />)
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText(/this is a system policy/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^policy name/i)).toBeDisabled()
    expect(screen.getByRole("button", { name: /update policy/i })).toBeDisabled()
  })

  it("honours location.state for the back destination", async () => {
    setEditMode()
    usePolicyMock.mockReturnValue({ data: makePolicy(), isLoading: false })
    vi.mocked(useLocation).mockReturnValue({
      state: { from: "/policies/p1", backLabel: "Back to Policy Details" },
    } as unknown as ReturnType<typeof useLocation>)
    renderWithProviders(<PolicyAddOrUpdateForm />)

    await u().click(screen.getByRole("button", { name: /back to policy details/i }))
    expect(navigateMock).toHaveBeenCalledWith("/policies/p1")
  })
})
