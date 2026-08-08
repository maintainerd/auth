import { describe, it, expect } from "vitest"
import { render } from "@testing-library/react"
import { ListSkeleton } from "./ListSkeleton"

// Each skeleton row is a bordered flex container; count those to assert row count.
function rowCount(container: HTMLElement) {
  return container.querySelectorAll(".rounded-lg.border.p-4").length
}

describe("ListSkeleton", () => {
  it("renders three rows by default", () => {
    const { container } = render(<ListSkeleton />)
    expect(rowCount(container)).toBe(3)
  })

  it("renders the explicit number of rows", () => {
    const { container } = render(<ListSkeleton rows={5} />)
    expect(rowCount(container)).toBe(5)
  })
})
