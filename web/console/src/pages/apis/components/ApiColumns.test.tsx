import { describe, it, expect, vi } from "vitest"
import { screen, within } from "@testing-library/react"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { apiColumns } from "./ApiColumns"
import { DataTable } from "@/components/data-table"
import type { Api } from "@/services/api/api/types"

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => vi.fn() }
})

vi.mock("@/hooks/useApis", () => ({
  useDeleteApi: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateApiStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

vi.mock("@/hooks/useToast", () => ({
  useToast: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

function makeApi(overrides: Partial<Api> = {}): Api {
  return {
    api_id: "a1",
    name: "billing-api",
    display_name: "Billing API",
    description: "Invoices and payments",
    identifier: "https://api.example.com/billing",
    service: {
      service_id: "s1", name: "billing-service", display_name: "Billing Service",
      description: "", version: "1.0.0", status: "active", is_system: false,
      api_count: 0, policy_count: 0, created_at: "", updated_at: "",
    },
    status: "active",
    is_system: false,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  }
}

function Harness({ data }: { data: Api[] }) {
  const table = useReactTable({
    data,
    columns: apiColumns,
    getCoreRowModel: getCoreRowModel(),
  })
  return <DataTable table={table} columnCount={apiColumns.length} />
}

describe("apiColumns", () => {
  it("renders all cell branches across multiple APIs", () => {
    const data: Api[] = [
      makeApi({ api_id: "a1", name: "billing-api", display_name: "Billing API", is_system: true, status: "active" }),
      makeApi({ api_id: "a2", name: "user-api", display_name: "User API", status: "inactive" }),
    ]

    renderWithProviders(<Harness data={data} />)

    // Headers
    expect(screen.getByRole("button", { name: "API" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Service" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Status" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Created" })).toBeInTheDocument()

    // Display names and system names
    expect(screen.getByText("Billing API")).toBeInTheDocument()
    expect(screen.getByText("billing-api")).toBeInTheDocument()
    expect(screen.getByText("User API")).toBeInTheDocument()

    // Owning service
    expect(screen.getAllByText("Billing Service").length).toBe(data.length)

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
