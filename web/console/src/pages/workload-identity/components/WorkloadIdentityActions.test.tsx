import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { WorkloadIdentityActions } from "./WorkloadIdentityActions"
import type { WorkloadIdentityFederation } from "@/services/api/workload-identity/types"

const { navigateMock, updateMutateAsync, deleteMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    updateMutateAsync: vi.fn(),
    deleteMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock(
  "react-router-dom",
  async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
    const actual = await importOriginal()
    return { ...actual, useParams: () => ({}), useNavigate: () => navigateMock }
  },
)

vi.mock("@/hooks/useWorkloadIdentity", () => ({
  useUpdateWorkloadIdentity: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
  useDeleteWorkloadIdentity: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeFederation(
  overrides: Partial<WorkloadIdentityFederation> = {},
): WorkloadIdentityFederation {
  return {
    workload_identity_federation_uuid: "fed-1",
    client_uuid: "client-1",
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
    ...overrides,
  }
}

const openMenu = async () => {
  const user = userEvent.setup()
  await user.click(screen.getByRole("button", { name: /open menu|actions/i }))
  return user
}

beforeEach(() => vi.clearAllMocks())

describe("WorkloadIdentityActions", () => {
  it("offers view, edit, deactivate and delete for an active federation", async () => {
    renderWithProviders(<WorkloadIdentityActions federation={makeFederation()} />)
    await openMenu()

    expect(screen.getByText("View Details")).toBeInTheDocument()
    expect(screen.getByText("Edit Federation")).toBeInTheDocument()
    expect(screen.getByText("Deactivate Federation")).toBeInTheDocument()
    expect(screen.getByText("Delete Federation")).toBeInTheDocument()
    expect(screen.queryByText("Activate Federation")).not.toBeInTheDocument()
  })

  it("offers activate instead when the federation is inactive", async () => {
    renderWithProviders(
      <WorkloadIdentityActions federation={makeFederation({ is_active: false })} />,
    )
    await openMenu()

    expect(screen.getByText("Activate Federation")).toBeInTheDocument()
    expect(screen.queryByText("Deactivate Federation")).not.toBeInTheDocument()
  })

  it("navigates to details and edit", async () => {
    renderWithProviders(<WorkloadIdentityActions federation={makeFederation()} />)
    const user = await openMenu()

    await user.click(screen.getByText("Edit Federation"))
    expect(navigateMock).toHaveBeenCalledWith("/workload-identity/fed-1/edit")
  })

  // Update is a full-replace PUT, so a status toggle must resend every field or the
  // omitted ones would be cleared server-side.
  it("resends the whole federation when toggling status", async () => {
    updateMutateAsync.mockResolvedValue({})
    renderWithProviders(<WorkloadIdentityActions federation={makeFederation()} />)
    const user = await openMenu()

    await user.click(screen.getByText("Deactivate Federation"))
    await user.click(screen.getByRole("button", { name: /^deactivate$/i }))

    await waitFor(() => expect(updateMutateAsync).toHaveBeenCalled())
    const { federationId, data } = updateMutateAsync.mock.calls[0][0]
    expect(federationId).toBe("fed-1")
    expect(data.is_active).toBe(false)
    expect(data.subject_pattern).toBe("repo:my-org/my-repo:*")
    expect(data.allowed_scopes).toEqual(["api:read"])
    expect(data.attribute_mapping).toEqual({ repository: "repository" })
  })

  // Deleting instantly revokes machine auth for a live workload with no credential to
  // fall back on, so it uses the type-to-confirm dialog.
  it("requires typing the name to confirm deletion", async () => {
    deleteMutateAsync.mockResolvedValue({})
    renderWithProviders(<WorkloadIdentityActions federation={makeFederation()} />)
    const user = await openMenu()

    await user.click(screen.getByText("Delete Federation"))

    const confirmButton = screen.getByRole("button", { name: /^delete$/i })
    expect(confirmButton).toBeDisabled()

    await user.type(screen.getByRole("textbox"), "github-actions")
    await waitFor(() => expect(confirmButton).toBeEnabled())
    await user.click(confirmButton)

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("fed-1"))
    expect(showSuccessMock).toHaveBeenCalled()
  })

  it("surfaces an error and does not report success when delete rejects", async () => {
    deleteMutateAsync.mockRejectedValue(new Error("boom"))
    renderWithProviders(<WorkloadIdentityActions federation={makeFederation()} />)
    const user = await openMenu()

    await user.click(screen.getByText("Delete Federation"))
    await user.type(screen.getByRole("textbox"), "github-actions")
    await user.click(screen.getByRole("button", { name: /^delete$/i }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalled())
    expect(showSuccessMock).not.toHaveBeenCalled()
  })
})
