import { useState } from "react"
import type { LucideIcon } from "lucide-react"
import { MoreHorizontal, ChevronDown } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ConfirmationDialog, DeleteConfirmationDialog } from "@/components/dialog"

export interface RowActionConfirm {
  title: string
  description: string
  confirmText?: string
  /** Use the type-to-confirm delete dialog (with `itemName`) instead of the simple one. */
  destructive?: boolean
  itemName?: string
}

export interface RowActionItem {
  key: string
  label: string
  icon: LucideIcon
  /** The work to run (navigate, or a mutation). Awaited when there's a confirm step. */
  onSelect: () => void | Promise<void>
  /** Styles the item as destructive. */
  destructive?: boolean
  /** When set, the item opens a confirmation dialog before running `onSelect`. */
  confirm?: RowActionConfirm
  /** Render a separator above this item. */
  separatorBefore?: boolean
}

/**
 * Shared row-actions menu for listings: a `⋯` dropdown built from a declarative
 * item list, with the confirm/delete dialog flow handled here — so each listing
 * only declares its actions instead of re-implementing the menu + dialogs.
 */
export function RowActions({
  items,
  variant = "row",
}: {
  items: RowActionItem[]
  /**
   * "row" (default) — the compact ghost `⋯` used inside table rows.
   * "header" — matches the main detail-card action button (outline, `⋮`), for
   * use in an InformationCard's `action` slot.
   */
  variant?: "row" | "header"
}) {
  const [activeConfirm, setActiveConfirm] = useState<RowActionItem | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSelect = (item: RowActionItem) => {
    if (item.confirm) setActiveConfirm(item)
    else void item.onSelect()
  }

  const runConfirm = async (item: RowActionItem) => {
    setIsSubmitting(true)
    try {
      await item.onSelect()
    } finally {
      setIsSubmitting(false)
      setActiveConfirm(null)
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          {variant === "header" ? (
            <Button data-md-action-button variant="outline" size="sm" className="h-9 gap-1.5">
              Actions
              <ChevronDown className="size-4 opacity-70" />
            </Button>
          ) : (
            <Button data-md-icon-action-button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">Open menu</span>
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          )}
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {items.map((item) => (
            <div key={item.key}>
              {item.separatorBefore && <DropdownMenuSeparator />}
              <DropdownMenuItem
                onClick={() => handleSelect(item)}
                className={item.destructive ? "text-destructive" : undefined}
              >
                <item.icon className="mr-2 h-4 w-4" />
                {item.label}
              </DropdownMenuItem>
            </div>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {activeConfirm?.confirm &&
        (activeConfirm.confirm.destructive ? (
          <DeleteConfirmationDialog
            open
            onOpenChange={(open) => !open && setActiveConfirm(null)}
            onConfirm={() => runConfirm(activeConfirm)}
            title={activeConfirm.confirm.title}
            description={activeConfirm.confirm.description}
            itemName={activeConfirm.confirm.itemName ?? ""}
            isDeleting={isSubmitting}
            confirmLabel={activeConfirm.confirm.confirmText}
          />
        ) : (
          <ConfirmationDialog
            open
            onOpenChange={(open) => !open && setActiveConfirm(null)}
            onConfirm={() => runConfirm(activeConfirm)}
            title={activeConfirm.confirm.title}
            description={activeConfirm.confirm.description}
            confirmText={activeConfirm.confirm.confirmText}
            variant={activeConfirm.destructive ? "destructive" : "default"}
            isLoading={isSubmitting}
          />
        ))}
    </>
  )
}
