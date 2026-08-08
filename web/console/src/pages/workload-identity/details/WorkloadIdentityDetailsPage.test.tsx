import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useParams } from "react-router-dom"
import { renderWithProviders } from "@/test/utils"
import WorkloadIdentityDetailsPage from "./WorkloadIdentityDetailsPage"

const { useWorkloadIdentityMock, navigateMock } = vi.hoisted(() => ({
  useWorkloadIdentityMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: vi.fn(() => ({ federationId: "fed-1" })),
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useWorkloadIdentity", () => ({
  useWorkloadIdentity: (...args: unknown[]) => useWorkloadIdentityMock(...args),
  useUpdateWorkloadIdentity: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteWorkloadIdentity: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

const federation = {
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
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(useParams).mockReturnValue({ federationId: "fed-1" })
  useWorkloadIdentityMock.mockReturnValue({ data: federation, isLoading: false, isError: false })
})

describe("WorkloadIdentityDetailsPage", () => {
  it("renders the header and both tabs", () => {
    renderWithProviders(<WorkloadIdentityDetailsPage />)
    expect(screen.getByText("github-actions")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: /trust/i })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: /issued token/i })).toBeInTheDocument()
  })

  it("shows the trust boundary on the default tab", () => {
    renderWithProviders(<WorkloadIdentityDetailsPage />)
    expect(screen.getByText("https://token.actions.githubusercontent.com")).toBeInTheDocument()
    expect(screen.getByText("repo:my-org/my-repo:*")).toBeInTheDocument()
  })

  it("shows scopes and the attribute mapping on the issued-token tab", async () => {
    const user = userEvent.setup()
    renderWithProviders(<WorkloadIdentityDetailsPage />)

    await user.click(screen.getByRole("tab", { name: /issued token/i }))

    await waitFor(() => expect(screen.getByText("api:read")).toBeInTheDocument())
    expect(screen.getByText(/repository/)).toBeInTheDocument()
  })

  it("renders the not-found state when the federation is missing", () => {
    useWorkloadIdentityMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<WorkloadIdentityDetailsPage />)
    expect(screen.getByText("Federation not found")).toBeInTheDocument()
  })

  it("does not render content while loading", () => {
    useWorkloadIdentityMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    renderWithProviders(<WorkloadIdentityDetailsPage />)
    expect(screen.queryByText("github-actions")).not.toBeInTheDocument()
  })
})
