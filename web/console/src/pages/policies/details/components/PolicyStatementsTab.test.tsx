import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/utils"
import { PolicyStatementsTab } from "./PolicyStatementsTab"
import type { PolicyStatement } from "@/services/api/policies/types"

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

const allowStatement: PolicyStatement = {
  effect: "allow",
  action: ["users:read", "users:*"],
  resource: ["*"],
}

const denyStatement: PolicyStatement = {
  effect: "deny",
  action: ["users:delete"],
  resource: ["tenant:acme"],
}

describe("PolicyStatementsTab", () => {
  it("renders the empty state when there are no statements", () => {
    renderWithProviders(<PolicyStatementsTab documentVersion="2012-10-17" statements={[]} />)
    expect(screen.getByText("No statements")).toBeInTheDocument()
    expect(screen.getByText("Document 2012-10-17")).toBeInTheDocument()
  })

  it("renders allow and deny statements with their summaries", () => {
    renderWithProviders(
      <PolicyStatementsTab documentVersion="2012-10-17" statements={[allowStatement, denyStatement]} />,
    )
    expect(screen.getByText("Statement 01")).toBeInTheDocument()
    expect(screen.getByText("Statement 02")).toBeInTheDocument()
    expect(screen.getByText("allow")).toBeInTheDocument()
    expect(screen.getByText("deny")).toBeInTheDocument()
    expect(screen.getByText("Overrides allows")).toBeInTheDocument()
    expect(screen.getByText(/Grants 2 actions against 1 resource pattern/)).toBeInTheDocument()
    expect(screen.getByText(/Blocks 1 action against 1 resource pattern/)).toBeInTheDocument()
  })

  it("expands a statement to reveal its action and resource patterns", async () => {
    renderWithProviders(
      <PolicyStatementsTab documentVersion="2012-10-17" statements={[allowStatement]} />,
    )
    expect(screen.queryByText("Matched Patterns")).not.toBeInTheDocument()

    await u().click(screen.getByRole("button", { name: /expand statement/i }))

    expect(screen.getByText("Matched Patterns")).toBeInTheDocument()
    expect(screen.getByText("users:read")).toBeInTheDocument()
    expect(screen.getByText("users:*")).toBeInTheDocument()
    expect(screen.getByText("Global wildcard")).toBeInTheDocument()
    expect(screen.getByText("Wildcard")).toBeInTheDocument()
  })
})
