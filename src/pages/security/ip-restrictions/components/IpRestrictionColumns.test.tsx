import { describe, it, expect, vi } from "vitest"
import { screen, within } from "@testing-library/react"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { ipRestrictionColumns } from "./IpRestrictionColumns"
import { DataTable } from "@/components/data-table"
import type { IpRestrictionRule } from "@/services/api/ip-restriction-rules/types"

vi.mock("@/hooks/useIpRestrictionRules", () => ({
  useDeleteIpRestrictionRule: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateIpRestrictionRuleStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeRule(overrides: Partial<IpRestrictionRule> = {}): IpRestrictionRule {
  return {
    ipRestrictionRuleId: "r1",
    description: "Office VPN",
    type: "allow",
    ipAddress: "203.0.113.0/24",
    status: "active",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function Harness({ data }: { data: IpRestrictionRule[] }) {
  const columns = ipRestrictionColumns(vi.fn())
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })
  return <DataTable table={table} columnCount={columns.length} />
}

describe("ipRestrictionColumns", () => {
  it("renders all cell branches across multiple rules", () => {
    const data: IpRestrictionRule[] = [
      makeRule({ ipRestrictionRuleId: "r1", type: "allow", status: "active", ipAddress: "10.0.0.1" }),
      makeRule({ ipRestrictionRuleId: "r2", type: "deny", status: "inactive", ipAddress: "10.0.0.2" }),
    ]

    renderWithProviders(<Harness data={data} />)

    // Headers
    expect(screen.getByRole("button", { name: "IP Address/Range" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Description" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Type" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Status" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Created" })).toBeInTheDocument()

    // IP addresses, types, and status badges
    expect(screen.getByText("10.0.0.1")).toBeInTheDocument()
    expect(screen.getByText("10.0.0.2")).toBeInTheDocument()
    expect(screen.getByText("Allow")).toBeInTheDocument()
    expect(screen.getByText("Deny")).toBeInTheDocument()

    const tbody = document.querySelector("tbody")!
    expect(within(tbody).getByText("active")).toBeInTheDocument()
    expect(within(tbody).getByText("inactive")).toBeInTheDocument()

    // Actions menu trigger per row
    expect(screen.getAllByRole("button", { name: /open menu/i }).length).toBe(data.length)
  })
})
