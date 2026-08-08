import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { Checkbox } from "./checkbox"

describe("Checkbox", () => {
  it("renders no indicator when unchecked", () => {
    render(<Checkbox checked={false} aria-label="pick" />)
    const box = screen.getByRole("checkbox", { name: "pick" })
    expect(box).toHaveAttribute("aria-checked", "false")
    expect(box.querySelector('[data-slot="checkbox-indicator"]')).toBeNull()
  })

  it("renders the tick when fully checked", () => {
    render(<Checkbox checked aria-label="pick" />)
    const box = screen.getByRole("checkbox", { name: "pick" })
    expect(box).toHaveAttribute("aria-checked", "true")
    // The minus glyph is mounted but display:none in this state; the tick is the
    // one that shows.
    expect(box.querySelector('[data-slot="checkbox-check"]')).not.toHaveClass("hidden")
    expect(box.querySelector('[data-slot="checkbox-minus"]')).toHaveClass("hidden")
  })

  it("renders a minus, not a tick, for the indeterminate state", () => {
    // Radix mounts the indicator for "indeterminate" as well as for true, so a
    // lone CheckIcon drew a full tick on a partially-selected tri-state box —
    // the glyph claimed "all selected" while aria-checked said "mixed".
    render(<Checkbox checked="indeterminate" aria-label="pick" />)
    const box = screen.getByRole("checkbox", { name: "pick" })
    expect(box).toHaveAttribute("aria-checked", "mixed")
    expect(box).toHaveAttribute("data-state", "indeterminate")

    const check = box.querySelector('[data-slot="checkbox-check"]')
    const minus = box.querySelector('[data-slot="checkbox-minus"]')
    expect(check).toBeInTheDocument()
    expect(minus).toBeInTheDocument()
    // Tailwind resolves these at build time, so assert the variant classes that
    // do the swap rather than computed styles jsdom will not produce.
    expect(check).toHaveClass("group-data-[state=indeterminate]/checkbox:hidden")
    expect(minus).toHaveClass("group-data-[state=indeterminate]/checkbox:block")
  })
})
