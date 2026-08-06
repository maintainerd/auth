import { describe, it, expect, vi } from "vitest"
import { render, screen, act } from "@testing-library/react"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { MemoryRouter, Routes, Route } from "react-router-dom"
import {
  useServerDataTable,
  type FilterGroup,
  type UseServerDataTableOptions,
  type UseServerDataTableResult,
} from "./useServerDataTable"

interface Row {
  id: string
  name: string
}

const COLUMNS: ColumnDef<Row>[] = [
  { id: "name", accessorKey: "name", header: "Name" },
  // Every listing labels this column for display while the API column is created_at.
  { id: "Created", accessorKey: "created_at", header: "Created" },
  { id: "actions", header: "" },
]
const DEFAULT_SORT: SortingState = [{ id: "name", desc: false }]
const SEARCH_FIELDS = ["name"]
const FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"] },
]
/** Backed by an endpoint that splits the param on "," into an IN (...) clause. */
const MULTI_FILTER_GROUPS: readonly FilterGroup[] = [
  { key: "status", label: "Status", options: ["active", "inactive"], multiple: true },
]

/** Spy hook capturing the params the engine assembles. */
function makeUseData(result?: { rows: Row[]; total: number }) {
  const spy = vi.fn()
  const useData = (params: Record<string, unknown>) => {
    spy(params)
    return {
      data: result,
      isLoading: false,
      error: null as Error | null,
    }
  }
  return { spy, useData }
}

interface HarnessProps {
  options: Omit<UseServerDataTableOptions<Row, Record<string, unknown>>, "columns">
  onResult?: (result: UseServerDataTableResult<Row>) => void
}

function Harness({ options, onResult }: HarnessProps) {
  const result = useServerDataTable<Row>({ columns: COLUMNS, ...options })
  onResult?.(result)
  return (
    <div>
      <span data-testid="rowcount">{result.table.getRowModel().rows.length}</span>
      <span data-testid="pagecount">{result.table.getPageCount()}</span>
      <span data-testid="search">{result.search}</span>
      <span data-testid="chips">{result.activeFilters.join("|")}</span>
    </div>
  )
}

function renderHarness(props: HarnessProps, route = "/") {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <Routes>
        <Route path="/*" element={<Harness {...props} />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe("useServerDataTable", () => {
  it("uses defaults when the URL has no query params", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness({
      options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: FILTER_GROUPS, useData },
    })
    expect(spy).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 1,
        limit: 10,
        sort_by: "name",
        sort_order: "asc",
      }),
    )
    // No search/filter params present
    const params = spy.mock.calls[0][0]
    expect(params.name).toBeUndefined()
    expect(params.status).toBeUndefined()
    // pageCount when total=0
    expect(screen.getByTestId("pagecount").textContent).toBe("0")
  })

  it("seeds state from URL query params (search, filter group, sort desc, page, limit)", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness(
      {
        options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: MULTI_FILTER_GROUPS, useData },
      },
      "/t1?search=bob&status=active,inactive&sortBy=name&sortOrder=desc&page=3&limit=25",
    )
    expect(spy).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 3,
        limit: 25,
        sort_by: "name",
        sort_order: "desc",
        name: "bob",
        status: "active,inactive",
      }),
    )
    expect(screen.getByTestId("search").textContent).toBe("bob")
    expect(screen.getByTestId("chips").textContent).toBe("Status: active, inactive")
  })

  it("uses a custom defaultPageSize when limit is absent", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness({
      options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, useData, defaultPageSize: 50 },
    })
    expect(spy).toHaveBeenCalledWith(expect.objectContaining({ limit: 50 }))
  })

  it("defaults filterGroups to empty (omitted) — no filter params, no chips", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness({
      options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, useData },
    })
    const params = spy.mock.calls[0][0]
    expect(params.status).toBeUndefined()
    expect(screen.getByTestId("chips").textContent).toBe("")
  })

  it("uses apiKey instead of key when assembling filter params", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    const groups: readonly FilterGroup[] = [
      { key: "status", apiKey: "state", label: "Status", options: ["active"] },
    ]
    renderHarness(
      {
        options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: groups, useData },
      },
      "/t1?status=active",
    )
    expect(spy).toHaveBeenCalledWith(expect.objectContaining({ state: "active" }))
    const params = spy.mock.calls.at(-1)?.[0]
    expect(params.status).toBeUndefined()
  })

  it("falls back to defaultSort[0].id in apiParams when sorting is emptied", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    let captured: UseServerDataTableResult<Row> | undefined
    renderHarness({
      options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, useData },
      onResult: (r) => {
        captured = r
      },
    })
    act(() => {
      captured!.table.setSorting([])
    })
    const params = spy.mock.calls.at(-1)?.[0]
    expect(params.sort_by).toBe("name")
    expect(params.sort_order).toBe("asc")
  })

  it("returns rows=[] when data is undefined", () => {
    const { useData } = makeUseData(undefined)
    renderHarness({
      options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, useData },
    })
    expect(screen.getByTestId("rowcount").textContent).toBe("0")
    expect(screen.getByTestId("pagecount").textContent).toBe("0")
  })

  it("computes pageCount from total and exposes rows", () => {
    const { useData } = makeUseData({ rows: [{ id: "1", name: "a" }], total: 25 })
    renderHarness({
      options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, useData },
    })
    expect(screen.getByTestId("rowcount").textContent).toBe("1")
    // ceil(25 / 10) = 3
    expect(screen.getByTestId("pagecount").textContent).toBe("3")
  })

  it("clearFilters resets all groups", () => {
    const { useData } = makeUseData({ rows: [], total: 0 })
    let captured: UseServerDataTableResult<Row> | undefined
    renderHarness(
      {
        options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: FILTER_GROUPS, useData },
        onResult: (r) => {
          captured = r
        },
      },
      "/t1?status=active",
    )
    expect(captured!.activeFilters).toEqual(["Status: active"])
    act(() => {
      captured!.clearFilters()
    })
    expect(captured!.activeFilters).toEqual([])
  })
})

