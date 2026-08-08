import { describe, it, expect, vi } from "vitest"
import { screen, within } from "@testing-library/react"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { policyColumns } from "./PolicyColumns"
import { DataTable } from "@/components/data-table"
import type { Policy } from "@/services/api/policies/types"

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => vi.fn() }
})

vi.mock("@/hooks/usePolicies", () => ({
  useDeletePolicy: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdatePolicyStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makePolicy(overrides: Partial<Policy> = {}): Policy {
  return {
    policy_id: "p1",
    name: "read-only",
    description: "Read only access",
    version: "1.0.0",
    status: "active",
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function Harness({ data }: { data: Policy[] }) {
  const table = useReactTable({
    data,
    columns: policyColumns,
    getCoreRowModel: getCoreRowModel(),
  })
  return <DataTable table={table} columnCount={policyColumns.length} />
}

describe("policyColumns", () => {
  it("renders all cell branches across multiple policies", () => {
    const data: Policy[] = [
      makePolicy({ policy_id: "p1", name: "read-only", version: "2.0.0", is_system: true, status: "active" }),
      makePolicy({ policy_id: "p2", name: "admin-policy", description: null, status: "inactive" }),
    ]

    renderWithProviders(<Harness data={data} />)

    // Headers
    expect(screen.getByRole("button", { name: "Policy" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Status" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Version" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Created" })).toBeInTheDocument()

    // Names and descriptions
    expect(screen.getByText("read-only")).toBeInTheDocument()
    expect(screen.getByText("Read only access")).toBeInTheDocument()
    expect(screen.getByText("admin-policy")).toBeInTheDocument()
    expect(screen.getByText("No description")).toBeInTheDocument()

    // Version
    expect(screen.getByText("2.0.0")).toBeInTheDocument()

    // Status badges
    const tbody = document.querySelector("tbody")!
    expect(within(tbody).getByText("active")).toBeInTheDocument()
    expect(within(tbody).getByText("inactive")).toBeInTheDocument()

    // System badge
    expect(screen.getByText("System")).toBeInTheDocument()

    // Actions menu trigger per row
    expect(screen.getAllByRole("button", { name: /open menu/i }).length).toBe(data.length)
  })
})
