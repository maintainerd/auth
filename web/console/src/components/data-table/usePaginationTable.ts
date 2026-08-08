import {
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type OnChangeFn,
  type PaginationState,
  type Table,
} from "@tanstack/react-table"

// The pagination control reads pagination state + page count only — never the
// rows (server-driven lists render their own rows), so these stay empty.
const EMPTY_ROWS: unknown[] = []
const EMPTY_COLUMNS: ColumnDef<unknown>[] = []

/**
 * Builds a minimal TanStack table whose only job is to drive
 * <DataTablePagination/> for a server-paginated list that renders its own rows.
 *
 * Replaces the empty-column `useReactTable` boilerplate each detail-tab list
 * used to repeat. Keep pagination state in the page (a single `useState`) and
 * pass it in alongside the server `pageCount`.
 */
export function usePaginationTable({
  pagination,
  onPaginationChange,
  pageCount,
}: {
  pagination: PaginationState
  onPaginationChange: OnChangeFn<PaginationState>
  pageCount: number
}): Table<unknown> {
  return useReactTable<unknown>({
    data: EMPTY_ROWS,
    columns: EMPTY_COLUMNS,
    pageCount,
    state: { pagination },
    onPaginationChange,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
  })
}
