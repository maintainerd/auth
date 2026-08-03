/**
 * Reusable Form Password Field Component
 * A flexible password field with label, validation, error handling, and show/hide toggle.
 *
 * Layout, error rendering and aria wiring all come from FieldShell — this
 * component only supplies the control and its visibility toggle.
 */

import { forwardRef, useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Eye, EyeOff } from "lucide-react"
import { cn } from "@/lib/utils"
import { FieldShell } from "./FieldShell"
import {
  fieldControlProps,
  resolveFieldId,
  FIELD_INVALID_CONTROL_CLASS,
  type FieldShellOwnProps,
} from "./fieldControl"

export interface FormPasswordFieldProps
  extends Omit<React.ComponentProps<typeof Input>, 'type'>,
    FieldShellOwnProps {
  showToggle?: boolean
}

export const FormPasswordField = forwardRef<HTMLInputElement, FormPasswordFieldProps>(
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
      showToggle = true,
      id,
      // Default to "new-password" so browsers (Chrome/Firefox) do NOT autofill
      // these fields with a saved login the way they do for a normal password
      // input. Every password field in this admin console is a secret-entry
      // field (client secrets, SMTP/SMS credentials, a user's new password) —
      // never a login — so unsolicited autofill is always unwanted. A caller can
      // still override (e.g. autoComplete="current-password" for a real login).
      autoComplete = "new-password",
      ...props
    },
    ref
  ) => {
    const [showPassword, setShowPassword] = useState(false)

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
        <div className="relative">
          <Input
            ref={ref}
            type={showPassword ? "text" : "password"}
            autoComplete={autoComplete}
            className={cn(
              error && FIELD_INVALID_CONTROL_CLASS,
              showToggle && "pr-10",
              className
            )}
            {...fieldControlProps(fieldId, error, description)}
            {...props}
          />

          {showToggle && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
              onClick={() => setShowPassword(!showPassword)}
            >
              {showPassword ? (
                <EyeOff className="h-4 w-4 text-muted-foreground" />
              ) : (
                <Eye className="h-4 w-4 text-muted-foreground" />
              )}
              <span className="sr-only">
                {showPassword ? "Hide password" : "Show password"}
              </span>
            </Button>
          )}
        </div>
      </FieldShell>
    )
  }
)

FormPasswordField.displayName = "FormPasswordField"
