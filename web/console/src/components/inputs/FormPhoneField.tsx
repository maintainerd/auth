import { forwardRef, useCallback } from "react"
import { FormInputField, type FormInputFieldProps } from "@/components/form"
import { formatPhone } from "@/lib/validations/regex"

export type FormPhoneFieldProps = Omit<FormInputFieldProps, 'type'>

export const FormPhoneField = forwardRef<HTMLInputElement, FormPhoneFieldProps>(
  ({ onChange, ...props }, ref) => {
    const handleChange = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        const formatted = formatPhone(e.target.value)
        if (formatted !== e.target.value) {
          e.target.value = formatted
        }
        onChange?.(e)
      },
      [onChange]
    )

    return (
      <FormInputField
        ref={ref}
        type="tel"
        autoComplete="tel"
        inputMode="numeric"
        placeholder="e.g., +1 555 123 4567"
        onChange={handleChange}
        {...props}
      />
    )
  }
)

FormPhoneField.displayName = "FormPhoneField"
