import { forwardRef } from 'react'
import { FormInputField, type FormInputFieldProps } from '@/components/form'

export type FormPhoneFieldProps = Omit<FormInputFieldProps, 'type' | 'inputMode' | 'autoComplete'>

export const FormPhoneField = forwardRef<HTMLInputElement, FormPhoneFieldProps>((props, ref) => {
  return (
    <FormInputField
      ref={ref}
      type="tel"
      autoComplete="tel"
      inputMode="tel"
      placeholder="+1 212 555 1234"
      {...props}
    />
  )
})

FormPhoneField.displayName = 'FormPhoneField'
