/**
 * Reusable Form Password Field Component
 * A flexible password field with label, validation, error handling, and show/hide toggle
 */

import { forwardRef, useState } from "react"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Eye, EyeOff } from "lucide-react"
import { cn } from "@/lib/utils"

export interface FormPasswordFieldProps extends Omit<React.ComponentProps<typeof Input>, 'type'> {
  label: string
  error?: string
  description?: string
  required?: boolean
  containerClassName?: string
  labelClassName?: string
  errorClassName?: string
  descriptionClassName?: string
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
    
    // Generate ID if not provided
    const fieldId = id || label.toLowerCase().replace(/\s+/g, '-')

    return (
      <Field className={cn(containerClassName)}>
        <FieldLabel 
          htmlFor={fieldId} 
          className={cn(labelClassName)}
        >
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </FieldLabel>
        
        <div className="relative">
          <Input
            id={fieldId}
            ref={ref}
            type={showPassword ? "text" : "password"}
            autoComplete={autoComplete}
            className={cn(
              error && "border-red-500 focus-visible:ring-red-500",
              showToggle && "pr-10",
              className
            )}
            aria-invalid={error ? "true" : "false"}
            aria-describedby={
              error ? `${fieldId}-error` :
              description ? `${fieldId}-description` :
              undefined
            }
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

FormPasswordField.displayName = "FormPasswordField"
