import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import RegistrationFlowDetailsPage from "./RegistrationFlowDetailsPage"
import type { RegistrationFlowDetail } from "@/services/api/registration-flows/types"

const { useRegistrationFlowMock, navigateMock } = vi.hoisted(() => ({
  useRegistrationFlowMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({ registrationFlowId: "f1" }),
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useRegistrationFlows", () => ({
  useRegistrationFlow: (...args: unknown[]) => useRegistrationFlowMock(...args),
  useRegistrationFlowRoles: () => ({ data: { rows: [], total: 0, total_pages: 0 }, isLoading: false, isError: false }),
  useRemoveRegistrationFlowRole: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useAssignRegistrationFlowRoles: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteRegistrationFlow: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateRegistrationFlowStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useRoles", () => ({
  useRoles: () => ({ data: { rows: [], total: 0 }, isLoading: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

vi.mock("@/store/hooks", () => ({
  useAppSelector: (selector: (state: never) => unknown) =>
    selector({ tenant: { identityUrl: "https://auth.tenant.example.com" } } as never),
}))

function makeFlow(overrides: Partial<RegistrationFlowDetail> = {}): RegistrationFlowDetail {
  return {
    registration_flow_id: "f1",
    name: "seller-signup",
    description: "Sellers onboard here",
    status: "active",
    client_id: "c-uuid",
    verification_required: true,
    required_fields: ["email"],
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    client: {
      client_id: "c-uuid",
      name: "storefront",
      display_name: "Storefront",
      identifier: "storefront-abc123",
      status: "active",
    },
    ...overrides,
  }
}

describe("RegistrationFlowDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useRegistrationFlowMock.mockReturnValue({ data: makeFlow(), isLoading: false, isError: false })
  })

  it("renders the loading state", () => {
    useRegistrationFlowMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<RegistrationFlowDetailsPage />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders the not-found state", () => {
    useRegistrationFlowMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<RegistrationFlowDetailsPage />)
    expect(screen.getByText("Registration flow not found")).toBeInTheDocument()
  })

  it("uses a back label that names the flows listing", () => {
    renderWithProviders(<RegistrationFlowDetailsPage />)
    expect(screen.getByRole("button", { name: /back to registration flows/i })).toBeInTheDocument()
  })

  it("shows the registration link on the default tab", () => {
    renderWithProviders(<RegistrationFlowDetailsPage />)
    expect(screen.getByText("Registration Link")).toBeInTheDocument()
    expect(
      screen.getByText(
        "https://auth.tenant.example.com/register?client_id=storefront-abc123&registration_flow=seller-signup",
      ),
    ).toBeInTheDocument()
  })

  it("defaults to the configuration tab for an unknown ?tab= value", () => {
    renderWithProviders(<RegistrationFlowDetailsPage />, { route: "/?tab=not-a-tab" })
    // An unvalidated tab value would leave every panel hidden.
    expect(screen.getByText("Registration Link")).toBeInTheDocument()
  })

  it("honours a valid ?tab= value", () => {
    renderWithProviders(<RegistrationFlowDetailsPage />, { route: "/?tab=roles" })
    expect(screen.getByText(/roles automatically assigned/i)).toBeInTheDocument()
  })
})
