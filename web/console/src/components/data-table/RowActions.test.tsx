import { describe, it, expect, vi } from "vitest"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Eye, Trash2 } from "lucide-react"
import { renderWithProviders } from "@/test/utils"
import { RowActions, type RowActionItem } from "./RowActions"

const u = () => userEvent.setup({ pointerEventsCheck: 0 })

async function openMenu(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: /open menu/i }))
}

describe("RowActions", () => {
  it("runs onSelect immediately for an item with no confirm", async () => {
    const onSelect = vi.fn()
    const items: RowActionItem[] = [
      { key: "view", label: "View", icon: Eye, onSelect },
    ]
    const user = u()
    renderWithProviders(<RowActions items={items} />)
    await openMenu(user)
    await user.click(await screen.findByText("View"))
    expect(onSelect).toHaveBeenCalled()
  })

  it("opens a non-destructive confirm dialog and runs onSelect on Confirm", async () => {
    const onSelect = vi.fn().mockResolvedValue(undefined)
    const items: RowActionItem[] = [
      {
        key: "activate",
        label: "Activate",
        icon: Eye,
        onSelect,
        confirm: { title: "Activate?", description: "Sure?", confirmText: "Confirm" },
      },
    ]
    const user = u()
    renderWithProviders(<RowActions items={items} />)
    await openMenu(user)
    await user.click(await screen.findByText("Activate"))

    const dialog = await screen.findByRole("dialog")
    await user.click(within(dialog).getByRole("button", { name: "Confirm" }))
    await waitFor(() => expect(onSelect).toHaveBeenCalled())
  })

  it("renders a separator before an item and applies destructive styling, with the delete dialog flow", async () => {
    const onSelect = vi.fn().mockResolvedValue(undefined)
    const items: RowActionItem[] = [
      { key: "view", label: "View", icon: Eye, onSelect: vi.fn() },
      {
        key: "delete",
        label: "Delete",
        icon: Trash2,
        destructive: true,
        separatorBefore: true,
        onSelect,
        confirm: {
          title: "Delete?",
          description: "This is permanent.",
          destructive: true,
          itemName: "thing-1",
        },
      },
    ]
    const user = u()
    renderWithProviders(<RowActions items={items} />)
    await openMenu(user)

    const deleteItem = await screen.findByText("Delete")
    // destructive styling
    expect(deleteItem.closest('[role="menuitem"]')).toHaveClass("text-destructive")
    await user.click(deleteItem)

    const dialog = await screen.findByRole("dialog")
    await user.type(within(dialog).getByRole("textbox"), "thing-1")
    await user.click(within(dialog).getByRole("button", { name: "Delete" }))
    await waitFor(() => expect(onSelect).toHaveBeenCalled())
  })

  it("falls back to an empty itemName when the destructive confirm omits one", async () => {
    const onSelect = vi.fn().mockResolvedValue(undefined)
    const items: RowActionItem[] = [
      {
        key: "delete",
        label: "Delete",
        icon: Trash2,
        onSelect,
        confirm: { title: "Delete?", description: "Permanent.", destructive: true },
      },
    ]
    const user = u()
    renderWithProviders(<RowActions items={items} />)
    await openMenu(user)
    await user.click(await screen.findByText("Delete"))
    // With an empty itemName, typing "" already satisfies the confirm guard.
    const dialog = await screen.findByRole("dialog")
    await user.click(within(dialog).getByRole("button", { name: "Delete" }))
    await waitFor(() => expect(onSelect).toHaveBeenCalled())
  })

  it("cancelling the dialog (onOpenChange false) clears the active confirm without running onSelect", async () => {
    const onSelect = vi.fn()
    const items: RowActionItem[] = [
      {
        key: "activate",
        label: "Activate",
        icon: Eye,
        onSelect,
        confirm: { title: "Activate?", description: "Sure?" },
      },
    ]
    const user = u()
    renderWithProviders(<RowActions items={items} />)
    await openMenu(user)
    await user.click(await screen.findByText("Activate"))

    const dialog = await screen.findByRole("dialog")
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(onSelect).not.toHaveBeenCalled()
  })
})
