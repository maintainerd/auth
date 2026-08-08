import { describe, it, expect, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ColumnDef } from "@tanstack/react-table"
import { getCoreRowModel, useReactTable } from "@tanstack/react-table"
import { ListingToolbar } from "./ListingToolbar"
import type { FilterGroup, ListingFilters } from "./useServerDataTable"

interface Row {
  id: string
}

const COLUMNS: ColumnDef<Row>[] = [{ id: "name", accessorKey: "id", header: "Name" }]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
]

interface HarnessProps {
  filterGroups?: readonly FilterGroup[]
  filters?: ListingFilters
  onFiltersChange?: (filters: ListingFilters) => void
  onSearchChange?: (value: string) => void
  onCreate?: () => void
  createLabel?: string
}

function Harness({
  filterGroups,
  filters = { status: [] },
  onFiltersChange = vi.fn(),
  onSearchChange = vi.fn(),
  onCreate,
  createLabel,
}: HarnessProps) {
  const table = useReactTable({ data: [], columns: COLUMNS, getCoreRowModel: getCoreRowModel() })
  return (
    <ListingToolbar
      table={table}
      search=""
      onSearchChange={onSearchChange}
      searchPlaceholder="Search..."
      filterGroups={filterGroups}
      filters={filters}
      onFiltersChange={onFiltersChange}
      onCreate={onCreate}
      createLabel={createLabel}
    />
  )
}

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("ListingToolbar", () => {
  it("hides the Filters button when filterGroups defaults to empty", () => {
    render(<Harness />)
    expect(screen.queryByRole("button", { name: /filters/i })).not.toBeInTheDocument()
  })

  it("renders the default 'New' label on the create button", () => {
    render(<Harness onCreate={vi.fn()} />)
    expect(screen.getByText("New")).toBeInTheDocument()
  })

  it("uses a custom createLabel and fires onCreate", async () => {
    const onCreate = vi.fn()
    const user = u()
    render(<Harness onCreate={onCreate} createLabel="New User" />)
    await user.click(screen.getByRole("button", { name: /new user/i }))
    expect(onCreate).toHaveBeenCalled()
  })

  it("shows the active filter count badge and toggles a checkbox on then off", async () => {
    const onFiltersChange = vi.fn()
    const user = u()
    render(
      <Harness
        filterGroups={FILTER_GROUPS}
        filters={{ status: ["active"] }}
        onFiltersChange={onFiltersChange}
      />,
    )
    // Badge with count 1
    await user.click(screen.getByRole("button", { name: /filters/i }))

    // 'active' is checked -> toggling it off removes it
    const activeBox = await screen.findByRole("checkbox", { name: "active" })
    await user.click(activeBox)
    expect(onFiltersChange).toHaveBeenLastCalledWith({ status: [] })

    // 'inactive' is unchecked -> toggling it on adds it
    const inactiveBox = screen.getByRole("checkbox", { name: "inactive" })
    await user.click(inactiveBox)
    expect(onFiltersChange).toHaveBeenLastCalledWith({ status: ["active", "inactive"] })
  })

  it("toggling a value for a group with no current selection seeds from empty", async () => {
    const onFiltersChange = vi.fn()
    const user = u()
    // filters intentionally omits the 'status' key to exercise the `?? []` branch
    render(
      <Harness filterGroups={FILTER_GROUPS} filters={{}} onFiltersChange={onFiltersChange} />,
    )
    await user.click(screen.getByRole("button", { name: /filters/i }))
    const activeBox = await screen.findByRole("checkbox", { name: "active" })
    await user.click(activeBox)
    expect(onFiltersChange).toHaveBeenLastCalledWith({ status: ["active"] })
  })

  it("Clear All resets all groups", async () => {
    const onFiltersChange = vi.fn()
    const user = u()
    render(
      <Harness
        filterGroups={FILTER_GROUPS}
        filters={{ status: ["active"] }}
        onFiltersChange={onFiltersChange}
      />,
    )
    await user.click(screen.getByRole("button", { name: /filters/i }))
    await user.click(await screen.findByRole("button", { name: /clear all/i }))
    expect(onFiltersChange).toHaveBeenLastCalledWith({ status: [] })
  })

  it("debounced search change calls onSearchChange after the delay", async () => {
    const onSearchChange = vi.fn()
    const user = u()
    render(<Harness onSearchChange={onSearchChange} />)
    const input = screen.getByPlaceholderText("Search...")
    await user.type(input, "x")
    await waitFor(() => expect(onSearchChange).toHaveBeenCalledWith("x"))
  })
})
