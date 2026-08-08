import { forwardRef } from 'react'
import { FormInputField, type FormInputFieldProps } from '@/components/form'

export type FormEmailFieldProps = Omit<FormInputFieldProps, 'type'>

export const FormEmailField = forwardRef<HTMLInputElement, FormEmailFieldProps>((props, ref) => {
  return (
    <FormInputField
      ref={ref}
      type="email"
      autoComplete="email"
      inputMode="email"
      placeholder="you@company.com"
      {...props}
    />
  )
})

FormEmailField.displayName = 'FormEmailField'
