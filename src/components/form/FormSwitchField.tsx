/**
 * Reusable Form Switch Field Component
 * A flexible switch field with label, description, and error handling
 */

import { Switch } from "@/components/ui/switch"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { cn } from "@/lib/utils"

export interface FormSwitchFieldProps {
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
  switchClassName?: string
  id?: string
  layout?: "horizontal" | "vertical"
}

export function FormSwitchField({
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
  switchClassName,
  id,
  layout = "horizontal",
}: FormSwitchFieldProps) {
  // Generate ID if not provided
  const fieldId = id || label.toLowerCase().replace(/\s+/g, '-')

  return (
    <Field className={cn(containerClassName)}>
      <div className={cn(
        "flex items-center",
        layout === "horizontal" ? "justify-between" : "flex-col items-start gap-2"
      )}>
        <div className="space-y-0.5 flex-1">
          <FieldLabel
            htmlFor={fieldId}
            className={cn("text-base", labelClassName)}
          >
            {label}
          </FieldLabel>
          {/* Suppressed while an error shows, so the control is never followed
              by two competing lines of helper text. */}
          {description && !error && (
            <FieldDescription id={`${fieldId}-description`} className={cn("text-sm", descriptionClassName)}>
              {description}
            </FieldDescription>
          )}
        </div>
        <Switch
          id={fieldId}
          checked={checked}
          onCheckedChange={onCheckedChange}
          disabled={disabled}
          className={switchClassName}
          aria-invalid={error ? "true" : "false"}
          aria-describedby={
            error ? `${fieldId}-error` :
            description ? `${fieldId}-description` :
            undefined
          }
        />
      </div>

      {error && (
        <FieldError id={`${fieldId}-error`} className={cn(errorClassName)}>
          {error}
        </FieldError>
      )}
    </Field>
  )
}

FormSwitchField.displayName = "FormSwitchField"
