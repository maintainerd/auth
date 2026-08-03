/**
 * Reusable Form Date Field Component
 * A beautiful date picker with calendar popover using shadcn components
 */

import { forwardRef, useState } from "react"
import { format } from "date-fns"
import { ChevronDownIcon } from "lucide-react"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { cn } from "@/lib/utils"

export interface FormDateFieldProps {
  label: string
  value?: string
  onChange?: (value: string) => void
  error?: string
  description?: string
  required?: boolean
  placeholder?: string
  disabled?: boolean
  containerClassName?: string
  labelClassName?: string
  errorClassName?: string
  descriptionClassName?: string
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
      className,
      id,
      name,
      ...props
    },
    ref
  ) => {
    const [open, setOpen] = useState(false)

    // Generate ID if not provided
    const fieldId = id || label.toLowerCase().replace(/\s+/g, '-')

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
      <Field className={cn(containerClassName)}>
        <FieldLabel
          htmlFor={fieldId}
          className={cn(labelClassName)}
        >
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </FieldLabel>

        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              data-md-date-picker-trigger
              id={fieldId}
              ref={ref}
              variant="outline"
              disabled={disabled}
              aria-invalid={error ? "true" : "false"}
              aria-describedby={
                error ? `${fieldId}-error` :
                description ? `${fieldId}-description` :
                undefined
              }
              className={cn(
                "w-full justify-between font-normal",
                !selectedDate && "text-muted-foreground",
                error && "border-red-500 focus-visible:ring-red-500",
                className
              )}
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

        {description && (
          <FieldDescription id={`${fieldId}-description`} className={cn(descriptionClassName)}>
            {description}
          </FieldDescription>
        )}

        {error && (
          <FieldError id={`${fieldId}-error`} className={cn("text-red-600", errorClassName)}>
            {error}
          </FieldError>
        )}
      </Field>
    )
  }
)

FormDateField.displayName = "FormDateField"
