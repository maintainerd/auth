import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Smartphone } from "lucide-react"

import { ListingItemCard } from "./ListingItemCard"

describe("ListingItemCard", () => {
  it("renders shared listing icon and action slots", () => {
    const { container } = render(
      <ListingItemCard icon={Smartphone} action={<span>Manage</span>}>
        Mobile session
      </ListingItemCard>,
    )

    expect(screen.getByText("Mobile session")).toBeInTheDocument()
    expect(screen.getByText("Manage")).toBeInTheDocument()
    expect(container.querySelector("[data-md-listing-item]")).toBeInTheDocument()
    expect(container.querySelector("[data-md-listing-icon]")).toBeInTheDocument()
  })

  it("can render as an interactive button row", async () => {
    const onClick = vi.fn()
    const user = userEvent.setup({ pointerEventsCheck: 0 })
    render(
      <ListingItemCard as="button" icon={Smartphone} action={<span>Open</span>} onClick={onClick}>
        Security method
      </ListingItemCard>,
    )

    await user.click(screen.getByRole("button", { name: /security method open/i }))
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})
