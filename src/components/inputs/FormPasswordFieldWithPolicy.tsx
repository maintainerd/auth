import { forwardRef, useState } from 'react'
import type { PasswordConfigPublic } from '@/services/api/tenants/types'
import { FormPasswordField, PasswordRequirements, type FormPasswordFieldProps } from '@/components/form'

export interface FormPasswordFieldWithPolicyProps extends FormPasswordFieldProps {
  passwordConfig?: PasswordConfigPublic
}

export const FormPasswordFieldWithPolicy = forwardRef<HTMLInputElement, FormPasswordFieldWithPolicyProps>(
  ({ passwordConfig, onChange, ...props }, ref) => {
    const [password, setPassword] = useState('')

    return (
      // The checklist rides in the field's own footer slot rather than a
      // wrapper div, so it inherits the same gap as the label and input
      // instead of introducing a third spacing value.
      <FormPasswordField
        ref={ref}
        {...props}
        footer={
          password.length > 0 ? (
            <PasswordRequirements password={password} config={passwordConfig} />
          ) : undefined
        }
        onChange={(event) => {
          setPassword(event.target.value)
          onChange?.(event)
        }}
      />
    )
  },
)

FormPasswordFieldWithPolicy.displayName = 'FormPasswordFieldWithPolicy'
