/**
 * The control half of the field standard — the bits a field's *input* needs,
 * kept out of FieldShell.tsx so that file exports only a component (react-refresh
 * requires it, and it keeps the scaffold's markup separate from its contract).
 */
import type { ReactNode } from "react"

export interface FieldShellOwnProps {
  label: string
  error?: string
  description?: string
  required?: boolean
  containerClassName?: string
  labelClassName?: string
  errorClassName?: string
  descriptionClassName?: string
  /**
   * Control rendered at the far end of the label row — the conventional home
   * for a "Forgot password?" link, which belongs beside the field it refers to.
   */
  labelAction?: ReactNode
  /**
   * Extra content rendered as the field's last row (e.g. a password-policy
   * checklist). Belongs here rather than in a wrapper around the field, so it
   * inherits the same gap as the label and control instead of inventing one.
   */
  footer?: ReactNode
}

/** Derive a stable field id from an explicit id or the label text. */
export function resolveFieldId(id: string | undefined, label: string): string {
  return id || label.toLowerCase().replace(/\s+/g, "-")
}

/**
 * The id + aria attributes a control must spread so assistive tech can reach
 * its error and description. Returned as one object so a field cannot wire up
 * half of it and silently ship an unlabelled invalid state.
 */
export function fieldControlProps(
  fieldId: string,
  error?: string,
  description?: string,
): {
  id: string
  "aria-invalid": "true" | "false"
  "aria-describedby": string | undefined
} {
  return {
    id: fieldId,
    "aria-invalid": error ? "true" : "false",
    "aria-describedby": error
      ? `${fieldId}-error`
      : description
        ? `${fieldId}-description`
        : undefined,
  }
}

/** The class a control adds while invalid. One definition, every field. */
export const FIELD_INVALID_CONTROL_CLASS = "border-destructive focus-visible:ring-destructive/20"
