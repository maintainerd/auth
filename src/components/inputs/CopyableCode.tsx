import { Copy } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useToast } from "@/hooks/useToast"
import { cn } from "@/lib/utils"

interface CopyableCodeProps {
  /** The exact text placed on the clipboard. */
  value: string
  /**
   * What the value is, in sentence case — used for both the screen-reader action
   * ("Copy registration link") and the confirmation toast ("Registration link
   * copied to clipboard"), so the two never drift apart.
   */
  label: string
  /**
   * `inline` fits a table cell: single line, truncated, small control.
   * `block` fits a card: wraps on any character so a long URL stays readable.
   */
  variant?: "inline" | "block"
  /**
   * Set when the surrounding element is itself clickable (a navigating table
   * row), so copying does not also trigger it.
   */
  stopPropagation?: boolean
  className?: string
}

/**
 * A machine-readable value rendered as code with a copy control.
 *
 * Extracted because the same clipboard-plus-`<code>` pairing was being rebuilt
 * per feature — each copy re-deciding the toast wording, the button size, and
 * whether to guard the click. Values like an OAuth client id, a webhook secret or
 * a registration link exist to be copied, so the affordance belongs to the value
 * rather than to the page.
 */
export function CopyableCode({
  value,
  label,
  variant = "inline",
  stopPropagation = false,
  className,
}: CopyableCodeProps) {
  const { showSuccess, showError } = useToast()

  const copy = async (event: React.MouseEvent) => {
    if (stopPropagation) event.stopPropagation()
    try {
      await navigator.clipboard.writeText(value)
      showSuccess(`${label} copied to clipboard`)
    } catch (error) {
      // Clipboard access can be denied (insecure context, permissions policy).
      // Surface it rather than silently appearing to succeed.
      showError(error)
    }
  }

  const isBlock = variant === "block"

  return (
    <div className={cn("flex min-w-0 items-center gap-2", className)}>
      <code
        className={cn(
          "min-w-0 rounded bg-muted font-mono",
          isBlock ? "flex-1 break-all px-2 py-1.5 text-sm" : "truncate px-2 py-1 text-sm",
        )}
      >
        {value}
      </code>
      <Button
        type="button"
        variant={isBlock ? "outline" : "ghost"}
        size="sm"
        className={cn("shrink-0 p-0", isBlock ? "h-8 w-8" : "h-7 w-7")}
        onClick={copy}
      >
        <span className="sr-only">Copy {label.toLowerCase()}</span>
        <Copy className={isBlock ? "size-4" : "size-3.5"} />
      </Button>
    </div>
  )
}
