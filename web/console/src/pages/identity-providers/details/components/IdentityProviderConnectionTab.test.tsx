import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import { IdentityProviderConnectionTab } from "./IdentityProviderConnectionTab"
import type { IdentityProviderDetail } from "@/services/api/identity-providers/types"

const { samlCardMock } = vi.hoisted(() => ({ samlCardMock: vi.fn() }))

vi.mock("./SamlServiceProviderCard", () => ({
  SamlServiceProviderCard: (props: { identifier: string }) => {
    samlCardMock(props)
    return <div data-testid="saml-sp-card">{props.identifier}</div>
  },
}))

function provider(overrides: Partial<IdentityProviderDetail> = {}): IdentityProviderDetail {
  return {
    identity_provider_id: "idp-1",
    name: "okta-sso",
    display_name: "Okta SSO",
    provider: "saml",
    provider_type: "saml",
    identifier: "okta-sso",
    issuer: null,
    provider_client_id: null,
    allow_jit_provisioning: true,
    allow_registration: false,
    allow_token_federation: false,
    allowed_audiences: [],
    email_domains: [],
    config: {
      sso_url: "https://idp.example.com/saml2/sso",
      entity_id: "https://idp.example.com/saml2/metadata",
    },
    tenant: null,
    status: "active",
    is_default: false,
    is_system: false,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

describe("IdentityProviderConnectionTab", () => {
  beforeEach(() => vi.clearAllMocks())

  it("publishes the service provider details for a SAML provider", () => {
    // Without this an admin had no way to learn Maintainerd's entity ID or ACS
    // URL from the console, so a SAML connection could not be completed at all.
    renderWithProviders(<IdentityProviderConnectionTab provider={provider()} />)

    expect(screen.getByTestId("saml-sp-card")).toBeInTheDocument()
    expect(samlCardMock).toHaveBeenCalledWith({ identifier: "okta-sso" })
  })

  it("does not call a SAML provider a Maintainerd-managed system provider", () => {
    // SAML has no OIDC connection schema, and the tab used to fall through to
    // the system-provider copy — which is false for a tenant-configured
    // enterprise connection and sent admins looking in the wrong place.
    renderWithProviders(<IdentityProviderConnectionTab provider={provider()} />)

    expect(screen.queryByText(/managed by Maintainerd/i)).not.toBeInTheDocument()
    expect(screen.getByText(/Configuration tab/i)).toBeInTheDocument()
  })

  it("keeps the system-provider notice for the built-in provider", () => {
    renderWithProviders(
      <IdentityProviderConnectionTab
        provider={provider({
          provider: "maintainerd",
          provider_type: "system",
          is_system: true,
        })}
      />,
    )

    expect(screen.getByText(/managed by Maintainerd/i)).toBeInTheDocument()
    expect(screen.queryByTestId("saml-sp-card")).not.toBeInTheDocument()
  })

  it("renders the OIDC broker fields and no SP card for a social provider", () => {
    renderWithProviders(
      <IdentityProviderConnectionTab
        provider={provider({
          provider: "google",
          provider_type: "social",
          issuer: "https://accounts.google.com",
          provider_client_id: "google-client-id",
        })}
      />,
    )

    expect(screen.getByText("https://accounts.google.com")).toBeInTheDocument()
    expect(screen.queryByTestId("saml-sp-card")).not.toBeInTheDocument()
  })

  it("treats a saml provider_type as SAML even when the provider key differs", () => {
    renderWithProviders(
      <IdentityProviderConnectionTab
        provider={provider({ provider: "maintainerd", provider_type: "saml" })}
      />,
    )

    expect(screen.getByTestId("saml-sp-card")).toBeInTheDocument()
  })
})
