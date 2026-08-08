import type { ReactNode } from "react"
import type { LucideIcon } from "lucide-react"
import { Inbox, Plus, SearchX } from "lucide-react"
import { Button } from "@/components/ui/button"

interface DataTableEmptyProps {
  /** Headline — short, e.g. "No users yet". */
  title: string
  /** One-line supporting copy. */
  description?: string
  /** Icon shown above the title. Defaults to Inbox / SearchX based on `variant`. */
  icon?: LucideIcon
  /** "empty" = nothing exists yet (offers a create CTA); "no-results" = filters/search hid everything. */
  variant?: "empty" | "no-results"
  /** Primary call-to-action (e.g. create the first row). */
  onAction?: () => void
  actionLabel?: string
  /** Custom action element rendered in the action slot, in place of the built-in button. */
  children?: ReactNode
}

/**
 * The shared table empty state: a centred icon + title + description, with an
 * optional primary action. Used in place of a bare "No results." text so an
 * empty listing reads as intentional rather than broken.
 */
export function DataTableEmpty({
  title,
  description,
  icon,
  variant = "empty",
  onAction,
  actionLabel,
  children,
}: DataTableEmptyProps) {
  const Icon = icon ?? (variant === "no-results" ? SearchX : Inbox)

  return (
    <div className="flex flex-col items-center justify-center gap-3 px-6 py-12 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-muted">
        <Icon className="size-6 text-muted-foreground" />
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium">{title}</p>
        {description && (
          <p className="max-w-sm text-sm text-muted-foreground">{description}</p>
        )}
      </div>
      {children
        ? children
        : onAction && actionLabel && (
            <Button size="sm" className="mt-1" onClick={onAction}>
              <Plus className="size-4" />
              {actionLabel}
            </Button>
          )}
    </div>
  )
}
