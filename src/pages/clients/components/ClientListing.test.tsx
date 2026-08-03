import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ClientListing } from "./ClientListing"
import type { Client } from "@/services/api/clients/types"

const { useClientsListMock, navigateMock } = vi.hoisted(() => ({
  useClientsListMock: vi.fn(),
  navigateMock: vi.fn(),
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
  useClientsList: (...args: unknown[]) => useClientsListMock(...args),
  useDeleteClient: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateClientStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
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
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function setClients(overrides: Record<string, unknown> = {}) {
  useClientsListMock.mockReturnValue({
    data: { rows: [], total: 0 },
    isLoading: false,
    error: null,
    ...overrides,
  })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ClientListing", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setClients()
  })

  it("renders the loading state", () => {
    setClients({ isLoading: true })
    const { container } = renderWithProviders(<ClientListing />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders rows and navigates on row click", async () => {
    setClients({ data: { rows: [makeClient({ client_id: "c9", display_name: "Storefront" })], total: 1 } })
    renderWithProviders(<ClientListing />)
    const cell = screen.getByText("Storefront")
    await u().click(cell)
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith("/clients/c9"))
  })

  it("clicking New Client navigates to the create route", async () => {
    renderWithProviders(<ClientListing />)
    const buttons = screen.getAllByRole("button", { name: /new client/i })
    await u().click(buttons[0])
    expect(navigateMock).toHaveBeenCalledWith("/clients/create")
  })
})
