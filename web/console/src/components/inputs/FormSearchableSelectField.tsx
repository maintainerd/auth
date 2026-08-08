/**
 * FormSearchableSelectField
 *
 * A reusable combobox: a trigger styled like a select/input (follows the
 * inputs theme config) that opens a searchable command list. Supports both
 * client-side filtering over a static option list and server-side search via
 * controlled `searchValue`/`onSearchChange` (e.g. a clients picker that
 * re-queries as the user types).
 *
 * Controlled: pass `value` + `onValueChange`; `onSelect` fires with the chosen
 * option's value (or the option object when `onSelectOption` is provided).
 */

import { useState } from "react"
import { ChevronsUpDown } from "lucide-react"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import { cn } from "@/lib/utils"

export interface SearchableSelectOption {
  value: string
  label: string
  description?: string
  /** Optional secondary key (e.g. the slug) matched by client-side search. */
  keywords?: string
  disabled?: boolean
}

export interface FormSearchableSelectFieldProps {
  label: string
  value?: string
  onValueChange?: (value: string) => void
  options?: SearchableSelectOption[]
  placeholder?: string
  emptyText?: string
  loading?: boolean
  error?: string
  description?: string
  required?: boolean
  disabled?: boolean
  id?: string
  containerClassName?: string
  /** Controlled search term (server-side search). When omitted, filtering is
   *  client-side over `options`. */
  searchValue?: string
  onSearchChange?: (value: string) => void
  /** Trigger search on open with the current term (server-side sources). */
  onOpenChange?: (open: boolean) => void
  /** Renders the selected value inside the trigger. */
  renderValue?: (option: SearchableSelectOption | undefined) => React.ReactNode
}

export function FormSearchableSelectField({
  label,
  value,
  onValueChange,
  options = [],
  placeholder = "Select an option",
  emptyText = "No results found.",
  loading = false,
  error,
  description,
  required = false,
  disabled = false,
  id,
  containerClassName,
  searchValue,
  onSearchChange,
  onOpenChange,
  renderValue,
}: FormSearchableSelectFieldProps) {
  const fieldId = id || label.toLowerCase().replace(/\s+/g, "-")
  const [open, setOpen] = useState(false)
  const [internalSearch, setInternalSearch] = useState("")
  const [internalValue, setInternalValue] = useState<string | undefined>(undefined)

  const effectiveValue = value ?? internalValue
  const isControlledSearch = onSearchChange !== undefined
  const searchTerm = isControlledSearch ? (searchValue ?? "") : internalSearch

  const selected = options.find((option) => option.value === effectiveValue)

  const filteredOptions = isControlledSearch
    ? options
    : options.filter(
        (option) =>
          option.label.toLowerCase().includes(searchTerm.toLowerCase()) ||
          (option.keywords ?? "").toLowerCase().includes(searchTerm.toLowerCase()) ||
          (option.description ?? "").toLowerCase().includes(searchTerm.toLowerCase()),
      )

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    onOpenChange?.(next)
  }

  const commit = (option: SearchableSelectOption) => {
    if (!option.disabled) {
      setInternalValue(option.value)
      onValueChange?.(option.value)
      setOpen(false)
      if (isControlledSearch) {
        onSearchChange?.("")
      } else {
        setInternalSearch("")
      }
    }
  }

  return (
    <Field className={cn(containerClassName)}>
      <FieldLabel htmlFor={fieldId}>
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </FieldLabel>

      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            id={fieldId}
            variant="outline"
            role="combobox"
            data-md-input-button
            aria-expanded={open}
            aria-invalid={error ? "true" : "false"}
            aria-describedby={
              error ? `${fieldId}-error` : description ? `${fieldId}-description` : undefined
            }
            disabled={disabled}
            className={cn(
              "w-full justify-between",
              error && "border-red-500 focus-visible:ring-red-500/20",
            )}
          >
            <span className={cn("truncate", !selected && "text-muted-foreground")}>
              {renderValue
                ? renderValue(selected)
                : selected
                  ? selected.label
                  : placeholder}
            </span>
            <ChevronsUpDown className="size-4 shrink-0 opacity-60" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-full min-w-[var(--radix-popover-trigger-width)] p-0" align="start">
          <Command shouldFilter={false}>
            <CommandInput
              placeholder="Search..."
              value={searchTerm}
              onValueChange={(next) => {
                if (isControlledSearch) onSearchChange?.(next)
                else setInternalSearch(next)
              }}
            />
            <CommandList>
              {filteredOptions.length === 0 && (
                <CommandEmpty>{loading ? "Loading..." : emptyText}</CommandEmpty>
              )}
              <CommandGroup>
                {filteredOptions.map((option) => (
                  <CommandItem
                    key={option.value}
                    value={option.label}
                    disabled={option.disabled}
                    onSelect={() => commit(option)}
                    className="gap-2"
                  >
                    <div className="min-w-0">
                      <span className="block truncate font-medium">{option.label}</span>
                      {option.description && (
                        <span
                          aria-hidden
                          className="block truncate text-xs text-muted-foreground"
                        >
                          {option.description}
                        </span>
                      )}
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>

      {description && !error && (
        <FieldDescription id={`${fieldId}-description`} className="text-muted-foreground">
          {description}
        </FieldDescription>
      )}
      {error && <FieldError id={`${fieldId}-error`}>{error}</FieldError>}
    </Field>
  )
}

FormSearchableSelectField.displayName = "FormSearchableSelectField"
