/**
 * FormCheckboxSubContainer
 *
 * A themed box that wraps a checkbox option list (e.g. roles or permissions
 * pickers). The box follows its own inputs theme config (checkbox sub-container
 * tokens), so checkbox lists keep a surface distinct from plain inputs.
 *
 * Controlled: pass `selected` (array of values) + `onToggle(value)`; options
 * render through SelectableOptionRow so the whole row is one toggle target.
 */

import { SelectableOptionRow } from "./SelectableOptionRow"
import { cn } from "@/lib/utils"

export interface CheckboxSubContainerOption {
  value: string
  title: string
  description?: string
  mono?: boolean
}

export interface FormCheckboxSubContainerProps {
  options: CheckboxSubContainerOption[]
  selected: string[]
  onToggle: (value: string) => void
  disabled?: boolean
  loading?: boolean
  emptyText?: string
  maxHeightClassName?: string
  containerClassName?: string
  footer?: React.ReactNode
}

export function FormCheckboxSubContainer({
  options,
  selected,
  onToggle,
  disabled,
  loading,
  emptyText = "No options available",
  maxHeightClassName = "max-h-64",
  containerClassName,
  footer,
}: FormCheckboxSubContainerProps) {
  if (loading) {
    return <div className="py-6 text-center text-sm text-muted-foreground">Loading options...</div>
  }

  if (options.length === 0) {
    return <div className="py-6 text-center text-sm text-muted-foreground">{emptyText}</div>
  }

  return (
    <div>
      <div
        data-md-checkbox-sub-container
        className={cn("divide-y overflow-y-auto rounded-md border", maxHeightClassName, containerClassName)}
      >
        {options.map((option) => (
          <SelectableOptionRow
            key={option.value}
            selected={selected.includes(option.value)}
            onToggle={() => onToggle(option.value)}
            disabled={disabled}
            title={option.title}
            description={option.description}
            mono={option.mono}
          />
        ))}
      </div>
      {footer}
    </div>
  )
}

FormCheckboxSubContainer.displayName = "FormCheckboxSubContainer"
