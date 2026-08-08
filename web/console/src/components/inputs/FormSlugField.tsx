import { forwardRef, useCallback } from "react"
import { FormInputField, type FormInputFieldProps } from "@/components/form"
import { sanitizeSlug } from "@/lib/validations/regex"

export type FormSlugFieldProps = Omit<FormInputFieldProps, 'type'> & {
  /**
   * Live input normalizer. Defaults to `sanitizeSlug` (allows the `:` used in
   * namespaced role names). Pass a stricter one (e.g. `sanitizeName`) for slugs
   * that must not contain colons, so the field can't produce a value the
   * validation would reject.
   */
  sanitize?: (raw: string) => string
}

export const FormSlugField = forwardRef<HTMLInputElement, FormSlugFieldProps>(
  ({ onChange, sanitize = sanitizeSlug, ...props }, ref) => {
    const handleChange = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        const sanitized = sanitize(e.target.value)
        if (sanitized !== e.target.value) {
          e.target.value = sanitized
        }
        onChange?.(e)
      },
      [onChange, sanitize]
    )

    return (
      <FormInputField
        ref={ref}
        type="text"
        autoComplete="off"
        onChange={handleChange}
        {...props}
      />
    )
  }
)

FormSlugField.displayName = "FormSlugField"
