import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import { flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { workloadIdentityColumns } from "./WorkloadIdentityColumns"
import type { WorkloadIdentityFederation } from "@/services/api/workload-identity/types"

vi.mock("./WorkloadIdentityActions", () => ({
  WorkloadIdentityActions: () => <div data-testid="actions" />,
}))

function makeFederation(
  overrides: Partial<WorkloadIdentityFederation> = {},
): WorkloadIdentityFederation {
  return {
    workload_identity_federation_uuid: "fed-1",
    client_uuid: "client-1",
    name: "github-actions",
    description: "",
    issuer_url: "https://token.actions.githubusercontent.com",
    audience: "https://auth.example.com",
    subject_claim: "sub",
    subject_pattern: "repo:my-org/my-repo:*",
    allowed_scopes: ["api:read", "api:write"],
    attribute_mapping: {},
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

/** Renders the column cells for one row, like the data table would. */
function renderRow(federation: WorkloadIdentityFederation) {
  function Harness() {
    const table = useReactTable({
      data: [federation],
      columns: workloadIdentityColumns,
      getCoreRowModel: getCoreRowModel(),
    })
    const row = table.getRowModel().rows[0]
    return (
      <div>
        {row.getVisibleCells().map((cell) => (
          <div key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</div>
        ))}
      </div>
    )
  }
  return renderWithProviders(<Harness />)
}

describe("workloadIdentityColumns", () => {
  it("shows the name and the issuer host", () => {
    renderRow(makeFederation())
    expect(screen.getByText("github-actions")).toBeInTheDocument()
    // The host, not the full URL — the scheme is always https.
    expect(screen.getByText("token.actions.githubusercontent.com")).toBeInTheDocument()
  })

  it("shows the subject pattern, which is the trust boundary", () => {
    renderRow(makeFederation())
    expect(screen.getByText("repo:my-org/my-repo:*")).toBeInTheDocument()
  })

  it("renders the status from is_active", () => {
    renderRow(makeFederation({ is_active: false }))
    expect(screen.getByText(/inactive/i)).toBeInTheDocument()
  })

  it("summarises scopes and says so when there are none", () => {
    renderRow(makeFederation({ allowed_scopes: ["a", "b", "c"] }))
    expect(screen.getByText("+1")).toBeInTheDocument()

    renderRow(makeFederation({ allowed_scopes: [] }))
    expect(screen.getByText("None")).toBeInTheDocument()
  })

  it("falls back to the raw value when the issuer is unparseable", () => {
    renderRow(makeFederation({ issuer_url: "not-a-url" }))
    expect(screen.getByText("not-a-url")).toBeInTheDocument()
  })

  // sort_by is resolved from accessorKey, and the backend allow-list has no
  // issuer_url / subject_pattern / allowed_scopes — clicking those would silently
  // fall back, so they must not look sortable.
  it("only marks backend-sortable columns as sortable", () => {
    const byId = Object.fromEntries(workloadIdentityColumns.map((c) => [c.id, c]))
    expect(byId["Subject Pattern"].enableSorting).toBe(false)
    expect(byId["Scopes"].enableSorting).toBe(false)
    expect(byId["actions"].enableSorting).toBe(false)
    expect(byId["Federation"].enableSorting).toBeUndefined()
    expect(byId["Status"].enableSorting).toBeUndefined()
    expect(byId["Created"].enableSorting).toBeUndefined()
  })
})
