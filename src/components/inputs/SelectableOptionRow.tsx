import { Check } from "lucide-react"
import { cn } from "@/lib/utils"

export interface SelectableOptionRowProps {
  selected: boolean
  onToggle: () => void
  title: string
  description?: string
  disabled?: boolean
  /** Render the title in a monospace font (e.g. for URIs). */
  mono?: boolean
}

/**
 * A single selectable row for checkbox-list pickers: the whole row is one
 * interactive, keyboard-accessible control that toggles the option. The
 * check indicator is purely presentational (no nested form control), which
 * avoids event/double-fire issues — including inside a <form>.
 */
export function SelectableOptionRow({
  selected,
  onToggle,
  title,
  description,
  disabled,
  mono,
}: SelectableOptionRowProps) {
  return (
    <div
      role="checkbox"
      aria-checked={selected}
      aria-disabled={disabled || undefined}
      tabIndex={disabled ? -1 : 0}
      onClick={() => {
        if (!disabled) onToggle()
      }}
      onKeyDown={(e) => {
        if (!disabled && (e.key === "Enter" || e.key === " ")) {
          e.preventDefault()
          onToggle()
        }
      }}
      className={cn(
        "flex items-start gap-3 p-3 outline-none transition-colors",
        disabled
          ? "cursor-not-allowed opacity-60"
          : "cursor-pointer hover:bg-accent/50 focus-visible:bg-accent/50",
      )}
    >
      <span
        aria-hidden
        className={cn(
          "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-[3px] border transition-colors",
          selected ? "border-primary bg-primary text-primary-foreground" : "border-input",
        )}
      >
        {selected && <Check className="size-3.5" />}
      </span>
      <div className="min-w-0 space-y-0.5">
        <p className={cn("text-sm font-medium", mono ? "break-all font-mono" : "break-words")}>
          {title}
        </p>
        {description && (
          <p className="break-words text-xs text-muted-foreground">{description}</p>
        )}
      </div>
    </div>
  )
}
