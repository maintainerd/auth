import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ClientApis } from "./ClientApis"
import type { ClientApiAssociation } from "@/services/api/clients/types"

const { useClientApisMock, removeApiMutateAsync, removePermissionMutateAsync, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    useClientApisMock: vi.fn(),
    removeApiMutateAsync: vi.fn(),
    removePermissionMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("@/hooks/useClients", () => ({
  useClientApis: (...args: unknown[]) => useClientApisMock(...args),
  useRemoveClientApi: () => ({ mutateAsync: removeApiMutateAsync, isPending: false }),
  useRemoveClientApiPermission: () => ({ mutateAsync: removePermissionMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

// The add dialogs manage their own flows; stub them so this tab is isolated.
vi.mock("./AddClientApiDialog", () => ({
  AddClientApiDialog: () => null,
}))
vi.mock("./AddClientApiPermissionsDialog", () => ({
  AddClientApiPermissionsDialog: () => null,
}))

function makeAssociation(overrides: Partial<ClientApiAssociation> = {}): ClientApiAssociation {
  return {
    client_api_id: "ca1",
    api: {
      api_id: "a1",
      name: "billing-api",
      display_name: "Billing API",
      description: "Invoices and payments",
      identifier: "https://api.example.com/billing",
      status: "active",
      is_system: false,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    },
    permissions: [
      {
        permission_id: "p1",
        name: "billing:read",
        description: "Read invoices",
        status: "active",
        is_system: false,
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
      },
    ],
    created_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ClientApis", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useClientApisMock.mockReturnValue({ data: undefined, isLoading: false, error: null })
  })

  it("renders loading", () => {
    useClientApisMock.mockReturnValue({ data: undefined, isLoading: true, error: null })
    const { container } = renderWithProviders(<ClientApis clientId="c1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useClientApisMock.mockReturnValue({ data: undefined, isLoading: false, error: new Error("x") })
    renderWithProviders(<ClientApis clientId="c1" />)
    expect(screen.getByText("Failed to load APIs")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useClientApisMock.mockReturnValue({ data: { apis: [] }, isLoading: false, error: null })
    renderWithProviders(<ClientApis clientId="c1" />)
    expect(screen.getByText("No API permissions")).toBeInTheDocument()
  })

  it("renders an API row with badges and permission count", () => {
    useClientApisMock.mockReturnValue({ data: { apis: [makeAssociation()] }, isLoading: false, error: null })
    renderWithProviders(<ClientApis clientId="c1" />)
    expect(screen.getByText("Billing API")).toBeInTheDocument()
    expect(screen.getByText("Invoices and payments")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
    expect(screen.getByText("1 permission")).toBeInTheDocument()
  })

  it("expands to show permissions and their empty state", async () => {
    useClientApisMock.mockReturnValue({ data: { apis: [makeAssociation()] }, isLoading: false, error: null })
    renderWithProviders(<ClientApis clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /expand permissions/i }))
    expect(screen.getByText("billing:read")).toBeInTheDocument()
    expect(screen.getByText("Read invoices")).toBeInTheDocument()
  })

  it("shows the no-permissions state inside an expanded API", async () => {
    useClientApisMock.mockReturnValue({
      data: { apis: [makeAssociation({ permissions: [] })] },
      isLoading: false,
      error: null,
    })
    renderWithProviders(<ClientApis clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /expand permissions/i }))
    expect(screen.getByText("No permissions")).toBeInTheDocument()
  })

  it("removes an API with type-to-confirm", async () => {
    removeApiMutateAsync.mockResolvedValueOnce(undefined)
    useClientApisMock.mockReturnValue({ data: { apis: [makeAssociation()] }, isLoading: false, error: null })
    renderWithProviders(<ClientApis clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Remove API"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "Billing API")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() =>
      expect(removeApiMutateAsync).toHaveBeenCalledWith({ clientId: "c1", apiId: "a1" }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("API removed from client successfully")
  })

  it("removes a permission with confirmation", async () => {
    removePermissionMutateAsync.mockResolvedValueOnce(undefined)
    useClientApisMock.mockReturnValue({ data: { apis: [makeAssociation()] }, isLoading: false, error: null })
    renderWithProviders(<ClientApis clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /expand permissions/i }))

    const menuButtons = screen.getAllByRole("button", { name: /open menu/i })
    await u().click(menuButtons[menuButtons.length - 1])
    await u().click(await screen.findByText("Remove Permission"))

    const dialog = await screen.findByRole("dialog")
    await u().click(within(dialog).getByRole("button", { name: "Remove" }))

    await waitFor(() =>
      expect(removePermissionMutateAsync).toHaveBeenCalledWith({
        clientId: "c1",
        apiId: "a1",
        permissionId: "p1",
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Permission removed from client API successfully")
  })

  it("shows an error when removing an API rejects", async () => {
    const err = new Error("fail")
    removeApiMutateAsync.mockRejectedValueOnce(err)
    useClientApisMock.mockReturnValue({ data: { apis: [makeAssociation()] }, isLoading: false, error: null })
    renderWithProviders(<ClientApis clientId="c1" />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Remove API"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "Billing API")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })
})
