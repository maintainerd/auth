import { describe, it, expect, vi } from "vitest"
import { screen, within } from "@testing-library/react"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { roleColumns } from "./RoleColumns"
import { DataTable } from "@/components/data-table"
import type { Role } from "@/services/api/roles/types"

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => vi.fn() }
})

vi.mock("@/hooks/useRoles", () => ({
  useDeleteRole: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateRoleStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeRole(overrides: Partial<Role> = {}): Role {
  return {
    role_id: "r1",
    name: "admin",
    description: "Administrator role",
    is_default: false,
    is_system: false,
    status: "active",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function Harness({ data }: { data: Role[] }) {
  const table = useReactTable({
    data,
    columns: roleColumns,
    getCoreRowModel: getCoreRowModel(),
  })
  return <DataTable table={table} columnCount={roleColumns.length} />
}

describe("roleColumns", () => {
  it("renders all cell branches across multiple roles", () => {
    const data: Role[] = [
      makeRole({ role_id: "r1", name: "admin", description: "Full access", is_system: true, is_default: true, status: "active" }),
      makeRole({ role_id: "r2", name: "viewer", description: "Read only", is_system: false, status: "inactive" }),
    ]

    renderWithProviders(<Harness data={data} />)

    // Headers
    expect(screen.getByRole("button", { name: "Role" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Status" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Created" })).toBeInTheDocument()

    // Role names and descriptions
    expect(screen.getByText("admin")).toBeInTheDocument()
    expect(screen.getByText("Full access")).toBeInTheDocument()
    expect(screen.getByText("viewer")).toBeInTheDocument()

    // Status badges
    const tbody = document.querySelector("tbody")!
    expect(within(tbody).getByText("active")).toBeInTheDocument()
    expect(within(tbody).getByText("inactive")).toBeInTheDocument()

    // System and default badges
    expect(screen.getByText("System")).toBeInTheDocument()
    expect(screen.getByText("Default")).toBeInTheDocument()

    // Actions menu trigger per row
    expect(screen.getAllByRole("button", { name: /open menu/i }).length).toBe(data.length)
  })
})
