import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { SamlServiceProviderCard } from "./SamlServiceProviderCard"
import type { SamlServiceProviderDetails } from "@/services/api/identity-providers"

const { useSamlServiceProviderMetadataMock, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  useSamlServiceProviderMetadataMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/useIdentityProviders", () => ({
  useSamlServiceProviderMetadata: (...args: unknown[]) =>
    useSamlServiceProviderMetadataMock(...args),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

const BASE = "https://identity-api.auth.maintainerd.local"

function metadata(overrides: Partial<SamlServiceProviderDetails> = {}): SamlServiceProviderDetails {
  return {
    entityId: `${BASE}/federation/saml/metadata/okta-sso`,
    acsUrl: `${BASE}/federation/saml/acs/okta-sso`,
    metadataUrl: `${BASE}/federation/saml/metadata/okta-sso`,
    sloUrl: null,
    nameIdFormats: ["urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"],
    authnRequestsSigned: false,
    wantAssertionsSigned: true,
    xml: "<EntityDescriptor/>",
    ...overrides,
  }
}

function mockQuery(overrides: Record<string, unknown> = {}) {
  useSamlServiceProviderMetadataMock.mockReturnValue({
    data: undefined,
    isLoading: false,
    isError: false,
    error: null,
    isFetching: false,
    refetch: vi.fn(),
    ...overrides,
  })
}

describe("SamlServiceProviderCard", () => {
  beforeEach(() => vi.clearAllMocks())

  it("surfaces the entity ID, ACS URL and metadata URL the upstream IdP needs", () => {
    mockQuery({ data: metadata() })

    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)

    expect(screen.getByText("SP Entity ID")).toBeInTheDocument()
    expect(screen.getByText("ACS URL")).toBeInTheDocument()
    expect(screen.getByText("SP Metadata URL")).toBeInTheDocument()
    expect(
      screen.getAllByText(`${BASE}/federation/saml/metadata/okta-sso`).length,
    ).toBeGreaterThan(0)
    expect(screen.getByText(`${BASE}/federation/saml/acs/okta-sso`)).toBeInTheDocument()
  })

  it("reads the values from the published metadata rather than composing them", () => {
    // The console does not know the authorization server's public hostname, so
    // whatever host the backend advertises is what must be shown — including a
    // host that is not the console's own origin.
    mockQuery({
      data: metadata({
        entityId: "https://sso.acme.example/federation/saml/metadata/acme",
        acsUrl: "https://sso.acme.example/federation/saml/acs/acme",
        metadataUrl: "https://sso.acme.example/federation/saml/metadata/acme",
      }),
    })

    renderWithProviders(<SamlServiceProviderCard identifier="acme" />)

    expect(screen.getByText("https://sso.acme.example/federation/saml/acs/acme")).toBeInTheDocument()
  })

  it("passes the provider identifier to the metadata query", () => {
    mockQuery({ data: metadata() })

    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)

    expect(useSamlServiceProviderMetadataMock).toHaveBeenCalledWith("okta-sso", true)
  })

  it("shows the signature expectations so the IdP is configured to match", () => {
    mockQuery({ data: metadata({ authnRequestsSigned: true }) })

    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)

    expect(screen.getByText("Signed assertions required")).toBeInTheDocument()
    expect(screen.getByText("AuthnRequests are signed")).toBeInTheDocument()
  })

  it("shows the Single Logout URL only when the SP publishes one", () => {
    mockQuery({ data: metadata() })
    const { unmount } = renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)
    expect(screen.queryByText("Single Logout URL")).not.toBeInTheDocument()
    unmount()

    mockQuery({ data: metadata({ sloUrl: `${BASE}/federation/saml/slo/okta-sso` }) })
    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)
    expect(screen.getByText("Single Logout URL")).toBeInTheDocument()
  })

  it("copies the raw metadata XML for IdPs that import a document", async () => {
    const user = userEvent.setup()
    mockQuery({ data: metadata({ xml: "<EntityDescriptor>real</EntityDescriptor>" }) })

    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)
    await user.click(screen.getByRole("button", { name: /copy xml/i }))

    await waitFor(() => expect(showSuccessMock).toHaveBeenCalled())
    await expect(navigator.clipboard.readText()).resolves.toBe(
      "<EntityDescriptor>real</EntityDescriptor>",
    )
  })

  it("downloads the metadata document for IdPs that only accept a file upload", async () => {
    const user = userEvent.setup()
    const createObjectURL = vi.fn().mockReturnValue("blob:metadata")
    const revokeObjectURL = vi.fn()
    vi.stubGlobal("URL", Object.assign(URL, { createObjectURL, revokeObjectURL }))
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {})
    mockQuery({ data: metadata() })

    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)
    await user.click(screen.getByRole("button", { name: /download/i }))

    expect(createObjectURL).toHaveBeenCalled()
    expect(click).toHaveBeenCalled()
    // The object URL pins the blob for the life of the document if not revoked.
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:metadata")

    click.mockRestore()
    vi.unstubAllGlobals()
  })

  it("prints no URLs at all when the metadata cannot be fetched", async () => {
    // Fail closed: a guessed entity ID or ACS URL would be pasted into a
    // production IdP and only fail at the end of a user's login.
    const refetch = vi.fn()
    mockQuery({ isError: true, error: new Error("Request failed with status code 404"), refetch })

    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)

    expect(screen.queryByText("SP Entity ID")).not.toBeInTheDocument()
    expect(screen.queryByText("ACS URL")).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /copy xml/i })).not.toBeInTheDocument()
    expect(screen.getByRole("alert")).toHaveTextContent(/could not publish/i)

    await userEvent.setup().click(screen.getByRole("button", { name: /try again/i }))
    expect(refetch).toHaveBeenCalled()
  })

  it("says the metadata URL is unavailable instead of inventing one", () => {
    mockQuery({ data: metadata({ metadataUrl: null }) })

    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)

    expect(screen.getByText(/not published/i)).toBeInTheDocument()
    expect(
      screen.queryByRole("link", { name: /open service provider metadata/i }),
    ).not.toBeInTheDocument()
  })

  it("links out to the metadata document when one is published", () => {
    mockQuery({ data: metadata() })

    renderWithProviders(<SamlServiceProviderCard identifier="okta-sso" />)

    expect(
      screen.getByRole("link", { name: /open service provider metadata/i }),
    ).toHaveAttribute("href", `${BASE}/federation/saml/metadata/okta-sso`)
  })
})
