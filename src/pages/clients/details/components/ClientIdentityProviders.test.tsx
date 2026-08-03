import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ClientIdentityProviders } from "./ClientIdentityProviders"
import type {
  ClientIdentityProviderConnection,
  ClientResponse,
} from "@/services/api/clients/types"

const { navigateMock, removeMutateAsync, updateMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    removeMutateAsync: vi.fn(),
    updateMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({ clientId: "c1" }),
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useClients", () => ({
  useRemoveClientIdentityProvider: () => ({ mutateAsync: removeMutateAsync, isPending: false }),
  useUpdateClientIdentityProvider: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

// ConnectProviderDialog manages its own flows; stub it so this tab is isolated.
vi.mock("./ConnectProviderDialog", () => ({
  ConnectProviderDialog: () => null,
}))

function makeConnection(
  overrides: Partial<ClientIdentityProviderConnection> = {},
  provider: Partial<ClientIdentityProviderConnection["identity_provider"]> = {},
): ClientIdentityProviderConnection {
  return {
    client_identity_provider_id: "conn1",
    is_default: false,
    enabled: true,
    display_order: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    identity_provider: {
      identity_provider_id: "idp1",
      name: "google-oauth",
      display_name: "Google",
      provider: "google",
      provider_type: "social",
      identifier: "google-identifier",
      status: "active",
      is_default: false,
      is_system: false,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
      ...provider,
    },
    ...overrides,
  }
}

function makeClient(overrides: Partial<ClientResponse> = {}): ClientResponse {
  return {
    client_id: "c1", name: "console", display_name: "Console",
    client_type: "spa", status: "active", is_default: false, is_system: false,
    allow_registration: true,
    allow_magic_link: false, created_at: "", updated_at: "", ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ClientIdentityProviders", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the empty state when nothing is connected", () => {
    renderWithProviders(<ClientIdentityProviders client={makeClient({ connections: [] })} />)
    expect(screen.getByText("No identity providers connected")).toBeInTheDocument()
  })

  it("renders a connection with provider, status, and connection badges", () => {
    const connection = makeConnection(
      { is_default: true, enabled: false },
      { display_name: "Google Login" },
    )
    renderWithProviders(<ClientIdentityProviders client={makeClient({ connections: [connection] })} />)
    expect(screen.getByText("Google Login")).toBeInTheDocument()
    // The provider label badge, distinct from the connection's display name.
    expect(screen.getByText("Google")).toBeInTheDocument()
    expect(screen.getByText("google-oauth")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
    expect(screen.getByText("Default")).toBeInTheDocument()
    expect(screen.getByText("Disabled")).toBeInTheDocument()
    expect(screen.getByText("conn1")).toBeInTheDocument()
  })

  it("navigates to the provider via the menu", async () => {
    renderWithProviders(
      <ClientIdentityProviders client={makeClient({ connections: [makeConnection()] })} />,
    )
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("View Provider"))
    expect(navigateMock).toHaveBeenCalledWith("/providers/identity/idp1")
  })

  it("hides the disconnect action for system provider connections", async () => {
    const connection = makeConnection({}, { is_system: true })
    renderWithProviders(<ClientIdentityProviders client={makeClient({ connections: [connection] })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    expect(await screen.findByText("View Provider")).toBeInTheDocument()
    expect(screen.queryByText("Disconnect")).not.toBeInTheDocument()
  })

  it("disconnects a provider with type-to-confirm", async () => {
    removeMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(
      <ClientIdentityProviders client={makeClient({ connections: [makeConnection()] })} />,
    )
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Disconnect"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "google-oauth")
    await u().click(within(dialog).getByRole("button", { name: "Disconnect" }))

    await waitFor(() =>
      expect(removeMutateAsync).toHaveBeenCalledWith({ clientId: "c1", connectionId: "conn1" }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Disconnected Google from Console")
  })

  it("toggles the inline editor and updates the default flag", async () => {
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(
      <ClientIdentityProviders client={makeClient({ connections: [makeConnection()] })} />,
    )
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Edit Connection"))

    const defaultSwitch = screen.getAllByRole("switch")[0]
    await u().click(defaultSwitch)

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith({
        clientId: "c1",
        connectionId: "conn1",
        data: { is_default: true },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Connection updated")
  })
})
