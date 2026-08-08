import { describe, it, expect, vi } from "vitest"
import { screen, within } from "@testing-library/react"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { clientColumns } from "./ClientColumns"
import { DataTable } from "@/components/data-table"
import type { Client } from "@/services/api/clients/types"

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom")
  return { ...actual, useNavigate: () => vi.fn() }
})

vi.mock("@/hooks/useClients", () => ({
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

function Harness({ data }: { data: Client[] }) {
  const table = useReactTable({
    data,
    columns: clientColumns,
    getCoreRowModel: getCoreRowModel(),
  })
  return <DataTable table={table} columnCount={clientColumns.length} />
}

describe("clientColumns", () => {
  it("renders all cell branches across multiple clients", () => {
    const data: Client[] = [
      makeClient({ client_id: "c1", name: "console", display_name: "Console", client_type: "spa", is_system: true, status: "active" }),
      makeClient({ client_id: "c2", name: "backend", display_name: "Backend Worker", client_type: "m2m", status: "inactive" }),
    ]

    renderWithProviders(<Harness data={data} />)

    // Headers
    expect(screen.getByRole("button", { name: "Client" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Type" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Status" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Created" })).toBeInTheDocument()

    // Display names and system names
    expect(screen.getByText("Console")).toBeInTheDocument()
    expect(screen.getByText("console")).toBeInTheDocument()
    expect(screen.getByText("Backend Worker")).toBeInTheDocument()

    // Client type badges
    expect(screen.getByText("Single Page App")).toBeInTheDocument()
    expect(screen.getByText("Machine to Machine")).toBeInTheDocument()

    // Status badges
    const tbody = document.querySelector("tbody")!
    expect(within(tbody).getByText("active")).toBeInTheDocument()
    expect(within(tbody).getByText("inactive")).toBeInTheDocument()

    // System badge
    expect(screen.getByText("System")).toBeInTheDocument()

    // Actions menu trigger per row
    expect(screen.getAllByRole("button", { name: /open menu/i }).length).toBe(data.length)
  })

  it("falls back to the raw type for unknown client types", () => {
    renderWithProviders(
      <Harness data={[makeClient({ client_type: "native" as Client["client_type"] })]} />,
    )
    expect(screen.getByText("native")).toBeInTheDocument()
  })
})