// Most list endpoints compare a filter param against the column verbatim; only the
// handlers that explicitly split on "," build an IN (...) clause. Serializing two
// selections as "a,b" to a verbatim endpoint matches the literal string "a,b" and
// returns zero rows, so the filter looked applied while guaranteeing an empty table.
// Multi-value is therefore opt-in per group via `multiple`.
describe("filter group arity", () => {
  it("clamps a single-select group seeded with several values from the URL", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness(
      {
        options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: FILTER_GROUPS, useData },
      },
      "/t1?status=active,inactive",
    )
    const params = spy.mock.calls.at(-1)?.[0]
    expect(params.status).toBe("inactive")
    expect(screen.getByTestId("chips").textContent).toBe("Status: inactive")
  })

  it("keeps every value for a group that opts in with multiple", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness(
      {
        options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: MULTI_FILTER_GROUPS, useData },
      },
      "/t1?status=active,inactive",
    )
    const params = spy.mock.calls.at(-1)?.[0]
    expect(params.status).toBe("active,inactive")
  })

  it("clamps a multi-value setFilters call on a single-select group", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    let captured: UseServerDataTableResult<Row> | undefined
    renderHarness({
      options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: FILTER_GROUPS, useData },
      onResult: (r) => {
        captured = r
      },
    })
    act(() => {
      captured!.setFilters({ status: ["active", "inactive"] })
    })
    expect(captured!.filters.status).toEqual(["inactive"])
    expect(spy.mock.calls.at(-1)?.[0].status).toBe("inactive")
  })

  it("keeps a multi-value setFilters call on a multiple group", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    let captured: UseServerDataTableResult<Row> | undefined
    renderHarness({
      options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: MULTI_FILTER_GROUPS, useData },
      onResult: (r) => {
        captured = r
      },
    })
    act(() => {
      captured!.setFilters({ status: ["active", "inactive"] })
    })
    expect(captured!.filters.status).toEqual(["active", "inactive"])
    expect(spy.mock.calls.at(-1)?.[0].status).toBe("active,inactive")
  })

  // The toolbar appends on check, so the newest value has to win — keeping the
  // first would leave the box the user just clicked unchecked and look like a
  // dead control.
  it("replaces the selection when a single-select group is toggled again", () => {
    const { useData } = makeUseData({ rows: [], total: 0 })
    let captured: UseServerDataTableResult<Row> | undefined
    renderHarness(
      {
        options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: FILTER_GROUPS, useData },
        onResult: (r) => {
          captured = r
        },
      },
      "/t1?status=active",
    )
    // How ListingToolbar reports a check: append to the group's current values.
    act(() => {
      captured!.setFilters({ status: [...captured!.filters.status, "inactive"] })
    })
    expect(captured!.filters.status).toEqual(["inactive"])
  })

  it("still clears to empty via an unchecking setFilters call", () => {
    const { useData } = makeUseData({ rows: [], total: 0 })
    let captured: UseServerDataTableResult<Row> | undefined
    renderHarness(
      {
        options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, filterGroups: FILTER_GROUPS, useData },
        onResult: (r) => {
          captured = r
        },
      },
      "/t1?status=active",
    )
    act(() => {
      captured!.setFilters({ status: [] })
    })
    expect(captured!.filters.status).toEqual([])
    expect(captured!.activeFilters).toEqual([])
  })
})

// Listing columns carry a display id so the view-options menu reads well, but the
// backend sanitizes sort_by against a per-resource allowlist — sending "Created"
// made it fall back to its default, so every header click looked like a no-op.
describe("sort field resolution", () => {
  it("sends the column's API field, not its display id", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness(
      { options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, useData } },
      "/t1?sortBy=Created&sortOrder=desc",
    )
    expect(spy).toHaveBeenCalledWith(
      expect.objectContaining({ sort_by: "created_at", sort_order: "desc" }),
    )
  })

  it("resolves the default sort the same way", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness({
      options: {
        defaultSort: [{ id: "Created", desc: true }],
        searchFields: SEARCH_FIELDS,
        useData,
      },
    })
    expect(spy).toHaveBeenCalledWith(expect.objectContaining({ sort_by: "created_at" }))
  })

  // A column with no accessorKey (a composite cell) has nothing to map to, so the id
  // is passed through and the server's allowlist remains the last word.
  it("passes an unmapped id through unchanged", () => {
    const { spy, useData } = makeUseData({ rows: [], total: 0 })
    renderHarness(
      { options: { defaultSort: DEFAULT_SORT, searchFields: SEARCH_FIELDS, useData } },
      "/t1?sortBy=actions",
    )
    expect(spy).toHaveBeenCalledWith(expect.objectContaining({ sort_by: "actions" }))
  })
})
