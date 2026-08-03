import { forwardRef } from 'react'
import { FormInputField, type FormInputFieldProps } from '@/components/form'
import { cn } from '@/lib/utils'

export interface FormCodeFieldProps extends Omit<FormInputFieldProps, 'inputMode'> {
  numeric?: boolean
}

export const FormCodeField = forwardRef<HTMLInputElement, FormCodeFieldProps>(
  ({ className, numeric = false, ...props }, ref) => {
    return (
      <FormInputField
        ref={ref}
        inputMode={numeric ? 'numeric' : 'text'}
        autoComplete="one-time-code"
        className={cn(
          'font-mono',
          numeric && 'text-center tracking-[0.4em]',
          className,
        )}
        {...props}
      />
    )
  },
)

FormCodeField.displayName = 'FormCodeField'
