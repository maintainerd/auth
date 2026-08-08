import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { PermissionFormDialog } from "./PermissionFormDialog"
import type { PermissionEntity } from "@/services/api/permissions/types"

const { createMutateAsync, updateMutateAsync, showSuccessMock, showErrorMock } = vi.hoisted(() => ({
  createMutateAsync: vi.fn(),
  updateMutateAsync: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock("@/hooks/usePermissions", () => ({
  useCreatePermission: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdatePermission: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
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
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

const baseProps = {
  open: true,
  onOpenChange: vi.fn(),
  apiId: "a1",
}

describe("PermissionFormDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("creates a permission and closes on success", async () => {
    createMutateAsync.mockResolvedValueOnce(undefined)
    const onOpenChange = vi.fn()
    renderWithProviders(<PermissionFormDialog {...baseProps} onOpenChange={onOpenChange} />)

    await u().type(screen.getByLabelText(/^permission name/i), "users:read")
    await u().type(screen.getByLabelText(/^description/i), "Read users")
    await u().click(screen.getByRole("button", { name: /create permission/i }))

    await waitFor(() =>
      expect(createMutateAsync).toHaveBeenCalledWith({
        name: "users:read",
        description: "Read users",
        status: "active",
        api_id: "a1",
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Permission created successfully")
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it("validates the resource:action name format", async () => {
    renderWithProviders(<PermissionFormDialog {...baseProps} />)
    await u().type(screen.getByLabelText(/^permission name/i), "invalid name")
    await u().type(screen.getByLabelText(/^description/i), "Read users")
    await u().click(screen.getByRole("button", { name: /create permission/i }))

    expect(
      await screen.findByText(/lowercase colon-separated segments/i),
    ).toBeInTheDocument()
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  // The client rule used to mandate exactly one colon, which blocked
  // `users:read:own` — a name the server accepts. A client stricter than the
  // server makes valid names unreachable, so pin the looser form.
  it("accepts a three-segment name the server allows", async () => {
    createMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<PermissionFormDialog {...baseProps} />)
    await u().type(screen.getByLabelText(/^permission name/i), "users:read:own")
    await u().type(screen.getByLabelText(/^description/i), "Read your own user")
    await u().click(screen.getByRole("button", { name: /create permission/i }))

    await waitFor(() => expect(createMutateAsync).toHaveBeenCalled())
  })

  it("pre-fills and updates an existing permission via the status select", async () => {
    updateMutateAsync.mockResolvedValueOnce(undefined)
    renderWithProviders(<PermissionFormDialog {...baseProps} permission={makePermission()} />)

    expect(screen.getByLabelText(/^permission name/i)).toHaveValue("billing:read")

    // Change status through the Radix select (drives the RHF Controller).
    await u().click(screen.getByRole("combobox"))
    await u().click(await screen.findByRole("option", { name: "Inactive" }))
    await u().click(screen.getByRole("button", { name: /update permission/i }))

    await waitFor(() =>
      expect(updateMutateAsync).toHaveBeenCalledWith({
        permissionId: "p1",
        data: {
          name: "billing:read",
          description: "Read invoices",
          status: "inactive",
        },
      }),
    )
    expect(showSuccessMock).toHaveBeenCalledWith("Permission updated successfully")
  })

  it("shows an error when the mutation rejects", async () => {
    const err = new Error("fail")
    createMutateAsync.mockRejectedValueOnce(err)
    renderWithProviders(<PermissionFormDialog {...baseProps} />)

    await u().type(screen.getByLabelText(/^permission name/i), "users:read")
    await u().type(screen.getByLabelText(/^description/i), "Read users")
    await u().click(screen.getByRole("button", { name: /create permission/i }))

    await waitFor(() => expect(showErrorMock).toHaveBeenCalledWith(err))
  })

  it("disables the name field for system permissions", () => {
    renderWithProviders(
      <PermissionFormDialog {...baseProps} permission={makePermission({ is_system: true })} />,
    )
    expect(screen.getByLabelText(/^permission name/i)).toBeDisabled()
  })

  it("cancel calls onOpenChange(false)", async () => {
    const onOpenChange = vi.fn()
    renderWithProviders(<PermissionFormDialog {...baseProps} onOpenChange={onOpenChange} />)
    await u().click(screen.getByRole("button", { name: "Cancel" }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
