/**
 * Reusable Form Input Field Component
 * A flexible input field with label, validation, and error handling.
 *
 * Layout, error rendering and aria wiring all come from FieldShell — this
 * component only supplies the control.
 */

import { forwardRef } from "react"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import { FieldShell } from "./FieldShell"
import {
  fieldControlProps,
  resolveFieldId,
  FIELD_INVALID_CONTROL_CLASS,
  type FieldShellOwnProps,
} from "./fieldControl"

export interface FormInputFieldProps
  extends React.ComponentProps<typeof Input>,
    FieldShellOwnProps {}

export const FormInputField = forwardRef<HTMLInputElement, FormInputFieldProps>(
  (
    {
      label,
      error,
      description,
      required = false,
      containerClassName,
      labelClassName,
      errorClassName,
      descriptionClassName,
      labelAction,
      footer,
      className,
      id,
      ...props
    },
    ref
  ) => {
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
        <Input
          ref={ref}
          className={cn(error && FIELD_INVALID_CONTROL_CLASS, className)}
          {...fieldControlProps(fieldId, error, description)}
          {...props}
        />
      </FieldShell>
    )
  }
)

FormInputField.displayName = "FormInputField"
