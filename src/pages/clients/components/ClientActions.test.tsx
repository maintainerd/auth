import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ClientActions } from "./ClientActions"
import type { Client } from "@/services/api/clients/types"

const { navigateMock, updateStatusMutateAsync, deleteMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    navigateMock: vi.fn(),
    updateStatusMutateAsync: vi.fn(),
    deleteMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async (importOriginal: () => Promise<typeof import("react-router-dom")>) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({}),
    useNavigate: () => navigateMock,
  }
})

vi.mock("@/hooks/useClients", () => ({
  useDeleteClient: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useUpdateClientStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeClient(overrides: Partial<Client> = {}): Client {
  return {
    client_id: "c1",
    name: "console",
    display_name: "Console",
    client_type: "spa",
    status: "active",
    is_default: false,
    is_system: false,
    allow_registration: true,
    allow_magic_link: false,
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ClientActions", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("navigates to view details", async () => {
    renderWithProviders(<ClientActions client={makeClient()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("View Details"))
    expect(navigateMock).toHaveBeenCalledWith("/clients/c1")
  })

  it("navigates to edit", async () => {
    renderWithProviders(<ClientActions client={makeClient()} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Edit Client"))
    expect(navigateMock).toHaveBeenCalledWith("/clients/c1/edit")
  })

  it("deactivates an active client with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ClientActions client={makeClient({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Client"))
    await u().click(screen.getByRole("button", { name: "Deactivate" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        clientId: "c1",
        data: { status: "inactive" },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Client status updated to inactive")
  })

  it("activates an inactive client with confirmation", async () => {
    updateStatusMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ClientActions client={makeClient({ status: "inactive" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Activate Client"))
    await u().click(screen.getByRole("button", { name: "Activate" }))

    await waitFor(() =>
      expect(updateStatusMutateAsync).toHaveBeenCalledWith({
        clientId: "c1",
        data: { status: "active" },
      }),
    )
  })

  it("shows error when status update rejects", async () => {
    const err = new Error("fail")
    updateStatusMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ClientActions client={makeClient({ status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Deactivate Client"))
    await u().click(screen.getByRole("button", { name: "Deactivate" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("deletes a non-system client with confirmation", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<ClientActions client={makeClient({ name: "console" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Client"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "console")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("c1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Client deleted successfully")
  })

  it("shows error when delete rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ClientActions client={makeClient({ name: "console" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Client"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "console")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("hides status and delete actions for system clients", async () => {
    renderWithProviders(<ClientActions client={makeClient({ is_system: true })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    expect(await screen.findByText("View Details")).toBeInTheDocument()
    expect(screen.queryByText("Deactivate Client")).not.toBeInTheDocument()
    expect(screen.queryByText("Delete Client")).not.toBeInTheDocument()
  })

  it("hides deactivate and delete for the default client", async () => {
    renderWithProviders(<ClientActions client={makeClient({ is_default: true, status: "active" })} />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    expect(await screen.findByText("View Details")).toBeInTheDocument()
    expect(screen.queryByText("Deactivate Client")).not.toBeInTheDocument()
    expect(screen.queryByText("Delete Client")).not.toBeInTheDocument()
  })
})
