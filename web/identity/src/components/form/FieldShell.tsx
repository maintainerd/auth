/**
 * The one field scaffold.
 *
 * Every form field is a label, a control, and an optional description or error.
 * Before this existed each Form*Field re-implemented that scaffolding by hand,
 * so they drifted: different label→input spacing, three different error colours,
 * some wiring `aria-describedby` and some not. FieldShell owns the arrangement
 * once; a field component supplies only its control.
 *
 * The standard it encodes:
 *  - Spacing comes solely from <Field> (`flex flex-col gap-3`). Nothing here adds
 *    a `space-y-*`/`mt-*`, because those STACK on the flex gap and knock one
 *    field's label out of line with its neighbours'.
 *  - Errors render through <FieldError> — `text-destructive text-sm`, `role="alert"`.
 *    Never a literal `text-red-*`, which does not adapt to a branded theme.
 *  - A description is suppressed while an error is showing, so the control is
 *    never followed by two competing lines of helper text.
 *
 * The control-side contract (`fieldControlProps`, `FIELD_INVALID_CONTROL_CLASS`)
 * lives in ./fieldControl.
 */
import type { ReactNode } from "react"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { cn } from "@/lib/utils"
import type { FieldShellOwnProps } from "./fieldControl"

interface Props extends FieldShellOwnProps {
  fieldId: string
  children: ReactNode
}

export function FieldShell({
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
  fieldId,
  children,
}: Props) {
  const fieldLabel = (
    <FieldLabel htmlFor={fieldId} className={cn(labelClassName)}>
      {label}
      {required && <span className="text-destructive ml-1">*</span>}
    </FieldLabel>
  )

  return (
    <Field className={cn(containerClassName)}>
      {labelAction ? (
        <div className="flex items-center justify-between gap-2">
          {fieldLabel}
          {labelAction}
        </div>
      ) : (
        fieldLabel
      )}

      {children}

      {description && !error && (
        <FieldDescription
          id={`${fieldId}-description`}
          className={cn(descriptionClassName)}
        >
          {description}
        </FieldDescription>
      )}

      {error && (
        <FieldError id={`${fieldId}-error`} className={cn(errorClassName)}>
          {error}
        </FieldError>
      )}

      {footer}
    </Field>
  )
}
