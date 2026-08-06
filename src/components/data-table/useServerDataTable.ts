import * as React from "react"
import { useSearchParams } from "react-router-dom"
import type {
  ColumnDef,
  ColumnFiltersState,
  PaginationState,
  SortingState,
  Table,
  VisibilityState,
} from "@tanstack/react-table"
import {
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table"

/**
 * A filter group for a listing.
 *
 * The `key` is the state key, the URL query key, and (by default) the API param
 * name; set `apiKey` if the backend expects a different name.
 */
export interface FilterGroup {
  key: string
  label: string
  options: readonly string[]
  apiKey?: string
  /**
   * Opt in to selecting several values at once. Defaults to `false`.
   *
   * Only some list endpoints split their filter param on commas into an
   * `IN (...)` clause; the rest compare the param against the column verbatim.
   * Sending `status=active,maintenance` to one of those compares the column
   * against the literal string "active,maintenance", which matches no row — so
   * the filter looks applied while guaranteeing an empty table. Set this only
   * for a group whose endpoint is known to split the value.
   */
  multiple?: boolean
}

/** Listing filter state: each group key → the selected values. */
export type ListingFilters = Record<string, string[]>

/** The shape every paginated list endpoint returns inside `data`. */
export interface ServerListResult<TRow> {
  rows: TRow[]
  total: number
}

/**
 * A TanStack-Query list hook: `(apiParams) => { data, isLoading, error }`.
 *
 * `TParams` lets a page pass its strongly-typed list hook (e.g. `useUsers`)
 * directly; the engine builds generic params internally and casts to `TParams`
 * once, here, so no listing page needs an adapter or cast.
 */
export type UseListData<TRow, TParams> = (params: TParams) => {
  data?: ServerListResult<TRow>
  isLoading: boolean
  error: Error | null
}

export interface UseServerDataTableOptions<TRow, TParams> {
  columns: ColumnDef<TRow>[]
  /** Default sort applied when the URL doesn't specify one. */
  defaultSort: SortingState
  /** API params the free-text search maps to (the server matches against them). */
  searchFields: string[]
  /** TanStack-Query list hook: `(apiParams) => { data, isLoading, error }`. */
  useData: UseListData<TRow, TParams>
  /** Stable (module-level) filter group config. */
  filterGroups?: readonly FilterGroup[]
  defaultPageSize?: number
}

export interface UseServerDataTableResult<TRow> {
  table: Table<TRow>
  isLoading: boolean
  error: Error | null
  search: string
  setSearch: (value: string) => void
  filters: ListingFilters
  setFilters: (filters: ListingFilters) => void
  /** Human-readable "Label: a, b" chips for the active filters. */
  activeFilters: string[]
  clearFilters: () => void
}

const EMPTY_FILTER_GROUPS: readonly FilterGroup[] = []

/**
 * Resolves a column id to the API field the server can sort on.
 *
 * Listing columns carry a human display id ("Created", "Client") so the view-options
 * menu reads well, but `sort_by` has to be a real column name — the backend
 * sanitizes against a per-resource allowlist and silently falls back to its default
 * otherwise, which made every header click look like it did nothing. The column's
 * own `accessorKey` is that name, so the mapping needs no per-listing config.
 */
function resolveSortField<TRow>(columns: ColumnDef<TRow>[], columnId: string): string {
  const column = columns.find((c) => c.id === columnId)
  const accessorKey = (column as { accessorKey?: unknown } | undefined)?.accessorKey
  return typeof accessorKey === "string" ? accessorKey : columnId
}

/**
 * Drops all but the newest selection from every group that isn't `multiple`.
 *
 * A group without `multiple` is backed by an endpoint that compares its filter
 * param verbatim, so a two-value selection serialized as "a,b" matches nothing
 * and the listing silently renders zero rows. Clamping here — rather than in
 * the toolbar — keeps the guarantee on every path into filter state (URL seed,
 * checkbox toggle, programmatic set), so no caller can produce that query.
 *
 * The newest value wins because the toolbar appends on check: keeping the first
 * would leave the box the user just clicked unchecked and look like a dead control.
 */
function clampToGroupArity(
  filterGroups: readonly FilterGroup[],
  filters: ListingFilters,
): ListingFilters {
  const clamped: ListingFilters = {}
  for (const group of filterGroups) {
    const values = filters[group.key] ?? []
    clamped[group.key] = group.multiple ? values : values.slice(-1)
  }
  return clamped
}

/**
 * The shared engine for server-driven listing tables: URL-synced search / filters /
 * sorting / pagination, API-param assembly, and the TanStack table — so a listing
 * page only declares its columns + a small config instead of re-implementing all of it.
 *
 * Pass `columns`, `filterGroups`, `searchFields`, and `defaultSort` as stable
 * (module-level) values.
 */
export function useServerDataTable<TRow, TParams = Record<string, unknown>>({
  columns,
  defaultSort,
  searchFields,
  useData,
  filterGroups = EMPTY_FILTER_GROUPS,
  defaultPageSize = 10,
}: UseServerDataTableOptions<TRow, TParams>): UseServerDataTableResult<TRow> {
  const [searchParams, setSearchParams] = useSearchParams()

  const [search, setSearch] = React.useState(() => searchParams.get("search") || "")
  const [filters, setFiltersState] = React.useState<ListingFilters>(() => {
    const initial: ListingFilters = {}
    for (const group of filterGroups) {
      initial[group.key] = searchParams.get(group.key)?.split(",").filter(Boolean) ?? []
    }
    return clampToGroupArity(filterGroups, initial)
  })
  const [sorting, setSorting] = React.useState<SortingState>(() => {
    const sortBy = searchParams.get("sortBy")
    return sortBy ? [{ id: sortBy, desc: searchParams.get("sortOrder") === "desc" }] : defaultSort
  })
  const [pagination, setPagination] = React.useState<PaginationState>(() => ({
    pageIndex: Math.max(0, Number(searchParams.get("page") || 1) - 1),
    pageSize: Number(searchParams.get("limit") || defaultPageSize),
  }))
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([])
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({})

  const apiParams = React.useMemo(() => {
    const params: Record<string, unknown> = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
      sort_by:
        sorting.length > 0
          ? resolveSortField(columns, sorting[0].id)
          : defaultSort[0]?.id && resolveSortField(columns, defaultSort[0].id),
      sort_order: sorting.length > 0 && sorting[0].desc ? "desc" : "asc",
    }
    if (search) {
      for (const field of searchFields) params[field] = search
    }
    for (const group of filterGroups) {
      const values = filters[group.key]
      // Comma is the server's own multi-value encoding — the handlers that accept
      // several values split the raw param on "," and build an IN (...) clause.
      // Repeating the param instead would not work: the query builder appends one
      // entry per key and the handlers read only the first occurrence. Groups
      // without `multiple` are clamped to one value, so this join is a no-op there.
      if (values?.length) params[group.apiKey ?? group.key] = values.join(",")
    }
    return params
  }, [search, filters, sorting, pagination, defaultSort, searchFields, filterGroups, columns])

  const { data, isLoading, error } = useData(apiParams as unknown as TParams)
  const rows = React.useMemo(() => data?.rows ?? [], [data])
  const rowCount = data?.total ?? 0

  const table = useReactTable({
    data: rows,
    columns,
    pageCount: Math.ceil(rowCount / pagination.pageSize) || 0,
    manualPagination: true,
    manualSorting: true,
    onSortingChange: setSorting,
    onPaginationChange: setPagination,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    state: { sorting, pagination, columnFilters, columnVisibility },
  })

  // Mirror state to the URL so listings are shareable / bookmarkable.
  React.useEffect(() => {
    const params = new URLSearchParams(searchParams)
    if (search) params.set("search", search)
    for (const group of filterGroups) {
      const values = filters[group.key]
      if (values?.length) params.set(group.key, values.join(","))
    }
    if (sorting.length > 0) {
      params.set("sortBy", sorting[0].id)
      params.set("sortOrder", sorting[0].desc ? "desc" : "asc")
    }
    params.set("page", String(pagination.pageIndex + 1))
    params.set("limit", String(pagination.pageSize))
    setSearchParams(params, { replace: true })
  }, [search, filters, sorting, pagination, filterGroups, searchParams, setSearchParams])

  const activeFilters = React.useMemo(() => {
    const chips: string[] = []
    for (const group of filterGroups) {
      const values = filters[group.key]
      if (values?.length) chips.push(`${group.label}: ${values.join(", ")}`)
    }
    return chips
  }, [filters, filterGroups])

  // Every write to filter state goes through the clamp, so a single-select group
  // can never hold the two values that would serialize to an unmatchable "a,b".
  const setFilters = React.useCallback(
    (next: ListingFilters) => setFiltersState(clampToGroupArity(filterGroups, next)),
    [filterGroups],
  )

  const clearFilters = React.useCallback(() => {
    const cleared: ListingFilters = {}
    for (const group of filterGroups) cleared[group.key] = []
    setFiltersState(cleared)
  }, [filterGroups])

  return {
    table,
    isLoading,
    error,
    search,
    setSearch,
    filters,
    setFilters,
    activeFilters,
    clearFilters,
  }
}
