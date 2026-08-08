import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ClientCredentials } from "./ClientCredentials"
import type { ClientResponse } from "@/services/api/clients/types"

const { rotateMutateAsync, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  rotateMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/useClients", () => ({
  useRotateClientSecret: () => ({ mutateAsync: rotateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeClient(overrides: Partial<ClientResponse> = {}): ClientResponse {
  return {
    client_id: "c1", name: "console", display_name: "Console",
    client_type: "traditional", status: "active", is_default: false, is_system: false,
    allow_registration: true,
    allow_magic_link: false, created_at: "", updated_at: "", ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ClientCredentials", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("shows the confidential model and rotate action for secret-bearing clients", () => {
    renderWithProviders(<ClientCredentials client={makeClient({ client_type: "m2m" })} />)
    expect(screen.getByText("Confidential client")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /rotate secret/i })).toBeInTheDocument()
  })

  it("shows the public model without a rotate action for PKCE clients", () => {
    renderWithProviders(<ClientCredentials client={makeClient({ client_type: "spa" })} />)
    expect(screen.getByText("Public client")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /rotate secret/i })).not.toBeInTheDocument()
    expect(screen.getByText(/does not use a client secret/i)).toBeInTheDocument()
  })

  it("copies the client UUID to the clipboard", async () => {
    // user-event installs its own clipboard stub; assert the observable toast.
    renderWithProviders(<ClientCredentials client={makeClient()} />)
    await u().click(screen.getByRole("button", { name: /copy client uuid/i }))
    await waitFor(() =>
      expect(showSuccessMock).toHaveBeenCalledWith("Client UUID copied to clipboard"),
    )
  })

  it("rotates the secret with confirmation and reveals the one-time value", async () => {
    rotateMutateAsync.mockResolvedValueOnce({
      client_secret: "new-secret-value",
      previous_secret_expires_at: "2024-06-01T00:00:00Z",
    })
    renderWithProviders(<ClientCredentials client={makeClient()} />)
    await u().click(screen.getByRole("button", { name: /rotate secret/i }))

    const dialog = await screen.findByRole("dialog")
    expect(within(dialog).getByText(/shown only once/i)).toBeInTheDocument()
    await u().click(within(dialog).getByRole("button", { name: "Rotate Secret" }))

    await waitFor(() =>
      expect(rotateMutateAsync).toHaveBeenCalledWith({
        clientId: "c1",
        data: { grace_period_hours: 24 },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Client secret rotated successfully")
    expect(await screen.findByText("new-secret-value")).toBeInTheDocument()
    expect(screen.getByText(/previous secret expires/i)).toBeInTheDocument()
  })

  it("shows an error when rotation rejects", async () => {
    const err = new Error("fail")
    rotateMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<ClientCredentials client={makeClient()} />)
    await u().click(screen.getByRole("button", { name: /rotate secret/i }))
    const dialog = await screen.findByRole("dialog")
    await u().click(within(dialog).getByRole("button", { name: "Rotate Secret" }))
    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })
})
