/**
 * Reusable Form Date Field Component
 * A beautiful date picker with calendar popover using shadcn components.
 *
 * Layout, error rendering and aria wiring all come from FieldShell — this
 * component only supplies the trigger, its calendar and the hidden input.
 */

import { forwardRef, useState } from "react"
import { format } from "date-fns"
import { ChevronDownIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { cn } from "@/lib/utils"
import { FieldShell } from "./FieldShell"
import {
  fieldControlProps,
  resolveFieldId,
  FIELD_INVALID_CONTROL_CLASS,
  type FieldShellOwnProps,
} from "./fieldControl"

export interface FormDateFieldProps extends FieldShellOwnProps {
  value?: string
  onChange?: (value: string) => void
  placeholder?: string
  disabled?: boolean
  className?: string
  id?: string
  name?: string
}

export const FormDateField = forwardRef<HTMLButtonElement, FormDateFieldProps>(
  (
    {
      label,
      value,
      onChange,
      error,
      description,
      required = false,
      placeholder = "Pick a date",
      disabled = false,
      containerClassName,
      labelClassName,
      errorClassName,
      descriptionClassName,
      labelAction,
      footer,
      className,
      id,
      name,
      ...props
    },
    ref
  ) => {
    const [open, setOpen] = useState(false)

    const fieldId = resolveFieldId(id, label)

    // Convert string value to Date object
    const selectedDate = value ? new Date(value) : undefined

    // Handle date selection
    const handleDateSelect = (date: Date | undefined) => {
      if (date) {
        // Format date as YYYY-MM-DD for form compatibility
        const formattedDate = format(date, 'yyyy-MM-dd')
        onChange?.(formattedDate)
      } else {
        onChange?.('')
      }
      setOpen(false)
    }

    return (
      <FieldShell
        fieldId={fieldId}
        label={label}
        error={error}
        description={description}
        required={required}
        containerClassName={containerClassName}
        labelClassName={labelClassName}
        errorClassName={errorClassName}
        descriptionClassName={descriptionClassName}
        labelAction={labelAction}
        footer={footer}
      >
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              data-md-date-picker-trigger
              ref={ref}
              variant="outline"
              disabled={disabled}
              className={cn(
                "w-full justify-between font-normal",
                !selectedDate && "text-muted-foreground",
                error && FIELD_INVALID_CONTROL_CLASS,
                className
              )}
              {...fieldControlProps(fieldId, error, description)}
              {...props}
            >
              {selectedDate ? (
                selectedDate.toLocaleDateString()
              ) : (
                <span>{placeholder}</span>
              )}
              <ChevronDownIcon className="h-4 w-4" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-auto overflow-hidden p-0" align="start">
            <Calendar
              mode="single"
              selected={selectedDate}
              onSelect={handleDateSelect}
              captionLayout="dropdown"
              disabled={(date) => {
                // Disable future dates (birth date validation)
                const today = new Date()
                today.setHours(23, 59, 59, 999)
                if (date > today) return true

                // Disable dates more than 150 years ago
                const maxAge = 150
                const minDate = new Date(today.getFullYear() - maxAge, today.getMonth(), today.getDate())
                if (date < minDate) return true

                return false
              }}
              autoFocus
            />
          </PopoverContent>
        </Popover>

        {/* Hidden input for form compatibility */}
        <input
          type="hidden"
          name={name}
          value={value || ''}
        />
      </FieldShell>
    )
  }
)

FormDateField.displayName = "FormDateField"
