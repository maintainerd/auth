import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { Inbox } from "lucide-react"
import { EmptyState } from "./EmptyState"

describe("EmptyState", () => {
  it("renders with only the required icon and title", () => {
    render(<EmptyState icon={Inbox} title="Nothing here" />)
    expect(screen.getByText("Nothing here")).toBeInTheDocument()
    // No description paragraph and no action when those props are omitted.
    expect(screen.queryByText(/try again/i)).not.toBeInTheDocument()
  })

  it("renders the optional description and action when provided", () => {
    render(
      <EmptyState
        icon={Inbox}
        title="Nothing here"
        description="There is nothing to show yet."
        action={<button type="button">Try again</button>}
      />,
    )
    expect(screen.getByText("There is nothing to show yet.")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument()
  })
})
