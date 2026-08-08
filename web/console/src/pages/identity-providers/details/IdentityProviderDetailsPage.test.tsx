import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import IdentityProviderDetailsPage from "./IdentityProviderDetailsPage"
import type { IdentityProviderDetailResponse } from "@/services/api/identity-providers/types"
import type { ClientListResponse } from "@/services/api/clients/types"

const { useIdentityProviderMock, useClientsMock, navigateMock } = vi.hoisted(() => ({
  useIdentityProviderMock: vi.fn(),
  useClientsMock: vi.fn(),
  navigateMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useIdentityProviders", () => ({
  useIdentityProvider: (...args: unknown[]) => useIdentityProviderMock(...args),
  useDeleteIdentityProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateIdentityProviderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useClients", () => ({
  useClients: (params: Record<string, unknown>) => useClientsMock(params),
  useRemoveClientIdentityProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useAddClientIdentityProvider: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function seededSystemProvider(): IdentityProviderDetailResponse {
  return {
    identity_provider_id: "50b31489-3516-4c25-9dd2-510e8eba1c9d",
    name: "maintainerd",
    display_name: "Built-in Authentication System",
    provider: "maintainerd",
    provider_type: "system",
    identifier: "bjcz6Pgn2JNeUzX",
    issuer: "",
    provider_client_id: "",
    allow_jit_provisioning: false,
    allow_registration: true,
    allow_token_federation: false,
    allowed_audiences: [],
    email_domains: [],
    config: {
      allow_login: true,
      require_mfa: false,
      allow_registration: true,
      allow_magic_link: false,
      max_login_attempts: 5,
      refresh_timeout_min: 1440,
      session_timeout_min: 60,
      allow_mfa_enrollment: true,
      allow_password_reset: true,
      lockout_duration_min: 15,
      require_email_verify: false,
      require_phone_verify: false,
    },
    tenant: null,
    status: "active",
    is_default: true,
    is_system: true,
    created_at: "2026-07-27T07:58:23.228308Z",
    updated_at: "2026-07-27T07:58:23.228308Z",
  }
}

describe("IdentityProviderDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useClientsMock.mockReturnValue({
      data: {
        rows: [],
        total: 0,
        page: 1,
        limit: 10,
        total_pages: 0,
      } satisfies ClientListResponse,
      isLoading: false,
      isError: false,
    })
  })

  it("renders the seeded system identity provider details without throwing", () => {
    useIdentityProviderMock.mockReturnValue({
      data: seededSystemProvider(),
      isLoading: false,
      isError: false,
    })

    renderWithProviders(<IdentityProviderDetailsPage />, {
      route: "/providers/identity/50b31489-3516-4c25-9dd2-510e8eba1c9d",
      path: "/providers/identity/:providerId",
    })

    expect(screen.getByText("Built-in Authentication System")).toBeInTheDocument()
    expect(screen.getAllByText("Built-in Authentication").length).toBeGreaterThan(0)
    expect(screen.getByRole("tab", { name: "Connection" })).toHaveAttribute("aria-selected", "true")
  })
})
