import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/utils"
import ClientDetailsPage from "./ClientDetailsPage"
import type { ClientResponse } from "@/services/api/clients/types"

const { useClientMock, navigateMock, searchParamsRef } = vi.hoisted(() => ({
  useClientMock: vi.fn(),
  navigateMock: vi.fn(),
  searchParamsRef: { current: new URLSearchParams() },
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({ clientId: "c1" }),
    useNavigate: () => navigateMock,
    useSearchParams: () => [searchParamsRef.current, vi.fn()],
  }
})

vi.mock("@/hooks/useClients", () => ({
  useClient: (...args: unknown[]) => useClientMock(...args),
}))

vi.mock("./components", () => ({
  ClientHeader: ({ client }: { client: ClientResponse }) => (
    <div data-testid="client-header">{client.display_name}</div>
  ),
  ClientInformation: () => <div data-testid="overview-tab" />,
  ClientIdentityProviders: () => <div data-testid="identity-providers-tab" />,
  ClientCredentials: () => <div data-testid="credentials-tab" />,
  ClientConfig: () => <div data-testid="config-tab" />,
  ClientUris: () => <div data-testid="uris-tab" />,
  ClientApis: () => <div data-testid="apis-tab" />,
  ClientRoles: () => <div data-testid="roles-tab" />,
  ClientMetadata: () => <div data-testid="metadata-tab" />,
}))

function makeClient(overrides: Partial<ClientResponse> = {}): ClientResponse {
  return {
    client_id: "c1", name: "console", display_name: "Console",
    client_type: "spa", status: "active", is_default: false, is_system: false,
    allow_registration: true,
    allow_magic_link: false, created_at: "", updated_at: "", ...overrides,
  }
}

describe("ClientDetailsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    searchParamsRef.current = new URLSearchParams()
    useClientMock.mockReturnValue({ data: undefined, isLoading: false })
  })

  it("renders the loading state", () => {
    useClientMock.mockReturnValue({ data: undefined, isLoading: true })
    const { container } = renderWithProviders(<ClientDetailsPage />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders the not-found state", () => {
    useClientMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ClientDetailsPage />)
    expect(screen.getByText("Client not found")).toBeInTheDocument()
  })

  it("renders the client header with display name", () => {
    useClientMock.mockReturnValue({ data: makeClient(), isLoading: false })
    renderWithProviders(<ClientDetailsPage />)
    expect(screen.getByTestId("client-header")).toHaveTextContent("Console")
  })

  it("renders all tab triggers", () => {
    useClientMock.mockReturnValue({ data: makeClient(), isLoading: false })
    renderWithProviders(<ClientDetailsPage />)
    for (const label of [
      "Overview", "Identity Providers", "Credentials", "OAuth & Tokens",
      "URIs & Origins", "API Permissions", "Roles", "Metadata",
    ]) {
      expect(screen.getByRole("tab", { name: label })).toBeInTheDocument()
    }
  })

  it("falls back to the overview tab for an unknown ?tab value", () => {
    searchParamsRef.current = new URLSearchParams("tab=bogus")
    useClientMock.mockReturnValue({ data: makeClient(), isLoading: false })
    renderWithProviders(<ClientDetailsPage />)
    expect(screen.getByTestId("overview-tab")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Overview" })).toHaveAttribute("aria-selected", "true")
  })

  it("activates the tab from the ?tab search param", () => {
    searchParamsRef.current = new URLSearchParams("tab=roles")
    useClientMock.mockReturnValue({ data: makeClient(), isLoading: false })
    renderWithProviders(<ClientDetailsPage />)
    expect(screen.getByTestId("roles-tab")).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Roles" })).toHaveAttribute("aria-selected", "true")
  })
})
