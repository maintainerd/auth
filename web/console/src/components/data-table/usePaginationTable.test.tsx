import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useState } from "react"
import type { PaginationState } from "@tanstack/react-table"
import { usePaginationTable } from "./usePaginationTable"
import { DataTablePagination } from "./DataTablePagination"

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

interface HarnessProps {
  pageCount: number
  rowCount: number
  initial?: PaginationState
  onChangeSpy?: (next: PaginationState) => void
}

/**
 * Drives the hook with real local pagination state and renders the pagination
 * control so its buttons exercise the `onPaginationChange` updater path.
 */
function Harness({
  pageCount,
  rowCount,
  initial = { pageIndex: 0, pageSize: 10 },
  onChangeSpy,
}: HarnessProps) {
  const [pagination, setPagination] = useState<PaginationState>(initial)
  const table = usePaginationTable({
    pagination,
    onPaginationChange: (updater) => {
      setPagination((prev) => {
        const next = typeof updater === "function" ? updater(prev) : updater
        onChangeSpy?.(next)
        return next
      })
    },
    pageCount,
  })

  return (
    <div>
      <div data-testid="page-index">{table.getState().pagination.pageIndex}</div>
      <div data-testid="page-size">{table.getState().pagination.pageSize}</div>
      <div data-testid="page-count">{table.getPageCount()}</div>
      <DataTablePagination table={table} rowCount={rowCount} />
    </div>
  )
}

describe("usePaginationTable", () => {
  it("reflects the supplied pageCount and initial pagination state", () => {
    render(<Harness pageCount={5} rowCount={42} />)
    expect(screen.getByTestId("page-count")).toHaveTextContent("5")
    expect(screen.getByTestId("page-index")).toHaveTextContent("0")
    expect(screen.getByTestId("page-size")).toHaveTextContent("10")
    expect(screen.getByText(/of 42 results/i)).toBeInTheDocument()
  })

  it("advances the page index via the next-page control", async () => {
    const onChangeSpy = vi.fn()
    const user = u()
    render(<Harness pageCount={5} rowCount={42} onChangeSpy={onChangeSpy} />)

    await user.click(screen.getByRole("button", { name: /go to next page/i }))
    expect(screen.getByTestId("page-index")).toHaveTextContent("1")
    expect(onChangeSpy).toHaveBeenCalledWith({ pageIndex: 1, pageSize: 10 })
  })

  it("jumps to the last page using the supplied pageCount", async () => {
    const user = u()
    render(<Harness pageCount={5} rowCount={42} />)
    await user.click(screen.getByRole("button", { name: /go to last page/i }))
    // pageCount 5 -> last page index is 4.
    expect(screen.getByTestId("page-index")).toHaveTextContent("4")
  })

  it("changes the page size via the rows-per-page select", async () => {
    const onChangeSpy = vi.fn()
    const user = u()
    render(<Harness pageCount={5} rowCount={42} onChangeSpy={onChangeSpy} />)

    await user.click(screen.getByRole("combobox"))
    await user.click(await screen.findByRole("option", { name: "20" }))

    expect(screen.getByTestId("page-size")).toHaveTextContent("20")
    expect(onChangeSpy).toHaveBeenCalledWith({ pageIndex: 0, pageSize: 20 })
  })
})
