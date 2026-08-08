import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { Mail, Calendar } from "lucide-react"
import { DetailHeaderCard, type DetailAttribute } from "./DetailHeaderCard"

describe("DetailHeaderCard", () => {
  it("renders every optional slot when provided", () => {
    const attributes: DetailAttribute[] = [
      { icon: Mail, label: "Email", value: "jdoe@example.com" },
      { icon: Calendar, label: "Created", value: "2024-01-01" },
    ]
    render(
      <DetailHeaderCard
        leading={<div data-testid="leading">L</div>}
        title="John Doe"
        badge={<span data-testid="badge">Active</span>}
        subtitle={<span data-testid="subtitle">jdoe</span>}
        actions={<button type="button">Edit</button>}
        attributes={attributes}
      />,
    )

    expect(screen.getByRole("heading", { name: "John Doe" })).toBeInTheDocument()
    expect(screen.getByTestId("leading")).toBeInTheDocument()
    expect(screen.getByTestId("badge")).toBeInTheDocument()
    expect(screen.getByTestId("subtitle")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument()
    expect(screen.getByText("Email")).toBeInTheDocument()
    expect(screen.getByText("jdoe@example.com")).toBeInTheDocument()
    expect(screen.getByText("Created")).toBeInTheDocument()
  })

  it("renders only the title when no optional props are provided", () => {
    render(<DetailHeaderCard title="Bare Title" />)
    expect(screen.getByRole("heading", { name: "Bare Title" })).toBeInTheDocument()
    // No actions and no attribute grid headings.
    expect(screen.queryByRole("button")).not.toBeInTheDocument()
    expect(screen.queryByText("Email")).not.toBeInTheDocument()
  })

  it("omits the attribute grid when attributes is an empty array", () => {
    render(<DetailHeaderCard title="Empty Attrs" attributes={[]} />)
    expect(screen.getByRole("heading", { name: "Empty Attrs" })).toBeInTheDocument()
    expect(screen.queryByText("Email")).not.toBeInTheDocument()
  })
})
