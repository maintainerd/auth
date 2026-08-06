import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

import { renderWithProviders } from "@/test/utils"
import { TenantActions } from "./TenantActions"
import type { TenantEntity, TenantStatus } from "@/services/api/tenants/types"

const { updateStatusMutateAsync, deleteMutateAsync, navigateMock, showSuccessMock, showErrorMock } =
  vi.hoisted(() => ({
    updateStatusMutateAsync: vi.fn(),
    deleteMutateAsync: vi.fn(),
    navigateMock: vi.fn(),
    showSuccessMock: vi.fn(),
    showErrorMock: vi.fn(),
  }))

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock("@/hooks/useTenants", () => ({
  useUpdateTenantStatus: () => ({ mutateAsync: updateStatusMutateAsync, isPending: false }),
  useDeleteTenant: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: showSuccessMock, showError: showErrorMock }),
}))

function makeTenant(overrides: Partial<TenantEntity> = {}): TenantEntity {
  return {
    tenant_id: "t-1",
    name: "acme-corp",
    display_name: "Acme Corporation",
    description: "The Acme tenant",
    status: "active",
    is_system: false,
    created_at: "2025-01-01T00:00:00.000Z",
    updated_at: "2025-01-01T00:00:00.000Z",
    ...overrides,
  }
}

async function openMenu() {
  const user = userEvent.setup({ pointerEventsCheck: 0 })
  await user.click(screen.getByRole("button", { name: "Open menu" }))
  return user
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("TenantActions", () => {
  it("renders the always-available actions for an active tenant", async () => {
    renderWithProviders(<TenantActions tenant={makeTenant()} />)
    await openMenu()

    expect(screen.getByRole("menuitem", { name: "View Details" })).toBeInTheDocument()
    expect(screen.getByRole("menuitem", { name: "Edit Tenant" })).toBeInTheDocument()
    expect(screen.getByRole("menuitem", { name: "Deactivate Tenant" })).toBeInTheDocument()
    expect(screen.getByRole("menuitem", { name: "Suspend Tenant" })).toBeInTheDocument()
    expect(screen.getByRole("menuitem", { name: "Delete Tenant" })).toBeInTheDocument()
  })

  // The blocker: `pending` had no STATUS_ACTIONS entry, so the lookup returned
  // undefined and .map() threw during render, blanking the whole listing.
  it("renders a pending tenant instead of throwing", async () => {
    expect(() =>
      renderWithProviders(<TenantActions tenant={makeTenant({ status: "pending" })} />),
    ).not.toThrow()

    await openMenu()
    expect(screen.getByRole("menuitem", { name: "View Details" })).toBeInTheDocument()
  })

  it("offers a pending tenant the transitions the backend accepts", async () => {
    renderWithProviders(<TenantActions tenant={makeTenant({ status: "pending" })} />)
    await openMenu()

    expect(screen.getByRole("menuitem", { name: "Activate Tenant" })).toBeInTheDocument()
    expect(screen.getByRole("menuitem", { name: "Suspend Tenant" })).toBeInTheDocument()
    // Already pending — never offer a no-op transition to itself.
    expect(screen.queryByRole("menuitem", { name: /Pending/ })).not.toBeInTheDocument()
  })

  it("renders every status the backend can emit without crashing", async () => {
    for (const status of ["active", "inactive", "pending", "suspended"] as TenantStatus[]) {
      const { unmount } = renderWithProviders(<TenantActions tenant={makeTenant({ status })} />)
      expect(screen.getByRole("button", { name: "Open menu" })).toBeInTheDocument()
      unmount()
    }
  })

  // Structural guarantee: a status this build has never heard of must degrade to
  // "no status actions", not take the listing down with it.
  it("degrades gracefully on an unknown status from a newer backend", async () => {
    const tenant = makeTenant({ status: "archived" as TenantStatus })

    expect(() => renderWithProviders(<TenantActions tenant={tenant} />)).not.toThrow()

    await openMenu()
    expect(screen.getByRole("menuitem", { name: "View Details" })).toBeInTheDocument()
    expect(screen.getByRole("menuitem", { name: "Delete Tenant" })).toBeInTheDocument()
    expect(screen.queryByRole("menuitem", { name: /Activate Tenant/ })).not.toBeInTheDocument()
    expect(screen.queryByRole("menuitem", { name: /Suspend Tenant/ })).not.toBeInTheDocument()
  })

  it("activates a pending tenant through the confirmation dialog", async () => {
    updateStatusMutateAsync.mockResolvedValue({})
    renderWithProviders(<TenantActions tenant={makeTenant({ status: "pending" })} />)

    const user = await openMenu()
    await user.click(screen.getByRole("menuitem", { name: "Activate Tenant" }))
    await user.click(screen.getByRole("button", { name: "Activate Tenant" }))

    expect(updateStatusMutateAsync).toHaveBeenCalledWith({ tenantId: "t-1", status: "active" })
  })
})
