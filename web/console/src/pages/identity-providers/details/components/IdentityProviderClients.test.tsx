import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { IdentityProviderClients } from "./IdentityProviderClients"
import type {
  Client,
  ClientIdentityProviderConnection,
} from "@/services/api/clients/types"
import type { ClientListResponse } from "@/services/api/clients/types"

const { navigateMock, removeMutateAsync, useClientsMock } = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  removeMutateAsync: vi.fn(),
  useClientsMock: vi.fn(),
}))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useClients", () => ({
  useClients: (params: Record<string, unknown>) => useClientsMock(params),
  useRemoveClientIdentityProvider: () => ({ mutateAsync: removeMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

vi.mock("./ConnectClientDialog", () => ({
  ConnectClientDialog: () => null,
}))

function makeConnection(
  overrides: Partial<ClientIdentityProviderConnection> = {},
): ClientIdentityProviderConnection {
  return {
    client_identity_provider_id: "conn1",
    identity_provider: {
      identity_provider_id: "idp1",
      name: "google-oauth",
      display_name: "Google",
      provider: "google",
      provider_type: "social",
      identifier: "google",
      status: "active",
      is_default: false,
      is_system: false,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    },
    is_default: false,
    enabled: true,
    display_order: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function makeClient(overrides: Partial<Client> = {}): Client {
  return {
    client_id: "client1",
    name: "console",
    display_name: "Console",
    client_type: "spa",
    status: "active",
    is_default: false,
    is_system: false,
    allow_registration: true,
    allow_magic_link: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function mockClients(rows: Client[]) {
  useClientsMock.mockReturnValue({
    data: {
      rows,
      total: rows.length,
      page: 1,
      limit: 10,
      total_pages: 1,
    } satisfies ClientListResponse,
    isLoading: false,
    isError: false,
  })
}

describe("IdentityProviderClients", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders connected clients and opens client details", async () => {
    mockClients([makeClient({ connections: [makeConnection()] })])

    renderWithProviders(<IdentityProviderClients providerId="idp1" providerName="Google" />)

    expect(screen.getByText("Console")).toBeInTheDocument()
    await userEvent.click(screen.getByRole("button", { name: /console/i }))
    expect(navigateMock).toHaveBeenCalledWith("/clients/client1")
  })

  it("does not crash when the filtered client row omits the nested identity provider relation", () => {
    const partialConnection = {
      ...makeConnection(),
      identity_provider: undefined,
    } as unknown as ClientIdentityProviderConnection
    mockClients([makeClient({ connections: [partialConnection] })])

    renderWithProviders(<IdentityProviderClients providerId="idp1" providerName="Google" />)

    expect(screen.getByText("Console")).toBeInTheDocument()
    expect(screen.queryByText("Disconnect")).not.toBeInTheDocument()
  })
})
