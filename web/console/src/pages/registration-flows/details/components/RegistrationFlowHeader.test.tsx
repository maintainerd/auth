import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { RegistrationFlowHeader } from "./RegistrationFlowHeader"
import type { RegistrationFlowDetail } from "@/services/api/registration-flows/types"

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return { ...actual, useNavigate: () => vi.fn() }
})

vi.mock("@/hooks/useRegistrationFlows", () => ({
  useDeleteRegistrationFlow: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateRegistrationFlowStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeFlow(overrides: Partial<RegistrationFlowDetail> = {}): RegistrationFlowDetail {
  return {
    registration_flow_id: "f1",
    name: "seller-signup",
    description: "Sellers onboard here",
    status: "active",
    client_id: "3f1b0c8e-2f4a-4a9b-9c1d-8e7f6a5b4c3d",
    verification_required: true,
    required_fields: ["email"],
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-02-01T00:00:00Z",
    client: {
      client_id: "3f1b0c8e-2f4a-4a9b-9c1d-8e7f6a5b4c3d",
      name: "storefront",
      display_name: "Storefront",
      identifier: "storefront-abc123",
      status: "active",
    },
    ...overrides,
  }
}

describe("RegistrationFlowHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the flow name and status", () => {
    renderWithProviders(<RegistrationFlowHeader registrationFlow={makeFlow()} registrationFlowId="f1" />)

    expect(screen.getByText("seller-signup")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
  })

  it("shows the client by display name, linked, with its identifier as secondary text", () => {
    renderWithProviders(<RegistrationFlowHeader registrationFlow={makeFlow()} registrationFlowId="f1" />)

    const link = screen.getByRole("link", { name: "Storefront" })
    expect(link).toHaveAttribute("href", "/clients/3f1b0c8e-2f4a-4a9b-9c1d-8e7f6a5b4c3d")
    expect(screen.getByText("storefront-abc123")).toBeInTheDocument()
    // The raw UUID is no longer the client's visible label.
    expect(screen.queryByText("3f1b0c8e-2f4a-4a9b-9c1d-8e7f6a5b4c3d")).not.toBeInTheDocument()
  })

  it("falls back to the client name when it has no display name", () => {
    const flow = makeFlow()
    renderWithProviders(
      <RegistrationFlowHeader
        registrationFlow={{ ...flow, client: { ...flow.client!, display_name: undefined } }}
        registrationFlowId="f1"
      />,
    )
    expect(screen.getByRole("link", { name: "storefront" })).toBeInTheDocument()
  })

  it("renders a dash when the flow has no client", () => {
    renderWithProviders(
      <RegistrationFlowHeader registrationFlow={makeFlow({ client: undefined })} registrationFlowId="f1" />,
    )
    expect(screen.getByText("—")).toBeInTheDocument()
    expect(screen.queryByRole("link", { name: /storefront/i })).not.toBeInTheDocument()
  })

  it("marks a system flow and hides its destructive actions", () => {
    renderWithProviders(
      <RegistrationFlowHeader registrationFlow={makeFlow({ is_system: true })} registrationFlowId="f1" />,
    )
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /open actions/i })).not.toBeInTheDocument()
  })
})
