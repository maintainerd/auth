import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { StatusBadge } from "./StatusBadge"

describe("StatusBadge", () => {
  // Each case maps a status to the dot colour class it should resolve to.
  const cases: { status: string; dot: string }[] = [
    { status: "active", dot: "bg-emerald-500" },
    { status: "pending", dot: "bg-amber-500" },
    { status: "inactive", dot: "bg-muted-foreground" },
    { status: "suspended", dot: "bg-red-500" },
  ]

  for (const { status, dot } of cases) {
    it(`renders the "${status}" label with the ${dot} dot`, () => {
      const { container } = render(<StatusBadge status={status} />)
      const badge = screen.getByText(status)
      expect(badge).toHaveClass("capitalize")
      // The dot is the first child span carrying the colour class.
      expect(container.querySelector(`.${dot}`)).toBeInTheDocument()
    })
  }

  it("falls back to the neutral dot for an unknown status", () => {
    const { container } = render(<StatusBadge status="mystery" />)
    expect(screen.getByText("mystery")).toBeInTheDocument()
    expect(container.querySelector(".bg-muted-foreground")).toBeInTheDocument()
  })

  it("matches statuses case-insensitively and merges a custom className", () => {
    const { container } = render(<StatusBadge status="ACTIVE" className="custom-class" />)
    expect(container.querySelector(".bg-emerald-500")).toBeInTheDocument()
    expect(screen.getByText("ACTIVE")).toHaveClass("custom-class")
  })
})
