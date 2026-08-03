import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ClientHeader } from "./ClientHeader"
import type { ClientResponse } from "@/services/api/clients/types"

const { navigateMock, deleteMutateAsync, updateStatusMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    deleteMutateAsync: vi.fn(),
    updateStatusMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useClients", () => ({
  useDeleteClient: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateClientStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useBranding", () => ({
  useBrandings: () => ({ data: [{ branding_id: "b1", name: "Acme Branding" }] }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeClient(overrides: Partial<ClientResponse> = {}): ClientResponse {
  return {
    client_id: "c1", name: "console", display_name: "Console",
    client_type: "traditional", status: "active", is_default: false, is_system: false,
    allow_registration: true,
    allow_magic_link: false,
    created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ClientHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders the display name, system name, status, and attributes", () => {
    renderWithProviders(<ClientHeader client={makeClient()} clientId="c1" />)
    expect(screen.getByText("Console")).toBeInTheDocument()
    expect(screen.getByText("console")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
    expect(screen.getByText("Traditional Web")).toBeInTheDocument()
    expect(screen.getByText("Tenant's active branding")).toBeInTheDocument()
  })

  it("shows system and default badges and the resolved branding name", () => {
    renderWithProviders(
      <ClientHeader
        client={makeClient({ is_system: true, is_default: true, branding_id: "b1" })}
        clientId="c1"
      />,
    )
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText("Default")).toBeInTheDocument()
    expect(screen.getByText("Acme Branding")).toBeInTheDocument()
  })

  it("navigates to edit with router state", async () => {
    renderWithProviders(<ClientHeader client={makeClient()} clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /edit/i }))
    expect(navigateMock).toHaveBeenCalledWith("/clients/c1/edit", expect.objectContaining({
      state: expect.objectContaining({ from: "/clients/c1" }),
    }))
  })

  it("activates an inactive client with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ClientHeader client={makeClient({ status: "inactive" })} clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Activate Client"))
    await u().click(screen.getByRole("button", { name: "Activate" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        clientId: "c1",
        data: { status: "active" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Client status updated to active")
  })

  it("deactivates an active client with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ClientHeader client={makeClient({ status: "active" })} clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Deactivate Client"))
    await u().click(screen.getByRole("button", { name: "Deactivate" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        clientId: "c1",
        data: { status: "inactive" },
      }),
    )
  })

  it("shows an error when the status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ClientHeader client={makeClient({ status: "active" })} clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Deactivate Client"))
    await u().click(screen.getByRole("button", { name: "Deactivate" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes the client, shows success, and navigates back", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ClientHeader client={makeClient({ name: "console" })} clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete Client"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "console")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("c1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Client deleted successfully")
    expect(navigateMock).toHaveBeenCalledWith("/clients")
  })

  it("shows an error and does not navigate when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ClientHeader client={makeClient({ name: "console" })} clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /open actions/i }))
    await u().click(await screen.findByText("Delete Client"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "console")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it("hides the actions menu for system clients", () => {
    renderWithProviders(<ClientHeader client={makeClient({ is_system: true })} clientId="c1" />)
    expect(screen.queryByRole("button", { name: /open actions/i })).not.toBeInTheDocument()
  })

  it("hides the actions menu for the active default client", () => {
    renderWithProviders(
      <ClientHeader client={makeClient({ is_default: true, status: "active" })} clientId="c1" />,
    )
    expect(screen.queryByRole("button", { name: /open actions/i })).not.toBeInTheDocument()
  })
})
