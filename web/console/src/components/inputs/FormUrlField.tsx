import { forwardRef } from "react"
import { FormInputField, type FormInputFieldProps } from "@/components/form"

export type FormUrlFieldProps = Omit<FormInputFieldProps, 'type'>

export const FormUrlField = forwardRef<HTMLInputElement, FormUrlFieldProps>(
  (props, ref) => {
    return (
      <FormInputField
        ref={ref}
        type="url"
        autoComplete="url"
        placeholder="https://example.com"
        {...props}
      />
    )
  }
)

FormUrlField.displayName = "FormUrlField"
