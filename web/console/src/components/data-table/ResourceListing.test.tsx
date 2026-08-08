import { describe, it, expect, vi } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { renderWithProviders } from "@/test/utils"
import { ResourceListing } from "./ResourceListing"
import type { FilterGroup, ServerListResult, UseListData } from "./useServerDataTable"

interface Row {
  id: string
  name: string
}

const COLUMNS: ColumnDef<Row>[] = [
  { id: "name", accessorKey: "name", header: "Name", cell: ({ row }) => row.original.name },
]
const DEFAULT_SORT: SortingState = [{ id: "name", desc: false }]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
]

function makeUseData(result?: ServerListResult<Row>): UseListData<Row, Record<string, unknown>> {
  return () => ({ data: result, isLoading: false, error: null })
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ResourceListing", () => {
  it("renders with all optional props (rowClick, create, filterGroups)", async () => {
    const onRowClick = vi.fn()
    const onCreate = vi.fn()
    const user = u()
    renderWithProviders(
      <ResourceListing<Row>
        columns={COLUMNS}
        defaultSort={DEFAULT_SORT}
        searchFields={["name"]}
        searchPlaceholder="Search..."
        useData={makeUseData({ rows: [{ id: "1", name: "Alice" }], total: 1 })}
        filterGroups={FILTER_GROUPS}
        onRowClick={onRowClick}
        onCreate={onCreate}
        createLabel="New Thing"
      />,
    )

    // create button present
    await user.click(screen.getByRole("button", { name: /new thing/i }))
    expect(onCreate).toHaveBeenCalled()

    // filters present
    expect(screen.getByRole("button", { name: /filters/i })).toBeInTheDocument()

    // row click navigates
    await user.click(screen.getByText("Alice"))
    await waitFor(() => expect(onRowClick).toHaveBeenCalledWith({ id: "1", name: "Alice" }))
  })

  it("renders without optional props (no rowClick, no create, no filterGroups)", () => {
    renderWithProviders(
      <ResourceListing<Row>
        columns={COLUMNS}
        defaultSort={DEFAULT_SORT}
        searchFields={["name"]}
        searchPlaceholder="Search..."
        useData={makeUseData({ rows: [{ id: "1", name: "Bob" }], total: 1 })}
      />,
    )
    expect(screen.getByText("Bob")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /filters/i })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /^new$/i })).not.toBeInTheDocument()
  })

  it("shows 'Clear search & filters' when filtered with no results and clears on click", async () => {
    const user = u()
    renderWithProviders(
      <ResourceListing<Row>
        columns={COLUMNS}
        defaultSort={DEFAULT_SORT}
        searchFields={["name"]}
        searchPlaceholder="Search..."
        useData={makeUseData({ rows: [], total: 0 })}
      />,
    )
    const input = screen.getByPlaceholderText("Search...")
    await user.type(input, "xyz")
    await waitFor(() => expect(screen.getByText("No results found")).toBeInTheDocument())
    const clearBtn = screen.getByRole("button", { name: /clear search & filters/i })
    await user.click(clearBtn)
    await waitFor(() => expect(input).toHaveValue(""))
  })
})
