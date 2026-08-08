import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { ApiPermissionsTab } from "./ApiPermissionsTab"
import type { PermissionEntity } from "@/services/api/permissions/types"

const { useApiPermissionsMock, deleteMutateAsync, showSuccessMock, showErrorMock, dialogSpy } =
  vi.hoisted(() => ({
    useApiPermissionsMock: vi.fn(),
    deleteMutateAsync: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
    dialogSpy: vi.fn(),
  }))

vi.mock("../hooks/useApiPermissions", () => ({
  useApiPermissions: (...args: unknown[]) => useApiPermissionsMock(...args),
}))

vi.mock("@/hooks/usePermissions", () => ({
  useDeletePermission: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

// The dialog is tested separately; record its props so open/edit flows are observable.
vi.mock("./PermissionFormDialog", () => ({
  PermissionFormDialog: (props: { open: boolean; permission?: PermissionEntity }) => {
    dialogSpy(props)
    return null
  },
}))

function makePermission(overrides: Partial<PermissionEntity> = {}): PermissionEntity {
  return {
    permission_id: "p1",
    name: "billing:read",
    description: "Read invoices",
    api: {
      api_id: "a1", name: "billing-api", display_name: "Billing API",
      description: "", identifier: "", status: "active", is_system: false,
      service: {
        service_id: "s1", name: "billing-service", display_name: "Billing Service",
        description: "", version: "1.0.0", status: "active", is_system: false,
        api_count: 0, policy_count: 0, created_at: "", updated_at: "",
      },
      created_at: "", updated_at: "",
    },
    status: "active",
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function response(rows: PermissionEntity[], total = rows.length) {
  return { rows, total, page: 1, limit: 10, total_pages: 1 }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ApiPermissionsTab", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useApiPermissionsMock.mockReturnValue({ data: undefined, isLoading: false, isError: false })
  })

  it("renders loading", () => {
    useApiPermissionsMock.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { container } = renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    expect(container.querySelector(".animate-pulse")).toBeTruthy()
  })

  it("renders error", () => {
    useApiPermissionsMock.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    expect(screen.getByText("Failed to load permissions")).toBeInTheDocument()
  })

  it("renders empty", () => {
    useApiPermissionsMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    expect(screen.getByText("No permissions")).toBeInTheDocument()
  })

  it("renders a permission with badges and pagination", () => {
    useApiPermissionsMock.mockReturnValue({ data: response([makePermission()], 5), isLoading: false, isError: false })
    renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    expect(screen.getByText("billing:read")).toBeInTheDocument()
    expect(screen.getByText("Read invoices")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
    expect(screen.getByText("Rows per page")).toBeInTheDocument()
  })

  it("hides all row actions for system permissions", () => {
    useApiPermissionsMock.mockReturnValue({
      data: response([makePermission({ is_system: true })]),
      isLoading: false,
      isError: false,
    })
    renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /open menu/i })).not.toBeInTheDocument()
  })

  it("opens the create dialog via Add Permission", async () => {
    useApiPermissionsMock.mockReturnValue({ data: response([]), isLoading: false, isError: false })
    renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /add permission/i }))
    await waitFor(() =>
      expect(dialogSpy).toHaveBeenCalledWith(
        expect.objectContaining({ open: true, permission: undefined }),
      ),
    )
  })

  it("opens the edit dialog with the selected permission", async () => {
    useApiPermissionsMock.mockReturnValue({ data: response([makePermission()]), isLoading: false, isError: false })
    renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Edit Permission"))
    await waitFor(() =>
      expect(dialogSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          open: true,
          permission: expect.objectContaining({ permission_id: "p1" }),
        }),
      ),
    )
  })

  it("deletes a permission with type-to-confirm", async () => {
    deleteMutateAsync.mockResolvedValueOnce(undefined)
    useApiPermissionsMock.mockReturnValue({ data: response([makePermission()]), isLoading: false, isError: false })
    renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Permission"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "billing:read")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith("p1"))
    expect(showSuccessMock).toHaveBeenCalledWith("Permission deleted successfully")
  })

  it("shows an error when deleting rejects", async () => {
    const err = new Error("fail")
    deleteMutateAsync.mockRejectedValueOnce(err)
    useApiPermissionsMock.mockReturnValue({ data: response([makePermission()]), isLoading: false, isError: false })
    renderWithProviders(<ApiPermissionsTab apiId="a1" />)
    await u().click(screen.getByRole("button", { name: /open menu/i }))
    await u().click(await screen.findByText("Delete Permission"))

    const dialog = await screen.findByRole("dialog")
    await u().type(within(dialog).getByRole("textbox"), "billing:read")
    await u().click(within(dialog).getByRole("button", { name: "Delete" }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })
})
