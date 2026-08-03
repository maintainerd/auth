/**
 * Reusable Form Select Field Component
 * A flexible select field with label, validation, and error handling.
 *
 * Layout, error rendering and aria wiring all come from FieldShell — this
 * component only supplies the control.
 */

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { cn } from "@/lib/utils"
import { FieldShell } from "./FieldShell"
import {
  fieldControlProps,
  resolveFieldId,
  FIELD_INVALID_CONTROL_CLASS,
  type FieldShellOwnProps,
} from "./fieldControl"

export interface SelectOption {
  value: string
  label: string
  disabled?: boolean
}

export interface FormSelectFieldProps extends FieldShellOwnProps {
  placeholder?: string
  options: SelectOption[]
  value?: string
  onValueChange?: (value: string) => void
  disabled?: boolean
  className?: string
  id?: string
}

export function FormSelectField({
  label,
  placeholder = "Select an option",
  options,
  value,
  onValueChange,
  error,
  description,
  required = false,
  disabled = false,
  containerClassName,
  labelClassName,
  errorClassName,
  descriptionClassName,
  labelAction,
  footer,
  className,
  id,
}: FormSelectFieldProps) {
  const fieldId = resolveFieldId(id, label)

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
      <Select
        key={value}
        value={value || undefined}
        onValueChange={onValueChange}
        disabled={disabled}
      >
        <SelectTrigger
          className={cn("w-full", error && FIELD_INVALID_CONTROL_CLASS, className)}
          {...fieldControlProps(fieldId, error, description)}
        >
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem
              key={option.value}
              value={option.value}
              disabled={option.disabled}
            >
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </FieldShell>
  )
}

FormSelectField.displayName = "FormSelectField"
