import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { RegistrationFlowLink } from "./RegistrationFlowLink"
import type { RegistrationFlowDetail } from "@/services/api/registration-flows/types"

const { showSuccessMock, identityUrlRef } = vi.hoisted(() => ({
  showSuccessMock: vi.fn(),
  identityUrlRef: { current: "https://auth.tenant.example.com" as string | null },
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: vi.fn() }),
}))

vi.mock("@/store/hooks", () => ({
  useAppSelector: (selector: (state: never) => unknown) =>
    selector({ tenant: { identityUrl: identityUrlRef.current } } as never),
}))

vi.mock("@/services/api/config", () => ({
  API_CONFIG: { IDENTITY_BASE_URL: "https://fallback.example.com" },
}))

function makeFlow(overrides: Partial<RegistrationFlowDetail> = {}): RegistrationFlowDetail {
  return {
    registration_flow_id: "f1",
    name: "seller-signup",
    description: "Sellers onboard here",
    status: "active",
    client_id: "3f1b0c8e-2f4a-4a9b-9c1d-8e7f6a5b4c3d",
    verification_required: true,
    is_system: false,
    required_fields: ["email"],
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
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

const EXPECTED_URL =
  "https://auth.tenant.example.com/register?client_id=storefront-abc123&registration_flow=seller-signup"

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("RegistrationFlowLink", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    identityUrlRef.current = "https://auth.tenant.example.com"
  })

  it("composes the link from the tenant identity URL, client identifier and flow name", () => {
    renderWithProviders(<RegistrationFlowLink flow={makeFlow()} />)
    expect(screen.getByText(EXPECTED_URL)).toBeInTheDocument()
  })

  it("uses the client identifier, not the client UUID", () => {
    renderWithProviders(<RegistrationFlowLink flow={makeFlow()} />)
    const link = screen.getByText(EXPECTED_URL).textContent!
    expect(link).toContain("client_id=storefront-abc123")
    expect(link).not.toContain("3f1b0c8e-2f4a-4a9b-9c1d-8e7f6a5b4c3d")
  })

  it("falls back to the configured identity base URL when the tenant has none", () => {
    identityUrlRef.current = null
    renderWithProviders(<RegistrationFlowLink flow={makeFlow()} />)
    expect(
      screen.getByText(
        "https://fallback.example.com/register?client_id=storefront-abc123&registration_flow=seller-signup",
      ),
    ).toBeInTheDocument()
  })

  it("copies the composed link to the clipboard", async () => {
    const user = u()
    renderWithProviders(<RegistrationFlowLink flow={makeFlow()} />)

    await user.click(screen.getByRole("button", { name: /copy registration link/i }))

    await waitFor(() =>
      expect(showSuccessMock).toHaveBeenCalledWith("Registration link copied to clipboard"),
    )
    await expect(navigator.clipboard.readText()).resolves.toBe(EXPECTED_URL)
  })

  it("offers an Open action pointing at the composed link", () => {
    renderWithProviders(<RegistrationFlowLink flow={makeFlow()} />)
    const open = screen.getByRole("link", { name: /open/i })
    expect(open).toHaveAttribute("href", EXPECTED_URL)
    expect(open).toHaveAttribute("target", "_blank")
    expect(open).toHaveAttribute("rel", expect.stringContaining("noopener"))
  })

  it("warns that renaming breaks the link and deactivation is the kill switch", () => {
    renderWithProviders(<RegistrationFlowLink flow={makeFlow()} />)
    expect(screen.getByText(/renaming the flow breaks it/i)).toBeInTheDocument()
    expect(screen.getByText(/deactivate this flow/i)).toBeInTheDocument()
  })

  it("states that an inactive flow already refuses registrations", () => {
    renderWithProviders(<RegistrationFlowLink flow={makeFlow({ status: "inactive" })} />)
    expect(screen.getByText(/currently refuses all registrations/i)).toBeInTheDocument()
  })

  it("renders no link and no actions when the flow has no client", () => {
    renderWithProviders(<RegistrationFlowLink flow={makeFlow({ client: undefined })} />)
    expect(screen.getByText(/no client attached/i)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /^copy$/i })).not.toBeInTheDocument()
    expect(screen.queryByRole("link", { name: /open/i })).not.toBeInTheDocument()
  })

  it("flags a system flow as not redeemable through a self-service link", () => {
    renderWithProviders(<RegistrationFlowLink flow={makeFlow({ is_system: true })} />)
    expect(screen.getByText(/only through an invite/i)).toBeInTheDocument()
  })
})
