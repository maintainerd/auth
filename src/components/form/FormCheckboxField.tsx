/**
 * Reusable Form Checkbox Field Component
 * A flexible checkbox field with label and error handling
 */

import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { cn } from "@/lib/utils"

export interface FormCheckboxFieldProps {
  label: string
  description?: string
  error?: string
  disabled?: boolean
  checked?: boolean
  onCheckedChange?: (checked: boolean) => void
  containerClassName?: string
  labelClassName?: string
  errorClassName?: string
  descriptionClassName?: string
  id?: string
}

export function FormCheckboxField({
  label,
  description,
  error,
  disabled = false,
  checked,
  onCheckedChange,
  containerClassName,
  labelClassName,
  errorClassName,
  descriptionClassName,
  id,
}: FormCheckboxFieldProps) {
  // Generate ID if not provided
  const fieldId = id || label.toLowerCase().replace(/\s+/g, '-')

  return (
    <Field className={cn(containerClassName)}>
      <div className="flex items-center space-x-2">
        <Checkbox
          id={fieldId}
          checked={checked}
          onCheckedChange={onCheckedChange}
          disabled={disabled}
          aria-invalid={error ? "true" : "false"}
          aria-describedby={
            error ? `${fieldId}-error` :
            description ? `${fieldId}-description` :
            undefined
          }
        />
        <div className="flex flex-col gap-1">
          <FieldLabel
            htmlFor={fieldId}
            className={cn("text-sm font-normal cursor-pointer", labelClassName)}
          >
            {label}
          </FieldLabel>
          {/* Suppressed while an error shows, so the control is never followed
              by two competing lines of helper text. */}
          {description && !error && (
            <FieldDescription id={`${fieldId}-description`} className={cn(descriptionClassName)}>
              {description}
            </FieldDescription>
          )}
        </div>
      </div>

      {error && (
        <FieldError id={`${fieldId}-error`} className={cn(errorClassName)}>
          {error}
        </FieldError>
      )}
    </Field>
  )
}

FormCheckboxField.displayName = "FormCheckboxField"

