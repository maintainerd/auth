/**
 * Reusable Form Textarea Field Component
 * A flexible textarea field with label, validation, and error handling.
 *
 * Layout, error rendering and aria wiring all come from FieldShell — this
 * component only supplies the control.
 */

import { forwardRef } from "react"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"
import { FieldShell } from "./FieldShell"
import {
  fieldControlProps,
  resolveFieldId,
  FIELD_INVALID_CONTROL_CLASS,
  type FieldShellOwnProps,
} from "./fieldControl"

export interface FormTextareaFieldProps
  extends React.ComponentProps<typeof Textarea>,
    FieldShellOwnProps {}

export const FormTextareaField = forwardRef<HTMLTextAreaElement, FormTextareaFieldProps>(
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
        <Textarea
          ref={ref}
          className={cn(error && FIELD_INVALID_CONTROL_CLASS, className)}
          {...fieldControlProps(fieldId, error, description)}
          {...props}
        />
      </FieldShell>
    )
  }
)

FormTextareaField.displayName = "FormTextareaField"
