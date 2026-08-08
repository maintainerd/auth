import { describe, it, expect, vi } from "vitest"
import { screen, within } from "@testing-library/react"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { serviceColumns } from "./ServiceColumns"
import { DataTable } from "@/components/data-table"
import type { Service } from "@/services/api/services/types"

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => vi.fn() }
})

vi.mock("@/hooks/useServices", () => ({
  useDeleteService: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateServiceStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeService(overrides: Partial<Service> = {}): Service {
  return {
    service_id: "s1",
    name: "auth-service",
    display_name: "Auth Service",
    description: "Handles authentication",
    version: "1.0.0",
    status: "active",
    is_system: false,
    api_count: 2,
    policy_count: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function Harness({ data }: { data: Service[] }) {
  const table = useReactTable({
    data,
    columns: serviceColumns,
    getCoreRowModel: getCoreRowModel(),
  })
  return <DataTable table={table} columnCount={serviceColumns.length} />
}

describe("serviceColumns", () => {
  it("renders all cell branches across multiple services", () => {
    const data: Service[] = [
      makeService({ service_id: "s1", name: "auth-service", display_name: "Auth Service", is_system: true, status: "active", version: "2.1.0", api_count: 3, policy_count: 5 }),
      makeService({ service_id: "s2", name: "billing-service", display_name: "Billing Service", status: "maintenance" }),
    ]

    renderWithProviders(<Harness data={data} />)

    // Headers
    expect(screen.getByRole("button", { name: "Service" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Status" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Version" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "APIs" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Policies" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Created" })).toBeInTheDocument()

    // Display names and system names
    expect(screen.getByText("Auth Service")).toBeInTheDocument()
    expect(screen.getByText("auth-service")).toBeInTheDocument()
    expect(screen.getByText("Billing Service")).toBeInTheDocument()

    // Version, API and policy counts
    expect(screen.getByText("2.1.0")).toBeInTheDocument()
    expect(screen.getByText("3")).toBeInTheDocument()
    expect(screen.getByText("5")).toBeInTheDocument()

    // Status badges
    const tbody = document.querySelector("tbody")!
    expect(within(tbody).getByText("active")).toBeInTheDocument()
    expect(within(tbody).getByText("maintenance")).toBeInTheDocument()

    // System badge
    expect(screen.getByText("System")).toBeInTheDocument()

    // Actions menu trigger per row
    expect(screen.getAllByRole("button", { name: /open menu/i }).length).toBe(data.length)
  })
})
