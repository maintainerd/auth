import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { DetailLayout } from "./DetailLayout"

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

describe("DetailLayout", () => {
  it("shows the loading skeleton and not the children while isLoading", () => {
    render(
      <DetailLayout backLabel="Back to users" onBack={vi.fn()} isLoading>
        <div>loaded content</div>
      </DetailLayout>,
    )
    expect(screen.queryByText("loaded content")).not.toBeInTheDocument()
    // Default not-found copy is not shown in the loading state.
    expect(screen.queryByText("Not found")).not.toBeInTheDocument()
  })

  it("shows the default not-found card on error and fires onBack from its button", async () => {
    const onBack = vi.fn()
    const user = u()
    render(
      <DetailLayout backLabel="Back to users" onBack={onBack} isError>
        <div>loaded content</div>
      </DetailLayout>,
    )
    expect(screen.getByRole("heading", { name: "Not found" })).toBeInTheDocument()
    expect(
      screen.getByText("The item you're looking for doesn't exist or may have been removed."),
    ).toBeInTheDocument()
    expect(screen.queryByText("loaded content")).not.toBeInTheDocument()

    // The card's own back button (there are two "Back to users" buttons total).
    const backButtons = screen.getAllByRole("button", { name: /back to users/i })
    await user.click(backButtons[backButtons.length - 1])
    expect(onBack).toHaveBeenCalledTimes(1)
  })

  it("shows custom not-found title and description when provided", () => {
    render(
      <DetailLayout
        backLabel="Back"
        onBack={vi.fn()}
        isError
        notFoundTitle="User missing"
        notFoundDescription="That user is gone."
      >
        <div>loaded content</div>
      </DetailLayout>,
    )
    expect(screen.getByRole("heading", { name: "User missing" })).toBeInTheDocument()
    expect(screen.getByText("That user is gone.")).toBeInTheDocument()
  })

  it("renders children on success and fires onBack from the top back link", async () => {
    const onBack = vi.fn()
    const user = u()
    render(
      <DetailLayout backLabel="Back to users" onBack={onBack}>
        <div>loaded content</div>
      </DetailLayout>,
    )
    expect(screen.getByText("loaded content")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /back to users/i }))
    expect(onBack).toHaveBeenCalledTimes(1)
  })
})
