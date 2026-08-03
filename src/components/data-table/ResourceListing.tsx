import type { ReactNode } from "react"
import { Button } from "@/components/ui/button"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { DataTable } from "./DataTable"
import { DataTableEmpty } from "./DataTableEmpty"
import { DataTablePagination } from "./DataTablePagination"
import { DataTableActiveFilters } from "./DataTableActiveFilters"
import { ListingToolbar } from "./ListingToolbar"
import {
  useServerDataTable,
  type FilterGroup,
  type UseListData,
} from "./useServerDataTable"

interface ResourceListingProps<TRow, TParams> {
  columns: ColumnDef<TRow>[]
  defaultSort: SortingState
  /** API params the free-text search maps to. */
  searchFields: string[]
  searchPlaceholder: string
  /** The resource's strongly-typed list hook (e.g. `useUsers`). */
  useData: UseListData<TRow, TParams>
  filterGroups?: readonly FilterGroup[]
  /** Navigate when a row body is clicked (the actions menu is ignored). */
  onRowClick?: (row: TRow) => void
  onCreate?: () => void
  createLabel?: string
  defaultPageSize?: number
  /** Empty-state headline shown when no rows exist yet (before any search/filter). */
  emptyTitle?: string
  /** Empty-state supporting copy shown alongside `emptyTitle`. */
  emptyDescription?: string
  /** When true, the table is wrapped in its own bordered card while the toolbar
   *  (above) and pagination (below) render outside it. The page must NOT add its
   *  own encapsulating card in this mode. */
  tableInCard?: boolean
  /** Additional action elements (e.g. Export) rendered between the column-toggle
   *  and the create button in the toolbar. */
  extraActions?: ReactNode
}

/**
 * The standard server-driven listing: toolbar (search + filters + create) →
 * active-filter chips → table → pagination. A listing page declares its columns
 * and a small config; all the state/URL/query/table wiring lives in the shared
 * `useServerDataTable` engine.
 */
export function ResourceListing<TRow, TParams = Record<string, unknown>>({
  columns,
  defaultSort,
  searchFields,
  searchPlaceholder,
  useData,
  filterGroups,
  onRowClick,
  onCreate,
  createLabel,
  defaultPageSize,
  emptyTitle = "Nothing here yet",
  emptyDescription,
  tableInCard = false,
  extraActions,
}: ResourceListingProps<TRow, TParams>) {
  const { table, isLoading, error, search, setSearch, filters, setFilters, activeFilters, clearFilters } =
    useServerDataTable<TRow, TParams>({
      columns,
      defaultSort,
      searchFields,
      filterGroups,
      useData,
      defaultPageSize,
    })

  // A search term or applied filter means rows may exist but are hidden — offer
  // a reset rather than a create CTA, which wouldn't help the user here.
  const isFiltered = search.trim() !== "" || activeFilters.length > 0
  const emptyState = isFiltered ? (
    <DataTableEmpty
      variant="no-results"
      title="No results found"
      description="No records match your current search and filters."
    >
      <Button
        variant="outline"
        size="sm"
        className="mt-1"
        onClick={() => {
          setSearch("")
          clearFilters()
        }}
      >
        Clear search & filters
      </Button>
    </DataTableEmpty>
  ) : (
    <DataTableEmpty
      title={emptyTitle}
      description={emptyDescription}
      onAction={onCreate}
      actionLabel={createLabel}
    />
  )

  return (
    <div className="space-y-4">
      <ListingToolbar
        table={table}
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder={searchPlaceholder}
        filterGroups={filterGroups}
        filters={filters}
        onFiltersChange={setFilters}
        onCreate={onCreate}
        createLabel={createLabel}
        extraActions={extraActions}
      />
      <DataTableActiveFilters activeFilters={activeFilters} onClearAll={clearFilters} />
      {tableInCard ? (
        // Table in its own bordered card; the toolbar (above) and pagination
        // (below) sit outside it. Cells re-pad to a 16px gutter, and the table's
        // own bottom rule + last-row rule are dropped so the card border is the
        // single bottom line.
        <div data-md-table-shell className="overflow-hidden rounded-[3px] border border-border bg-card [&_table]:border-b-0 [&_tbody_tr:last-child]:border-b-0 md:[&_td:first-child]:pl-4 md:[&_td:last-child]:pr-4 md:[&_th:first-child]:pl-4 md:[&_th:last-child]:pr-4">
          <DataTable
            table={table}
            columnCount={columns.length}
            isLoading={isLoading}
            error={error}
            emptyState={emptyState}
            onRowClick={onRowClick}
          />
        </div>
      ) : (
        // The table bleeds past the card's px-6 to touch its side edges (-mx-6),
        // while the first/last columns re-pad to the same 24px gutter so cell
        // content stays aligned with the toolbar and pagination above/below.
        <div className="-mx-6 md:[&_td:first-child]:pl-6 md:[&_td:last-child]:pr-6 md:[&_th:first-child]:pl-6 md:[&_th:last-child]:pr-6">
          <DataTable
            table={table}
            columnCount={columns.length}
            isLoading={isLoading}
            error={error}
            emptyState={emptyState}
            onRowClick={onRowClick}
          />
        </div>
      )}
      <DataTablePagination table={table} />
    </div>
  )
}
