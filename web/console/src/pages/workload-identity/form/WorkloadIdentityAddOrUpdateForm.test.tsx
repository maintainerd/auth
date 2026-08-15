import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useParams } from "react-router-dom"
import { renderWithProviders } from "@/test/utils"
import WorkloadIdentityAddOrUpdateForm from "./WorkloadIdentityAddOrUpdateForm"

const {
  useWorkloadIdentityMock,
  createMutateAsync,
  updateMutateAsync,
  navigateMock,
  showSuccessMock,
  showErrorMock,
} = vi.hoisted(() => ({
  useWorkloadIdentityMock: vi.fn(),
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

vi.mock("@/hooks/useWorkloadIdentity", () => ({
  useWorkloadIdentity: (...args: unknown[]) => useWorkloadIdentityMock(...args),
  useCreateWorkloadIdentity: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateWorkloadIdentity: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useClients", () => ({
  useClients: () => ({
    data: {
      rows: [
        { client_id: "client-uuid-1", name: "api", display_name: "API Client" },
      ],
    },
  }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock,
    parseError: (e: unknown) => ({ message: String(e) }),
  }),
}))

const existingFederation = {
  workload_identity_federation_id: "fed-uuid-1",
  client_id: "client-uuid-1",
  name: "github-actions",
  description: "CI deploys",
  issuer_url: "https://token.actions.githubusercontent.com",
  audience: "https://auth.example.com",
  subject_claim: "sub",
  subject_pattern: "repo:my-org/my-repo:*",
  allowed_scopes: ["api:read"],
  attribute_mapping: { repository: "repository" },
  is_active: true,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
}

const u = () => userEvent.setup()

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(useParams).mockReturnValue({})
  useWorkloadIdentityMock.mockReturnValue({ data: undefined, isLoading: false })
})

describe("WorkloadIdentityAddOrUpdateForm", () => {
  it("renders the create form with its sections", () => {
    renderWithProviders(<WorkloadIdentityAddOrUpdateForm />)
    expect(screen.getByText("New Workload Identity Federation")).toBeInTheDocument()
    expect(screen.getByText("Trust")).toBeInTheDocument()
    expect(screen.getByText("Issued Token")).toBeInTheDocument()
  })

  it("blocks submit and reports the offending field for a blank form", async () => {
    const user = u()
    renderWithProviders(<WorkloadIdentityAddOrUpdateForm />)

    await user.click(screen.getByRole("button", { name: /create federation/i }))

    await waitFor(() => expect(screen.getByText("Name is required")).toBeInTheDocument())
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  // The console used to accept http:// (yup's .url()) and let the server reject it.
  it("rejects a non-https issuer before sending", async () => {
    const user = u()
    renderWithProviders(<WorkloadIdentityAddOrUpdateForm />)

    await user.type(screen.getByLabelText(/^name/i), "github-actions")
    await user.type(screen.getByLabelText(/issuer url/i), "http://token.actions.githubusercontent.com")
    await user.type(screen.getByLabelText(/^audience/i), "https://auth.example.com")
    await user.type(screen.getByLabelText(/subject pattern/i), "repo:my-org/my-repo:*")
    await user.click(screen.getByRole("button", { name: /create federation/i }))

    await waitFor(() => expect(screen.getByText(/must use https/i)).toBeInTheDocument())
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  // An unanchored pattern lets any workload on a shared public issuer mint this
  // tenant's token, so the console must refuse it rather than rely on the server.
  it("rejects an unanchored subject pattern before sending", async () => {
    const user = u()
    renderWithProviders(<WorkloadIdentityAddOrUpdateForm />)

    await user.type(screen.getByLabelText(/^name/i), "github-actions")
    await user.type(screen.getByLabelText(/issuer url/i), "https://token.actions.githubusercontent.com")
    await user.type(screen.getByLabelText(/^audience/i), "https://auth.example.com")
    await user.type(screen.getByLabelText(/subject pattern/i), "*")
    await user.click(screen.getByRole("button", { name: /create federation/i }))

    await waitFor(() =>
      expect(screen.getByText(/must not start with a wildcard/i)).toBeInTheDocument(),
    )
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  describe("editing", () => {
    beforeEach(() => {
      vi.mocked(useParams).mockReturnValue({ federationId: "fed-uuid-1" })
      useWorkloadIdentityMock.mockReturnValue({ data: existingFederation, isLoading: false })
    })

    it("pre-fills the form including the attribute mapping rows", async () => {
      renderWithProviders(<WorkloadIdentityAddOrUpdateForm />)

      await waitFor(() =>
        expect(screen.getByLabelText(/^name/i)).toHaveValue("github-actions"),
      )
      expect(screen.getByLabelText(/subject pattern/i)).toHaveValue("repo:my-org/my-repo:*")
      // The mapping arrives as structured rows, not a JSON blob.
      expect(screen.getByLabelText(/source claim in the workload token/i)).toHaveValue("repository")
      expect(screen.getByLabelText(/claim name to write in the issued token/i)).toHaveValue(
        "repository",
      )
    })

    // THE regression: the mapping used to be a JSON textarea whose parse failure was
    // swallowed, so a typo silently replaced a saved mapping with {} and still
    // reported success. A structured editor plus this guard makes that impossible.
    it("blocks the save when a mapping targets a reserved claim", async () => {
      const user = u()
      renderWithProviders(<WorkloadIdentityAddOrUpdateForm />)

      await waitFor(() =>
        expect(screen.getByLabelText(/source claim in the workload token/i)).toHaveValue(
          "repository",
        ),
      )

      const targetInput = screen.getByLabelText(/claim name to write in the issued token/i)
      await user.clear(targetInput)
      await user.type(targetInput, "sub")
      await user.click(screen.getByRole("button", { name: /save changes/i }))

      await waitFor(() =>
        expect(screen.getByRole("alert")).toHaveTextContent(/cannot be overridden/i),
      )
      expect(updateMutateAsync).not.toHaveBeenCalled()
      expect(showSuccessMock).not.toHaveBeenCalled()
    })

    it("submits the mapping as an object and navigates back on success", async () => {
      const user = u()
      updateMutateAsync.mockResolvedValue({})
      renderWithProviders(<WorkloadIdentityAddOrUpdateForm />)

      await waitFor(() =>
        expect(screen.getByLabelText(/source claim in the workload token/i)).toHaveValue(
          "repository",
        ),
      )
      await user.click(screen.getByRole("button", { name: /save changes/i }))

      await waitFor(() => expect(updateMutateAsync).toHaveBeenCalled())
      const payload = updateMutateAsync.mock.calls[0][0].data
      expect(payload.attribute_mapping).toEqual({ repository: "repository" })
      expect(payload.allowed_scopes).toEqual(["api:read"])
      expect(navigateMock).toHaveBeenCalledWith("/workload-identity")
    })

    // The mapped client is what the issued token acts as, so repointing it would
    // silently change which identity the federation grants.
    it("does not allow changing the mapped client", async () => {
      renderWithProviders(<WorkloadIdentityAddOrUpdateForm />)
      await waitFor(() => expect(screen.getByLabelText(/^name/i)).toHaveValue("github-actions"))
      expect(screen.getByText(/cannot be changed after creation/i)).toBeInTheDocument()
    })
  })
})
